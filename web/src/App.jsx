import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import AppLayout from './layouts/AppLayout'
import Dashboard from './pages/Dashboard'
import ContentList from './pages/Content'
import ContentCreate from './pages/Content/Create'
import ContentPreview from './pages/Content/Preview'
import Queue from './pages/Queue'
import Schedule from './pages/Schedule'
import Analytics from './pages/Analytics'
import Topics from './pages/Topics'
import Settings from './pages/Settings'

const qc = new QueryClient({
  defaultOptions: {
    queries: { refetchOnWindowFocus: false, retry: 1, staleTime: 15_000 },
  },
})

export default function App() {
  return (
    <QueryClientProvider client={qc}>
      <BrowserRouter>
        <Routes>
          <Route element={<AppLayout />}>
            <Route index element={<Dashboard />} />
            <Route path="content" element={<ContentList />} />
            <Route path="create" element={<ContentCreate />} />
            <Route path="queue" element={<Queue />} />
            <Route path="analytics" element={<Analytics />} />
            <Route path="topics" element={<Topics />} />
            <Route path="settings" element={<Settings />} />
          </Route>
          <Route element={<AppLayout bare />}>
            <Route path="content/:id/preview" element={<ContentPreview />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}