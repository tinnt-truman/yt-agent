import { useState } from 'react'
import { useAuth } from '../context/AuthContext'
import { Logo } from '../components/Logo'

export default function LoginPage() {
  const { login } = useAuth()
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError(null)
    try {
      await login(password)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sai mật khẩu')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-50 px-6">
      <form
        onSubmit={handleSubmit}
        className="w-full max-w-sm rounded-lg border border-slate-200 bg-white p-6 shadow-sm"
      >
        <div className="flex items-center gap-2">
          <Logo size={32} />
          <h1 className="text-lg font-semibold text-slate-900">YT Agent</h1>
        </div>
        <p className="mt-3 text-sm text-slate-600">Nhập mật khẩu để tiếp tục.</p>

        <input
          type="password"
          autoFocus
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Mật khẩu"
          className="mt-4 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500"
        />

        {error && <p className="mt-2 text-sm text-red-600">{error}</p>}

        <button
          type="submit"
          disabled={loading}
          className="mt-4 w-full rounded-lg bg-slate-900 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-slate-700 disabled:opacity-50"
        >
          {loading ? 'Đang kiểm tra...' : 'Đăng nhập'}
        </button>
      </form>
    </div>
  )
}
