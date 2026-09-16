package gemini

import (
	"strings"
	"testing"

	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/port"
)

func sampleGenerated() port.GeneratedContent {
	return port.GeneratedContent{
		ContentType:          "COMMUNITY",
		Hook:                 "Banyak orang baru belajar hukum setelah masalah datang.",
		HookVariants:         []port.HookVariant{{Hook: "A", HookType: entity.HookQuestion}},
		Body:                 "Padahal memahami hak sejak awal itu penting.",
		CTA:                  "Kenalan dengan KAWAN.",
		Topic:                "Komunitas",
		HookType:             entity.HookRelatable,
		ConversationQuestion: "Pernah merasa seperti ini?",
		ConversationScore:    88,
		Quality: entity.QualityScore{
			Hook: 90, Conversation: 88, Readability: 92,
			Originality: 86, Relevance: 95, Shareability: 87,
		},
	}
}

func TestValidateGeneratedContent_Valid(t *testing.T) {
	if err := ValidateGeneratedContent(sampleGenerated()); err != nil {
		t.Fatalf("expected valid content, got error: %v", err)
	}
}

func TestValidateGeneratedContent_MissingFields(t *testing.T) {
	gc := sampleGenerated()
	gc.Hook = ""
	gc.Body = ""

	err := ValidateGeneratedContent(gc)
	if err == nil {
		t.Fatal("expected error for missing hook/body")
	}
	if !strings.Contains(err.Error(), "hook") || !strings.Contains(err.Error(), "body") {
		t.Fatalf("error should mention hook and body, got: %v", err)
	}
}

func TestValidateGeneratedContent_InvalidHookType(t *testing.T) {
	gc := sampleGenerated()
	gc.HookType = entity.HookType("CLICKBAIT")
	if err := ValidateGeneratedContent(gc); err == nil {
		t.Fatal("expected error for invalid hook type")
	}
}

func TestValidateGeneratedContent_ScoresOutOfRange(t *testing.T) {
	gc := sampleGenerated()
	gc.Quality.Hook = 120
	if err := ValidateGeneratedContent(gc); err == nil {
		t.Fatal("expected error for out-of-range quality score")
	}
}

func TestValidateGeneratedContent_NoVariants(t *testing.T) {
	gc := sampleGenerated()
	gc.HookVariants = nil
	if err := ValidateGeneratedContent(gc); err == nil {
		t.Fatal("expected error for missing hook variants")
	}
}

func TestParseGeneratedContent_StripsCodeFence(t *testing.T) {
	raw := "```json\n" +
		`{"contentType":"COMMUNITY","hook":"H","hookVariants":[{"hook":"A","hookType":"QUESTION"}],"body":"B","cta":"C","topic":"T","hookType":"RELATABLE","conversationQuestion":"Q","conversationScore":80,"quality":{"hook":80,"conversation":81,"readability":82,"originality":83,"relevance":84,"shareability":85}}` +
		"\n```"

	gc, err := parseGeneratedContent(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if gc.Hook != "H" || gc.Body != "B" || gc.CTA != "C" {
		t.Fatalf("unexpected parsed content: %+v", gc)
	}
	if gc.Quality.Shareability != 85 {
		t.Fatalf("expected shareability 85, got %d", gc.Quality.Shareability)
	}
}

func TestParseGeneratedContent_NoJSON(t *testing.T) {
	if _, err := parseGeneratedContent("hello world"); err == nil {
		t.Fatal("expected error for non-JSON response")
	}
}
