import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  createAnalysis,
  getTrending,
  getTrendingInsight,
  getVideoCategories,
} from '../api/client'
import type { TrendingInsight, TrendingReport, VideoCategory } from '../types'
import { formatCompact, formatDate } from '../lib/format'

const REGIONS = [
  { value: 'VN', label: 'Việt Nam' },
  { value: 'US', label: 'Mỹ' },
  { value: 'JP', label: 'Nhật Bản' },
  { value: 'KR', label: 'Hàn Quốc' },
  { value: 'IN', label: 'Ấn Độ' },
  { value: 'GB', label: 'Anh' },
  { value: 'TH', label: 'Thái Lan' },
  { value: 'ID', label: 'Indonesia' },
]

export default function TrendingPage() {
  const [region, setRegion] = useState('VN')
  const [category, setCategory] = useState('')
  const [categories, setCategories] = useState<VideoCategory[]>([])
  const [report, setReport] = useState<TrendingReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [insight, setInsight] = useState<TrendingInsight | null>(null)
  const [insightLoading, setInsightLoading] = useState(false)
  const [insightError, setInsightError] = useState<string | null>(null)

  const [analyzingChannelId, setAnalyzingChannelId] = useState<string | null>(null)
  const navigate = useNavigate()

  // Category IDs/availability vary by region, so the list is refetched
  // whenever the region changes — and any category picked for the previous
  // region is reset, since it may not exist (or mean something else) here.
  useEffect(() => {
    setCategory('')
    getVideoCategories(region)
      .then(setCategories)
      .catch(() => setCategories([]))
  }, [region])

  useEffect(() => {
    setLoading(true)
    setError(null)
    setInsight(null)
    setInsightError(null)
    getTrending(region, category || undefined)
      .then(setReport)
      .catch((err) => setError(err instanceof Error ? err.message : 'Không tải được dữ liệu trending'))
      .finally(() => setLoading(false))
  }, [region, category])

  async function handleGenerateInsight() {
    if (!report) return
    setInsightLoading(true)
    setInsightError(null)
    try {
      setInsight(await getTrendingInsight(report))
    } catch (err) {
      setInsightError(err instanceof Error ? err.message : 'Không tạo được nhận xét AI')
    } finally {
      setInsightLoading(false)
    }
  }

  async function handleAnalyzeChannel(channelId: string) {
    setAnalyzingChannelId(channelId)
    try {
      const analysis = await createAnalysis(`https://www.youtube.com/channel/${channelId}`)
      navigate(`/analyses/${analysis.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Không tạo được phân tích')
      setAnalyzingChannelId(null)
    }
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Kênh đang trending</h1>
          <p className="mt-1 text-sm text-slate-600">
            Gộp từ danh sách video đang thịnh hành (YouTube "mostPopular") theo khu vực.
          </p>
        </div>
        <div className="flex gap-2">
          <select
            value={region}
            onChange={(e) => setRegion(e.target.value)}
            className="rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-slate-500"
          >
            {REGIONS.map((r) => (
              <option key={r.value} value={r.value}>
                {r.label}
              </option>
            ))}
          </select>
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-slate-500"
          >
            <option value="">Tất cả chủ đề</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.title}
              </option>
            ))}
          </select>
        </div>
      </div>

      {loading && <p className="mt-6 text-slate-600">Đang tải...</p>}
      {error && <p className="mt-6 text-sm text-red-600">{error}</p>}

      {report && !loading && (
        <>
          <div className="mt-6 flex items-center justify-between">
            <p className="text-sm text-slate-500">
              {report.channels.length} kênh · cập nhật {formatDate(report.generatedAt)}
            </p>
            <button
              onClick={handleGenerateInsight}
              disabled={insightLoading}
              className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-100 disabled:opacity-50"
            >
              {insightLoading ? 'Đang phân tích...' : 'Xem nhận xét AI'}
            </button>
          </div>

          {insightError && <p className="mt-3 text-sm text-red-600">{insightError}</p>}
          {insight && (
            <div className="mt-4 rounded-lg border border-indigo-200 bg-indigo-50 p-4">
              <p className="text-sm text-slate-800">{insight.summary}</p>
              {insight.opportunities.length > 0 && (
                <ul className="mt-3 list-inside list-disc space-y-1 text-sm text-slate-700">
                  {insight.opportunities.map((o, i) => (
                    <li key={i}>{o}</li>
                  ))}
                </ul>
              )}
            </div>
          )}

          <div className="mt-6 space-y-3">
            {report.channels.map((ch) => (
              <div
                key={ch.channelId}
                className="rounded-lg border border-slate-200 bg-white p-4"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex items-center gap-3">
                    {ch.channelThumbnail && (
                      <img
                        src={ch.channelThumbnail}
                        alt={ch.channelTitle}
                        className="h-12 w-12 rounded-full"
                      />
                    )}
                    <div>
                      <p className="font-medium text-slate-900">{ch.channelTitle}</p>
                      <p className="text-sm text-slate-500">
                        {formatCompact(ch.subscriberCount)} subscribers ·{' '}
                        {ch.trendingVideoCount} video trending ·{' '}
                        {formatCompact(ch.totalViews)} lượt xem
                      </p>
                    </div>
                  </div>
                  <button
                    onClick={() => handleAnalyzeChannel(ch.channelId)}
                    disabled={analyzingChannelId === ch.channelId}
                    className="shrink-0 rounded-lg bg-slate-900 px-4 py-2 text-xs font-medium text-white transition hover:bg-slate-700 disabled:opacity-50"
                  >
                    {analyzingChannelId === ch.channelId ? 'Đang tạo...' : 'Phân tích kênh này'}
                  </button>
                </div>

                <div className="mt-3 flex gap-3 overflow-x-auto">
                  {ch.videos.map((v) => (
                    <a
                      key={v.id}
                      href={`https://www.youtube.com/watch?v=${v.id}`}
                      target="_blank"
                      rel="noreferrer"
                      className="shrink-0"
                    >
                      {v.thumbnail && (
                        <img src={v.thumbnail} alt={v.title} className="h-20 w-36 rounded object-cover" />
                      )}
                      <p className="mt-1 w-36 truncate text-xs text-slate-600">{v.title}</p>
                      <p className="text-xs text-slate-400">{formatCompact(v.viewCount)} views</p>
                    </a>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}
