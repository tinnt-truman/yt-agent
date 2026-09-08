import type { AnalysisResult } from '../types'
import { formatCompact, formatPercent } from '../lib/format'

export function ChannelOverview({ analysis }: { analysis: AnalysisResult }) {
  const { channel, stats, videosSampled } = analysis

  return (
    <div className="space-y-4">
      <section className="rounded-lg border border-slate-200 bg-white p-5">
        <div className="flex flex-wrap items-center gap-4">
          {channel.thumbnail ? (
            <img src={channel.thumbnail} alt={channel.title} className="h-14 w-14 shrink-0 rounded-full" />
          ) : (
            <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-red-300 to-red-400 text-sm font-bold text-white">
              {channel.title.slice(0, 2).toUpperCase()}
            </div>
          )}
          <div className="min-w-[180px] flex-1">
            <h2 className="text-base font-semibold text-slate-900">{channel.title}</h2>
            <p className="mt-0.5 text-sm text-slate-500">Kênh tham khảo cho chiến lược mới</p>
          </div>
          <div className="flex gap-7">
            <HeadlineStat label="Subscribers" value={formatCompact(channel.subscriberCount)} />
            <HeadlineStat label="Videos" value={formatCompact(channel.videoCount)} />
            <HeadlineStat label="Tổng lượt xem" value={formatCompact(channel.viewCount)} />
          </div>
        </div>
      </section>

      <section className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <Stat label="Lượt xem TB / video" value={formatCompact(stats.avgViews)} />
        <Stat label="Lượt xem trung vị" value={formatCompact(stats.medianViews)} />
        <Stat
          label="Tỷ lệ tương tác"
          value={formatPercent(stats.engagementRate * 100, 2)}
          barPct={Math.min(stats.engagementRate * 100 * 10, 100)}
          barColor="bg-indigo-500"
        />
        <Stat label="Tần suất đăng" value={`${stats.uploadFrequencyPerWeek.toFixed(1)}/tuần`} />
        <Stat label="Lượt thích TB" value={formatCompact(stats.avgLikes)} />
        <Stat label="Bình luận TB" value={formatCompact(stats.avgComments)} />
        <Stat
          label="Tỷ lệ Shorts"
          value={formatPercent(stats.shortsRatio * 100)}
          barPct={stats.shortsRatio * 100}
          barColor="bg-slate-400"
        />
        <Stat label="Mẫu video phân tích" value={`${videosSampled}`} />
      </section>
    </div>
  )
}

function HeadlineStat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-slate-500">{label}</p>
      <p className="text-xl font-bold text-slate-900">{value}</p>
    </div>
  )
}

function Stat({
  label,
  value,
  barPct,
  barColor,
}: {
  label: string
  value: string
  barPct?: number
  barColor?: string
}) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 text-lg font-semibold text-slate-900">{value}</p>
      {barPct !== undefined && (
        <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-slate-100">
          <div className={`h-full rounded-full ${barColor}`} style={{ width: `${barPct}%` }} />
        </div>
      )}
    </div>
  )
}
