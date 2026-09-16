export function Card({ children, className = '' }) {
  return <div className={`card ${className}`}>{children}</div>
}

export function Spinner({ label }) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-10 text-slate-500">
      <div className="h-6 w-6 animate-spin rounded-full border-2 border-slate-300 border-t-brand-600" />
      {label && <p className="text-xs font-medium">{label}</p>}
    </div>
  )
}

export function EmptyState({ title, hint, icon }) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-14 text-center text-slate-400">
      {icon}
      <p className="text-sm font-semibold text-slate-500">{title}</p>
      {hint && <p className="max-w-xs text-xs">{hint}</p>}
    </div>
  )
}

export function ErrorBox({ message }) {
  return (
    <div className="rounded-xl border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700">
      {message}
    </div>
  )
}