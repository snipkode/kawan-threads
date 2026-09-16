package port

import (
	"context"

	"kawan-threads/internal/domain/entity"
)

// GenerateContentRequest carries the parameters needed to generate content.
type GenerateContentRequest struct {
	Pillar   entity.ContentPillar `json:"pillar"`
	Topic    string               `json:"topic"`
	Audience string               `json:"audience"`
	Tone     string               `json:"tone"`
	Format   entity.ContentFormat `json:"format"`
	CTA      string               `json:"cta"`
}

// HookVariant is a single alternative hook with its classified type.
type HookVariant struct {
	Hook     string          `json:"hook"`
	HookType entity.HookType `json:"hook_type"`
}

// GeneratedContent is the structured response returned by an AI provider.
type GeneratedContent struct {
	ContentType          string              `json:"content_type"`
	Hook                 string              `json:"hook"`
	HookVariants         []HookVariant       `json:"hook_variants"`
	Body                 string              `json:"body"`
	CTA                  string              `json:"cta"`
	Topic                string              `json:"topic"`
	HookType             entity.HookType     `json:"hook_type"`
	ConversationQuestion string              `json:"conversation_question"`
	ConversationScore    int                 `json:"conversation_score"`
	Quality              entity.QualityScore `json:"quality"`
	PromptVersion        string              `json:"prompt_version"`
	Model                string              `json:"model"`
}

// AIProvider is the port that must be implemented by any AI backend
// (e.g. Gemini) used to generate social-media content.
type AIProvider interface {
	// GenerateContent produces a GeneratedContent response for the given request.
	GenerateContent(ctx context.Context, req GenerateContentRequest) (GeneratedContent, error)
}
