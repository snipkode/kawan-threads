import { Link } from 'react-router-dom'
import { Plus, Clock } from 'lucide-react'
import { Header } from '../../components/layout/Header'
import { Card } from '../../components/ui/Card'
import { StatusBadge, PillarBadge } from '../../components/ui/Badge'
import { useDashboard, useSchedules, useContentList } from '../../hooks/useApi'
import { formatClock, timeAgo, formatNumber } from '../../types'
import { Spinner, EmptyState } from '../../components/ui/Card'

const statCard = (label, value, sub) => (
  <Card className="p-3">
    <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">{label}</p>
    <p className="mt-1 text-2xl font-bold text-slate-900">{value}</p>
    {sub && <p className="text-[11px] text-slate-400">{sub}</p>}
  </Card>
)

export default function Dashboard() {
  const dash = useDashboard()
  const schedules = useSchedules()
  const content = useContentList({ limit: 6 })

  const d = dash.data || {}

  const upcoming = (schedules.data || [])
    .filter((s) => new Date(s.scheduled_at) >= new Date())
    .sort((a, b) => new Date(a.scheduled_at) - new Date(b.scheduled_at))

  return (
    <>
      <Header
        title="KAWAN AI"
        subtitle="Threads Automation"
        right={
          <Link to="/create" className="btn-primary px-3 py-2 text-xs">
            <Plus size={14} /> Generate
          </Link>
        }
      />

      <div className="space-y-4 p-4">
        {dash.isLoading ? (
          <Spinner label="Memuat dashboard…" />
        ) : (
          <>
            <div className="grid grid-cols-2 gap-2">
              {statCard('Posts', d.published_count ?? 0, 'published')}
              {statCard('Queue', d.queue_count ?? 0, 'waiting')}
              {statCard('Scheduled', d.scheduled_count ?? 0, 'upcoming')}
              {statCard('Views', formatNumber(d.total_views ?? 0), 'lifetime')}
            </div>

            <section>
              <h2 className="mb-2 flex items-center gap-1.5 text-sm font-bold text-slate-800">
                <Clock size={15} className="text-brand-600" /> Today&apos;s Schedule
              </h2>
              {upcoming.length === 0 ? (
                <EmptyState title="Belum ada jadwal" hint="Approved content akan dijadwalkan otomatis oleh AMAB." />
              ) : (
                <div className="space-y-2">
                  {upcoming.slice(0, 3).map((s) => (
                    <Link key={s.id} to={`/content/${s.content_id}/preview`}>
                      <Card className="flex items-center justify-between gap-3 py-3">
                        <div>
                          <p className="text-lg font-bold text-slate-900">{formatClock(s.scheduled_at)}</p>
                          <p className="text-xs text-slate-500">{new Date(s.scheduled_at).toLocaleDateString('id-ID')}</p>
                        </div>
                        <span className="chip bg-slate-100 text-slate-600">{s.scheduler_type}</span>
                      </Card>
                    </Link>
                  ))}
                </div>
              )}
            </section>

            <section>
              <div className="mb-2 flex items-center justify-between">
                <h2 className="text-sm font-bold text-slate-800">Recent Content</h2>
                <Link to="/content" className="text-xs font-semibold text-brand-600">
                  View all
                </Link>
              </div>
              {content.data?.length ? (
                <div className="divide-y divide-slate-100 overflow-hidden rounded-2xl bg-white ring-1 ring-slate-200">
                  {content.data.map((c) => (
                    <Link key={c.id} to={`/content/${c.id}/preview`} className="flex items-center justify-between gap-2 px-4 py-3 active:bg-slate-50">
                      <div className="min-w-0">
                        <p className="truncate text-[13px] font-medium leading-snug text-slate-800">
                          {c.hook || c.body}
                        </p>
                        <div className="mt-1 flex items-center gap-1">
                          <PillarBadge pillar={c.pillar} />
                          <StatusBadge status={c.status} />
                        </div>
                      </div>
                      <span className="shrink-0 text-[11px] text-slate-400">{timeAgo(c.created_at)}</span>
                    </Link>
                  ))}
                </div>
              ) : (
                <EmptyState
                  title="Belum ada konten"
                  hint="Mulai buat konten pertama dengan tombol Generate."
                />
              )}
            </section>
          </>
        )}
      </div>
    </>
  )
}