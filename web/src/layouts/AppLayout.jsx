import { Outlet } from 'react-router-dom'
import { BottomNav } from '../components/layout/BottomNav'

export default function AppLayout({ bare }) {
  return (
    <div className="mx-auto flex min-h-screen max-w-lg flex-col bg-slate-100">
      <main className="flex-1 pb-20">
        <Outlet />
      </main>
      {!bare && <BottomNav />}
    </div>
  )
}