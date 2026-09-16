// Package gemini provides a Gemini AI adapter implementing the AIProvider port.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"kawan-threads/internal/application/runtimeconfig"
	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/port"
	"kawan-threads/internal/infrastructure/gemini/prompts"
)

// truncate limits a string for log/error inclusion.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

const geminiAPIBase = "https://generativelanguage.googleapis.com/v1beta/models"

// GeminiAdapter implements port.AIProvider using the Gemini REST API.
// Credentials and model are read from the runtime settings store so they can
// be changed through the UI without restarting the process.
type GeminiAdapter struct {
	settings   *runtimeconfig.Store
	httpClient *http.Client
	logger     *slog.Logger
	maxRetries int
}

// NewGeminiAdapter constructs a GeminiAdapter backed by the runtime settings.
func NewGeminiAdapter(settings *runtimeconfig.Store, logger *slog.Logger) *GeminiAdapter {
	return &GeminiAdapter{
		settings: settings,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger:     logger,
		maxRetries: 3,
	}
}

// GenerateContent implements port.AIProvider.
func (g *GeminiAdapter) GenerateContent(ctx context.Context, req port.GenerateContentRequest) (port.GeneratedContent, error) {
	apiKey := g.settings.Str(runtimeconfig.GeminiAPIKey, "")
	if apiKey == "" {
		return port.GeneratedContent{}, errors.New("gemini: API key not configured — set it in Settings (AI section)")
	}
	model := g.settings.Str(runtimeconfig.GeminiModel, "gemini-3.6-flash")

	prompt := g.buildPrompt(req)

	// On a failed attempt, nudge the model to return ONLY the JSON object.
	const correction = "\n\nCATATAN PENTING: Jawaban sebelumnya tidak valid." +
		" Balas dengan SATU objeck JSON valid saja (tanpa teks lain, tanpa markdown, tanpa kalimat pengantar)."

	var (
		result      port.GeneratedContent
		lastErr     error
		retryPrompt = prompt
	)

	backoffs := []time.Duration{1 * time.Second, 3 * time.Second, 9 * time.Second}

	for attempt := 0; attempt < g.maxRetries; attempt++ {
		if attempt > 0 {
			wait := backoffs[attempt-1]
			g.logger.Warn("gemini: retrying generation",
				"attempt", attempt,
				"backoff", wait,
				"last_error", lastErr,
			)
			select {
			case <-ctx.Done():
				return port.GeneratedContent{}, ctx.Err()
			case <-time.After(wait):
			}
		}

		raw, err := g.callAPI(ctx, retryPrompt, model, apiKey)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: API call failed: %w", attempt+1, err)
			// Quota and auth errors are terminal — retrying won't help.
			if IsQuotaError(err) || IsAuthError(err) {
				return port.GeneratedContent{}, lastErr
			}
			retryPrompt = prompt + correction
			continue
		}

		result, err = parseGeneratedContent(raw)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: parse failed: %w", attempt+1, err)
			retryPrompt = prompt + correction
			continue
		}

		if err := ValidateGeneratedContent(result); err != nil {
			lastErr = fmt.Errorf("attempt %d: validation failed: %w", attempt+1, err)
			continue
		}

		// Success — tag model and prompt version
		result.Model = model
		result.PromptVersion = promptVersion(req.Pillar)
		return result, nil
	}

	return port.GeneratedContent{}, fmt.Errorf("gemini: max retries (%d) exceeded: %w", g.maxRetries, lastErr)
}

