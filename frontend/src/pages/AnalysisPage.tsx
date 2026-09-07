import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { stepAnalysis } from '../api/client'
import type { Analysis, AnalysisResult, StrategyOutput } from '../types'
import { StatusBadge } from '../components/StatusBadge'
import { ChannelOverview } from '../components/ChannelOverview'
import { TopVideos } from '../components/TopVideos'
import { TagsAndTitles } from '../components/TagsAndTitles'
import { StrategySections } from '../components/StrategySections'

const POLL_INTERVAL_MS = 3000
const ACTIVE_STATUSES = new Set(['pending', 'fetching', 'analyzing', 'generating'])

export default function AnalysisPage() {
  const { id } = useParams<{ id: string }>()
  const [analysis, setAnalysis] = useState<Analysis | null>(null)
  const [error, setError] = useState<string | null>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (!id) return
    let cancelled = false

    async function poll() {
      try {
        // Also advances the job by one stage if it isn't finished yet — the
        // backend has no background worker, so this drives the pipeline
        // forward. It's a safe no-op once the job is done or failed.
        const data = await stepAnalysis(id!)
        if (cancelled) return
        setAnalysis(data)
        if (ACTIVE_STATUSES.has(data.status)) {
          timerRef.current = setTimeout(poll, POLL_INTERVAL_MS)
        }
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Không tải được kết quả')
      }
    }

    poll()
    return () => {
      cancelled = true
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [id])

  if (error) return <p className="text-red-600">{error}</p>
  if (!analysis) return <p className="text-slate-600">Đang tải...</p>

  const result: AnalysisResult | undefined = analysis.analysis
  const strategy: StrategyOutput | undefined = analysis.aiOutput

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">
            {analysis.channelTitle || analysis.inputUrl}
          </h1>
          {analysis.stageMessage && analysis.status !== 'done' && (
            <p className="mt-1 text-sm text-slate-500">{analysis.stageMessage}</p>
          )}
        </div>
        <StatusBadge status={analysis.status} />
      </div>

      {analysis.status === 'failed' && (
        <p className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {analysis.errorMessage || 'Đã có lỗi xảy ra khi phân tích.'}
        </p>
      )}

      {ACTIVE_STATUSES.has(analysis.status) && (
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-center text-slate-600">
          <div className="mx-auto mb-3 h-6 w-6 animate-spin rounded-full border-2 border-slate-300 border-t-slate-900" />
          {analysis.stageMessage || 'Đang xử lý...'}
        </div>
      )}

      {result && <ChannelOverview analysis={result} />}
      {result && <TopVideos videos={result.topVideos} />}
      {result && <TagsAndTitles tags={result.commonTags} titlePatterns={result.titlePatterns} />}
      {strategy && <StrategySections strategy={strategy} />}
    </div>
  )
}
