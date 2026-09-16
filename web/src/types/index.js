export const STATUS = {
  DRAFT: 'DRAFT',
  REJECTED: 'REJECTED',
  QUEUED: 'QUEUED',
  SCHEDULED: 'SCHEDULED',
  PUBLISHING: 'PUBLISHING',
  PUBLISHED: 'PUBLISHED',
  FAILED: 'FAILED',
}

export const PILLARS = [
  { value: 'EDUKASI_HUKUM', label: 'Edukasi Hukum' },
  { value: 'TIPS_HUKUM', label: 'Tips Hukum' },
  { value: 'CERITA_WARGA', label: 'Cerita Warga' },
  { value: 'MYTH_VS_FACT', label: 'Myth vs Fact' },
  { value: 'COMMUNITY', label: 'Community' },
  { value: 'EVENT', label: 'Event' },
  { value: 'TRAINING', label: 'Training' },
  { value: 'MEMBERSHIP', label: 'Membership' },
  { value: 'ENGAGEMENT', label: 'Engagement' },
]

export const AUDIENCES = ['Gen Z', 'Millennial', 'Mahasiswa', 'Pekerja', 'UMKM', 'Masyarakat umum']

export const TONES = ['casual', 'friendly', 'edukatif', 'santai', 'inspiratif', 'human']

export const FORMATS = [
  { value: 'SINGLE_POST', label: 'Single Post' },
  { value: 'THREAD_SERIES', label: 'Thread Series' },
]

export const HOOK_TYPES = [
  'QUESTION',
  'RELATABLE',
  'CURIOSITY',
  'STORY',
  'MYTH',
  'PROBLEM',
  'SURPRISING_FACT',
  'HOT_TAKE',
]

export const PRIORITIES = [
  { value: 100, label: 'Event' },
  { value: 80, label: 'Campaign' },
  { value: 50, label: 'Normal' },
  { value: 20, label: 'Evergreen' },
]

export const STATUS_STYLES = {
  DRAFT: 'bg-slate-100 text-slate-700',
  REJECTED: 'bg-rose-100 text-rose-700',
  QUEUED: 'bg-amber-100 text-amber-700',
  SCHEDULED: 'bg-blue-100 text-blue-700',
  PUBLISHING: 'bg-indigo-100 text-indigo-700',
  PUBLISHED: 'bg-emerald-100 text-emerald-700',
  FAILED: 'bg-rose-100 text-rose-700',
}

export const pillarLabel = (value) =>
  PILLARS.find((p) => p.value === value)?.label || value || '—'

export const permissionDropdown = (arr) => arr

export const timeAgo = (iso) => {
  if (!iso) return '—'
  const then = new Date(iso).getTime()
  const diff = Date.now() - then
  const min = Math.floor(diff / 60000)
  if (min < 1) return 'just now'
  if (min < 60) return `${min}m ago`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}h ago`
  const day = Math.floor(hr / 24)
  return `${day}d ago`
}

export const formatTime = (iso) => {
  if (!iso) return '—'
  return new Date(iso).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' })
}

export const formatClock = (iso) => {
  if (!iso) return '—'
  return new Date(iso).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
}

export const formatNumber = (n) => {
  if (n == null) return '0'
  return new Intl.NumberFormat('en', { notation: 'compact', maximumFractionDigits: 1 }).format(n)
}

export const apiMessage = (err, fallback = 'Terjadi kesalahan') => {
  return err?.response?.data?.message || err?.message || fallback
}