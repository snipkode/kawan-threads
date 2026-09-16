import { useState } from 'react'
import { Save, RefreshCw, Sparkles, KeyRound, CalendarClock, Bot } from 'lucide-react'
import { Header } from '../../components/layout/Header'
import { Card, Spinner, ErrorBox } from '../../components/ui/Card'
import { useSettings, useSettingsStatus, useUpdateSettings } from '../../hooks/useApi'
import { apiMessage } from '../../types'

export default function Settings() {
  const settings = useSettings()
  const status = useSettingsStatus()
  const update = useUpdateSettings()
  const [form, setForm] = useState(null)

  const s = form ?? settings.data ?? {}
  const st = status.data ?? {}

  const set = (k) => (e) => {
    const v = e.target.type === 'checkbox' ? e.target.checked : e.target.value
    setForm({ ...s, [k]: v })
  }

  // For boolean toggles (button-based, no real event)
  const toggle = (k) => () => setForm({ ...s, [k]: !s[k] })

  const save = () => update.mutate(form ?? s, { onSuccess: () => setForm(null) })

  return (
    <>
      <Header title="Settings" subtitle="Konfigurasi platform (berlaku langsung)" />
      <div className="space-y-3 p-4">
        {settings.isLoading ? (
          <Spinner label="Memuat settings…" />
        ) : (
          <>
            {update.isError && <ErrorBox message={apiMessage(update.error)} />}

            <Card className="space-y-3">
              <h3 className="flex items-center gap-2 text-sm font-bold text-slate-800">
                <Sparkles size={15} className="text-brand-500" /> AI — Gemini
              </h3>
              <p className="text-[11px] text-slate-400">
                API key & model dibaca saat request; simpan lalu buat konten baru untuk memakai.
              </p>
              <IntegrationChip
                label="Status AI"
                ok={!!st.gemini_configured}
                okText="Terkonfigurasi"
                failText="Belum ada API key"
              />
              <div>
                <label className="label">API key</label>
                <input type="password" className="input" placeholder="AIza…" value={s.gemini_api_key ?? ''} onChange={set('gemini_api_key')} />
              </div>
              <div>
                <label className="label">Model</label>
                <input className="input" placeholder="gemini-3.6-flash" value={s.gemini_model ?? ''} onChange={set('gemini_model')} />
              </div>
            </Card>

            <Card className="space-y-3">
              <h3 className="flex items-center gap-2 text-sm font-bold text-slate-800">
                <KeyRound size={15} className="text-brand-500" /> Threads API
              </h3>
              <p className="text-[11px] text-slate-400">
                Kredensial OAuth. Token bisa didapat lewat alur OAuth (<code className="text-slate-500">/api/auth/threads</code>) dan tersimpan otomatis.
              </p>
              <IntegrationChip
                label="Status integrasi"
                ok={!!st.threads_configured}
                okText="Client dikonfigurasi"
                failText="Belum ada client_id / user_id"
              />
              <IntegrationChip
                label="Status koneksi"
                ok={!!st.threads_connected}
                okText="Terkoneksi (token tersimpan)"
                failText="Belum ada access token"
              />
              <div>
                <label className="label">Client ID</label>
                <input className="input" value={s.threads_client_id ?? ''} onChange={set('threads_client_id')} />
              </div>
              <div>
                <label className="label">Client secret</label>
                <input type="password" className="input" value={s.threads_client_secret ?? ''} onChange={set('threads_client_secret')} />
              </div>
              <div>
                <label className="label">Redirect URI</label>
                <input className="input" placeholder="https://api.kawan.app/api/auth/threads/callback" value={s.threads_redirect_uri ?? ''} onChange={set('threads_redirect_uri')} />
              </div>
              <div>
                <label className="label">User ID</label>
                <input className="input" value={s.threads_user_id ?? ''} onChange={set('threads_user_id')} />
              </div>
              <div>
                <label className="label">Access token (long-lived)</label>
                <input type="password" className="input" value={s.threads_access_token ?? ''} onChange={set('threads_access_token')} />
              </div>
            </Card>

            <Card className="space-y-3">
              <h3 className="flex items-center gap-2 text-sm font-bold text-slate-800">
                <CalendarClock size={15} className="text-brand-500" /> Scheduler (AMAB)
              </h3>
              <div>
                <label className="label">Timezone</label>
                <input className="input" value={s.timezone || 'Asia/Jakarta'} onChange={set('timezone')} />
              </div>
              <div>
                <label className="label">Interval scheduler (menit)</label>
                <input type="number" min="1" className="input" value={s.scheduler_interval_minutes ?? 5} onChange={set('scheduler_interval_minutes')} />
                <p className="text-[11px] text-slate-400">Interval dicek ulang tiap tick — berlaku tanpa restart worker.</p>
              </div>
            </Card>

            <Card className="space-y-3">
              <h3 className="text-sm font-bold text-slate-800">Workflow Otomatisasi</h3>
              <ToggleRow
                label="Auto-approval"
                hint="Content langsung disetujui & masuk queue"
                checked={!!s.auto_approval}
                onChange={toggle('auto_approval')}
              />
              <ToggleRow
                label="Auto-publish"
                hint="Threads diterbitkan otomatis saat waktunya tiba"
                checked={!!s.auto_publish}
                onChange={toggle('auto_publish')}
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
                <label className="label">Exploration rate (0–1)</label>
                <input type="number" min="0" max="1" step="0.05" className="input" value={s.exploration_rate ?? 0} onChange={set('exploration_rate')} />
              </div>
              <div>
                <label className="label">Max retry publish</label>
                <input type="number" min="0" className="input" value={s.max_retry ?? 0} onChange={set('max_retry')} />
              </div>
            </Card>

            {/* ── Autopilot ── */}
            <Card className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="flex items-center gap-2 text-sm font-bold text-slate-800">
                  <Bot size={15} className="text-brand-500" /> Autopilot
                </h3>
                <span className={`chip ${s.autopilot_enabled ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-500'}`}>
                  {s.autopilot_enabled ? 'Aktif' : 'Nonaktif'}
                </span>
              </div>
              <p className="text-[11px] text-slate-400 leading-relaxed">
                Worker otomatis generate draft konten setiap hari tanpa konfigurasi manual.
                Semua output tetap <strong>DRAFT</strong> — perlu review sebelum publish.
              </p>
              <ToggleRow
                label="Aktifkan Autopilot"
                hint="Generate draft harian otomatis saat jam yang ditentukan"
                checked={!!s.autopilot_enabled}
                onChange={toggle('autopilot_enabled')}
              />
              <div>
                <label className="label">Jumlah konten per hari</label>
                <input
                  type="number" min="1" max="10" className="input"
                  value={s.autopilot_daily_count ?? 3}
                  onChange={set('autopilot_daily_count')}
                />
                <p className="mt-1 text-[11px] text-slate-400">Mix otomatis: ~70% single post, ~30% thread series. Pillar dipilih random.</p>
              </div>
              <div>
                <label className="label">Jam generate (0–23, waktu lokal)</label>
                <input
                  type="number" min="0" max="23" className="input"
                  value={s.autopilot_run_hour ?? 7}
                  onChange={set('autopilot_run_hour')}
                />
                <p className="mt-1 text-[11px] text-slate-400">Default jam 07:00 — draft sudah siap di pagi hari untuk direview.</p>
              </div>
              <div>
                <label className="label">Goal konten</label>
                <select
                  className="input"
                  value={s.autopilot_goal ?? 'website_visit'}
                  onChange={set('autopilot_goal')}
                >
                  <option value="website_visit">Kunjungan Website</option>
                  <option value="training_signup">Pendaftaran Pelatihan</option>
                  <option value="community_growth">Pertumbuhan Komunitas</option>
                </select>
                <p className="mt-1 text-[11px] text-slate-400">Goal menentukan arah CTA yang disuntikkan ke setiap konten yang digenerate.</p>
              </div>
            </Card>

            {st.data_store && (
              <p className="px-1 text-[11px] text-slate-400">
                Datastore: <span className="font-mono">{st.data_store}</span> — mode penyimpanan
                & Firebase service account hanya diatur via environment (tidak lewat UI).
              </p>
            )}

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

function IntegrationChip({ label, ok, okText, failText }) {
  return (
    <div className="flex items-center justify-between gap-2">
      <span className="text-xs text-slate-500">{label}</span>
      <span className={`chip ${ok ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'}`}>
        {ok ? okText : failText}
      </span>
    </div>
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