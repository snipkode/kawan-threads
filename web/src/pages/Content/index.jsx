import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Filter, ChevronLeft, ChevronRight } from 'lucide-react'
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

const PAGE_SIZE = 8

export default function ContentList() {
  const [status, setStatus] = useState('')
  const [pillar, setPillar] = useState('')
  const [page, setPage] = useState(0)
  const list = useContentList({ status: status || undefined, pillar: pillar || undefined, limit: PAGE_SIZE, offset: page * PAGE_SIZE })

  const items = list.data || []
  const hasMore = items.length === PAGE_SIZE
  const total = page * PAGE_SIZE + items.length

  const switchFilter = (setter) => (v) => {
    setter(v)
    setPage(0)
  }

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
        <div className="flex gap-1 rounded-xl bg-white p-1 ring-1 ring-slate-200">
          {TABS.map((t) => (
            <button
              key={t.value}
              type="button"
              onClick={() => switchFilter(setStatus)(t.value)}
              className={`min-w-0 flex-1 rounded-lg px-1 py-2 text-[11px] font-semibold leading-tight transition sm:text-xs ${
                status === t.value ? 'bg-brand-600 text-white' : 'text-slate-500'
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
            onChange={(e) => switchFilter(setPillar)(e.target.value)}
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
        ) : items.length ? (
          <>
            <div className="space-y-3">
              {items.map((c) => (
                <Link key={c.id} to={`/content/${c.id}/preview`}>
                  <Card className="py-3 mb-3">
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

            <div className="flex items-center justify-between gap-2 pt-1">
              <button
                className="btn-secondary px-3 py-2 text-xs"
                disabled={page === 0}
                onClick={() => setPage((p) => Math.max(0, p - 1))}
              >
                <ChevronLeft size={14} /> Sebelumnya
              </button>
              <span className="text-xs text-slate-400">Halaman {page + 1}</span>
              <button
                className="btn-secondary px-3 py-2 text-xs"
                disabled={!hasMore}
                onClick={() => setPage((p) => p + 1)}
              >
                Berikutnya <ChevronRight size={14} />
              </button>
            </div>
            {!hasMore && page > 0 && (
              <p className="text-center text-[11px] text-slate-400">
                {total} konten ditampilkan
              </p>
            )}
          </>
        ) : (
          <EmptyState title="Tidak ada konten" hint="Buat konten baru lewat tombol Create." />
        )}
      </div>
    </>
  )
}