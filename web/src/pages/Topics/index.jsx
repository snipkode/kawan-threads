import { useState } from 'react'
import { Plus, Hash } from 'lucide-react'
import { Header } from '../../components/layout/Header'
import { Card, Spinner, EmptyState } from '../../components/ui/Card'
import { useTopics, useCreateTopic } from '../../hooks/useApi'
import { PILLARS, timeAgo, apiMessage } from '../../types'

export default function Topics() {
  const topics = useTopics()
  const create = useCreateTopic()
  const [name, setName] = useState('')
  const [pillar, setPillar] = useState('EDUKASI_HUKUM')

  const submit = (e) => {
    e.preventDefault()
    if (!name.trim()) return
    create.mutate(
      { name: name.trim(), pillar, active: true },
      { onSuccess: () => setName('') }
    )
  }

  return (
    <>
      <Header title="Topics" subtitle="Topik konten & pengetahuan KAWAN" />
      <div className="space-y-3 p-4">
        <form onSubmit={submit} className="card space-y-2">
          <label className="label">Tambah Topik</label>
          <input
            className="input"
            placeholder="Nama topik (contoh: KAWAN 2026)"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <select className="input" value={pillar} onChange={(e) => setPillar(e.target.value)}>
            {PILLARS.map((p) => (
              <option key={p.value} value={p.value}>{p.label}</option>
            ))}
          </select>
          {create.isError && (
            <p className="text-xs text-rose-600">{apiMessage(create.error)}</p>
          )}
          <button type="submit" disabled={!name.trim() || create.isPending} className="btn-primary w-full">
            <Plus size={15} /> {create.isPending ? 'Menyimpan…' : 'Tambah Topik'}
          </button>
        </form>

        {topics.isLoading ? (
          <Spinner label="Memuat topik…" />
        ) : topics.data?.length ? (
          <div className="space-y-2">
            {topics.data.map((t) => (
              <Card key={t.id} className="flex items-center justify-between py-3">
                <div className="flex min-w-0 items-center gap-2">
                  <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-brand-50 text-brand-600">
                    <Hash size={15} />
                  </span>
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-slate-800">{t.name}</p>
                    <p className="text-[11px] text-slate-400">
                      {PILLARS.find((p) => p.value === t.pillar)?.label || t.pillar} · {timeAgo(t.created_at)}
                    </p>
                  </div>
                </div>
                <span className={`chip ${t.active ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-500'}`}>
                  {t.active ? 'Aktif' : 'Nonaktif'}
                </span>
              </Card>
            ))}
          </div>
        ) : (
          <EmptyState title="Belum ada topik" hint="Tambahkan topik agar tim punya panduan konten bersama." />
        )}
      </div>
    </>
  )
}