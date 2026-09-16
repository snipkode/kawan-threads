package prompts

import "fmt"

// EngagementPrompt builds a prompt for ENGAGEMENT pillar content.
func EngagementPrompt(topic, audience, tone, cta string) string {
	context := `
KONTEKS PILLAR ENGAGEMENT:
Fokus pada conversation starter, situasi relatable, pertanyaan komunitas.
Buat orang ingin berbagi pengalaman dan mendiskusikan topik.
Pertanyaan harus natural, bukan engagement bait.
`
	return BuildPrompt("ENGAGEMENT", topic, audience, tone, "SINGLE_POST", cta) + fmt.Sprintf("\nKONTEKS TAMBAHAN:%s", context)
}
