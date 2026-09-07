import type { StrategyOutput } from '../types'

export function StrategySections({ strategy }: { strategy: StrategyOutput }) {
  return (
    <div className="space-y-6">
      <Section title="Định vị kênh mới">
        <p className="text-sm text-slate-700">
          <span className="font-medium">Ngách:</span> {strategy.positioning.niche}
        </p>
        <p className="mt-2 text-sm text-slate-700">
          <span className="font-medium">Đối tượng mục tiêu:</span>{' '}
          {strategy.positioning.targetAudience}
        </p>
        <p className="mt-2 text-sm text-slate-700">
          <span className="font-medium">Góc nhìn khác biệt:</span> {strategy.positioning.uniqueAngle}
        </p>
        {strategy.positioning.channelNameIdeas.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-2">
            {strategy.positioning.channelNameIdeas.map((name) => (
              <span
                key={name}
                className="rounded-full bg-indigo-50 px-3 py-1 text-xs font-medium text-indigo-700"
              >
                {name}
              </span>
            ))}
          </div>
        )}
      </Section>

      <Section title="Trụ cột nội dung (Content Pillars)">
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {strategy.contentPillars.map((p) => (
            <div key={p.name} className="rounded-md border border-slate-200 p-3">
              <div className="flex items-center justify-between">
                <p className="font-medium text-slate-900">{p.name}</p>
                <span className="text-xs font-semibold text-slate-500">{p.percentage}%</span>
              </div>
              <p className="mt-1 text-sm text-slate-600">{p.description}</p>
            </div>
          ))}
        </div>
      </Section>

      <Section title="Ý tưởng nội dung">
        <div className="space-y-3">
          {strategy.contentIdeas.map((idea, i) => (
            <div key={i} className="rounded-md border border-slate-200 p-3">
              <div className="flex items-start justify-between gap-2">
                <p className="font-medium text-slate-900">{idea.title}</p>
                <span className="shrink-0 rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                  {idea.format}
                </span>
              </div>
              <p className="mt-1 text-sm text-slate-600">{idea.description}</p>
              <p className="mt-1 text-xs text-slate-500">
                Hook: {idea.hook} · Thời lượng: {idea.estimatedLength}
              </p>
            </div>
          ))}
        </div>
      </Section>

      <Section title="Lịch đăng gợi ý">
        <p className="text-sm text-slate-700">
          {strategy.postingSchedule.frequencyPerWeek} video/tuần · Ngày tốt nhất:{' '}
          {strategy.postingSchedule.bestDays.join(', ')} · Giờ vàng:{' '}
          {strategy.postingSchedule.bestTimeOfDay}
        </p>
      </Section>

      <Section title="Mẫu tiêu đề">
        <ul className="list-inside list-disc space-y-1 text-sm text-slate-700">
          {strategy.titleTemplates.map((t, i) => (
            <li key={i}>{t}</li>
          ))}
        </ul>
      </Section>

      <Section title="Từ khoá SEO">
        <div className="flex flex-wrap gap-2">
          {strategy.seoKeywords.map((k) => (
            <span key={k} className="rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-700">
              {k}
            </span>
          ))}
        </div>
      </Section>

      <Section title="Bộ hashtag">
        <div className="space-y-3">
          {strategy.hashtagSets.map((set) => (
            <div key={set.theme}>
              <p className="text-sm font-medium text-slate-900">{set.theme}</p>
              <p className="mt-1 text-sm text-indigo-600">{set.hashtags.join(' ')}</p>
            </div>
          ))}
        </div>
      </Section>

      <Section title="Chiến thuật tăng trưởng">
        <ul className="list-inside list-disc space-y-1 text-sm text-slate-700">
          {strategy.growthTactics.map((t, i) => (
            <li key={i}>{t}</li>
          ))}
        </ul>
      </Section>

      {strategy.risks.length > 0 && (
        <Section title="Rủi ro cần lưu ý">
          <ul className="list-inside list-disc space-y-1 text-sm text-amber-700">
            {strategy.risks.map((t, i) => (
              <li key={i}>{t}</li>
            ))}
          </ul>
        </Section>
      )}
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5">
      <h2 className="text-lg font-semibold text-slate-900">{title}</h2>
      <div className="mt-3">{children}</div>
    </section>
  )
}
