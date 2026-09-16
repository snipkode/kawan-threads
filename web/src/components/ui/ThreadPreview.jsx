import { Card } from './Card'

export function ThreadPreview({ content, onPickHook, pickedHook }) {
  const hook = pickedHook || content.hook
  return (
    <div className="mx-auto w-full max-w-sm">
      <Card className="overflow-hidden px-0 pt-0 pb-0">

        {/* Masthead */}
        <div className="border-b-2 border-slate-900 px-4 pt-4 pb-2">
          <p className="text-center text-[10px] font-semibold uppercase tracking-[0.2em] text-slate-400">
            {content.pillar || 'Komunitas'} · {content.topic || 'KAWAN'}
          </p>
          <div className="my-1 border-t border-slate-200" />
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="flex h-7 w-7 items-center justify-center rounded-full bg-slate-900 text-xs font-black text-white">
                K
              </div>
              <div>
                <p className="text-xs font-bold leading-none text-slate-900">KAWAN</p>
                <p className="text-[10px] text-slate-400">@kawan</p>
              </div>
            </div>
            <p className="text-[10px] text-slate-400">
              {new Date().toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' })}
            </p>
          </div>
        </div>

        {/* Hook variant selector */}
        {onPickHook && content.hook_variants?.length > 0 && (
          <div className="flex flex-wrap gap-1.5 border-b border-slate-100 px-4 py-2">
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

        {/* Editorial body */}
        <div className="px-4 py-4">
          {hook && (
            <h2
              className="mb-3 font-serif text-[18px] font-bold leading-snug tracking-tight text-slate-900 hyphens-auto"
              lang="id"
            >
              {hook}
            </h2>
          )}

          <div className="border-t-2 border-slate-900 mb-3" />

          {content.body && (
            <p
              className="font-serif text-[13.5px] leading-relaxed text-slate-700 text-justify hyphens-auto"
              lang="id"
            >
              {content.body}
            </p>
          )}

          {content.cta && (
            <p
              className="mt-3 font-serif text-[13px] font-semibold text-slate-800"
            >
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

        {/* Footer bar */}
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