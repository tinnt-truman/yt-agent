import type { AnalysisResult } from '../types'
import { formatCompact, formatPercent } from '../lib/format'

export function ChannelOverview({ analysis }: { analysis: AnalysisResult }) {
  const { channel, stats, videosSampled } = analysis

  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5">
      <div className="flex items-center gap-4">
        {channel.thumbnail && (
          <img src={channel.thumbnail} alt={channel.title} className="h-16 w-16 rounded-full" />
        )}
        <div>
          <h2 className="text-lg font-semibold text-slate-900">{channel.title}</h2>
          <p className="text-sm text-slate-500">
            {formatCompact(channel.subscriberCount)} subscribers ·{' '}
            {formatCompact(channel.videoCount)} videos · {formatCompact(channel.viewCount)} lượt xem
          </p>
        </div>
      </div>

      <div className="mt-5 grid grid-cols-2 gap-4 sm:grid-cols-4">
        <Stat label="Lượt xem TB" value={formatCompact(stats.avgViews)} />
        <Stat label="Lượt xem trung vị" value={formatCompact(stats.medianViews)} />
        <Stat label="Tỷ lệ tương tác" value={formatPercent(stats.engagementRate * 100, 2)} />
        <Stat label="Tần suất đăng" value={`${stats.uploadFrequencyPerWeek.toFixed(1)}/tuần`} />
        <Stat label="Lượt thích TB" value={formatCompact(stats.avgLikes)} />
        <Stat label="Bình luận TB" value={formatCompact(stats.avgComments)} />
        <Stat label="Tỷ lệ Shorts" value={formatPercent(stats.shortsRatio * 100)} />
        <Stat label="Mẫu video" value={`${videosSampled}`} />
      </div>
    </section>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-slate-500">{label}</p>
      <p className="text-lg font-semibold text-slate-900">{value}</p>
    </div>
  )
}
