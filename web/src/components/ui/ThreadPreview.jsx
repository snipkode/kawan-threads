import { Card } from './Card'

// ─── Rich text renderer ───────────────────────────────────────────────────────
// Detects numbered lists (1. ...), bullet lists (- / • ...), and plain paragraphs.
// Splits by newline, groups consecutive same-type lines into <ol> or <ul>.

function parseSegments(text) {
  if (!text) return []
  const lines = text.split('\n').map((l) => l.trimEnd())
  const segments = []

  for (const line of lines) {
    const ol = line.match(/^(\d+)\.\s+(.*)$/)
    const ul = line.match(/^[-•]\s+(.*)$/)
    if (ol) {
      segments.push({ type: 'ol', num: parseInt(ol[1], 10), text: ol[2] })
    } else if (ul) {
      segments.push({ type: 'ul', text: ul[1] })
    } else {
      segments.push({ type: 'p', text: line })
    }
  }
  return segments
}

function RichText({ text, className = '' }) {
  const segments = parseSegments(text)
  const nodes = []
  let i = 0

  while (i < segments.length) {
    const seg = segments[i]

    if (seg.type === 'ol') {
      const items = []
      while (i < segments.length && segments[i].type === 'ol') {
        items.push(segments[i])
        i++
      }
      nodes.push(
        <ol key={`ol-${i}`} className="mt-2 space-y-2.5 pl-0 list-none">
          {items.map((item, idx) => (
            <li key={idx} className="flex gap-2.5 items-start">
              <span className="flex-none w-5 text-right font-bold text-slate-400 font-serif text-[13px] pt-0.5">{item.num}.</span>
              <span className="flex-1 font-serif text-[13.5px] leading-snug" lang="id">
                {renderInline(item.text)}
              </span>
            </li>
          ))}
        </ol>
      )
    } else if (seg.type === 'ul') {
      const items = []
      while (i < segments.length && segments[i].type === 'ul') {
        items.push(segments[i])
        i++
      }
      nodes.push(
        <ul key={`ul-${i}`} className="mt-2 space-y-2.5 pl-0 list-none">
          {items.map((item, idx) => (
            <li key={idx} className="flex gap-2.5 items-start">
              <span className="flex-none text-slate-300 font-bold text-[13px] pt-0.5">–</span>
              <span className="flex-1 font-serif text-[13.5px] leading-snug" lang="id">
                {renderInline(item.text)}
              </span>
            </li>
          ))}
        </ul>
      )
    } else {
      // blank line = spacer, text = paragraph
      if (seg.text.trim()) {
        nodes.push(
          <p key={`p-${i}`} className={`font-serif text-[13.5px] leading-relaxed text-slate-700 text-justify hyphens-auto ${nodes.length > 0 ? 'mt-2' : ''}`} lang="id">
            {renderInline(seg.text)}
          </p>
        )
      }
      i++
    }
  }

  return <div className={className}>{nodes}</div>
}

// Bold **text** and em-dash highlight: "Term — description" renders as two lines
function renderInline(text) {
  // Split on **bold**
  const parts = text.split(/(\*\*[^*]+\*\*)/g)

  // Check if entire text matches "Term — description" pattern (em-dash or en-dash)
  const em = text.match(/^(.+?)\s+[—–]\s+(.+)$/)
  if (em) {
    return (
      <span className="block">
        <strong className="font-bold text-slate-900">{em[1]}</strong>
        <span className="block mt-0.5 text-slate-500 text-[12.5px] font-normal">{em[2]}</span>
      </span>
    )
  }

  return parts.map((part, i) => {
    if (part.startsWith('**') && part.endsWith('**')) {
      return <strong key={i} className="font-bold text-slate-900">{part.slice(2, -2)}</strong>
    }
    return <span key={i}>{part}</span>
  })
}
// ─────────────────────────────────────────────────────────────────────────────

export function ThreadPreview({ content, onPickHook, pickedHook }) {
  const hook = pickedHook || content.hook

  return (
    <div className="mx-auto w-full max-w-sm">
      <Card className="overflow-hidden px-0 pt-0 pb-0">

        {/* ── Masthead ── */}
        <div className="bg-slate-900 px-4 pt-3 pb-2.5">
          <p className="text-center text-[9px] font-semibold uppercase tracking-[0.25em] text-slate-400 mb-2">
            {content.pillar || 'Komunitas'} · {content.topic || 'KAWAN'}
          </p>
          <div className="border-t border-slate-700 mb-2" />
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="flex h-7 w-7 items-center justify-center rounded-full bg-white text-xs font-black text-slate-900">
                K
              </div>
              <div>
                <p className="text-xs font-bold leading-none text-white tracking-wide">KAWAN</p>
                <p className="text-[10px] text-slate-400 mt-0.5">@kawan · Threads</p>
              </div>
            </div>
            <p className="text-[10px] text-slate-500">
              {new Date().toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' })}
            </p>
          </div>
        </div>

        {/* ── Hook variant selector ── */}
        {onPickHook && content.hook_variants?.length > 0 && (
          <div className="flex flex-wrap gap-1.5 border-b border-slate-100 px-4 py-2 bg-slate-50">
            {content.hook_variants.map((v) => (
              <button
                key={v.hook}
                type="button"
                onClick={() => onPickHook(v)}
                className={`chip border transition ${
                  hook === v.hook
                    ? 'border-brand-600 bg-brand-50 text-brand-700'
                    : 'border-slate-200 bg-white text-slate-500 hover:border-slate-300'
                }`}
              >
                {v.hook_type?.toLowerCase()}
              </button>
            ))}
          </div>
        )}

        {/* ── Editorial body ── */}
        <div className="px-4 py-4">
          {hook && (
            <h2
              className="mb-2 font-serif text-[18px] font-bold leading-snug tracking-tight text-slate-900 hyphens-auto"
              lang="id"
            >
              {hook}
            </h2>
          )}

          <div className="border-t-2 border-slate-900 mb-3" />

          {content.body && <RichText text={content.body} />}

          {content.cta && (
            <p className="mt-3 font-serif text-[13px] font-semibold text-slate-800">
              {content.cta}
            </p>
          )}

          {content.conversation_question && (
            <p
              className="mt-3 border-l-2 border-slate-300 pl-3 font-serif text-[13px] italic text-slate-600 text-justify hyphens-auto"
              lang="id"
            >
              {content.conversation_question}
            </p>
          )}
        </div>

        {/* ── Footer ── */}
        <div className="flex items-center gap-5 border-t border-slate-100 px-4 py-2 text-slate-400">
          <span className="flex items-center gap-1 text-xs">♥ 0</span>
          <span className="flex items-center gap-1 text-xs">↗ 0</span>
          <span className="flex items-center gap-1 text-xs">💬 0</span>
          <span className="flex items-center gap-1 text-xs">⟳ 0</span>
        </div>
      </Card>
    </div>
  )
}