// buildPrompt selects the pillar-appropriate prompt.
func (g *GeminiAdapter) buildPrompt(req port.GenerateContentRequest) string {
	topic := req.Topic
	audience := req.Audience
	tone := req.Tone
	cta := req.CTA

	if req.Format == entity.FormatThreadSeries {
		return prompts.ThreadSeriesPrompt(topic, audience, tone, cta)
	}

	switch req.Pillar {
	case entity.PillarCommunity:
		return prompts.CommunityPrompt(topic, audience, tone, cta)
	case entity.PillarEdukasiHukum, entity.PillarTipsHukum, entity.PillarMythVsFact, entity.PillarCeritaWarga:
		return prompts.EducationPrompt(topic, audience, tone, cta)
	case entity.PillarEngagement:
		return prompts.EngagementPrompt(topic, audience, tone, cta)
	case entity.PillarEvent, entity.PillarTraining:
		return prompts.EventPrompt(topic, audience, tone, cta)
	case entity.PillarMembership:
		return prompts.MembershipPrompt(topic, audience, tone, cta)
	default:
		return prompts.BuildPrompt(string(req.Pillar), topic, audience, tone, string(req.Format), cta)
	}
}

// ---------------------------------------------------------------------------
// Gemini REST API
// ---------------------------------------------------------------------------

