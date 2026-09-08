import { useEffect, useState } from 'react'
import { generateVideoPrompt } from '../api/client'
import type { VideoPrompt, VideoPromptRequest } from '../types'
import { CopyButton } from './CopyButton'

export function VideoPromptModal({
  video,
  onClose,
}: {
  video: VideoPromptRequest
  onClose: () => void
}) {
  const [prompt, setPrompt] = useState<VideoPrompt | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setLoading(true)
    setError(null)
    generateVideoPrompt(video)
      .then(setPrompt)
      .catch((err) => setError(err instanceof Error ? err.message : 'Không tạo được prompt'))
      .finally(() => setLoading(false))
    // Keyed on title only: this modal is mounted fresh per video (see call
    // sites), so title alone is enough to detect "a different video opened".
  }, [video.title])

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-lg rounded-lg bg-white p-5 shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs font-semibold uppercase tracking-wide text-violet-700">
              Prompt AI video (text-to-video)
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

        {loading && <p className="mt-4 text-sm text-slate-600">Đang tạo prompt...</p>}
        {error && <p className="mt-4 text-sm text-red-600">{error}</p>}

        {prompt && (
          <div className="mt-4 space-y-3">
            <div>
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium text-slate-500">Prompt</p>
                <CopyButton text={prompt.prompt} />
              </div>
              <p className="mt-1 rounded-md bg-slate-50 p-3 font-mono text-xs leading-relaxed text-slate-800">
                {prompt.prompt}
              </p>
            </div>

            {prompt.negativePrompt && (
              <div>
                <p className="text-xs font-medium text-slate-500">Negative prompt</p>
                <p className="mt-1 rounded-md bg-slate-50 p-3 font-mono text-xs leading-relaxed text-slate-600">
                  {prompt.negativePrompt}
                </p>
              </div>
            )}

            <div className="grid grid-cols-2 gap-3">
              <div>
                <p className="text-xs font-medium text-slate-500">Phong cách</p>
                <p className="mt-1 text-sm text-slate-800">{prompt.style}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Độ dài gợi ý</p>
                <p className="mt-1 text-sm text-slate-800">{prompt.durationHint}</p>
              </div>
            </div>

            <p className="text-xs text-slate-400">
              Dùng với các công cụ text-to-video như Kling, Runway, Sora...
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
