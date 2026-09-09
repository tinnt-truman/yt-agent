import { Link, Navigate, Route, Routes } from 'react-router-dom'
import HomePage from './pages/HomePage'
import AnalysisPage from './pages/AnalysisPage'
import HistoryPage from './pages/HistoryPage'
import ScriptsPage from './pages/ScriptsPage'
import VideoPromptSeriesPage from './pages/VideoPromptSeriesPage'
import SettingsPage from './pages/SettingsPage'
import TrendingPage from './pages/TrendingPage'
import ChannelsPage from './pages/ChannelsPage'
import ChannelDetailPage from './pages/ChannelDetailPage'
import LoginPage from './pages/LoginPage'
import { Logo } from './components/Logo'
import { SettingsProvider, useSettingsContext } from './context/SettingsContext'
import { AuthProvider, useAuth } from './context/AuthContext'

export default function App() {
  return (
    <AuthProvider>
      <AuthGate />
    </AuthProvider>
  )
}

function AuthGate() {
  const { isAuthenticated, logout } = useAuth()

  if (!isAuthenticated) return <LoginPage />

  return (
    <SettingsProvider>
      <div className="min-h-screen bg-slate-50 text-slate-900">
        <header className="border-b border-slate-200 bg-white">
          <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
            <Link to="/" className="flex items-center gap-2 text-lg font-semibold tracking-tight">
              <Logo size={28} />
              YT Agent
            </Link>
            <nav className="flex items-center gap-4 text-sm text-slate-600">
              <Link to="/" className="hover:text-slate-900">
                Phân tích mới
              </Link>
              <Link to="/trending" className="hover:text-slate-900">
                Xu hướng
              </Link>
              <Link to="/channels" className="hover:text-slate-900">
                Kênh của tôi
              </Link>
              <Link to="/history" className="hover:text-slate-900">
                Lịch sử
              </Link>
              <Link to="/scripts" className="hover:text-slate-900">
                Kịch bản
              </Link>
              <Link to="/video-prompts" className="hover:text-slate-900">
                Prompt video
              </Link>
              <Link to="/config" className="hover:text-slate-900">
                Cài đặt
              </Link>
              <button onClick={logout} className="hover:text-slate-900">
                Đăng xuất
              </button>
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
      <Route path="/trending" element={<TrendingPage />} />
      <Route path="/channels" element={<ChannelsPage />} />
      <Route path="/channels/:id" element={<ChannelDetailPage />} />
      <Route path="/history" element={<HistoryPage />} />
      <Route path="/scripts" element={<ScriptsPage />} />
      <Route path="/video-prompts" element={<VideoPromptSeriesPage />} />
      <Route path="/config" element={<SettingsPage />} />
    </Routes>
  )
}
