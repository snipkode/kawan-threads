import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Sparkles } from 'lucide-react'
import { BackHeader } from '../../components/layout/Header'
import { Card, Spinner, ErrorBox } from '../../components/ui/Card'
import { useGenerate } from '../../hooks/useApi'
import { PILLARS, AUDIENCES, TONES, FORMATS, apiMessage } from '../../types'

const EMPTY = {
  pillar: 'EDUKASI_HUKUM',
  topic: '',
  audience: 'Gen Z',
  tone: 'casual',
  format: 'SINGLE_POST',
  cta: 'Kenalan dengan KAWAN.',
}

export default function ContentCreate() {
  const [form, setForm] = useState(EMPTY)
  const generate = useGenerate()
  const navigate = useNavigate()

  const set = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }))

  const submit = (e) => {
    e.preventDefault()
    if (!form.topic.trim()) return
    generate.mutate(form, {
      onSuccess: (content) => navigate(`/content/${content.id}/preview`),
    })
  }

  return (
    <>
      <BackHeader to="/content" title="Content" />
      <div className="p-4">
        <Card className="mb-3 bg-gradient-to-br from-brand-600 to-brand-900 text-white">
          <h1 className="text-lg font-bold">Create Content</h1>
          <p className="mt-0.5 text-xs text-white/80">
            Hasil akan tersimpan sebagai DRAFT dan menunggu approval — tidak langsung publish.
          </p>
        </Card>

        {generate.isError && <div className="mb-3"><ErrorBox message={apiMessage(generate.error, 'Gagal generate konten')} /></div>}

        <form onSubmit={submit} className="space-y-3">
          <div>
            <label className="label">Content Pillar</label>
            <select className="input" value={form.pillar} onChange={set('pillar')}>
              {PILLARS.map((p) => (
                <option key={p.value} value={p.value}>{p.label}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="label">Topic</label>
            <input
              className="input"
              placeholder="Contoh: Kenapa KAWAN dibuat?"
              value={form.topic}
              onChange={set('topic')}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Audience</label>
              <select className="input" value={form.audience} onChange={set('audience')}>
                {AUDIENCES.map((a) => (
                  <option key={a} value={a}>{a}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">Tone</label>
              <select className="input" value={form.tone} onChange={set('tone')}>
                {TONES.map((t) => (
                  <option key={t} value={t}>{t}</option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className="label">Format</label>
            <div className="flex gap-2">
              {FORMATS.map((fmt) => (
                <button
                  type="button"
                  key={fmt.value}
                  onClick={() => setForm((f) => ({ ...f, format: fmt.value }))}
                  className={`flex-1 rounded-xl border px-3 py-2.5 text-sm font-semibold transition ${
                    form.format === fmt.value
                      ? 'border-brand-600 bg-brand-50 text-brand-700'
                      : 'border-slate-300 bg-white text-slate-500'
                  }`}
                >
                  {fmt.label}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="label">CTA</label>
            <input className="input" value={form.cta} onChange={set('cta')} />
          </div>

          <button
            type="submit"
            disabled={!form.topic.trim() || generate.isPending}
            className="btn-primary w-full py-3 text-base"
          >
            {generate.isPending ? (
              <>{' '}<Spinner label="Menyusun konten…" /></>
            ) : (
              <>
                <Sparkles size={17} /> Generate with Gemini
              </>
            )}
          </button>
        </form>
      </div>
    </>
  )
}