import { Header } from '../../components/layout/Header'
import { Card, Spinner, EmptyState } from '../../components/ui/Card'
import {
  useAnalytics,
  useAnalyticsHour,
  useAnalyticsPillar,
} from '../../hooks/useApi'
import { formatNumber, pillarLabel } from '../../types'

const pct = (part, total) => (total > 0 ? Math.round(((part || 0) / total) * 100) : 0)

export default function Analytics() {
  const overview = useAnalytics()
  const hour = useAnalyticsHour()
  const pillar = useAnalyticsPillar()

  const o = overview.data || {}
  const total = (o.total_likes || 0) + (o.total_replies || 0) + (o.total_reposts || 0)

  const hours = hour.data || []
  const maxHour = Math.max(1, ...hours.map((h) => h.posts || 0))

  const pillars = pillar.data || []
  const maxPillar = Math.max(1, ...pillars.map((p) => p.posts || 0))

  const stat = (label, value, sub) => (
    <Card className="p-3">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">{label}</p>
      <p className="mt-1 text-xl font-bold text-slate-900">{value}</p>
      {sub && <p className="text-[11px] text-slate-400">{sub}</p>}
    </Card>
  )

  return (
    <>
      <Header title="Analytics" subtitle="Performansi konten" />
      <div className="space-y-4 p-4">
        {overview.isLoading ? (
          <Spinner label="Memuat analytics…" />
        ) : (
          <>
            <div className="grid grid-cols-2 gap-2">
              {stat('Views', formatNumber(o.total_views))}
              {stat('Likes', formatNumber(o.total_likes))}
              {stat('Replies', formatNumber(o.total_replies))}
              {stat('Reposts', formatNumber(o.total_reposts))}
              {stat('Posts', o.total_posts ?? 0)}
              {stat('Engagement', o.total_engagement ?? 0)}
            </div>

            <Card>
              <h3 className="mb-3 text-sm font-bold text-slate-800">Best Posting Hours</h3>
              {hours.length ? (
                <div className="flex items-end gap-1" style={{ height: '120px' }}>
                  {hours.map((h) => (
                    <div key={h.hour} className="flex flex-1 flex-col items-center gap-1">
                      <span className="text-[9px] text-slate-400">{h.posts || ''}</span>
                      <div
                        className="w-full rounded-t-md bg-brand-600/80"
                        style={{ height: `${Math.max(4, ((h.posts || 0) / maxHour) * 90)}px` }}
                      />
                      <span className="text-[9px] text-slate-400">{h.hour}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <EmptyState title="Belum ada data" hint="Posting & analytics belum tersedia." />
              )}
            </Card>

            <Card>
              <h3 className="mb-3 text-sm font-bold text-slate-800">Pillar Distribution</h3>
              {pillars.length ? (
                <div className="space-y-2.5">
                  {pillars.map((p) => (
                    <div key={p.pillar}>
                      <div className="mb-1 flex justify-between text-xs">
                        <span className="font-medium text-slate-600">{pillarLabel(p.pillar)}</span>
                        <span className="text-slate-400">{p.posts} post</span>
                      </div>
                      <div className="h-2 overflow-hidden rounded-full bg-slate-100">
                        <div
                          className="h-full rounded-full bg-gradient-to-r from-brand-500 to-brand-700"
                          style={{ width: `${pct(p.posts, maxPillar)}%` }}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <EmptyState title="Belum ada data" hint="Posting per pillar akan muncul di sini." />
              )}
            </Card>

            <Card>
              <h3 className="mb-2 text-sm font-bold text-slate-800">Summary</h3>
              <p className="text-xs text-slate-500">
                {total || 0} total engagement dari {o.total_posts ?? 0} post yang terbit.
                {o.total_replies > 0 && ` Konversasi menyumbang ${pct(o.total_replies, total)}% engagement.`}
              </p>
            </Card>
          </>
        )}
      </div>
    </>
  )
}