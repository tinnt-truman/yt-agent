import { useState } from 'react'
import type { VideoInfo } from '../types'
import { formatCompact, formatDate, formatDuration } from '../lib/format'
import { ScriptModal } from './ScriptModal'
import { VideoPromptModal } from './VideoPromptModal'

export function TopVideos({ videos }: { videos: VideoInfo[] }) {
  const [promptVideo, setPromptVideo] = useState<VideoInfo | null>(null)
  const [scriptVideo, setScriptVideo] = useState<VideoInfo | null>(null)

  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5">
      <h2 className="text-base font-semibold text-slate-900">Video nổi bật</h2>
      <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
        {videos.map((v) => (
          <div key={v.id}>
            <a
              href={`https://www.youtube.com/watch?v=${v.id}`}
              target="_blank"
              rel="noreferrer"
              className="group block"
            >
              {v.thumbnail ? (
                <img
                  src={v.thumbnail}
                  alt=""
                  className="aspect-video w-full rounded-md object-cover"
                />
              ) : (
                <div className="aspect-video w-full rounded-md bg-slate-100" />
              )}
              <p className="mt-1.5 line-clamp-2 text-xs font-medium text-slate-900 group-hover:text-indigo-600">
                {v.title}
              </p>
              <p className="mt-1 text-[11px] text-slate-400">
                {formatCompact(v.viewCount)} views · {formatDate(v.publishedAt)}
              </p>
            </a>
            <button
              onClick={() => setPromptVideo(v)}
              className="mt-1.5 block text-[11px] font-medium text-violet-700 hover:text-violet-900"
            >
              ✦ Tạo prompt AI video
            </button>
            <button
              onClick={() => setScriptVideo(v)}
              className="mt-1 block text-[11px] font-medium text-indigo-700 hover:text-indigo-900"
            >
              Tạo kịch bản tóm tắt
            </button>
          </div>
        ))}
      </div>

      {promptVideo && (
        <VideoPromptModal
          video={{
            title: promptVideo.title,
            description: promptVideo.description,
            tags: promptVideo.tags,
          }}
          onClose={() => setPromptVideo(null)}
        />
      )}

      {scriptVideo && (
        <ScriptModal
          idea={{
            title: scriptVideo.title,
            description: scriptVideo.description ?? '',
            hook: scriptVideo.title,
            format: scriptVideo.isShort ? 'Shorts' : 'Long-form',
            estimatedLength: formatDuration(scriptVideo.durationSeconds),
          }}
          onClose={() => setScriptVideo(null)}
        />
      )}
    </section>
  )
}
