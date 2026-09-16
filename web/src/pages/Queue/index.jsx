import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Trash2, ArrowUpDown } from 'lucide-react'
import { Header } from '../../components/layout/Header'
import { Card, Spinner, EmptyState } from '../../components/ui/Card'
import { PillarBadge } from '../../components/ui/Badge'
import { useQueue, useRemoveFromQueue, useSetPriority } from '../../hooks/useApi'
import { PRIORITIES, formatTime } from '../../types'

export default function Queue() {
  const queue = useQueue()
  const remove = useRemoveFromQueue()
  const setPrio = useSetPriority()

  const [confirmId, setConfirmId] = useState(null)

  const sorted = (queue.data || []).slice().sort((a, b) => (b.priority || 0) - (a.priority || 0))

  const priorityLabel = (p) => PRIORITIES.find((x) => x.value === p)?.label || idToLabel(p)

  return (
    <>
      <Header title="Queue" subtitle="Antrian konten menunggu jadwal AMAB" />
      <div className="space-y-3 p-4">
        {queue.isLoading ? (
          <Spinner label="Memuat antrian…" />
        ) : !sorted.length ? (
          <EmptyState title="Antrian kosong" hint="Approved content akan masuk ke sini untuk dijadwalkan AMAB." />
        ) : (
          <div className="space-y-2">
            {sorted.map((q) => (
              <Card key={q.id}>
                <div className="flex flex-col gap-2">
                  <Link to={`/content/${q.content_id}/preview`} className="group">
                    <p className="text-sm font-medium text-slate-800 group-hover:text-brand-600">
                      {q.hook || 'Tanpa hook'}
                    </p>
                    <p className="mt-0.5 line-clamp-1 text-xs text-slate-500">{q.body}</p>
                  </Link>
                  <div className="flex flex-wrap items-center gap-1.5">
                    <PillarBadge pillar={q.pillar} />
                    <span className="chip bg-amber-100 text-amber-700">
                      #{q.priority} {priorityLabel(q.priority)}
                    </span>
                    {q.queued_at && (
                      <span className="text-[11px] text-slate-400">Added {formatTime(q.queued_at)}</span>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="flex flex-1 items-center gap-1 rounded-lg border border-slate-200 px-2 py-1">
                      <ArrowUpDown size={13} className="text-slate-400" />
                      <select
                        value={q.priority ?? 50}
                        onChange={(e) => setPrio.mutate({ id: q.id, priority: Number(e.target.value) })}
                        className="w-full bg-transparent text-xs font-semibold outline-none"
                      >
                        {PRIORITIES.map((p) => (
                          <option key={p.value} value={p.value}>{p.label}</option>
                        ))}
                      </select>
                    </div>
                    {confirmId === q.id ? (
                      <button
                        className="btn-danger px-3 py-1.5 text-xs"
                        onClick={() => remove.mutate(q.id, { onSuccess: () => setConfirmId(null) })}
                      >
                        Yakin? Hapus
                      </button>
                    ) : (
                      <button
                        className="btn-secondary px-3 py-1.5 text-xs"
                        onClick={() => setConfirmId(q.id)}
                      >
                        <Trash2 size={13} /> Keluar
                      </button>
                    )}
                  </div>
                </div>
              </Card>
            ))}
          </div>
        )}
      </div>
    </>
  )
}

const idToLabel = (p) => (p == null ? '' : `Prioritas ${p}`)