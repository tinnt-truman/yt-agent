import type { VideoInfo } from '../types'
import { formatCompact, formatDate } from '../lib/format'

export function TopVideos({ videos }: { videos: VideoInfo[] }) {
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5">
      <h2 className="text-base font-semibold text-slate-900">Video nổi bật</h2>
      <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
        {videos.map((v) => (
          <a
            key={v.id}
            href={`https://www.youtube.com/watch?v=${v.id}`}
            target="_blank"
            rel="noreferrer"
            className="group"
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
        ))}
      </div>
    </section>
  )
}
