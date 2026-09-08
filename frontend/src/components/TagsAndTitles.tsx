import type { TagCount, TitlePatterns } from '../types'
import { formatPercent } from '../lib/format'

export function TagsAndTitles({
  tags,
  titlePatterns,
}: {
  tags: TagCount[]
  titlePatterns: TitlePatterns
}) {
  const maxCount = tags.length > 0 ? Math.max(...tags.map((t) => t.count)) : 0

  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5">
      <h2 className="text-base font-semibold text-slate-900">DNA nội dung</h2>

      {tags.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-2">
          {tags.map((t) => {
            const weight = maxCount > 0 ? t.count / maxCount : 0
            const sizeClass =
              weight > 0.66 ? 'text-sm px-3.5 py-1.5' : weight > 0.33 ? 'text-xs px-3 py-1' : 'text-[11px] px-2.5 py-0.5'
            return (
              <span
                key={t.tag}
                className={`rounded-full bg-slate-100 font-medium text-slate-700 ${sizeClass}`}
              >
                {t.tag} <span className="text-slate-400">×{t.count}</span>
              </span>
            )
          })}
        </div>
      )}

      <dl className="mt-5 grid grid-cols-2 gap-4 border-t border-slate-100 pt-4 text-sm sm:grid-cols-4">
        <div>
          <dt className="text-xs text-slate-500">Độ dài tiêu đề TB</dt>
          <dd className="mt-1 font-semibold text-slate-900">{titlePatterns.avgLength.toFixed(0)} ký tự</dd>
        </div>
        <div>
          <dt className="text-xs text-slate-500">Có số</dt>
          <dd className="mt-1 font-semibold text-slate-900">{formatPercent(titlePatterns.usesNumbersPct)}</dd>
        </div>
        <div>
          <dt className="text-xs text-slate-500">Dạng câu hỏi</dt>
          <dd className="mt-1 font-semibold text-slate-900">{formatPercent(titlePatterns.usesQuestionPct)}</dd>
        </div>
        <div>
          <dt className="text-xs text-slate-500">Có ngoặc</dt>
          <dd className="mt-1 font-semibold text-slate-900">{formatPercent(titlePatterns.usesBracketsPct)}</dd>
        </div>
      </dl>

      {titlePatterns.commonWords.length > 0 && (
        <p className="mt-4 text-sm text-slate-600">
          Từ khoá lặp lại: {titlePatterns.commonWords.join(', ')}
        </p>
      )}
    </section>
  )
}
