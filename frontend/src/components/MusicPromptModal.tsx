import { useEffect, useRef, useState } from 'react'
import { createMusicPromptJob, stepMusicPromptJob } from '../api/client'
import type { MusicPromptJob, MusicPromptRequest } from '../types'
import { CopyButton } from './CopyButton'

const POLL_INTERVAL_MS = 2000
const FLOW_MUSIC_URL = 'https://flowmusic.google.com'

export function MusicPromptModal({
  video,
  onClose,
}: {
  video: MusicPromptRequest
  onClose: () => void
}) {
  const [job, setJob] = useState<MusicPromptJob | null>(null)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [openStatus, setOpenStatus] = useState<'idle' | 'done'>('idle')
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Queued generation: the backend has no background worker (same reason as
  // analyses/scripts/video prompts — see stepAnalysis), so this drives the
  // job forward by calling Step on an interval while it's still pending.
  useEffect(() => {
    if (!job || job.status !== 'pending') return
    let cancelled = false

    async function poll() {
      try {
        const updated = await stepMusicPromptJob(job!.id)
        if (cancelled) return
        setJob(updated)
        if (updated.status === 'pending') {
          timerRef.current = setTimeout(poll, POLL_INTERVAL_MS)
        }
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Không tạo được prompt nhạc')
      }
    }

    poll()
    return () => {
      cancelled = true
      if (timerRef.current) clearTimeout(timerRef.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [job?.id])

  async function handleGenerate() {
    setCreating(true)
    setError(null)
    try {
      const created = await createMusicPromptJob(video)
      setJob(created)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Không tạo được prompt nhạc')
    } finally {
      setCreating(false)
    }
  }

  async function handleOpenFlowMusic() {
    if (!job?.musicPrompt) return
    try {
      await navigator.clipboard.writeText(job.musicPrompt.prompt)
    } catch {
      // Clipboard denied/unavailable — the prompt is still visible to copy manually.
    }
    window.open(FLOW_MUSIC_URL, '_blank', 'noopener,noreferrer')
    setOpenStatus('done')
    setTimeout(() => setOpenStatus('idle'), 3000)
  }

  const music = job?.musicPrompt

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="max-h-[85vh] w-full max-w-lg overflow-y-auto rounded-lg bg-white p-5 shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs font-semibold uppercase tracking-wide text-emerald-700">
              Prompt nhạc nền (Google Flow Music)
            </p>
            <p className="mt-1 truncate text-sm text-slate-500">Dựa trên: {video.title}</p>
          </div>
          <button
            onClick={onClose}
            className="shrink-0 rounded-md p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
            aria-label="Đóng"
          >
            ✕
          </button>
        </div>

        <button
          onClick={handleGenerate}
          disabled={creating}
          className="mt-4 w-full rounded-md bg-emerald-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-emerald-700 disabled:opacity-50"
        >
          {creating ? 'Đang xếp hàng...' : job ? 'Tạo lại' : 'Tạo prompt nhạc nền'}
        </button>

        {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

        {job && job.status === 'pending' && (
          <div className="mt-4 rounded-lg border border-slate-200 bg-slate-50 p-4 text-center text-sm text-slate-600">
            <div className="mx-auto mb-2 h-5 w-5 animate-spin rounded-full border-2 border-slate-300 border-t-slate-900" />
            Đang sáng tác prompt nhạc...
          </div>
        )}

        {job && job.status === 'failed' && (
          <p className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
            {job.errorMessage || 'Đã có lỗi xảy ra khi tạo prompt nhạc.'}
          </p>
        )}

        {music && (
          <div className="mt-4 space-y-3">
            <div>
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium text-slate-500">Prompt cho Flow Music (English)</p>
                <CopyButton text={music.prompt} />
              </div>
              <p className="mt-1 rounded-md bg-slate-50 p-3 font-mono text-xs leading-relaxed text-slate-800">
                {music.prompt}
              </p>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <p className="text-xs font-medium text-slate-500">Thể loại</p>
                <p className="mt-1 text-sm text-slate-800">{music.genre}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Độ dài gợi ý</p>
                <p className="mt-1 text-sm text-slate-800">{music.durationHint}</p>
              </div>
            </div>

            {music.moodTags.length > 0 && (
              <div>
                <p className="text-xs font-medium text-slate-500">Tâm trạng</p>
                <div className="mt-1.5 flex flex-wrap gap-1">
                  {music.moodTags.map((tag) => (
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

            {music.lyricsHint && (
              <div>
                <div className="flex items-center justify-between">
                  <p className="text-xs font-medium text-slate-500">Gợi ý lời bài hát</p>
                  <CopyButton text={music.lyricsHint} />
                </div>
                <p className="mt-1 whitespace-pre-line rounded-md bg-slate-50 p-3 text-sm leading-relaxed text-slate-800">
                  {music.lyricsHint}
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
              <p className="mt-1.5 text-[11px] text-slate-400">
                Prompt đã copy vào clipboard — dán vào Flow Music để tạo nhạc.
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
