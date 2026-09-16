import { NavLink } from 'react-router-dom'
import { Home, LayoutGrid, PlusCircle, Inbox, BarChart3, Settings } from 'lucide-react'

const items = [
  { to: '/', label: 'Home', icon: Home, end: true },
  { to: '/content', label: 'Content', icon: LayoutGrid },
  { to: '/create', label: 'Create', icon: PlusCircle, accent: true },
  { to: '/queue', label: 'Queue', icon: Inbox },
  { to: '/analytics', label: 'Analytics', icon: BarChart3 },
]

export function BottomNav() {
  return (
    <nav className="fixed inset-x-0 bottom-0 z-30 border-t border-slate-200 bg-white/95 pb-[env(safe-area-inset-bottom)] backdrop-blur">
      <div className="mx-auto flex max-w-lg items-stretch justify-between px-2 py-1">
        {items.map(({ to, label, icon: Icon, end, accent }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            className={({ isActive }) =>
              `flex flex-1 flex-col items-center gap-0.5 rounded-lg px-1 py-1.5 text-[10px] font-medium transition ${
                isActive ? 'text-brand-600' : 'text-slate-400 hover:text-slate-600'
              }`
            }
          >
            <span
              className={`flex h-7 w-12 items-center justify-center rounded-full ${
                accent ? 'bg-brand-600 text-white' : ''
              }`}
            >
              <Icon size={18} />
            </span>
            {label}
          </NavLink>
        ))}
      </div>
      <NavLink
        to="/settings"
        end
        className={({ isActive }) =>
          `fixed right-3 bottom-[4.6rem] z-30 flex h-9 w-9 items-center justify-center rounded-full border shadow-md transition ${
            isActive
              ? 'border-brand-600 bg-brand-600 text-white'
              : 'border-slate-200 bg-white text-slate-500 hover:text-slate-700'
          }`
        }
        style={{ display: 'inline-flex' }}
      >
        <Settings size={16} />
      </NavLink>
    </nav>
  )
}