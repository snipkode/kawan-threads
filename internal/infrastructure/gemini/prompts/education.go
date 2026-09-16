package prompts

import "fmt"

// EducationPrompt builds a prompt for EDUKASI_HUKUM and TIPS_HUKUM pillar content.
func EducationPrompt(topic, audience, tone, cta string) string {
	context := `
KONTEKS PILLAR EDUKASI:
Fokus pada literasi hukum, kesadaran hak, tips praktis hukum sehari-hari.
Buat materi hukum terasa mudah dipahami dan relevan.
Gunakan analogi atau contoh nyata.
Hindari bahasa hukum yang terlalu teknis.
`
	return BuildPrompt("EDUKASI_HUKUM", topic, audience, tone, "SINGLE_POST", cta) + fmt.Sprintf("\nKONTEKS TAMBAHAN:%s", context)
}
