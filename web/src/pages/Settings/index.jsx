import { useState } from 'react'
import { Save, RefreshCw, ChevronDown } from 'lucide-react'
import { Header } from '../../components/layout/Header'
import { Spinner, ErrorBox } from '../../components/ui/Card'
import { useSettings, useSettingsStatus, useUpdateSettings } from '../../hooks/useApi'
import { apiMessage } from '../../types'

const GOALS = [
  { value: 'website_visit', label: 'Kunjungan Website' },
  { value: 'training_signup', label: 'Pendaftaran Pelatihan' },
  { value: 'community_growth', label: 'Pertumbuhan Komunitas' },
]

const TABS = [
  { id: 'integrasi', label: 'Integrasi' },
  { id: 'posting', label: 'Posting' },
  { id: 'autopilot', label: 'Autopilot' },
]

export default function Settings() {
  const settings = useSettings()
  const status = useSettingsStatus()
  const update = useUpdateSettings()
  const [form, setForm] = useState(null)
  const [tab, setTab] = useState('integrasi')

  const s = form ?? settings.data ?? {}
  const st = status.data ?? {}

  const set = (k) => (e) => setForm({ ...s, [k]: e.target.value })

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

            <div className="flex gap-1 rounded-xl bg-white p-1 ring-1 ring-slate-200">
              {TABS.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => setTab(t.id)}
                  className={`flex-1 rounded-lg py-2 text-[13px] font-semibold transition ${
                    tab === t.id ? 'bg-brand-600 text-white' : 'text-slate-500'
                  }`}
                >
                  {t.label}
                </button>
              ))}
            </div>

            {tab === 'integrasi' && (
              <>
                <Section title="AI · Gemini">
                  <StatusRow
                    label="Status AI"
                    ok={!!st.gemini_configured}
                    okText="Terkonfigurasi"
                    failText="Belum ada API key"
                  />
                  <InputRow
                    label="API key"
                    type="password"
                    placeholder="AIza…"
                    value={s.gemini_api_key ?? ''}
                    onChange={set('gemini_api_key')}
                  />
                  <InputRow
                    label="Model"
                    placeholder="gemini-3.6-flash"
                    value={s.gemini_model ?? ''}
                    onChange={set('gemini_model')}
                  />
                </Section>
                <p className="px-1 text-[11px] leading-relaxed text-slate-400">
                  API key &amp; model dibaca saat request; simpan lalu buat konten baru untuk
                  memakai.
                </p>

                <Section title="Threads API">
                  <StatusRow
                    label="Status integrasi"
                    ok={!!st.threads_configured}
                    okText="Client dikonfigurasi"
                    failText="Belum ada client_id / user_id"
                  />
                  <StatusRow
                    label="Status koneksi"
                    ok={!!st.threads_connected}
                    okText="Terkoneksi"
                    failText="Belum ada access token"
                  />
                  <InputRow
                    label="Client ID"
                    value={s.threads_client_id ?? ''}
                    onChange={set('threads_client_id')}
                  />
                  <InputRow
                    label="Client secret"
                    type="password"
                    value={s.threads_client_secret ?? ''}
                    onChange={set('threads_client_secret')}
                  />
                  <InputRow
                    label="Redirect URI"
                    placeholder="https://api.kawan.app/api/auth/threads/callback"
                    value={s.threads_redirect_uri ?? ''}
                    onChange={set('threads_redirect_uri')}
                  />
                  <InputRow
                    label="User ID"
                    value={s.threads_user_id ?? ''}
                    onChange={set('threads_user_id')}
                  />
                  <InputRow
                    label="Access token"
                    type="password"
                    value={s.threads_access_token ?? ''}
                    onChange={set('threads_access_token')}
                  />
                </Section>
                <p className="px-1 text-[11px] leading-relaxed text-slate-400">
                  Kredensial OAuth. Token bisa didapat lewat alur OAuth ({`/api/auth/threads`}) dan
                  tersimpan otomatis.
                </p>
              </>
            )}

            {tab === 'posting' && (
              <>
                <Section title="Scheduler · AMAB">
                  <InputRow
                    label="Timezone"
                    value={s.timezone || 'Asia/Jakarta'}
                    onChange={set('timezone')}
                  />
                  <InputRow
                    label="Interval scheduler"
                    hint="menit"
                    type="number"
                    min="1"
                    value={s.scheduler_interval_minutes ?? 5}
                    onChange={set('scheduler_interval_minutes')}
                  />
                </Section>
                <p className="px-1 text-[11px] leading-relaxed text-slate-400">
                  Interval dicek ulang tiap tick — berlaku tanpa restart worker.
                </p>

                <Section title="Workflow Otomatisasi">
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
                </Section>

                <Section title="Kapasitas Posting">
                  <InputRow
                    label="Max posts per hari"
                    type="number"
                    min="0"
                    value={s.max_posts_per_day ?? 0}
                    onChange={set('max_posts_per_day')}
                  />
                  <InputRow
                    label="Interval minimal"
                    hint="menit"
                    type="number"
                    min="0"
                    value={s.min_post_interval_minutes ?? 0}
                    onChange={set('min_post_interval_minutes')}
                  />
                  <InputRow
                    label="Exploration rate"
                    hint="0–1"
                    type="number"
                    min="0"
                    max="1"
                    step="0.05"
                    value={s.exploration_rate ?? 0}
                    onChange={set('exploration_rate')}
                  />
                  <InputRow
                    label="Max retry publish"
                    type="number"
                    min="0"
                    value={s.max_retry ?? 0}
                    onChange={set('max_retry')}
                  />
                </Section>
              </>
            )}

            {tab === 'autopilot' && (
              <>
                <Section
                  title="Autopilot"
                  titleRight={
                    <span
                      className={`chip ${
                        s.autopilot_enabled
                          ? 'bg-emerald-100 text-emerald-700'
                          : 'bg-slate-100 text-slate-500'
                      }`}
                    >
                      {s.autopilot_enabled ? 'Aktif' : 'Nonaktif'}
                    </span>
                  }
                >
                  <ToggleRow
                    label="Aktifkan Autopilot"
                    hint="Generate draft harian otomatis"
                    checked={!!s.autopilot_enabled}
                    onChange={toggle('autopilot_enabled')}
                  />
                  <InputRow
                    label="Jumlah konten per hari"
                    type="number"
                    min="1"
                    max="10"
                    value={s.autopilot_daily_count ?? 3}
                    onChange={set('autopilot_daily_count')}
                  />
                  <InputRow
                    label="Jam generate"
                    hint="0–23, waktu lokal"
                    type="number"
                    min="0"
                    max="23"
                    value={s.autopilot_run_hour ?? 7}
                    onChange={set('autopilot_run_hour')}
                  />
                  <SelectRow
                    label="Goal konten"
                    value={s.autopilot_goal ?? 'website_visit'}
                    onChange={set('autopilot_goal')}
                    options={GOALS}
                  />
                </Section>
                <p className="px-1 text-[11px] leading-relaxed text-slate-400">
                  Worker generate draft konten setiap hari tanpa konfigurasi manual. Semua output
                  tetap <b className="font-semibold text-slate-500">DRAFT</b> — perlu review
                  sebelum publish. Goal menentukan arah CTA konten.
                </p>
              </>
            )}

            {st.data_store && (
              <p className="px-1 text-[11px] text-slate-400">
                Datastore: <span className="font-mono">{st.data_store}</span> — mode penyimpanan
                &amp; Firebase service account hanya diatur via environment (tidak lewat UI).
              </p>
            )}

            <button className="btn-primary w-full py-2.5" onClick={save} disabled={update.isPending}>
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

