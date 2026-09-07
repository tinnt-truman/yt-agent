import type { VideoInfo } from '../types'
import { formatCompact, formatDate } from '../lib/format'

export function TopVideos({ videos }: { videos: VideoInfo[] }) {
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5">
      <h2 className="text-lg font-semibold text-slate-900">Video nổi bật</h2>
      <div className="mt-4 space-y-3">
        {videos.map((v) => (
          <a
            key={v.id}
            href={`https://www.youtube.com/watch?v=${v.id}`}
            target="_blank"
            rel="noreferrer"
            className="flex gap-3 rounded-md p-2 hover:bg-slate-50"
          >
            {v.thumbnail && (
              <img src={v.thumbnail} alt="" className="h-16 w-28 shrink-0 rounded object-cover" />
            )}
            <div className="min-w-0">
              <p className="truncate text-sm font-medium text-slate-900">{v.title}</p>
              <p className="mt-1 text-xs text-slate-500">
                {formatCompact(v.viewCount)} views · {formatCompact(v.likeCount)} likes ·{' '}
                {formatDate(v.publishedAt)}
              </p>
            </div>
          </a>
        ))}
      </div>
    </section>
  )
}
