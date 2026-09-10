import { useEffect, useRef, useState } from 'react'
import { createVideoPromptSeriesJob, stepVideoPromptSeriesJob } from '../api/client'
import type { VideoPromptRequest, VideoPromptSeriesJob } from '../types'
import { CopyButton } from './CopyButton'

const EPISODE_OPTIONS = [3, 5, 10]
const SCENES_OPTIONS = [2, 3, 5]
const POLL_INTERVAL_MS = 2000
// Mirrors the backend's ClampVideoPromptSeriesDimensions cap: episodeCount x
// scenesPerEpisode is bounded so the AI's JSON output stays within its
// completion-token budget. Disabling invalid combos here avoids the backend
// silently generating fewer scenes than picked.
const MAX_TOTAL_SCENES = 30

export function VideoPromptModal({
  video,
  onClose,
}: {
  video: VideoPromptRequest
  onClose: () => void
}) {
  const [episodeCount, setEpisodeCount] = useState(5)
  const [scenesPerEpisode, setScenesPerEpisode] = useState(3)
  const [job, setJob] = useState<VideoPromptSeriesJob | null>(null)

  // Keep the selected scene count valid whenever episodeCount changes —
  // e.g. switching to 10 episodes invalidates "5 cảnh" (would be 50 scenes).
  useEffect(() => {
    const validOptions = SCENES_OPTIONS.filter((n) => n * episodeCount <= MAX_TOTAL_SCENES)
    if (validOptions.length > 0 && !validOptions.includes(scenesPerEpisode)) {
      setScenesPerEpisode(validOptions[validOptions.length - 1])
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [episodeCount])
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Queued generation: the backend has no background worker (same reason as
  // analyses/scripts — see stepAnalysis), so this drives the job forward by
  // calling Step on an interval while it's still pending.
  useEffect(() => {
    if (!job || job.status !== 'pending') return
    let cancelled = false

    async function poll() {
      try {
        const updated = await stepVideoPromptSeriesJob(job!.id)
        if (cancelled) return
        setJob(updated)
        if (updated.status === 'pending') {
          timerRef.current = setTimeout(poll, POLL_INTERVAL_MS)
        }
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Không tạo được prompt')
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
      const created = await createVideoPromptSeriesJob({ ...video, episodeCount, scenesPerEpisode })
      setJob(created)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Không tạo được prompt')
    } finally {
      setCreating(false)
    }
  }

  const series = job?.series

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="max-h-[85vh] w-full max-w-xl overflow-y-auto rounded-lg bg-white p-5 shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs font-semibold uppercase tracking-wide text-violet-700">
              Prompt AI video nhiều tập
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

        <div className="mt-4">
          <p className="text-xs font-medium text-slate-500">Số tập</p>
          <div className="mt-1.5 flex gap-2">
            {EPISODE_OPTIONS.map((n) => (
              <button
                key={n}
                onClick={() => setEpisodeCount(n)}
                disabled={creating}
                className={`flex-1 rounded-md border px-3 py-2 text-sm font-medium transition disabled:opacity-50 ${
                  episodeCount === n
                    ? 'border-violet-600 bg-violet-50 text-violet-700'
                    : 'border-slate-200 text-slate-600 hover:bg-slate-50'
                }`}
              >
                {n} tập
              </button>
            ))}
          </div>
        </div>

        <div className="mt-3">
          <p className="text-xs font-medium text-slate-500">Số cảnh mỗi tập</p>
          <div className="mt-1.5 flex gap-2">
            {SCENES_OPTIONS.map((n) => {
              const tooMany = n * episodeCount > MAX_TOTAL_SCENES
              return (
                <button
                  key={n}
                  onClick={() => setScenesPerEpisode(n)}
                  disabled={creating || tooMany}
                  title={tooMany ? `${n} cảnh × ${episodeCount} tập vượt quá ${MAX_TOTAL_SCENES} cảnh tối đa` : undefined}
                  className={`flex-1 rounded-md border px-3 py-2 text-sm font-medium transition disabled:opacity-40 ${
                    scenesPerEpisode === n
                      ? 'border-violet-600 bg-violet-50 text-violet-700'
                      : 'border-slate-200 text-slate-600 hover:bg-slate-50'
                  }`}
                >
                  {n} cảnh
                </button>
              )
            })}
          </div>
          <p className="mt-1 text-[11px] text-slate-400">
            Tổng {episodeCount * scenesPerEpisode} cảnh (tối đa {MAX_TOTAL_SCENES})
          </p>
        </div>

        <button
          onClick={handleGenerate}
          disabled={creating}
          className="mt-4 w-full rounded-md bg-violet-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-violet-700 disabled:opacity-50"
        >
          {creating ? 'Đang xếp hàng...' : job ? 'Tạo lại' : 'Tạo prompt nhiều tập'}
        </button>

        {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

        {job && job.status === 'pending' && (
          <div className="mt-4 rounded-lg border border-slate-200 bg-slate-50 p-4 text-center text-sm text-slate-600">
            <div className="mx-auto mb-2 h-5 w-5 animate-spin rounded-full border-2 border-slate-300 border-t-slate-900" />
            Đang xây dựng cốt truyện...
          </div>
        )}

        {job && job.status === 'failed' && (
          <p className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
            {job.errorMessage || 'Đã có lỗi xảy ra khi tạo prompt.'}
          </p>
        )}

        {series && (
          <div className="mt-4 space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <p className="text-xs font-medium text-slate-500">Thể loại</p>
                <p className="mt-1 text-sm text-slate-800">{series.genre}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Đối tượng</p>
                <p className="mt-1 text-sm text-slate-800">{series.targetAudience}</p>
              </div>
            </div>

            <div>
              <p className="text-xs font-medium text-slate-500">Logline</p>
              <p className="mt-1 text-sm italic leading-relaxed text-slate-700">{series.logline}</p>
            </div>

            <div>
              <p className="text-xs font-medium text-slate-500">Cốt truyện tổng thể</p>
              <p className="mt-1 rounded-md bg-violet-50 p-3 text-sm leading-relaxed text-slate-800">
                {series.synopsis}
              </p>
            </div>

            <div>
              <p className="text-xs font-medium text-slate-500">
                Nhân vật — dán "Ngoại hình" kèm mỗi cảnh để giữ hình ảnh đồng nhất
              </p>
              <div className="mt-1.5 space-y-2">
                {series.characters.map((char) => (
                  <div key={char.name} className="rounded-md border border-slate-100 p-3">
                    <div className="flex items-center justify-between gap-2">
                      <p className="text-sm font-semibold text-slate-900">{char.name}</p>
                      <span className="shrink-0 rounded-full bg-violet-50 px-2 py-0.5 text-[10px] font-medium text-violet-700">
                        {char.role}
                      </span>
                    </div>
                    <p className="mt-1 text-xs text-slate-600">{char.personalInfo}</p>
                    <p className="mt-1 text-xs leading-relaxed text-slate-600">{char.personality}</p>
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
                      <p className="text-[11px] font-medium text-slate-500">Ngoại hình (English)</p>
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
                <p className="mt-1 text-sm text-slate-800">{series.style}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Độ dài mỗi cảnh</p>
                <p className="mt-1 text-sm text-slate-800">{series.durationHint}</p>
              </div>
            </div>

            <div className="space-y-3 border-t border-slate-100 pt-3">
              {series.episodes.map((ep) => (
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
                      <div key={scene.sceneNumber} className="rounded border border-slate-100 bg-slate-50/60 p-2">
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

            <p className="text-[11px] text-slate-400">
              Tạo từng cảnh riêng trên công cụ AI video (Kling, Google Flow, Sora...), luôn dán kèm
              phần "Ngoại hình" của nhân vật xuất hiện trong cảnh để giữ hình ảnh đồng nhất. Xem lại
              trong trang "Prompt video" ở menu.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
