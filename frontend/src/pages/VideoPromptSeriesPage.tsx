import { useEffect, useRef, useState } from 'react'
import {
  deleteVideoPromptSeriesJob,
  listVideoPromptSeriesJobs,
  retryVideoPromptSeriesJob,
  stepVideoPromptSeriesJob,
} from '../api/client'
import { formatDate } from '../lib/format'
import type { VideoPromptSeriesJob } from '../types'
import { CopyButton } from '../components/CopyButton'
import { StatusBadge } from '../components/StatusBadge'

const POLL_INTERVAL_MS = 2000

export default function VideoPromptSeriesPage() {
  const [items, setItems] = useState<VideoPromptSeriesJob[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [retryingId, setRetryingId] = useState<string | null>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    listVideoPromptSeriesJobs()
      .then(setItems)
      .catch((err) => setError(err instanceof Error ? err.message : 'Không tải được danh sách prompt'))
      .finally(() => setLoading(false))
  }, [])

  const selected = items.find((i) => i.id === selectedId) ?? null

  // A job left "pending" (e.g. the tab was closed mid-generation) has no
  // background worker to finish it — advance it on an interval, same as
  // VideoPromptModal, while it's the one currently open.
  useEffect(() => {
    if (!selected || selected.status !== 'pending') return
    let cancelled = false

    async function poll() {
      try {
        const updated = await stepVideoPromptSeriesJob(selected!.id)
        if (cancelled) return
        setItems((prev) => prev.map((i) => (i.id === updated.id ? updated : i)))
        if (updated.status === 'pending') {
          timerRef.current = setTimeout(poll, POLL_INTERVAL_MS)
        }
      } catch {
        // Selected job just stays pending — the user can reopen it later.
      }
    }

    poll()
    return () => {
      cancelled = true
      if (timerRef.current) clearTimeout(timerRef.current)
    }
    // selected.status is a dep (not just id) so retrying a failed job — which
    // flips it back to "pending" in place, same id — restarts this poll.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected?.id, selected?.status])

  async function handleRetry(item: VideoPromptSeriesJob) {
    setActionError(null)
    setRetryingId(item.id)
    try {
      const updated = await retryVideoPromptSeriesJob(item.id)
      setItems((prev) => prev.map((i) => (i.id === updated.id ? updated : i)))
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Không thử lại được prompt')
    } finally {
      setRetryingId(null)
    }
  }

  async function handleDelete(item: VideoPromptSeriesJob, e: React.MouseEvent) {
    e.stopPropagation() // don't trigger the row's own select-on-click
    if (!window.confirm(`Xoá prompt "${item.videoTitle}"?`)) return
    setActionError(null)
    setDeletingId(item.id)
    try {
      await deleteVideoPromptSeriesJob(item.id)
      setItems((prev) => prev.filter((i) => i.id !== item.id))
      if (selectedId === item.id) setSelectedId(null)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Không xoá được prompt')
    } finally {
      setDeletingId(null)
    }
  }

  if (loading) return <p className="text-slate-600">Đang tải...</p>
  if (error) return <p className="text-red-600">{error}</p>

  return (
    <div>
      <h1 className="text-xl font-semibold text-slate-900">Prompt video nhiều tập</h1>

      {actionError && <p className="mt-3 text-sm text-red-600">{actionError}</p>}

      {items.length === 0 ? (
        <p className="mt-4 text-slate-600">
          Chưa có prompt nào được tạo. Vào một video trong kết quả phân tích hoặc trang xu hướng để
          tạo prompt AI video nhiều tập.
        </p>
      ) : (
        <div className="mt-6 grid gap-6 md:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
          <div className="divide-y divide-slate-200 rounded-lg border border-slate-200 bg-white md:max-h-[70vh] md:overflow-y-auto">
            {items.map((item) => (
              <div
                key={item.id}
                onClick={() => setSelectedId(item.id)}
                className={`flex w-full cursor-pointer items-center justify-between gap-3 px-4 py-3 text-left transition hover:bg-slate-50 ${
                  selectedId === item.id ? 'bg-indigo-50' : ''
                }`}
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium text-slate-900">{item.videoTitle}</p>
                  <p className="mt-0.5 text-xs text-slate-500">
                    {item.episodeCount} tập · {formatDate(item.createdAt)}
                  </p>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <StatusBadge status={item.status} />
                  <button
                    onClick={(e) => handleDelete(item, e)}
                    disabled={deletingId === item.id}
                    className="rounded-md border border-red-200 bg-white px-2 py-1 text-xs font-medium text-red-600 transition hover:bg-red-50 disabled:opacity-50"
                  >
                    {deletingId === item.id ? 'Đang xoá...' : 'Xoá'}
                  </button>
                </div>
              </div>
            ))}
          </div>

          <div className="rounded-lg border border-slate-200 bg-white p-5">
            {!selected && (
              <p className="text-sm text-slate-500">Chọn một prompt bên trái để xem chi tiết.</p>
            )}

            {selected && selected.status === 'pending' && (
              <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-center text-sm text-slate-600">
                <div className="mx-auto mb-2 h-5 w-5 animate-spin rounded-full border-2 border-slate-300 border-t-slate-900" />
                Đang xây dựng cốt truyện...
              </div>
            )}

            {selected && selected.status === 'failed' && (
              <div className="rounded-lg border border-red-200 bg-red-50 p-3">
                <p className="text-sm text-red-700">
                  {selected.errorMessage || 'Đã có lỗi xảy ra khi tạo prompt.'}
                </p>
                <button
                  onClick={() => handleRetry(selected)}
                  disabled={retryingId === selected.id}
                  className="mt-2 rounded-md border border-red-300 bg-white px-3 py-1.5 text-xs font-medium text-red-700 transition hover:bg-red-100 disabled:opacity-50"
                >
                  {retryingId === selected.id ? 'Đang thử lại...' : 'Thử lại'}
                </button>
              </div>
            )}

            {selected && selected.series && (
              <div className="space-y-3">
                <div>
                  <p className="text-xs font-semibold uppercase tracking-wide text-violet-700">
                    {selected.videoTitle}
                  </p>
                  <p className="mt-1 text-xs text-slate-500">
                    {selected.episodeCount} tập · {formatDate(selected.createdAt)}
                  </p>
                </div>

                <div>
                  <p className="text-xs font-medium text-slate-500">Cốt truyện tổng thể</p>
                  <p className="mt-1 rounded-md bg-violet-50 p-3 text-sm leading-relaxed text-slate-800">
                    {selected.series.synopsis}
                  </p>
                </div>

                <div>
                  <p className="text-xs font-medium text-slate-500">
                    Nhân vật — dán "Ngoại hình" kèm mỗi cảnh để giữ hình ảnh đồng nhất
                  </p>
                  <div className="mt-1.5 space-y-2">
                    {selected.series.characters.map((char) => (
                      <div key={char.name} className="rounded-md border border-slate-100 p-3">
                        <div className="flex items-center justify-between gap-2">
                          <p className="text-sm font-semibold text-slate-900">{char.name}</p>
                          <span className="shrink-0 rounded-full bg-violet-50 px-2 py-0.5 text-[10px] font-medium text-violet-700">
                            {char.role}
                          </span>
                        </div>
                        <p className="mt-1 text-xs text-slate-600">{char.personalInfo}</p>
                        <p className="mt-1 text-xs leading-relaxed text-slate-600">
                          {char.personality}
                        </p>
                        {char.coreTags.length > 0 && (
                          <div className="mt-1.5 flex flex-wrap gap-1">
                            {char.coreTags.map((tag) => (
                              <span
                                key={tag}
                                className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] text-slate-600"
                              >
                                {tag}
                              </span>
                            ))}
                          </div>
                        )}
                        <div className="mt-2 flex items-center justify-between gap-2">
                          <p className="text-[11px] font-medium text-slate-500">
                            Ngoại hình (English)
                          </p>
                          <CopyButton text={char.appearance} />
                        </div>
                        <p className="mt-1 rounded bg-slate-50 p-2 font-mono text-[11px] leading-relaxed text-slate-800">
                          {char.appearance}
                        </p>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <p className="text-xs font-medium text-slate-500">Phong cách</p>
                    <p className="mt-1 text-sm text-slate-800">{selected.series.style}</p>
                  </div>
                  <div>
                    <p className="text-xs font-medium text-slate-500">Độ dài mỗi cảnh</p>
                    <p className="mt-1 text-sm text-slate-800">{selected.series.durationHint}</p>
                  </div>
                </div>

                <div className="space-y-3 border-t border-slate-100 pt-3">
                  {selected.series.episodes.map((ep) => (
                    <div key={ep.episodeNumber} className="rounded-md border border-slate-100 p-3">
                      <p className="text-xs font-semibold text-violet-700">
                        Tập {ep.episodeNumber}: {ep.title}
                      </p>
                      <p className="mt-1 text-xs leading-relaxed text-slate-600">{ep.plotSummary}</p>

                      {!ep.scenes?.length && ep.prompt && (
                        <>
                          <div className="mt-1.5 flex items-center justify-between gap-2">
                            <p className="text-[11px] font-medium text-slate-500">Prompt (English)</p>
                            <CopyButton text={ep.prompt} label="Copy" />
                          </div>
                          <p className="mt-1 rounded bg-slate-50 p-2 font-mono text-[11px] leading-relaxed text-slate-800">
                            {ep.prompt}
                          </p>
                        </>
                      )}

                      <div className="mt-2 space-y-2">
                        {(ep.scenes ?? []).map((scene) => (
                          <div
                            key={scene.sceneNumber}
                            className="rounded border border-slate-100 bg-slate-50/60 p-2"
                          >
                            <div className="flex items-center justify-between gap-2">
                              <p className="text-[11px] font-medium text-slate-500">
                                Cảnh {scene.sceneNumber} · {scene.setting}
                              </p>
                              <span className="shrink-0 rounded-full bg-violet-50 px-2 py-0.5 text-[10px] font-medium text-violet-700">
                                {scene.shotType}
                              </span>
                            </div>
                            {scene.characters.length > 0 && (
                              <p className="mt-1 text-[11px] text-slate-500">
                                Nhân vật: {scene.characters.join(', ')}
                              </p>
                            )}
                            <p className="mt-1 text-xs leading-relaxed text-slate-700">{scene.action}</p>
                            <div className="mt-1.5 flex items-center justify-between gap-2">
                              <p className="text-[11px] font-medium text-slate-500">Prompt (English)</p>
                              <CopyButton text={scene.prompt} label="Copy" />
                            </div>
                            <p className="mt-1 rounded bg-white p-2 font-mono text-[11px] leading-relaxed text-slate-800">
                              {scene.prompt}
                            </p>
                          </div>
                        ))}
                      </div>

                      {ep.negativePrompt && (
                        <p className="mt-2 text-[11px] leading-relaxed text-slate-400">
                          Negative (cả tập): {ep.negativePrompt}
                        </p>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
