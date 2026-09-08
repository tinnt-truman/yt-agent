import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { createAnalysis, listAnalyses } from '../api/client'
import type { Analysis } from '../types'
import { StatusBadge } from '../components/StatusBadge'

export default function HistoryPage() {
  const [items, setItems] = useState<Analysis[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [reanalyzingId, setReanalyzingId] = useState<string | null>(null)
  const navigate = useNavigate()

  useEffect(() => {
    listAnalyses()
      .then(setItems)
      .catch((err) => setError(err instanceof Error ? err.message : 'Không tải được lịch sử'))
      .finally(() => setLoading(false))
  }, [])

  async function handleReanalyze(item: Analysis, e: React.MouseEvent) {
    e.preventDefault() // don't follow the row's own Link
    e.stopPropagation()
    setActionError(null)
    setReanalyzingId(item.id)
    try {
      // Same URL, brand new job — the old row (and its saved result) stays
      // untouched in history; this doesn't overwrite it.
      const fresh = await createAnalysis(item.inputUrl)
      navigate(`/analyses/${fresh.id}`)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Không tạo được phân tích mới')
      setReanalyzingId(null)
    }
  }

  if (loading) return <p className="text-slate-600">Đang tải...</p>
  if (error) return <p className="text-red-600">{error}</p>
  if (items.length === 0) return <p className="text-slate-600">Chưa có phân tích nào.</p>

  return (
    <div>
      <h1 className="text-xl font-semibold text-slate-900">Lịch sử phân tích</h1>

      {actionError && <p className="mt-3 text-sm text-red-600">{actionError}</p>}

      <div className="mt-6 divide-y divide-slate-200 rounded-lg border border-slate-200 bg-white">
        {items.map((item) => (
          <div key={item.id} className="flex items-center justify-between gap-4 px-4 py-3 hover:bg-slate-50">
            <Link to={`/analyses/${item.id}`} className="min-w-0 flex-1">
              <p className="truncate font-medium text-slate-900">
                {item.channelTitle || item.inputUrl}
              </p>
              <p className="truncate text-sm text-slate-500">{item.inputUrl}</p>
            </Link>
            <div className="flex shrink-0 items-center gap-3">
              <StatusBadge status={item.status} />
              <button
                onClick={(e) => handleReanalyze(item, e)}
                disabled={reanalyzingId === item.id}
                className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:bg-slate-100 disabled:opacity-50"
              >
                {reanalyzingId === item.id ? 'Đang tạo...' : 'Phân tích lại'}
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
