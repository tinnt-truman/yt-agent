import type { StrategyOutput } from '../types'
import { CopyButton } from './CopyButton'

const PILLAR_COLORS = ['bg-indigo-600', 'bg-violet-500', 'bg-amber-400', 'bg-slate-300', 'bg-emerald-400']
const PILLAR_DOT_COLORS = ['bg-indigo-600', 'bg-violet-500', 'bg-amber-400', 'bg-slate-300', 'bg-emerald-400']

const WEEK_DAYS = [
  { short: 'T2', full: 'thu hai' },
  { short: 'T3', full: 'thu ba' },
  { short: 'T4', full: 'thu tu' },
  { short: 'T5', full: 'thu nam' },
  { short: 'T6', full: 'thu sau' },
  { short: 'T7', full: 'thu bay' },
  { short: 'CN', full: 'chu nhat' },
]

function normalize(s: string): string {
  return s
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .trim()
}

export function StrategySections({ strategy }: { strategy: StrategyOutput }) {
  const bestDaysNormalized = strategy.postingSchedule.bestDays.map(normalize)

  return (
    <div className="space-y-4">
      {/* Positioning */}
      <section className="rounded-lg border border-indigo-100 bg-indigo-50 p-5">
        <p className="text-xs font-semibold uppercase tracking-wide text-indigo-700">Ngách</p>
        <p className="mt-1.5 text-lg font-semibold leading-snug text-slate-900">
          {strategy.positioning.niche}
        </p>

        <div className="mt-5 grid grid-cols-1 gap-5 sm:grid-cols-2">
          <div>
            <p className="text-xs font-semibold text-indigo-700">Đối tượng mục tiêu</p>
            <p className="mt-1 text-sm leading-relaxed text-slate-700">
              {strategy.positioning.targetAudience}
            </p>
          </div>
          <div>
            <p className="text-xs font-semibold text-indigo-700">Góc nhìn khác biệt</p>
            <p className="mt-1 text-sm leading-relaxed text-slate-700">
              {strategy.positioning.uniqueAngle}
            </p>
          </div>
        </div>

        {strategy.positioning.channelNameIdeas.length > 0 && (
          <div className="mt-5">
            <p className="text-xs font-semibold text-indigo-700">Gợi ý tên kênh</p>
            <div className="mt-2 flex flex-wrap gap-2">
              {strategy.positioning.channelNameIdeas.map((name) => (
                <CopyButton
                  key={name}
                  text={name}
                  label={name}
                  bare
                  className="rounded-full border border-indigo-200 bg-white px-3 py-1 text-indigo-700 hover:bg-indigo-50"
                />
              ))}
            </div>
          </div>
        )}
      </section>

      {/* Content pillars */}
      <section className="rounded-lg border border-slate-200 bg-white p-5">
        <h2 className="text-base font-semibold text-slate-900">Trụ cột nội dung</h2>
        <div className="mt-4 flex h-3 w-full overflow-hidden rounded-full">
          {strategy.contentPillars.map((p, i) => (
            <div
              key={p.name}
              className={PILLAR_COLORS[i % PILLAR_COLORS.length]}
              style={{ width: `${p.percentage}%` }}
            />
          ))}
        </div>
        <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {strategy.contentPillars.map((p, i) => (
            <div key={p.name}>
              <div className="flex items-center gap-2">
                <span className={`h-2 w-2 shrink-0 rounded-full ${PILLAR_DOT_COLORS[i % PILLAR_DOT_COLORS.length]}`} />
                <span className="text-sm font-semibold text-slate-900">
                  {p.percentage}% · {p.name}
                </span>
              </div>
              <p className="mt-1 text-xs leading-relaxed text-slate-500">{p.description}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Content ideas */}
      <section className="rounded-lg border border-slate-200 bg-white p-5">
        <h2 className="text-base font-semibold text-slate-900">Ý tưởng nội dung</h2>
        <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
          {strategy.contentIdeas.map((idea, i) => (
            <div key={i} className="rounded-md border border-slate-100 p-3.5">
              <div className="flex items-start justify-between gap-2">
                <p className="text-sm font-semibold leading-snug text-slate-900">{idea.title}</p>
                <span
                  className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium ${
                    idea.format.toLowerCase().includes('short')
                      ? 'bg-violet-50 text-violet-700'
                      : 'bg-indigo-50 text-indigo-700'
                  }`}
                >
                  {idea.format}
                </span>
              </div>
              <p className="mt-1.5 text-xs leading-relaxed text-slate-500">{idea.description}</p>
              <div className="mt-2 flex items-center justify-between gap-2">
                <p className="text-[11px] text-slate-400">
                  Hook: {idea.hook} · {idea.estimatedLength}
                </p>
                <CopyButton text={idea.title} label="Copy tiêu đề" />
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Posting schedule */}
      <section className="rounded-lg border border-slate-200 bg-white p-5">
        <h2 className="text-base font-semibold text-slate-900">Lịch đăng gợi ý</h2>
        <div className="mt-4 flex flex-wrap items-center justify-between gap-5">
          <div className="flex gap-2">
            {WEEK_DAYS.map((day) => {
              const isBest = bestDaysNormalized.some(
                (d) => d.includes(day.full) || day.full.includes(d),
              )
              return (
                <div
                  key={day.short}
                  className={`flex h-9 w-9 items-center justify-center rounded-full text-xs font-semibold ${
                    isBest ? 'bg-indigo-600 text-white' : 'bg-slate-100 text-slate-400'
                  }`}
                >
                  {day.short}
                </div>
              )
            })}
          </div>
          <div className="flex gap-7">
            <div>
              <p className="text-xs text-slate-500">Tần suất</p>
              <p className="text-lg font-bold text-slate-900">
                {strategy.postingSchedule.frequencyPerWeek} video
                <span className="text-xs font-medium text-slate-500">/tuần</span>
              </p>
            </div>
            <div>
              <p className="text-xs text-slate-500">Giờ vàng</p>
              <p className="text-lg font-bold text-slate-900">{strategy.postingSchedule.bestTimeOfDay}</p>
            </div>
          </div>
        </div>
      </section>

      {/* Title templates + SEO keywords */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <section className="rounded-lg border border-slate-200 bg-white p-5">
          <h2 className="text-base font-semibold text-slate-900">Mẫu tiêu đề</h2>
          <ul className="mt-3 space-y-2">
            {strategy.titleTemplates.map((t, i) => (
              <li key={i} className="flex items-start justify-between gap-2">
                <span className="text-sm leading-relaxed text-slate-700">
                  {i + 1}. {t}
                </span>
                <CopyButton text={t} className="shrink-0" />
              </li>
            ))}
          </ul>
        </section>

        <section className="rounded-lg border border-slate-200 bg-white p-5">
          <div className="flex items-center justify-between gap-2">
            <h2 className="text-base font-semibold text-slate-900">Từ khoá SEO</h2>
            <CopyButton text={strategy.seoKeywords.join(', ')} label="Copy tất cả" />
          </div>
          <div className="mt-3 flex flex-wrap gap-2">
            {strategy.seoKeywords.map((k) => (
              <span key={k} className="rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-700">
                {k}
              </span>
            ))}
          </div>
        </section>
      </div>

      {/* Hashtag sets */}
      <section className="rounded-lg border border-slate-200 bg-white p-5">
        <h2 className="text-base font-semibold text-slate-900">Bộ hashtag</h2>
        <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
          {strategy.hashtagSets.map((set) => (
            <div key={set.theme} className="rounded-md border border-slate-100 p-3">
              <div className="flex items-center justify-between gap-2">
                <p className="text-xs font-semibold text-slate-900">{set.theme}</p>
                <CopyButton text={set.hashtags.join(' ')} />
              </div>
              <p className="mt-1.5 text-sm leading-relaxed text-indigo-600">{set.hashtags.join(' ')}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Growth tactics */}
      <section className="rounded-lg border border-slate-200 bg-white p-5">
        <h2 className="text-base font-semibold text-slate-900">Chiến thuật tăng trưởng</h2>
        <div className="mt-3 flex flex-col gap-2.5">
          {strategy.growthTactics.map((t, i) => (
            <div key={i} className="flex items-start gap-2.5">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="#16a34a"
                strokeWidth="2.5"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="mt-0.5 shrink-0"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
              <p className="text-sm leading-relaxed text-slate-700">{t}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Risks */}
      {strategy.risks.length > 0 && (
        <section className="rounded-lg border border-amber-200 bg-amber-50 p-5">
          <div className="flex items-center gap-2">
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="#b45309"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
              <line x1="12" y1="9" x2="12" y2="13" />
              <line x1="12" y1="17" x2="12.01" y2="17" />
            </svg>
            <h2 className="text-base font-semibold text-amber-900">Rủi ro cần lưu ý</h2>
          </div>
          <div className="mt-3 flex flex-col gap-1.5">
            {strategy.risks.map((t, i) => (
              <p key={i} className="text-sm leading-relaxed text-amber-900">
                • {t}
              </p>
            ))}
          </div>
        </section>
      )}
    </div>
  )
}
