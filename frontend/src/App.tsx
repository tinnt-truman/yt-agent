import { Link, Navigate, Route, Routes } from 'react-router-dom'
import HomePage from './pages/HomePage'
import AnalysisPage from './pages/AnalysisPage'
import HistoryPage from './pages/HistoryPage'
import SettingsPage from './pages/SettingsPage'
import { SettingsProvider, useSettingsContext } from './context/SettingsContext'

export default function App() {
  return (
    <SettingsProvider>
      <div className="min-h-screen bg-slate-50 text-slate-900">
        <header className="border-b border-slate-200 bg-white">
          <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
            <Link to="/" className="text-lg font-semibold tracking-tight">
              YT Agent
            </Link>
            <nav className="flex gap-4 text-sm text-slate-600">
              <Link to="/" className="hover:text-slate-900">
                Phân tích mới
              </Link>
              <Link to="/history" className="hover:text-slate-900">
                Lịch sử
              </Link>
              <Link to="/config" className="hover:text-slate-900">
                Cài đặt
              </Link>
            </nav>
          </div>
        </header>

        <main className="mx-auto max-w-5xl px-6 py-10">
          <AppRoutes />
        </main>
      </div>
    </SettingsProvider>
  )
}

function AppRoutes() {
  const { settings, loading } = useSettingsContext()

  if (loading) return <p className="text-slate-600">Đang tải...</p>

  return (
    <Routes>
      <Route
        path="/"
        element={settings?.configured ? <HomePage /> : <Navigate to="/config" replace />}
      />
      <Route path="/analyses/:id" element={<AnalysisPage />} />
      <Route path="/history" element={<HistoryPage />} />
      <Route path="/config" element={<SettingsPage />} />
    </Routes>
  )
}
