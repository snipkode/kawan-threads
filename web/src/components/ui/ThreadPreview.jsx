import { Card } from './Card'

export function ThreadPreview({ content, onPickHook, pickedHook }) {
  const hook = pickedHook || content.hook
  return (
    <div className="mx-auto w-full max-w-sm">
      <Card className="px-4 pt-3 pb-2">
        <div className="mb-3 flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-brand-600 to-brand-900 text-sm font-bold text-white">
            K
          </div>
          <div>
            <p className="flex items-center gap-1 text-sm font-semibold">
              KAWAN <span className="text-brand-600 text-xs font-bold">·</span>
            </p>
            <p className="text-xs text-slate-400">@kawan</p>
          </div>
        </div>

        {onPickHook && content.hook_variants?.length > 0 && (
          <div className="mb-3 flex flex-wrap gap-1.5">
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

        <div className="space-y-2 text-[15px] leading-relaxed text-slate-800">
          {hook && <p>{hook}</p>}
          {content.body && <p>{content.body}</p>}
          {content.cta && <p>{content.cta}</p>}
          {content.conversation_question && (
            <p className="font-medium">{content.conversation_question}</p>
          )}
        </div>

        {(content.topic || content.pillar) && (
          <p className="mt-3 text-xs text-slate-400">
            {content.topic || 'KAWAN'} · {content.pillar || 'Komunitas'}
          </p>
        )}

        <div className="mt-3 flex items-center gap-5 border-t border-slate-100 py-2 text-slate-400">
          <span className="flex items-center gap-1 text-xs">♥ 0</span>
          <span className="flex items-center gap-1 text-xs">↗ 0</span>
          <span className="flex items-center gap-1 text-xs">💬 0</span>
          <span className="flex items-center gap-1 text-xs">⟳ 0</span>
        </div>
      </Card>
    </div>
  )
}