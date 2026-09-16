package prompts

import "fmt"

// MembershipPrompt builds a prompt for MEMBERSHIP pillar content.
func MembershipPrompt(topic, audience, tone, cta string) string {
	context := `
KONTEKS PILLAR MEMBERSHIP:
Fokus pada ajakan bergabung KAWAN, manfaat komunitas, social proof.
Tunjukkan kenapa komunitas ini penting, bukan sekadar promosi.
Gunakan cerita atau pengalaman nyata anggota jika relevan.
`
	return BuildPrompt("MEMBERSHIP", topic, audience, tone, "SINGLE_POST", cta) + fmt.Sprintf("\nKONTEKS TAMBAHAN:%s", context)
}
