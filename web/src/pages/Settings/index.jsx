import { useState } from 'react'
import { Save, KeyRound, RefreshCw } from 'lucide-react'
import { Header } from '../../components/layout/Header'
import { Card, Spinner, ErrorBox } from '../../components/ui/Card'
import { useSettings, useUpdateSettings } from '../../hooks/useApi'
import { apiMessage } from '../../types'

export default function Settings() {
  const settings = useSettings()
  const update = useUpdateSettings()
  const [form, setForm] = useState(null)

  const s = form ?? settings.data ?? {}
  const set = (k) => (e) => {
    const v = e.target.type === 'checkbox' ? e.target.checked : e.target.value
    setForm({ ...s, [k]: v })
  }

  const save = () => update.mutate(form ?? s, { onSuccess: () => setForm(null) })

  return (
    <>
      <Header title="Settings" subtitle="Konfigurasi platform" />
      <div className="space-y-3 p-4">
        {settings.isLoading ? (
          <Spinner label="Memuat settings…" />
        ) : (
          <>
            {update.isError && <ErrorBox message={apiMessage(update.error)} />}

            <Card className="space-y-3">
              <h3 className="text-sm font-bold text-slate-800">Workflow Otomatisasi</h3>
              <ToggleRow
                label="Auto-approval"
                hint="Content langsung disetujui & masuk queue"
                checked={!!s.auto_approval}
                onChange={set('auto_approval')}
              />
              <ToggleRow
                label="Auto-publish"
                hint="Threads diterbitkan otomatis saat waktunya tiba"
                checked={!!s.auto_publish}
                onChange={set('auto_publish')}
              />
            </Card>

            <Card className="space-y-3">
              <h3 className="text-sm font-bold text-slate-800">Kapasitas Posting</h3>
              <div>
                <label className="label">Max posts per hari</label>
                <input type="number" min="0" className="input" value={s.max_posts_per_day ?? 0} onChange={set('max_posts_per_day')} />
              </div>
              <div>
                <label className="label">Interval minimal (menit)</label>
                <input type="number" min="0" className="input" value={s.min_post_interval_minutes ?? 0} onChange={set('min_post_interval_minutes')} />
              </div>
              <div>
                <label className="label">Exploration rate</label>
                <input type="number" min="0" max="1" step="0.05" className="input" value={s.exploration_rate ?? 0} onChange={set('exploration_rate')} />
              </div>
              <div>
                <label className="label">Max retry</label>
                <input type="number" min="0" className="input" value={s.max_retry ?? 0} onChange={set('max_retry')} />
              </div>
            </Card>

            <Card className="space-y-3">
              <h3 className="text-sm font-bold text-slate-800">Jadwal & Integrasi</h3>
              <div>
                <label className="label">Timezone</label>
                <input className="input" value={s.timezone || 'Asia/Jakarta'} onChange={set('timezone')} />
              </div>
              <div className="flex items-center justify-between gap-2 border-t border-slate-100 pt-3">
                <div className="flex items-center gap-2 text-xs text-slate-500">
                  <KeyRound size={14} /> Status Integrasi Threads
                </div>
                <span className="chip bg-amber-100 text-amber-700">Lihat log (server)</span>
              </div>
            </Card>

            <button className="btn-primary w-full py-3" onClick={save} disabled={update.isPending}>
              {update.isPending ? (
                <RefreshCw size={16} className="animate-spin" />
              ) : (
                <Save size={16} />
              )}
              Simpan Pengaturan
            </button>
          </>
        )}
      </div>
    </>
  )
}

function ToggleRow({ label, hint, checked, onChange }) {
  return (
    <label className="flex items-center justify-between gap-3">
      <div>
        <p className="text-sm font-medium text-slate-800">{label}</p>
        {hint && <p className="text-[11px] text-slate-400">{hint}</p>}
      </div>
      <button
        type="button"
        onClick={onChange}
        aria-pressed={checked}
        className={`relative h-6 w-11 shrink-0 rounded-full transition ${
          checked ? 'bg-brand-600' : 'bg-slate-300'
        }`}
      >
        <span
          className={`absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-all ${
            checked ? 'left-[22px]' : 'left-0.5'
          }`}
        />
      </button>
    </label>
  )
}