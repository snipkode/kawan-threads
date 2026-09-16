import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { Check, X, RefreshCw, Pencil, PencilOff } from 'lucide-react'
import { BackHeader } from '../../components/layout/Header'
import { Card, Spinner, ErrorBox } from '../../components/ui/Card'
import { ThreadPreview } from '../../components/ui/ThreadPreview'
import { StatusBadge, PillarBadge, QualityScore } from '../../components/ui/Badge'
import {
  usePreview,
  useApprove,
  useReject,
  useRegenerate,
  useUpdateContent,
} from '../../hooks/useApi'
import { apiMessage, formatTime } from '../../types'

export default function ContentPreview() {
  const { id } = useParams()
  const preview = usePreview(id)
  const approve = useApprove(id)
  const reject = useReject(id)
  const regen = useRegenerate(id)
  const update = useUpdateContent(id)

  const [editing, setEditing] = useState(false)
  const [rejecting, setRejecting] = useState(false)
  const [reason, setReason] = useState('')
  const [pickedHook, setPickedHook] = useState(null)
  const [draft, setDraft] = useState(null)

  const c = preview.data?.content
  const isDraft = c?.status === 'DRAFT'

  const startEdit = () => {
    setDraft({ hook: c.hook, hook_type: c.hook_type, body: c.body, cta: c.cta, conversation_question: c.conversation_question })
    setEditing(true)
  }

  const saveEdit = () => {
    if (!draft) return
    update.mutate(draft, { onSuccess: () => { setEditing(false); setDraft(null) } })
  }

  return (
    <div className="flex flex-col min-h-screen">
      <BackHeader to="/content" title="Preview" />

      <div className="flex-1 space-y-3 p-4 pb-28">
        {preview.isLoading && <Spinner label="Memuat preview…" />}
        {preview.isError && <ErrorBox message={apiMessage(preview.error)} />}

        {c && (
          <>
            {/* Meta info */}
            <div className="rounded-2xl bg-white px-4 py-3 ring-1 ring-slate-200">
              <div className="flex items-center justify-between gap-2 mb-1.5">
                <div className="flex flex-wrap items-center gap-1">
                  <PillarBadge pillar={c.pillar} />
                  <StatusBadge status={c.status} />
                </div>
                <QualityScore quality={c.quality} conversation={c.conversation_score} />
              </div>
              <p className="text-[11px] text-slate-400 leading-relaxed">
                {[c.topic, c.audience, c.tone, c.format].filter(Boolean).join(' · ')}
              </p>
            </div>

            {/* Thread preview */}
            {!editing && (
              <ThreadPreview content={c} onPickHook={isDraft ? setPickedHook : null} pickedHook={pickedHook?.hook} />
            )}

            {/* Edit form */}
            {editing && (
              <Card className="space-y-3">
                <div>
                  <label className="label">Hook</label>
                  <input className="input" value={draft.hook || ''} onChange={(e) => setDraft({ ...draft, hook: e.target.value })} />
                </div>
                <div>
                  <label className="label">Body</label>
                  <textarea rows={6} className="input" value={draft.body || ''} onChange={(e) => setDraft({ ...draft, body: e.target.value })} />
                </div>
                <div>
                  <label className="label">CTA</label>
                  <input className="input" value={draft.cta || ''} onChange={(e) => setDraft({ ...draft, cta: e.target.value })} />
                </div>
                <div>
                  <label className="label">Conversation Question</label>
                  <input className="input" value={draft.conversation_question || ''} onChange={(e) => setDraft({ ...draft, conversation_question: e.target.value })} />
                </div>
                <button className="btn-primary w-full" onClick={saveEdit} disabled={update.isPending}>
                  {update.isPending ? '…' : 'Simpan'}
                </button>
              </Card>
            )}

            {/* Scheduled info */}
            {c.scheduled_at && (
              <div className="flex items-center gap-2 rounded-2xl bg-white px-4 py-3 ring-1 ring-slate-200">
                <div className="flex-1">
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Dijadwalkan</p>
                  <p className="text-sm font-bold text-slate-800">{formatTime(c.scheduled_at)}</p>
                </div>
              </div>
            )}

            {/* Review note */}
            {c.approval && (
              <div className="rounded-2xl bg-white px-4 py-3 ring-1 ring-slate-200">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-400 mb-1">Catatan Review</p>
                <p className="text-[13px] text-slate-600">{c.approval.reason || 'Approved otomatis — siap masuk antrian.'}</p>
              </div>
            )}

            {/* Reject form */}
            {rejecting && (
              <Card className="space-y-2">
                <label className="label">Alasan reject</label>
                <textarea rows={2} className="input" placeholder="Tulis alasan…" value={reason} onChange={(e) => setReason(e.target.value)} />
                <div className="flex gap-2">
                  <button
                    className="btn-secondary flex-1"
                    onClick={() => { setRejecting(false); setReason('') }}
                  >
                    Batal
                  </button>
                  <button
                    className="btn-danger flex-1"
                    disabled={!reason.trim() || reject.isPending}
                    onClick={() =>
                      reject.mutate(reason, { onSuccess: () => { setRejecting(false); setReason('') } })
                    }
                  >
                    {reject.isPending ? '…' : 'Konfirmasi'}
                  </button>
                </div>
              </Card>
            )}
          </>
        )}
      </div>

      {/* Sticky action bar */}
      {c && isDraft && (
        <div className="fixed bottom-0 left-0 right-0 z-20 border-t border-slate-200 bg-white/95 backdrop-blur-sm px-4 py-3 safe-area-bottom">
          <div className="flex items-center gap-2">
            {/* Edit / Batal */}
            <button
              className="btn-secondary flex-none px-3 py-2"
              onClick={editing ? () => { setEditing(false); setDraft(null) } : startEdit}
              title={editing ? 'Batal edit' : 'Edit'}
            >
              {editing ? <PencilOff size={16} /> : <Pencil size={16} />}
            </button>

            {/* Regenerate */}
            <button
              className="btn-secondary flex-none px-3 py-2"
              onClick={() => regen.mutate({})}
              disabled={regen.isPending}
              title="Regenerate"
            >
              <RefreshCw size={16} className={regen.isPending ? 'animate-spin' : ''} />
            </button>

            {/* Reject */}
            <button
              className="btn-danger flex-1 py-2"
              onClick={() => setRejecting((v) => !v)}
            >
              <X size={15} /> Reject
            </button>

            {/* Approve */}
            <button
              className="btn-primary flex-1 py-2"
              onClick={() => approve.mutate()}
              disabled={approve.isPending}
            >
              <Check size={15} /> {approve.isPending ? '…' : 'Approve'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}