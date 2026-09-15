import { useEffect, useRef, useState } from 'react'
import {
  deleteMusicPromptJob,
  listMusicPromptJobs,
  retryMusicPromptJob,
  stepMusicPromptJob,
} from '../api/client'
import { formatDate } from '../lib/format'
import type { MusicPromptJob } from '../types'
import { CopyButton } from '../components/CopyButton'
import { StatusBadge } from '../components/StatusBadge'

const POLL_INTERVAL_MS = 2000
const FLOW_MUSIC_URL = 'https://flowmusic.google.com'

export default function MusicPromptsPage() {
  const [items, setItems] = useState<MusicPromptJob[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [retryingId, setRetryingId] = useState<string | null>(null)
  const [openStatus, setOpenStatus] = useState<'idle' | 'done'>('idle')
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    listMusicPromptJobs()
      .then(setItems)
      .catch((err) => setError(err instanceof Error ? err.message : 'Không tải được danh sách prompt'))
      .finally(() => setLoading(false))
  }, [])

  const selected = items.find((i) => i.id === selectedId) ?? null

  // A job left "pending" (e.g. the tab was closed mid-generation) has no
  // background worker to finish it — advance it on an interval, same as
  // MusicPromptModal, while it's the one currently open.
  useEffect(() => {
    if (!selected || selected.status !== 'pending') return
    let cancelled = false

    async function poll() {
      try {
        const updated = await stepMusicPromptJob(selected!.id)
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

  async function handleRetry(item: MusicPromptJob, e: React.MouseEvent) {
    e.stopPropagation() // don't trigger the row's own select-on-click
    setActionError(null)
    setRetryingId(item.id)
    try {
      const updated = await retryMusicPromptJob(item.id)
      setItems((prev) => prev.map((i) => (i.id === updated.id ? updated : i)))
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Không thử lại được prompt')
    } finally {
      setRetryingId(null)
    }
  }

  async function handleDelete(item: MusicPromptJob, e: React.MouseEvent) {
    e.stopPropagation() // don't trigger the row's own select-on-click
    if (!window.confirm(`Xoá prompt nhạc "${item.videoTitle}"?`)) return
    setActionError(null)
    setDeletingId(item.id)
    try {
      await deleteMusicPromptJob(item.id)
      setItems((prev) => prev.filter((i) => i.id !== item.id))
      if (selectedId === item.id) setSelectedId(null)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Không xoá được prompt')
    } finally {
      setDeletingId(null)
    }
  }

  async function handleOpenFlowMusic() {
    if (!selected?.musicPrompt) return
    try {
      await navigator.clipboard.writeText(selected.musicPrompt.prompt)
    } catch {
      // Clipboard denied/unavailable — the prompt is still visible to copy manually.
    }
    window.open(FLOW_MUSIC_URL, '_blank', 'noopener,noreferrer')
    setOpenStatus('done')
    setTimeout(() => setOpenStatus('idle'), 3000)
  }

  if (loading) return <p className="text-slate-600">Đang tải...</p>
  if (error) return <p className="text-red-600">{error}</p>

  return (
    <div>
      <h1 className="text-xl font-semibold text-slate-900">Prompt nhạc nền</h1>

      {actionError && <p className="mt-3 text-sm text-red-600">{actionError}</p>}

      {items.length === 0 ? (
        <p className="mt-4 text-slate-600">
          Chưa có prompt nhạc nào được tạo. Vào một video trong kết quả phân tích để tạo prompt nhạc
          nền cho Google Flow Music.
        </p>
      ) : (
        <div className="mt-6 grid gap-6 md:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
          <div className="divide-y divide-slate-200 rounded-lg border border-slate-200 bg-white md:max-h-[70vh] md:overflow-y-auto">
            {items.map((item) => (
              <div
                key={item.id}
                onClick={() => setSelectedId(item.id)}
                className={`flex w-full cursor-pointer items-center justify-between gap-3 px-4 py-3 text-left transition hover:bg-slate-50 ${
                  selectedId === item.id ? 'bg-emerald-50' : ''
                }`}
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium text-slate-900">{item.videoTitle}</p>
                  <p className="mt-0.5 text-xs text-slate-500">{formatDate(item.createdAt)}</p>
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
                Đang sáng tác prompt nhạc...
              </div>
            )}

            {selected && selected.status === 'failed' && (
              <div className="rounded-lg border border-red-200 bg-red-50 p-3">
                <p className="text-sm text-red-700">
                  {selected.errorMessage || 'Đã có lỗi xảy ra khi tạo prompt nhạc.'}
                </p>
                <button
                  onClick={(e) => handleRetry(selected, e)}
                  disabled={retryingId === selected.id}
                  className="mt-2 rounded-md border border-red-300 bg-white px-3 py-1.5 text-xs font-medium text-red-700 transition hover:bg-red-100 disabled:opacity-50"
                >
                  {retryingId === selected.id ? 'Đang thử lại...' : 'Thử lại'}
                </button>
              </div>
            )}

            {selected && selected.musicPrompt && (
              <div className="space-y-3">
                <div>
                  <p className="text-xs font-semibold uppercase tracking-wide text-emerald-700">
                    {selected.videoTitle}
                  </p>
                  <p className="mt-1 text-xs text-slate-500">{formatDate(selected.createdAt)}</p>
                </div>

                <div>
                  <div className="flex items-center justify-between">
                    <p className="text-xs font-medium text-slate-500">Prompt cho Flow Music (English)</p>
                    <CopyButton text={selected.musicPrompt.prompt} />
                  </div>
                  <p className="mt-1 rounded-md bg-slate-50 p-3 font-mono text-xs leading-relaxed text-slate-800">
                    {selected.musicPrompt.prompt}
                  </p>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <p className="text-xs font-medium text-slate-500">Thể loại</p>
                    <p className="mt-1 text-sm text-slate-800">{selected.musicPrompt.genre}</p>
                  </div>
                  <div>
                    <p className="text-xs font-medium text-slate-500">Độ dài gợi ý</p>
                    <p className="mt-1 text-sm text-slate-800">{selected.musicPrompt.durationHint}</p>
                  </div>
                </div>

                {selected.musicPrompt.moodTags.length > 0 && (
                  <div>
                    <p className="text-xs font-medium text-slate-500">Tâm trạng</p>
                    <div className="mt-1.5 flex flex-wrap gap-1">
                      {selected.musicPrompt.moodTags.map((tag) => (
                        <span
                          key={tag}
                          className="rounded-full bg-emerald-50 px-2 py-0.5 text-[10px] text-emerald-700"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
                  </div>
                )}

                {selected.musicPrompt.lyricsHint && (
                  <div>
                    <div className="flex items-center justify-between">
                      <p className="text-xs font-medium text-slate-500">Gợi ý lời bài hát</p>
                      <CopyButton text={selected.musicPrompt.lyricsHint} />
                    </div>
                    <p className="mt-1 whitespace-pre-line rounded-md bg-slate-50 p-3 text-sm leading-relaxed text-slate-800">
                      {selected.musicPrompt.lyricsHint}
                    </p>
                  </div>
                )}

                <div className="border-t border-slate-100 pt-3">
                  <button
                    onClick={handleOpenFlowMusic}
                    className="rounded-md bg-emerald-100 px-3 py-1.5 text-xs font-medium text-emerald-700 transition hover:bg-emerald-200"
                  >
                    {openStatus === 'done' ? 'Đã copy prompt — đã mở Flow Music!' : 'Mở Google Flow Music'}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