type geminiRequest struct {
	Contents         []geminiContent  `json:"contents"`
	GenerationConfig generationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type generationConfig struct {
	Temperature      float64 `json:"temperature"`
	TopK             int     `json:"topK"`
	TopP             float64 `json:"topP"`
	MaxOutputTokens  int     `json:"maxOutputTokens"`
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (g *GeminiAdapter) callAPI(ctx context.Context, prompt, model, apiKey string) (string, error) {
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: generationConfig{
			Temperature:      0.8,
			TopK:             40,
			TopP:             0.95,
			MaxOutputTokens:  2048,
			ResponseMimeType: "application/json",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiAPIBase, model, apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", parseGeminiError(resp.StatusCode, respBody)
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	return gemResp.Candidates[0].Content.Parts[0].Text, nil
}

// ---------------------------------------------------------------------------
// Parsing
// ---------------------------------------------------------------------------

// geminiOutputJSON maps the JSON structure Gemini returns.
type geminiOutputJSON struct {
	ContentType  string `json:"contentType"`
	Hook         string `json:"hook"`
	HookVariants []struct {
		Hook     string `json:"hook"`
		HookType string `json:"hookType"`
	} `json:"hookVariants"`
	Body                 string `json:"body"`
	CTA                  string `json:"cta"`
	Topic                string `json:"topic"`
	HookType             string `json:"hookType"`
	ConversationQuestion string `json:"conversationQuestion"`
	ConversationScore    int    `json:"conversationScore"`
	Quality              struct {
		Hook         int `json:"hook"`
		Conversation int `json:"conversation"`
		Readability  int `json:"readability"`
		Originality  int `json:"originality"`
		Relevance    int `json:"relevance"`
		Shareability int `json:"shareability"`
	} `json:"quality"`
}

// extractJSONObject returns the first balanced JSON object, scanning for
// braces while respecting strings and escapes — so a literal `}` inside e.g. a
// CTA or hook no longer truncates the payload.
func extractJSONObject(text string) string {
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		ch := text[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case ch == '\\':
				escaped = true
			case ch == '"':
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}
	return ""
}

func parseGeneratedContent(raw string) (port.GeneratedContent, error) {
	// Strip markdown code fences if present.
	text := strings.TrimSpace(raw)
	if idx := strings.Index(text, "```json"); idx != -1 {
		text = text[idx+7:]
		if end := strings.LastIndex(text, "```"); end != -1 {
			text = text[:end]
		}
	} else if idx := strings.Index(text, "```"); idx != -1 {
		text = text[idx+3:]
		if end := strings.LastIndex(text, "```"); end != -1 {
			text = text[:end]
		}
	}
	// Extract the first balanced JSON object (string/escape aware).
	snippet := extractJSONObject(text)
	if snippet == "" {
		return port.GeneratedContent{}, fmt.Errorf("no JSON object found in response (got %q)", truncate(text, 300))
	}

	var out geminiOutputJSON
	if err := json.Unmarshal([]byte(snippet), &out); err != nil {
		return port.GeneratedContent{}, fmt.Errorf("JSON unmarshal: %w", err)
	}

	variants := make([]port.HookVariant, 0, len(out.HookVariants))
	for _, v := range out.HookVariants {
		variants = append(variants, port.HookVariant{
			Hook:     v.Hook,
			HookType: entity.HookType(v.HookType),
		})
	}

	return port.GeneratedContent{
		ContentType:          out.ContentType,
		Hook:                 out.Hook,
		HookVariants:         variants,
		Body:                 out.Body,
		CTA:                  out.CTA,
		Topic:                out.Topic,
		HookType:             entity.HookType(out.HookType),
		ConversationQuestion: out.ConversationQuestion,
		ConversationScore:    out.ConversationScore,
		Quality: entity.QualityScore{
			Hook:         out.Quality.Hook,
			Conversation: out.Quality.Conversation,
			Readability:  out.Quality.Readability,
			Originality:  out.Quality.Originality,
			Relevance:    out.Quality.Relevance,
			Shareability: out.Quality.Shareability,
		},
	}, nil
}

// promptVersion returns a version tag for audit/history tracking.
func promptVersion(pillar entity.ContentPillar) string {
	return fmt.Sprintf("%s-v1", strings.ToLower(string(pillar)))
}

// ---------------------------------------------------------------------------
// Gemini error handling
// ---------------------------------------------------------------------------

// geminiErrorResponse is the structure Gemini returns for non-200 responses.
type geminiErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// ErrQuotaExceeded is returned when the Gemini API quota or rate limit is hit.
var ErrQuotaExceeded = errors.New("gemini: API quota atau rate limit tercapai — coba beberapa saat lagi atau upgrade plan Gemini API")

// ErrInvalidAPIKey is returned when the API key is rejected.
var ErrInvalidAPIKey = errors.New("gemini: API key tidak valid atau tidak memiliki akses — periksa API key di Settings")

// parseGeminiError converts a non-200 Gemini HTTP response into a meaningful error.
func parseGeminiError(statusCode int, body []byte) error {
	// Try to parse Gemini's structured error body.
	var gemErr geminiErrorResponse
	msg := ""
	if err := json.Unmarshal(body, &gemErr); err == nil && gemErr.Error.Message != "" {
		msg = gemErr.Error.Message
	}

	switch statusCode {
	case http.StatusTooManyRequests: // 429
		if msg != "" {
			return fmt.Errorf("%w\nDetail: %s", ErrQuotaExceeded, msg)
		}
		return ErrQuotaExceeded

	case http.StatusUnauthorized, http.StatusForbidden: // 401, 403
		if msg != "" {
			return fmt.Errorf("%w\nDetail: %s", ErrInvalidAPIKey, msg)
		}
		return ErrInvalidAPIKey

	case http.StatusBadRequest: // 400
		if msg != "" {
			return fmt.Errorf("gemini: permintaan tidak valid — %s", msg)
		}
		return fmt.Errorf("gemini: permintaan tidak valid (HTTP 400)")

	case http.StatusServiceUnavailable, http.StatusInternalServerError: // 503, 500
		if msg != "" {
			return fmt.Errorf("gemini: layanan sedang gangguan — %s", msg)
		}
		return fmt.Errorf("gemini: layanan Gemini sedang tidak tersedia (HTTP %d), coba lagi nanti", statusCode)

	default:
		if msg != "" {
			return fmt.Errorf("gemini: HTTP %d — %s", statusCode, msg)
		}
		return fmt.Errorf("gemini: HTTP %d — %s", statusCode, truncate(string(body), 200))
	}
}

// IsQuotaError reports whether err is (or wraps) a quota/rate-limit error.
func IsQuotaError(err error) bool {
	return errors.Is(err, ErrQuotaExceeded)
}

// IsAuthError reports whether err is (or wraps) an auth/key error.
func IsAuthError(err error) bool {
	return errors.Is(err, ErrInvalidAPIKey)
}
