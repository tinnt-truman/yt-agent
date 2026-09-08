import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getChannelAnalytics, listConnectedChannels } from '../api/client'
import type { ChannelAnalytics, ConnectedChannel } from '../types'
import { formatCompact, formatDuration } from '../lib/format'

const MONETIZATION_LABEL: Record<ChannelAnalytics['monetization'], string> = {
  enabled: 'Đã bật kiếm tiền',
  disabled: 'Chưa bật kiếm tiền',
  unknown: 'Chưa xác định',
}

const MONETIZATION_COLOR: Record<ChannelAnalytics['monetization'], string> = {
  enabled: 'text-emerald-700',
  disabled: 'text-slate-500',
  unknown: 'text-slate-500',
}

export default function ChannelDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [channel, setChannel] = useState<ConnectedChannel | null>(null)
  const [analytics, setAnalytics] = useState<ChannelAnalytics | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    setLoading(true)
    setError(null)
    Promise.all([
      listConnectedChannels().then((list) => list.find((c) => c.id === id) ?? null),
      getChannelAnalytics(id),
    ])
      .then(([ch, data]) => {
        setChannel(ch)
        setAnalytics(data)
      })
      .catch((err) => setError(err instanceof Error ? err.message : 'Không tải được Analytics'))
      .finally(() => setLoading(false))
  }, [id])

  if (loading) return <p className="text-slate-600">Đang tải...</p>
  if (error) return <p className="text-red-600">{error}</p>
  if (!analytics) return null

  return (
    <div>
      <Link to="/channels" className="text-sm text-slate-500 hover:text-slate-900">
        ← Kênh của tôi
      </Link>

      <div className="mt-3 flex items-center justify-between">
        <div className="flex items-center gap-3">
          {channel?.channelThumbnail ? (
            <img src={channel.channelThumbnail} alt="" className="h-12 w-12 rounded-full" />
          ) : (
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-gradient-to-br from-indigo-400 to-violet-400 text-sm font-bold text-white">
              {(channel?.channelTitle ?? '?').slice(0, 2).toUpperCase()}
            </div>
          )}
          <div>
            <h1 className="text-xl font-semibold text-slate-900">{channel?.channelTitle}</h1>
            {channel?.googleEmail && <p className="text-sm text-slate-500">{channel.googleEmail}</p>}
          </div>
        </div>
        <span className="rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-500">
          {analytics.windowDays} ngày qua
        </span>
      </div>

      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="rounded-lg border border-slate-200 bg-white p-5">
          <p className="text-xs text-slate-500">Trạng thái kiếm tiền</p>
          <p className={`mt-2 text-lg font-bold ${MONETIZATION_COLOR[analytics.monetization]}`}>
            {MONETIZATION_LABEL[analytics.monetization]}
          </p>
        </div>
        <div className="rounded-lg border border-slate-200 bg-white p-5">
          <p className="text-xs text-slate-500">Doanh thu ước tính · {analytics.windowDays} ngày</p>
          <p className="mt-2 text-2xl font-bold text-slate-900">
            {analytics.estimatedRevenueUsd !== undefined ? `$${analytics.estimatedRevenueUsd.toFixed(2)}` : '—'}
          </p>
          {analytics.revenueNote && <p className="mt-1 text-xs text-slate-400">{analytics.revenueNote}</p>}
        </div>
      </div>

      <div className="mt-4 rounded-lg border border-slate-200 bg-white p-5">
        <h2 className="text-base font-semibold text-slate-900">
          Hiệu suất · {analytics.windowDays} ngày qua
        </h2>
        <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-3">
          <Stat label="Lượt xem" value={formatCompact(analytics.views)} />
          <Stat label="Giờ xem" value={`${formatCompact(analytics.estimatedMinutesWatched / 60)} giờ`} />
          <Stat label="Thời lượng xem TB" value={formatDuration(analytics.averageViewDurationSeconds)} />
          <Stat
            label="Subscribers mới"
            value={`+${formatCompact(analytics.subscribersGained)}`}
            valueClassName="text-emerald-700"
          />
          {analytics.impressions !== undefined && (
            <Stat label="Lượt hiển thị" value={formatCompact(analytics.impressions)} />
          )}
          {analytics.impressionsCtr !== undefined && (
            <Stat label="CTR trung bình" value={`${analytics.impressionsCtr.toFixed(1)}%`} />
          )}
        </div>
      </div>

      <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
        {analytics.topVideos && analytics.topVideos.length > 0 && (
          <div className="rounded-lg border border-slate-200 bg-white p-5">
            <h2 className="text-base font-semibold text-slate-900">Video xem nhiều giờ nhất</h2>
            <div className="mt-4 flex flex-col gap-3">
              {analytics.topVideos.map((v) => (
                <a
                  key={v.videoId}
                  href={`https://www.youtube.com/watch?v=${v.videoId}`}
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-3 rounded-md p-1 hover:bg-slate-50"
                >
                  {v.thumbnail ? (
                    <img src={v.thumbnail} alt="" className="h-9 w-16 shrink-0 rounded object-cover" />
                  ) : (
                    <div className="h-9 w-16 shrink-0 rounded bg-slate-100" />
                  )}
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-slate-900">{v.title || v.videoId}</p>
                    <p className="text-xs text-slate-400">
                      {formatCompact(v.estimatedMinutesWatched)} phút xem
                    </p>
                  </div>
                </a>
              ))}
            </div>
          </div>
        )}

        {analytics.trafficSources && analytics.trafficSources.length > 0 && (
          <div className="rounded-lg border border-slate-200 bg-white p-5">
            <h2 className="text-base font-semibold text-slate-900">Nguồn traffic</h2>
            <div className="mt-4 flex flex-col gap-3">
              {analytics.trafficSources.map((s) => (
                <div key={s.source}>
                  <div className="flex justify-between text-xs">
                    <span className="text-slate-700">{s.source}</span>
                    <span className="font-semibold text-slate-900">{s.sharePct.toFixed(0)}%</span>
                  </div>
                  <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-slate-100">
                    <div
                      className="h-full rounded-full bg-indigo-500"
                      style={{ width: `${s.sharePct}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function Stat({
  label,
  value,
  valueClassName = 'text-slate-900',
}: {
  label: string
  value: string
  valueClassName?: string
}) {
  return (
    <div>
      <p className="text-xs text-slate-500">{label}</p>
      <p className={`mt-1 text-lg font-semibold ${valueClassName}`}>{value}</p>
    </div>
  )
}
