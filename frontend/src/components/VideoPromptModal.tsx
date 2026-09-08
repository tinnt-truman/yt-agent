import { useState } from 'react'
import { generateVideoPromptSeries } from '../api/client'
import type { VideoPromptRequest, VideoPromptSeries } from '../types'
import { CopyButton } from './CopyButton'

const EPISODE_OPTIONS = [3, 5, 10]

export function VideoPromptModal({
  video,
  onClose,
}: {
  video: VideoPromptRequest
  onClose: () => void
}) {
  const [episodeCount, setEpisodeCount] = useState(5)
  const [series, setSeries] = useState<VideoPromptSeries | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleGenerate() {
    setLoading(true)
    setError(null)
    try {
      const result = await generateVideoPromptSeries({ ...video, episodeCount })
      setSeries(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Không tạo được prompt')
    } finally {
      setLoading(false)
    }
  }

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
                disabled={loading}
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

        <button
          onClick={handleGenerate}
          disabled={loading}
          className="mt-4 w-full rounded-md bg-violet-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-violet-700 disabled:opacity-50"
        >
          {loading ? 'Đang xây dựng cốt truyện...' : series ? 'Tạo lại' : 'Tạo prompt nhiều tập'}
        </button>

        {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

        {series && (
          <div className="mt-4 space-y-3">
            <div>
              <p className="text-xs font-medium text-slate-500">Cốt truyện tổng thể</p>
              <p className="mt-1 rounded-md bg-violet-50 p-3 text-sm leading-relaxed text-slate-800">
                {series.synopsis}
              </p>
            </div>

            <div>
              <p className="text-xs font-medium text-slate-500">
                Nhân vật — dán "Ngoại hình" kèm mỗi tập để giữ hình ảnh đồng nhất
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
                <p className="text-xs font-medium text-slate-500">Độ dài mỗi tập</p>
                <p className="mt-1 text-sm text-slate-800">{series.durationHint}</p>
              </div>
            </div>

            <div className="space-y-2 border-t border-slate-100 pt-3">
              {series.episodes.map((ep) => (
                <div key={ep.episodeNumber} className="rounded-md border border-slate-100 p-3">
                  <div className="flex items-start justify-between gap-2">
                    <p className="text-xs font-semibold text-violet-700">
                      Tập {ep.episodeNumber}: {ep.title}
                    </p>
                    <CopyButton text={ep.prompt} label="Copy prompt" />
                  </div>
                  <p className="mt-1 text-xs leading-relaxed text-slate-600">{ep.plotSummary}</p>
                  <p className="mt-1.5 rounded bg-slate-50 p-2 font-mono text-[11px] leading-relaxed text-slate-800">
                    {ep.prompt}
                  </p>
                  {ep.negativePrompt && (
                    <p className="mt-1 text-[11px] leading-relaxed text-slate-400">
                      Negative: {ep.negativePrompt}
                    </p>
                  )}
                </div>
              ))}
            </div>

            <p className="text-[11px] text-slate-400">
              Tạo từng tập riêng trên công cụ AI video (Kling, Google Flow, Sora...), luôn dán kèm
              phần "Ngoại hình" của nhân vật xuất hiện trong tập để giữ hình ảnh đồng nhất.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
