import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { createAnalysis } from '../api/client'

export default function HomePage() {
  const [url, setUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!url.trim()) return
    setLoading(true)
    setError(null)
    try {
      const { id } = await createAnalysis(url.trim())
      navigate(`/analyses/${id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Đã có lỗi xảy ra')
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-2xl text-center">
      <h1 className="text-3xl font-bold tracking-tight text-slate-900">
        Phân tích kênh YouTube &amp; tạo chiến lược kênh mới
      </h1>
      <p className="mt-3 text-slate-600">
        Dán link video hoặc kênh YouTube. Hệ thống sẽ phân tích số liệu, sau đó dùng AI để
        biến tấu thành chiến lược nội dung, ý tưởng, hashtag cho một kênh mới của bạn.
      </p>

      <form onSubmit={handleSubmit} className="mt-8 flex flex-col gap-3 sm:flex-row">
        <input
          type="text"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://www.youtube.com/@ten-kenh hoặc link video"
          className="flex-1 rounded-lg border border-slate-300 px-4 py-3 text-sm outline-none focus:border-slate-500"
        />
        <button
          type="submit"
          disabled={loading}
          className="rounded-lg bg-slate-900 px-6 py-3 text-sm font-medium text-white transition hover:bg-slate-700 disabled:opacity-50"
        >
          {loading ? 'Đang gửi...' : 'Phân tích'}
        </button>
      </form>

      {error && <p className="mt-4 text-sm text-red-600">{error}</p>}

      <div className="mt-10 grid grid-cols-1 gap-4 text-left sm:grid-cols-3">
        <FeatureCard title="Phân tích số liệu" desc="Tần suất đăng, lượt xem trung bình, video nổi bật, tag phổ biến." />
        <FeatureCard title="Chiến lược biến tấu" desc="Định vị, đối tượng mục tiêu, content pillar cho kênh mới." />
        <FeatureCard title="Content sẵn dùng" desc="Ý tưởng video, mẫu tiêu đề, bộ hashtag, lịch đăng gợi ý." />
      </div>
    </div>
  )
}

function FeatureCard({ title, desc }: { title: string; desc: string }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <h3 className="font-medium text-slate-900">{title}</h3>
      <p className="mt-1 text-sm text-slate-600">{desc}</p>
    </div>
  )
}
