import { STATUS, STATUS_STYLES, PILLARS } from '../../types'

const STATUS_LABEL = {
  [STATUS.DRAFT]: 'Draft',
  [STATUS.REJECTED]: 'Rejected',
  [STATUS.QUEUED]: 'Queued',
  [STATUS.SCHEDULED]: 'Scheduled',
  [STATUS.PUBLISHING]: 'Publishing',
  [STATUS.PUBLISHED]: 'Published',
  [STATUS.FAILED]: 'Failed',
}

export function StatusBadge({ status, className = '' }) {
  return (
    <span className={`chip ${STATUS_STYLES[status] || 'bg-slate-100 text-slate-600'} ${className}`}>
      {STATUS_LABEL[status] || status}
    </span>
  )
}

export function PillarBadge({ pillar }) {
  const label = PILLARS.find((p) => p.value === pillar)?.label || pillar || '—'
  return <span className="chip bg-slate-100 text-slate-600">{label}</span>
}

const qualityColor = (v) => {
  if (v == null) return 'text-slate-400'
  if (v >= 85) return 'text-emerald-600'
  if (v >= 70) return 'text-amber-600'
  return 'text-rose-600'
}

export function QualityScore({ quality, conversation }) {
  const main = quality?.relevance ?? quality?.hook ?? 0
  return (
    <div className="flex items-center gap-3 text-xs">
      {quality && (
        <span>
          Quality <b className={qualityColor(main)}>{main}</b>
        </span>
      )}
      {conversation != null && (
        <span>
          Conversation <b className={qualityColor(conversation)}>{conversation}</b>
        </span>
      )}
    </div>
  )
}