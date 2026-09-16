package prompts

import "fmt"

// EventPrompt builds a prompt for EVENT/TRAINING pillar content.
func EventPrompt(topic, audience, tone, cta string) string {
	context := `
KONTEKS PILLAR EVENT:
Fokus pada pengumuman kegiatan, undangan, reminder pendaftaran.
Buat orang merasa excited dan ingin ikut.
Tampilkan value dari kegiatan, bukan hanya info teknis.
`
	return BuildPrompt("EVENT", topic, audience, tone, "SINGLE_POST", cta) + fmt.Sprintf("\nKONTEKS TAMBAHAN:%s", context)
}
