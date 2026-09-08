import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { disconnectChannel, getGoogleAuthUrl, listConnectedChannels } from '../api/client'
import type { ConnectedChannel } from '../types'
import { formatCompact } from '../lib/format'

export default function ChannelsPage() {
  const [channels, setChannels] = useState<ConnectedChannel[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [connecting, setConnecting] = useState(false)
  const [disconnectingId, setDisconnectingId] = useState<string | null>(null)
  const [searchParams, setSearchParams] = useSearchParams()

  function refresh() {
    setLoading(true)
    listConnectedChannels()
      .then(setChannels)
      .catch((err) => setError(err instanceof Error ? err.message : 'Không tải được danh sách kênh'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    refresh()
  }, [])

  // Reads the OAuth redirect's query params and clears them. Depending on
  // searchParams/setSearchParams is safe here: clearing them re-runs this
  // once more with both params gone, which is a no-op the second time.
  useEffect(() => {
    const oauthError = searchParams.get('error')
    if (oauthError) setError(oauthError)
    if (oauthError || searchParams.get('connected')) {
      setSearchParams({}, { replace: true })
    }
  }, [searchParams, setSearchParams])

  async function handleConnect() {
    setConnecting(true)
    setError(null)
    try {
      const url = await getGoogleAuthUrl()
      window.location.href = url
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Không khởi tạo được kết nối Google')
      setConnecting(false)
    }
  }

  async function handleDisconnect(id: string) {
    if (!confirm('Ngắt kết nối kênh này? Bạn có thể kết nối lại bất cứ lúc nào.')) return
    setDisconnectingId(id)
    try {
      await disconnectChannel(id)
      setChannels((prev) => prev.filter((c) => c.id !== id))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Không ngắt kết nối được')
    } finally {
      setDisconnectingId(null)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Kênh của tôi</h1>
          <p className="mt-1 text-sm text-slate-600">
            Kết nối kênh YouTube của bạn qua Google để xem Analytics riêng tư: giờ xem, nguồn
            traffic, doanh thu ước tính và trạng thái kiếm tiền.
          </p>
        </div>
        <button
          onClick={handleConnect}
          disabled={connecting}
          className="shrink-0 rounded-lg bg-slate-900 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-slate-700 disabled:opacity-50"
        >
          {connecting ? 'Đang chuyển hướng...' : '+ Kết nối kênh mới'}
        </button>
      </div>

      {error && (
        <p className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </p>
      )}

      {loading && <p className="mt-6 text-slate-600">Đang tải...</p>}

      {!loading && channels.length === 0 && (
        <div className="mt-6 rounded-lg border border-dashed border-slate-300 bg-white p-8 text-center text-sm text-slate-500">
          Chưa có kênh nào được kết nối. Bấm "+ Kết nối kênh mới" để đăng nhập Google và liên kết
          kênh YouTube của bạn.
        </div>
      )}

      <div className="mt-6 space-y-3">
        {channels.map((ch) => (
          <div
            key={ch.id}
            className="flex flex-wrap items-center justify-between gap-4 rounded-lg border border-slate-200 bg-white p-4"
          >
            <div className="flex items-center gap-3">
              {ch.channelThumbnail ? (
                <img src={ch.channelThumbnail} alt={ch.channelTitle} className="h-12 w-12 rounded-full" />
              ) : (
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-gradient-to-br from-indigo-400 to-violet-400 text-sm font-bold text-white">
                  {ch.channelTitle.slice(0, 2).toUpperCase()}
                </div>
              )}
              <div>
                <p className="font-medium text-slate-900">{ch.channelTitle}</p>
                <p className="text-sm text-slate-500">
                  {formatCompact(ch.subscriberCount)} subscribers
                  {ch.googleEmail ? ` · ${ch.googleEmail}` : ''}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Link
                to={`/channels/${ch.id}`}
                className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-700"
              >
                Xem Analytics
              </Link>
              <button
                onClick={() => handleDisconnect(ch.id)}
                disabled={disconnectingId === ch.id}
                className="rounded-lg border border-slate-300 px-4 py-2 text-sm text-slate-600 transition hover:bg-slate-100 disabled:opacity-50"
              >
                {disconnectingId === ch.id ? 'Đang ngắt...' : 'Ngắt kết nối'}
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
