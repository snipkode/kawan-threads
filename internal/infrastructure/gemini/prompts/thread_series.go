package prompts

import "fmt"

// threadSeriesJSONFormat is the JSON schema for a 6-post thread series.
const threadSeriesJSONFormat = `
Kembalikan HANYA JSON valid (tanpa markdown) dengan format persis ini:
{
  "contentType": "string",
  "posts": [
    {"position": 1, "role": "HOOK", "text": "string"},
    {"position": 2, "role": "PROBLEM", "text": "string"},
    {"position": 3, "role": "EXPLANATION", "text": "string"},
    {"position": 4, "role": "EXAMPLE", "text": "string"},
    {"position": 5, "role": "SOLUTION", "text": "string"},
    {"position": 6, "role": "CTA", "text": "string"}
  ],
  "hook": "string (same as posts[0].text)",
  "hookVariants": [
    {"hook": "string", "hookType": "QUESTION|RELATABLE|CURIOSITY|STORY|MYTH|PROBLEM|SURPRISING_FACT|HOT_TAKE"},
    {"hook": "string", "hookType": "QUESTION|RELATABLE|CURIOSITY|STORY|MYTH|PROBLEM|SURPRISING_FACT|HOT_TAKE"},
    {"hook": "string", "hookType": "QUESTION|RELATABLE|CURIOSITY|STORY|MYTH|PROBLEM|SURPRISING_FACT|HOT_TAKE"}
  ],
  "body": "string (gabungan posts 2-5)",
  "cta": "string (same as posts[5].text)",
  "topic": "string",
  "hookType": "QUESTION|RELATABLE|CURIOSITY|STORY|MYTH|PROBLEM|SURPRISING_FACT|HOT_TAKE",
  "conversationQuestion": "string",
  "conversationScore": 85,
  "quality": {
    "hook": 85,
    "conversation": 88,
    "readability": 90,
    "originality": 82,
    "relevance": 95,
    "shareability": 87
  }
}
`

// ThreadSeriesPrompt builds a prompt for THREAD_SERIES format content.
func ThreadSeriesPrompt(topic, audience, tone, cta string) string {
	return fmt.Sprintf(`%s

TUGAS:
Buat konten Thread Series untuk KAWAN (6 post berurutan) dengan spesifikasi:
- Topik: %s
- Target Audience: %s
- Tone: %s
- CTA: %s

STRUKTUR THREAD:
Post 1 — HOOK: Kalimat pembuka yang menarik perhatian
Post 2 — PROBLEM: Masalah atau situasi yang relevan
Post 3 — EXPLANATION: Penjelasan edukatif
Post 4 — EXAMPLE: Contoh nyata atau kasus
Post 5 — SOLUTION: Solusi atau insight
Post 6 — CTA: Ajakan bertindak yang soft

%s`, BrandVoicePrompt, topic, audience, tone, cta, threadSeriesJSONFormat)
}
