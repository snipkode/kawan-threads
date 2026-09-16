import { Link } from 'react-router-dom'

export function Header({ title, subtitle, right }) {
  return (
    <header className="sticky top-0 z-20 border-b border-slate-200 bg-white/90 backdrop-blur">
      <div className="mx-auto flex max-w-lg items-center justify-between px-4 py-3">
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-brand-600 to-brand-900 text-xs font-bold text-white">
            K
          </div>
          <div>
            <h1 className="text-sm font-bold leading-tight text-slate-900">{title}</h1>
            {subtitle && <p className="text-[11px] leading-tight text-slate-400">{subtitle}</p>}
          </div>
        </div>
        {right && <div className="flex items-center gap-2">{right}</div>}
      </div>
    </header>
  )
}

export function BackHeader({ to = '..', title = 'Back' }) {
  return (
    <header className="sticky top-0 z-20 border-b border-slate-200 bg-white/90 backdrop-blur">
      <div className="mx-auto flex max-w-lg items-center gap-2 px-4 py-3">
        <Link to={to} className="text-sm font-semibold text-brand-600 hover:text-brand-700">
          ← {title}
        </Link>
      </div>
    </header>
  )
}