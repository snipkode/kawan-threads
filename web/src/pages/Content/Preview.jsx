import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
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

  const c = preview.data
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
    <>
      <BackHeader to="/content" title="Content" />
      <div className="space-y-3 p-4">
        {preview.isLoading && <Spinner label="Memuat preview…" />}
        {preview.isError && <ErrorBox message={apiMessage(preview.error)} />}

        {c && (
          <>
            <div className="flex items-center justify-between gap-2">
              <div className="flex flex-wrap items-center gap-1.5">
                <PillarBadge pillar={c.pillar} />
                <StatusBadge status={c.status} />
              </div>
              <QualityScore quality={c.quality} conversation={c.conversation_score} />
            </div>

            <p className="text-xs text-slate-500">
              Topic: {c.topic} · {c.audience} · {c.tone} · {c.format}
            </p>

            {!editing && (
              <ThreadPreview content={c} onPickHook={isDraft ? setPickedHook : null} pickedHook={pickedHook?.hook} />
            )}

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
                  Simpan
                </button>
              </Card>
            )}

            {c.scheduled_at && (
              <Card className="flex items-center justify-between py-3">
                <div>
                  <p className="text-xs font-semibold text-slate-500">Dijadwalkan</p>
                  <p className="text-sm font-bold text-slate-800">{formatTime(c.scheduled_at)}</p>
                </div>
              </Card>
            )}

            {c.approval && (
              <Card className="py-3 text-xs text-slate-500">
                <p className="font-semibold text-slate-600">Catatan Review</p>
                <p>{c.approval.reason || 'Approved otomatis — siap masuk antrian.'}</p>
              </Card>
            )}

            {/* Action bar */}
            {isDraft && (
              <div className="grid grid-cols-2 gap-2">
                {!editing ? (
                  <button className="btn-secondary" onClick={startEdit}>
                    <Pencil size={15} /> Edit
                  </button>
                ) : (
                  <button className="btn-secondary" onClick={() => { setEditing(false); setDraft(null) }}>
                    <PencilOff size={15} /> Batal
                  </button>
                )}
                <button
                  className="btn-secondary"
                  onClick={() => regen.mutate({})}
                  disabled={regen.isPending}
                >
                  <RefreshCw size={15} /> {regen.isPending ? '…' : 'Regenerate'}
                </button>
                <button className="btn-danger" onClick={() => setRejecting((v) => !v)}>
                  <X size={15} /> Reject
                </button>
                <button className="btn-primary" onClick={() => approve.mutate()} disabled={approve.isPending}>
                  <Check size={15} /> {approve.isPending ? '…' : 'Approve & Queue'}
                </button>
              </div>
            )}

            {rejecting && (
              <Card className="space-y-2">
                <label className="label">Alasan reject</label>
                <textarea rows={2} className="input" value={reason} onChange={(e) => setReason(e.target.value)} />
                <button
                  className="btn-danger w-full"
                  disabled={!reason.trim() || reject.isPending}
                  onClick={() =>
                    reject.mutate(reason, { onSuccess: () => { setRejecting(false); setReason('') } })
                  }
                >
                  {reject.isPending ? '…' : 'Konfirmasi Reject'}
                </button>
              </Card>
            )}
          </>
        )}
      </div>
    </>
  )
}