import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Filter } from 'lucide-react'
import { Header } from '../../components/layout/Header'
import { Card, Spinner, EmptyState } from '../../components/ui/Card'
import { StatusBadge, PillarBadge } from '../../components/ui/Badge'
import { useContentList } from '../../hooks/useApi'
import { STATUS, PILLARS, timeAgo } from '../../types'

const TABS = [
  { value: '', label: 'Semua' },
  { value: STATUS.DRAFT, label: 'Draft' },
  { value: STATUS.QUEUED, label: 'Queued' },
  { value: STATUS.SCHEDULED, label: 'Jadwal' },
  { value: STATUS.PUBLISHED, label: 'Terbit' },
]

export default function ContentList() {
  const [status, setStatus] = useState('')
  const [pillar, setPillar] = useState('')
  const list = useContentList({ status: status || undefined, pillar: pillar || undefined, limit: 50 })

  return (
    <>
      <Header
        title="Content"
        subtitle="Semua konten KAWAN"
        right={
          <Link to="/create" className="btn-primary px-3 py-2 text-xs">
            <Plus size={14} /> Buat
          </Link>
        }
      />

      <div className="space-y-3 p-4">
        <div className="flex gap-1 overflow-x-auto pb-1">
          {TABS.map((t) => (
            <button
              key={t.value}
              onClick={() => setStatus(t.value)}
              className={`shrink-0 rounded-full px-3 py-1.5 text-xs font-semibold transition ${
                status === t.value
                  ? 'bg-brand-600 text-white'
                  : 'bg-white text-slate-500 ring-1 ring-slate-200'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3 py-2">
          <Filter size={14} className="text-slate-400" />
          <select
            value={pillar}
            onChange={(e) => setPillar(e.target.value)}
            className="flex-1 bg-transparent text-sm outline-none"
          >
            <option value="">Semua Pillar</option>
            {PILLARS.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </select>
        </div>

        {list.isLoading ? (
          <Spinner label="Memuat konten…" />
        ) : list.data?.length ? (
          <div className="space-y-2">
            {list.data.map((c) => (
              <Link key={c.id} to={`/content/${c.id}/preview`}>
                <Card className="py-3">
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-slate-800">
                        {c.hook || 'Tanpa hook'}
                      </p>
                      <p className="mt-1 line-clamp-2 text-xs text-slate-500">{c.body}</p>
                      <div className="mt-2 flex flex-wrap items-center gap-1.5">
                        <PillarBadge pillar={c.pillar} />
                        <StatusBadge status={c.status} />
                        {c.quality?.hook != null && (
                          <span className="chip bg-emerald-50 text-emerald-600">Q{c.quality.hook}</span>
                        )}
                      </div>
                    </div>
                    <span className="shrink-0 text-[11px] text-slate-400">{timeAgo(c.created_at)}</span>
                  </div>
                </Card>
              </Link>
            ))}
          </div>
        ) : (
          <EmptyState title="Tidak ada konten" hint="Buat konten baru lewat tombol Create." />
        )}
      </div>
    </>
  )
}