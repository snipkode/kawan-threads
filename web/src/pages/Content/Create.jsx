import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Sparkles, ChevronRight, ChevronLeft } from 'lucide-react'
import { BackHeader } from '../../components/layout/Header'
import { ErrorBox } from '../../components/ui/Card'
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

const STEPS = ['Pillar', 'Topik', 'Format', 'Detail']

// Pillar icon map
const PILLAR_ICON = {
  EDUKASI_HUKUM: '📚',
  TIPS_HUKUM:    '💡',
  CERITA_WARGA:  '🗣️',
  MYTH_VS_FACT:  '⚖️',
  COMMUNITY:     '🤝',
  EVENT:         '📅',
  TRAINING:      '🎓',
  MEMBERSHIP:    '🌟',
  ENGAGEMENT:    '💬',
}

export default function ContentCreate() {
  const [form, setForm] = useState(EMPTY)
  const [step, setStep] = useState(0)
  const generate = useGenerate()
  const navigate = useNavigate()

  const set = (k, v) => setForm((f) => ({ ...f, [k]: v }))

  const canNext = () => {
    if (step === 1 && !form.topic.trim()) return false
    return true
  }

  const next = () => { if (canNext()) setStep((s) => Math.min(s + 1, STEPS.length - 1)) }
  const back = () => setStep((s) => Math.max(s - 1, 0))

  const submit = () => {
    if (!form.topic.trim()) return
    generate.mutate(form, {
      onSuccess: (content) => navigate(`/content/${content.id}/preview`),
    })
  }

  return (
    <div className="flex flex-col min-h-screen">
      <BackHeader to="/content" title="Buat Konten" />

      {/* Step indicator */}
      <div className="flex items-center gap-0 border-b border-slate-200 bg-white px-4 py-2.5">
        {STEPS.map((label, i) => (
          <div key={i} className="flex items-center">
            <button
              type="button"
              onClick={() => i < step && setStep(i)}
              className="flex items-center gap-1.5"
            >
              <span className={`flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-bold transition ${
                i === step
                  ? 'bg-brand-600 text-white'
                  : i < step
                  ? 'bg-emerald-500 text-white'
                  : 'bg-slate-200 text-slate-400'
              }`}>
                {i < step ? '✓' : i + 1}
              </span>
              <span className={`text-[11px] font-semibold ${i === step ? 'text-brand-600' : i < step ? 'text-emerald-600' : 'text-slate-400'}`}>
                {label}
              </span>
            </button>
            {i < STEPS.length - 1 && (
              <span className="mx-1.5 text-slate-200 text-[10px]">›</span>
            )}
          </div>
        ))}
      </div>

      <div className="flex-1 p-4 pb-40">
        {generate.isError && (
          <div className="mb-3">
            <ErrorBox message={apiMessage(generate.error, 'Gagal generate konten')} />
          </div>
        )}

        {/* ── Step 0: Pillar ── */}
        {step === 0 && (
          <div className="space-y-3">
            <div>
              <p className="text-base font-bold text-slate-900">Pilih Pillar Konten</p>
              <p className="text-[11px] text-slate-400 mt-0.5">Tema utama konten yang akan dibuat</p>
            </div>
            <div className="grid grid-cols-2 gap-2">
              {PILLARS.map((p) => (
                <button
                  key={p.value}
                  type="button"
                  onClick={() => set('pillar', p.value)}
                  className={`flex items-center gap-2 rounded-2xl border px-3 py-3 text-left transition active:scale-[0.98] ${
                    form.pillar === p.value
                      ? 'border-brand-600 bg-brand-50 ring-1 ring-brand-600'
                      : 'border-slate-200 bg-white'
                  }`}
                >
                  <span className="text-lg">{PILLAR_ICON[p.value]}</span>
                  <span className={`text-[12px] font-semibold leading-tight ${form.pillar === p.value ? 'text-brand-700' : 'text-slate-700'}`}>
                    {p.label}
                  </span>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* ── Step 1: Topik ── */}
        {step === 1 && (
          <div className="space-y-3">
            <div>
              <p className="text-base font-bold text-slate-900">Topik Konten</p>
              <p className="text-[11px] text-slate-400 mt-0.5">Deskripsikan apa yang ingin dibahas</p>
            </div>
            <textarea
              autoFocus
              rows={3}
              className="input resize-none"
              placeholder="Contoh: Syarat sah perjanjian menurut hukum Indonesia"
              value={form.topic}
              onChange={(e) => set('topic', e.target.value)}
            />
            {/* CTA di sini karena satu layar dengan topik */}
            <div>
              <label className="label">CTA (opsional)</label>
              <input
                className="input"
                placeholder="Kenalan dengan KAWAN."
                value={form.cta}
                onChange={(e) => set('cta', e.target.value)}
              />
            </div>
          </div>
        )}

        {/* ── Step 2: Format ── */}
        {step === 2 && (
          <div className="space-y-3">
            <div>
              <p className="text-base font-bold text-slate-900">Format Konten</p>
              <p className="text-[11px] text-slate-400 mt-0.5">Pilih struktur posting</p>
            </div>
            <div className="space-y-2">
              {FORMATS.map((fmt) => (
                <button
                  key={fmt.value}
                  type="button"
                  onClick={() => set('format', fmt.value)}
                  className={`flex w-full items-center justify-between rounded-2xl border px-4 py-3.5 text-left transition active:scale-[0.98] ${
                    form.format === fmt.value
                      ? 'border-brand-600 bg-brand-50 ring-1 ring-brand-600'
                      : 'border-slate-200 bg-white'
                  }`}
                >
                  <div>
                    <p className={`text-sm font-bold ${form.format === fmt.value ? 'text-brand-700' : 'text-slate-800'}`}>
                      {fmt.label}
                    </p>
                    <p className="text-[11px] text-slate-400 mt-0.5">
                      {fmt.value === 'SINGLE_POST'
                        ? 'Satu postingan ringkas & padat'
                        : 'Rangkaian thread bersambung (3–7 post)'}
                    </p>
                  </div>
                  <span className={`text-xl ${fmt.value === 'SINGLE_POST' ? '📝' : '🧵'}`}>
                    {fmt.value === 'SINGLE_POST' ? '📝' : '🧵'}
                  </span>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* ── Step 3: Detail (audience + tone) ── */}
        {step === 3 && (
          <div className="space-y-4">
            <div>
              <p className="text-base font-bold text-slate-900">Detail Konten</p>
              <p className="text-[11px] text-slate-400 mt-0.5">Sesuaikan target & gaya bahasa</p>
            </div>

            <div>
              <label className="label">Target Audiens</label>
              <div className="flex flex-wrap gap-1.5">
                {AUDIENCES.map((a) => (
                  <button
                    key={a}
                    type="button"
                    onClick={() => set('audience', a)}
                    className={`chip border transition ${
                      form.audience === a
                        ? 'border-brand-600 bg-brand-50 text-brand-700'
                        : 'border-slate-200 bg-white text-slate-500'
                    }`}
                  >
                    {a}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <label className="label">Tone / Gaya Bahasa</label>
              <div className="flex flex-wrap gap-1.5">
                {TONES.map((t) => (
                  <button
                    key={t}
                    type="button"
                    onClick={() => set('tone', t)}
                    className={`chip border transition capitalize ${
                      form.tone === t
                        ? 'border-brand-600 bg-brand-50 text-brand-700'
                        : 'border-slate-200 bg-white text-slate-500'
                    }`}
                  >
                    {t}
                  </button>
                ))}
              </div>
            </div>

            {/* Summary sebelum generate */}
            <div className="rounded-2xl bg-slate-50 px-4 py-3 ring-1 ring-slate-200 space-y-1.5">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Ringkasan</p>
              <SummaryRow label="Pillar" value={PILLARS.find(p => p.value === form.pillar)?.label} />
              <SummaryRow label="Topik" value={form.topic} />
              <SummaryRow label="Format" value={FORMATS.find(f => f.value === form.format)?.label} />
              <SummaryRow label="Audiens" value={form.audience} />
              <SummaryRow label="Tone" value={form.tone} />
            </div>
          </div>
        )}
      </div>

      {/* Sticky bottom nav */}
      <div className="fixed bottom-[4.5rem] left-0 right-0 z-40 border-t border-slate-200 bg-white/95 backdrop-blur-sm px-4 py-3">
        <div className="flex items-center gap-2">
          {step > 0 && (
            <button type="button" className="btn-secondary flex-none px-3 py-2" onClick={back}>
              <ChevronLeft size={16} />
            </button>
          )}

          {step < STEPS.length - 1 ? (
            <button
              type="button"
              className="btn-primary flex-1 py-2"
              onClick={next}
              disabled={!canNext()}
            >
              Lanjut <ChevronRight size={15} />
            </button>
          ) : (
            <button
              type="button"
              className="btn-primary flex-1 py-2.5"
              onClick={submit}
              disabled={!form.topic.trim() || generate.isPending}
            >
              {generate.isPending ? (
                <span className="flex items-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  Menyusun konten…
                </span>
              ) : (
                <span className="flex items-center gap-2">
                  <Sparkles size={15} /> Generate dengan Gemini
                </span>
              )}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

function SummaryRow({ label, value }) {
  return (
    <div className="flex items-start justify-between gap-2">
      <span className="text-[11px] text-slate-400 shrink-0">{label}</span>
      <span className="text-[12px] font-medium text-slate-700 text-right">{value || '—'}</span>
    </div>
  )
}
