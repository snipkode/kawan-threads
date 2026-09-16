package gemini

import (
	"strings"
	"testing"
)

func TestParseGeneratedContent(t *testing.T) {
	base := `{
  "contentType": "edukasi_hukum",
  "hook": "Apakah sah perjanjian lewat WA?",
  "hookVariants": [{"hook": "Varian A", "hookType": "question"}],
  "body": "Body konten.",
  "cta": "Ikuti KAWAN.",
  "topic": "Syarat sah perjanjian",
  "hookType": "question",
  "conversationQuestion": "Pernah bikin perjanjian lisan?",
  "conversationScore": 80,
  "quality": {"hook": 90, "conversation": 80, "readability": 95, "originality": 85, "relevance": 90, "shareability": 70}
}`

	cases := []struct {
		name string
		in   string
	}{
		{"plain", base},
		{"fenced", "```json\n" + base + "\n```"},
		{"prefixed_text", "Berikut hasilnya:\n" + base},
		{"brace_in_cta", strings.Replace(base, `"cta": "Ikuti KAWAN."`, `"cta": "Baca selengkapnya di {link}"`, 1)},
		{"brace_in_hook", strings.Replace(base, `"hook": "Apakah sah perjanjian lewat WA?"`, `"hook": "Syarat sah {poin} perjanjian"`, 1)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseGeneratedContent(tc.in)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got.Hook == "" || got.Body == "" {
				t.Fatalf("missing content fields: %+v", got)
			}
		})
	}
}

func TestParseGeneratedContent_NoJSONBare(t *testing.T) {
	if _, err := parseGeneratedContent("maaf, saya tidak bisa membuat konten itu."); err == nil {
		t.Fatal("expected error for non-JSON response")
	}
}
