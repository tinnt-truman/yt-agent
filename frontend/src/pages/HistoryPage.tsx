import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listAnalyses } from '../api/client'
import type { Analysis } from '../types'
import { StatusBadge } from '../components/StatusBadge'

export default function HistoryPage() {
  const [items, setItems] = useState<Analysis[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listAnalyses()
      .then(setItems)
      .catch((err) => setError(err instanceof Error ? err.message : 'Không tải được lịch sử'))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <p className="text-slate-600">Đang tải...</p>
  if (error) return <p className="text-red-600">{error}</p>
  if (items.length === 0) return <p className="text-slate-600">Chưa có phân tích nào.</p>

  return (
    <div>
      <h1 className="text-xl font-semibold text-slate-900">Lịch sử phân tích</h1>
      <div className="mt-6 divide-y divide-slate-200 rounded-lg border border-slate-200 bg-white">
        {items.map((item) => (
          <Link
            key={item.id}
            to={`/analyses/${item.id}`}
            className="flex items-center justify-between gap-4 px-4 py-3 hover:bg-slate-50"
          >
            <div className="min-w-0">
              <p className="truncate font-medium text-slate-900">
                {item.channelTitle || item.inputUrl}
              </p>
              <p className="truncate text-sm text-slate-500">{item.inputUrl}</p>
            </div>
            <StatusBadge status={item.status} />
          </Link>
        ))}
      </div>
    </div>
  )
}
