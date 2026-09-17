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

  const toggle = (k) => () => setForm({ ...s, [k]: !s[k] })

  const save = () => update.mutate(form ?? s, { onSuccess: () => setForm(null) })

  return (
    <div className="font-sans">
      <Header title="Settings" subtitle="Konfigurasi platform (berlaku langsung)" />
      <div className="space-y-2 p-4">
        {settings.isLoading ? (
          <Spinner label="Memuat settings…" />
        ) : (
          <>
            {update.isError && <ErrorBox message={apiMessage(update.error)} />}

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
            <p className="px-4 text-[12px] leading-relaxed text-[#8E8E93]">
              API key &amp; model dibaca saat request; simpan lalu buat konten baru untuk memakai.
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
            <p className="px-4 text-[12px] leading-relaxed text-[#8E8E93]">
              Kredensial OAuth. Token bisa didapat lewat alur OAuth {`(/api/auth/threads)`} dan
              tersimpan otomatis.
            </p>

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
            <p className="px-4 text-[12px] leading-relaxed text-[#8E8E93]">
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

            <Section
              title="Autopilot"
              titleRight={
                <span
                  className={`rounded-full px-2.5 py-0.5 text-[12px] font-semibold ${
                    s.autopilot_enabled
                      ? 'bg-[#34C759]/15 text-[#248A3D]'
                      : 'bg-[#E9E9EB] text-[#8E8E93]'
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
            <p className="px-4 text-[12px] leading-relaxed text-[#8E8E93]">
              Worker generate draft konten setiap hari tanpa konfigurasi manual. Semua output
              tetap <b>DRAFT</b> — perlu review sebelum publish. Goal menentukan arah CTA konten.
            </p>

            {st.data_store && (
              <p className="px-4 text-[12px] leading-relaxed text-[#8E8E93]">
                Datastore: <span className="font-mono">{st.data_store}</span> — mode penyimpanan
                &amp; Firebase service account hanya diatur via environment (tidak lewat UI).
              </p>
            )}

            <button
              className="mt-5 flex w-full items-center justify-center gap-2 rounded-full bg-[#007AFF] py-3 text-[15px] font-semibold text-white shadow-[0_8px_24px_rgba(0,122,255,0.3)] transition hover:brightness-105 active:scale-[0.98] disabled:opacity-50"
              onClick={save}
              disabled={update.isPending}
            >
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
    </div>
  )
}

function Section({ title, titleRight, children }) {
  return (
    <section>
      {(title || titleRight) && (
        <div className="flex items-center justify-between px-4 pb-1.5 pt-3">
          <h3 className="text-[13px] font-semibold uppercase tracking-wide text-[#8E8E93]">
            {title}
          </h3>
          {titleRight}
        </div>
      )}
      <div className="divide-y divide-[#E5E5EA] overflow-hidden rounded-2xl bg-white ring-1 ring-black/5">
        {children}
      </div>
    </section>
  )
}

function Row({ label, hint, children }) {
  return (
    <div className="flex min-h-[46px] items-center gap-3 bg-white px-4 py-2.5 active:bg-[#F2F2F7]">
      <div className="min-w-0 flex-1">
        <p className="text-[15px] leading-tight text-black">{label}</p>
        {hint && <p className="mt-0.5 text-[12px] leading-tight text-[#8E8E93]">{hint}</p>}
      </div>
      {children}
    </div>
  )
}

function StatusRow({ label, ok, okText, failText }) {
  return (
    <Row label={label}>
      <span className="flex shrink-0 items-center gap-1.5">
        <span className={`h-2 w-2 rounded-full ${ok ? 'bg-[#34C759]' : 'bg-[#C7C7CC]'}`} />
        <span
          className={`text-[13px] font-medium ${ok ? 'text-[#248A3D]' : 'text-[#8E8E93]'}`}
        >
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
        className="w-40 shrink-0 bg-transparent text-right text-[15px] text-[#007AFF] placeholder:text-[#C7C7CC] focus:outline-none"
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
          className="appearance-none rounded-lg bg-[#F2F2F7] py-1.5 pl-3 pr-8 text-right text-[15px] font-medium text-[#007AFF] focus:outline-none"
        >
          {options.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        <ChevronDown
          size={14}
          className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 text-[#C7C7CC]"
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
        checked ? 'bg-[#34C759]' : 'bg-[#E9E9EB]'
      }`}
    >
      <span
        className={`absolute top-[2px] h-[27px] w-[27px] rounded-full bg-white shadow-[0_3px_8px_rgba(0,0,0,0.2)] transition-all duration-200 ${
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