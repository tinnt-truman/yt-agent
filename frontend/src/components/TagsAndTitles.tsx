import type { TagCount, TitlePatterns } from '../types'
import { formatPercent } from '../lib/format'

export function TagsAndTitles({
  tags,
  titlePatterns,
}: {
  tags: TagCount[]
  titlePatterns: TitlePatterns
}) {
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5">
      <h2 className="text-lg font-semibold text-slate-900">Tag &amp; mẫu tiêu đề</h2>

      {tags.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-2">
          {tags.map((t) => (
            <span
              key={t.tag}
              className="rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-700"
            >
              {t.tag} <span className="text-slate-400">×{t.count}</span>
            </span>
          ))}
        </div>
      )}

      <dl className="mt-4 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
        <div>
          <dt className="text-xs text-slate-500">Độ dài tiêu đề TB</dt>
          <dd className="font-medium text-slate-900">{titlePatterns.avgLength.toFixed(0)} ký tự</dd>
        </div>
        <div>
          <dt className="text-xs text-slate-500">Có số</dt>
          <dd className="font-medium text-slate-900">{formatPercent(titlePatterns.usesNumbersPct)}</dd>
        </div>
        <div>
          <dt className="text-xs text-slate-500">Dạng câu hỏi</dt>
          <dd className="font-medium text-slate-900">{formatPercent(titlePatterns.usesQuestionPct)}</dd>
        </div>
        <div>
          <dt className="text-xs text-slate-500">Có ngoặc</dt>
          <dd className="font-medium text-slate-900">{formatPercent(titlePatterns.usesBracketsPct)}</dd>
        </div>
      </dl>

      {titlePatterns.commonWords.length > 0 && (
        <p className="mt-3 text-sm text-slate-600">
          Từ khoá lặp lại: {titlePatterns.commonWords.join(', ')}
        </p>
      )}
    </section>
  )
}
