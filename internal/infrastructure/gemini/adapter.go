// Package gemini provides a Gemini AI adapter implementing the AIProvider port.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"kawan-threads/internal/config"
	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/port"
	"kawan-threads/internal/infrastructure/gemini/prompts"
)

const geminiAPIBase = "https://generativelanguage.googleapis.com/v1beta/models"

// GeminiAdapter implements port.AIProvider using the Gemini REST API.
type GeminiAdapter struct {
	apiKey     string
	model      string
	httpClient *http.Client
	logger     *slog.Logger
	maxRetries int
}

// NewGeminiAdapter constructs a GeminiAdapter from app config.
func NewGeminiAdapter(cfg *config.Config, logger *slog.Logger) (*GeminiAdapter, error) {
	if cfg.Gemini.APIKey == "" {
		return nil, fmt.Errorf("gemini: GEMINI_API_KEY is required")
	}
	model := cfg.Gemini.Model
	if model == "" {
		model = "gemini-1.5-flash"
	}
	return &GeminiAdapter{
		apiKey: cfg.Gemini.APIKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger:     logger,
		maxRetries: 3,
	}, nil
}

// GenerateContent implements port.AIProvider.
func (g *GeminiAdapter) GenerateContent(ctx context.Context, req port.GenerateContentRequest) (port.GeneratedContent, error) {
	prompt := g.buildPrompt(req)

	var (
		result  port.GeneratedContent
		lastErr error
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

		raw, err := g.callAPI(ctx, prompt)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: API call failed: %w", attempt+1, err)
			continue
		}

		result, err = parseGeneratedContent(raw)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: parse failed: %w", attempt+1, err)
			continue
		}

		if err := ValidateGeneratedContent(result); err != nil {
			lastErr = fmt.Errorf("attempt %d: validation failed: %w", attempt+1, err)
			continue
		}

		// Success — tag model and prompt version
		result.Model = g.model
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
	Temperature     float64 `json:"temperature"`
	TopK            int     `json:"topK"`
	TopP            float64 `json:"topP"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
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

func (g *GeminiAdapter) callAPI(ctx context.Context, prompt string) (string, error) {
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: generationConfig{
			Temperature:     0.8,
			TopK:            40,
			TopP:            0.95,
			MaxOutputTokens: 2048,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiAPIBase, g.model, g.apiKey)

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
		return "", fmt.Errorf("gemini API HTTP %d: %s", resp.StatusCode, string(respBody))
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
	// Find JSON object boundaries.
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return port.GeneratedContent{}, fmt.Errorf("no JSON object found in response")
	}
	text = text[start : end+1]

	var out geminiOutputJSON
	if err := json.Unmarshal([]byte(text), &out); err != nil {
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