function Section({ title, titleRight, children }) {
  return (
    <section>
      {(title || titleRight) && (
        <div className="flex items-center justify-between px-1 pb-1.5">
          <h3 className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">
            {title}
          </h3>
          {titleRight}
        </div>
      )}
      <div className="divide-y divide-slate-100 overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-slate-200">
        {children}
      </div>
    </section>
  )
}

function Row({ label, hint, children }) {
  return (
    <div className="flex min-h-[46px] items-center gap-3 bg-white px-4 py-2.5">
      <div className="min-w-0 flex-1">
        <p className="text-[15px] leading-tight text-slate-800">{label}</p>
        {hint && <p className="mt-0.5 text-[11px] leading-tight text-slate-400">{hint}</p>}
      </div>
      {children}
    </div>
  )
}

function StatusRow({ label, ok, okText, failText }) {
  return (
    <Row label={label}>
      <span className="flex shrink-0 items-center gap-1.5">
        <span className={`h-2 w-2 rounded-full ${ok ? 'bg-emerald-500' : 'bg-slate-300'}`} />
        <span className={`text-[12px] font-medium ${ok ? 'text-emerald-600' : 'text-slate-400'}`}>
          {ok ? okText : failText}
        </span>
      </span>
    </Row>
  )
}

function InputRow({ label, hint, type = 'text', value, onChange, placeholder, min, max, step }) {
  return (
    <Row label={label} hint={hint}>
      <input
        type={type}
        min={min}
        max={max}
        step={step}
        value={value ?? ''}
        onChange={onChange}
        placeholder={placeholder}
        className="w-40 shrink-0 bg-transparent text-right text-[15px] text-brand-600 placeholder:text-slate-300 focus:outline-none"
      />
    </Row>
  )
}

function SelectRow({ label, hint, value, onChange, options }) {
  return (
    <Row label={label} hint={hint}>
      <div className="relative shrink-0">
        <select
          value={value}
          onChange={onChange}
          className="appearance-none rounded-lg bg-slate-100 py-1.5 pl-3 pr-8 text-right text-[15px] font-medium text-brand-600 focus:outline-none"
        >
          {options.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        <ChevronDown
          size={14}
          className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 text-slate-400"
        />
      </div>
    </Row>
  )
}

function Switch({ checked, onChange }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={onChange}
      className={`relative h-[31px] w-[51px] shrink-0 rounded-full transition-colors duration-200 ${
        checked ? 'bg-brand-600' : 'bg-slate-300'
      }`}
    >
      <span
        className={`absolute top-[2px] h-[27px] w-[27px] rounded-full bg-white shadow-md transition-all duration-200 ${
          checked ? 'left-[22px]' : 'left-[2px]'
        }`}
      />
    </button>
  )
}

function ToggleRow({ label, hint, checked, onChange }) {
  return (
    <Row label={label} hint={hint}>
      <Switch checked={checked} onChange={onChange} />
    </Row>
  )
}