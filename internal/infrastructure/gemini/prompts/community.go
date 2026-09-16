package prompts

import "fmt"

// CommunityPrompt builds a prompt for COMMUNITY pillar content.
func CommunityPrompt(topic, audience, tone, cta string) string {
	context := `
KONTEKS PILLAR COMMUNITY:
Fokus pada solidaritas warga, akses keadilan bersama, kekuatan komunitas.
Ceritakan bagaimana bersama-sama kita bisa lebih kuat.
Tunjukkan nilai komunitas tanpa memaksa orang untuk join.
`
	return BuildPrompt("COMMUNITY", topic, audience, tone, "SINGLE_POST", cta) + fmt.Sprintf("\nKONTEKS TAMBAHAN:%s", context)
}
