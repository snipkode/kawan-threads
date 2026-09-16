import { useState } from 'react'
import { Link } from 'react-router-dom'
import { CalendarDays, X } from 'lucide-react'
import { Header } from '../../components/layout/Header'
import { Card, Spinner, EmptyState } from '../../components/ui/Card'
import { StatusBadge } from '../../components/ui/Badge'
import { useSchedules, useCancelSchedule } from '../../hooks/useApi'
import { formatTime } from '../../types'

export default function Schedule() {
  const schedules = useSchedules()
  const cancel = useCancelSchedule()
  const [tab, setTab] = useState('upcoming')

  const list = (schedules.data || [])
    .filter((s) => (tab === 'upcoming' ? new Date(s.scheduled_at) >= new Date() : new Date(s.scheduled_at) < new Date()))
    .sort((a, b) => new Date(a.scheduled_at) - new Date(b.scheduled_at))

  return (
    <>
      <Header title="Schedule" subtitle="Jadwal posting AMAB" />
      <div className="space-y-3 p-4">
        <div className="flex gap-1 rounded-xl bg-white p-1 ring-1 ring-slate-200">
          {['upcoming', 'past'].map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`flex-1 rounded-lg py-1.5 text-xs font-semibold transition ${
                tab === t ? 'bg-brand-600 text-white' : 'text-slate-500'
              }`}
            >
              {t === 'upcoming' ? 'Akan Datang' : 'Riwayat'}
            </button>
          ))}
        </div>

        {schedules.isLoading ? (
          <Spinner label="Memuat jadwal…" />
        ) : !list.length ? (
          <EmptyState title="Tidak ada jadwal" hint="AMAB akan menjadwalkan konten dari antrian secara otomatis." />
        ) : (
          <div className="space-y-2">
            {list.map((s) => (
              <Card key={s.id}>
                <div className="flex items-center justify-between gap-2">
                  <Link to={`/content/${s.content_id}/preview`} className="min-w-0">
                    <p className="truncate text-sm font-medium text-slate-800">{s.hook || 'Content'}</p>
                    <p className="mt-0.5 flex items-center gap-1 text-xs text-slate-500">
                      <CalendarDays size={12} /> {formatTime(s.scheduled_at)}
                    </p>
                  </Link>
                  <div className="flex shrink-0 items-center gap-1.5">
                    <span className={`chip ${s.type === 'AUTO' ? 'bg-blue-100 text-blue-700' : 'bg-violet-100 text-violet-700'}`}>
                      {s.type || s.scheduler_type || 'AUTO'}
                    </span>
                    {tab === 'upcoming' && s.type !== 'AUTO' && (
                      <button
                        className="btn-secondary px-2 py-1 text-xs"
                        onClick={() => cancel.mutate(s.content_id)}
                        disabled={cancel.isPending}
                      >
                        <X size={13} />
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