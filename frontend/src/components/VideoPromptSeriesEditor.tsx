import type { Character, VideoPromptEpisode, VideoPromptScene, VideoPromptSeries } from '../types'

const SHOT_TYPES = ['Toàn cảnh', 'Trung cảnh', 'Cận cảnh', 'Hành động', 'Không gian trống']

// normalizeSeriesForEdit upgrades a legacy episode (one flat `prompt`
// string, from series generated before the per-scene breakdown existed)
// into a single-scene array, so the editor only ever has to deal with the
// current shape. Saving afterwards always writes the current schema.
export function normalizeSeriesForEdit(series: VideoPromptSeries): VideoPromptSeries {
  return {
    ...series,
    episodes: series.episodes.map((ep) => ({
      ...ep,
      scenes:
        ep.scenes && ep.scenes.length > 0
          ? ep.scenes
          : ep.prompt
            ? [{ sceneNumber: 1, setting: '', characters: [], shotType: '', action: '', prompt: ep.prompt }]
            : [],
      prompt: undefined,
    })),
  }
}

function splitList(s: string): string[] {
  return s
    .split(',')
    .map((t) => t.trim())
    .filter(Boolean)
}

const inputClass = 'w-full rounded-md border border-slate-200 px-2 py-1.5 text-sm text-slate-800'
const textareaClass = `${inputClass} leading-relaxed`
const monoTextareaClass = `${textareaClass} font-mono text-[11px]`
const labelClass = 'text-xs font-medium text-slate-500'
const removeButtonClass =
  'shrink-0 rounded-md border border-red-200 px-2 py-1 text-[11px] font-medium text-red-600 transition hover:bg-red-50'
const addButtonClass =
  'rounded-md border border-dashed border-slate-300 px-2.5 py-1.5 text-xs font-medium text-slate-500 transition hover:border-slate-400 hover:text-slate-700'

const emptyCharacter: Character = {
  name: '',
  role: '',
  appearance: '',
  coreTags: [],
  personalInfo: '',
  personality: '',
}

// Fully controlled structured editor for a VideoPromptSeries — story bible
// fields, cast, and every episode/scene, with add/remove for the repeating
// pieces. Used identically from VideoPromptSeriesPage's detail panel and
// VideoPromptModal.
export function VideoPromptSeriesEditor({
  value,
  onChange,
}: {
  value: VideoPromptSeries
  onChange: (next: VideoPromptSeries) => void
}) {
  function updateCharacter(index: number, patch: Partial<Character>) {
    const characters = [...value.characters]
    characters[index] = { ...characters[index], ...patch }
    onChange({ ...value, characters })
  }
  function addCharacter() {
    onChange({ ...value, characters: [...value.characters, { ...emptyCharacter }] })
  }
  function removeCharacter(index: number) {
    onChange({ ...value, characters: value.characters.filter((_, i) => i !== index) })
  }

  function updateEpisode(index: number, patch: Partial<VideoPromptEpisode>) {
    const episodes = [...value.episodes]
    episodes[index] = { ...episodes[index], ...patch }
    onChange({ ...value, episodes })
  }
  function addEpisode() {
    const episodes: VideoPromptEpisode[] = [
      ...value.episodes,
      {
        episodeNumber: value.episodes.length + 1,
        title: '',
        plotSummary: '',
        negativePrompt: '',
        scenes: [],
      },
    ]
    onChange({ ...value, episodes })
  }
  function removeEpisode(index: number) {
    const episodes = value.episodes
      .filter((_, i) => i !== index)
      .map((ep, i) => ({ ...ep, episodeNumber: i + 1 }))
    onChange({ ...value, episodes })
  }

  function updateScene(epIndex: number, sceneIndex: number, patch: Partial<VideoPromptScene>) {
    const episodes = [...value.episodes]
    const scenes = [...(episodes[epIndex].scenes ?? [])]
    scenes[sceneIndex] = { ...scenes[sceneIndex], ...patch }
    episodes[epIndex] = { ...episodes[epIndex], scenes }
    onChange({ ...value, episodes })
  }
  function addScene(epIndex: number) {
    const episodes = [...value.episodes]
    const scenes = episodes[epIndex].scenes ?? []
    episodes[epIndex] = {
      ...episodes[epIndex],
      scenes: [
        ...scenes,
        { sceneNumber: scenes.length + 1, setting: '', characters: [], shotType: '', action: '', prompt: '' },
      ],
    }
    onChange({ ...value, episodes })
  }
  function removeScene(epIndex: number, sceneIndex: number) {
    const episodes = [...value.episodes]
    const scenes = (episodes[epIndex].scenes ?? [])
      .filter((_, i) => i !== sceneIndex)
      .map((s, i) => ({ ...s, sceneNumber: i + 1 }))
    episodes[epIndex] = { ...episodes[epIndex], scenes }
    onChange({ ...value, episodes })
  }

  return (
    <div className="space-y-4">
      <div>
        <label className={labelClass}>Tiêu đề series</label>
        <input
          className={`mt-1 ${inputClass}`}
          value={value.seriesTitle ?? ''}
          onChange={(e) => onChange({ ...value, seriesTitle: e.target.value })}
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className={labelClass}>Thể loại</label>
          <input
            className={`mt-1 ${inputClass}`}
            value={value.genre}
            onChange={(e) => onChange({ ...value, genre: e.target.value })}
          />
        </div>
        <div>
          <label className={labelClass}>Đối tượng</label>
          <input
            className={`mt-1 ${inputClass}`}
            value={value.targetAudience}
            onChange={(e) => onChange({ ...value, targetAudience: e.target.value })}
          />
        </div>
      </div>

      <div>
        <label className={labelClass}>Logline</label>
        <textarea
          className={`mt-1 ${textareaClass}`}
          rows={2}
          value={value.logline}
          onChange={(e) => onChange({ ...value, logline: e.target.value })}
        />
      </div>

      <div>
        <label className={labelClass}>Cốt truyện tổng thể</label>
        <textarea
          className={`mt-1 ${textareaClass}`}
          rows={4}
          value={value.synopsis}
          onChange={(e) => onChange({ ...value, synopsis: e.target.value })}
        />
      </div>

      <div>
        <div className="flex items-center justify-between">
          <label className={labelClass}>Nhân vật</label>
          <button type="button" onClick={addCharacter} className={addButtonClass}>
            + Thêm nhân vật
          </button>
        </div>
        <div className="mt-1.5 space-y-2">
          {value.characters.map((char, i) => (
            <div key={i} className="space-y-1.5 rounded-md border border-slate-100 p-3">
              <div className="flex items-center gap-1.5">
                <input
                  className={inputClass}
                  placeholder="Tên"
                  value={char.name}
                  onChange={(e) => updateCharacter(i, { name: e.target.value })}
                />
                <input
                  className={inputClass}
                  placeholder="Vai trò"
                  value={char.role}
                  onChange={(e) => updateCharacter(i, { role: e.target.value })}
                />
                <button type="button" onClick={() => removeCharacter(i)} className={removeButtonClass}>
                  Xoá
                </button>
              </div>
              <input
                className={inputClass}
                placeholder="Thông tin cá nhân"
                value={char.personalInfo}
                onChange={(e) => updateCharacter(i, { personalInfo: e.target.value })}
              />
              <textarea
                className={textareaClass}
                rows={2}
                placeholder="Tính cách"
                value={char.personality}
                onChange={(e) => updateCharacter(i, { personality: e.target.value })}
              />
              <input
                className={inputClass}
                placeholder="Từ khoá (cách nhau bởi dấu phẩy)"
                value={char.coreTags.join(', ')}
                onChange={(e) => updateCharacter(i, { coreTags: splitList(e.target.value) })}
              />
              <textarea
                className={monoTextareaClass}
                rows={2}
                placeholder="Ngoại hình (tiếng Anh, dán kèm mỗi cảnh)"
                value={char.appearance}
                onChange={(e) => updateCharacter(i, { appearance: e.target.value })}
              />
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className={labelClass}>Phong cách</label>
          <input
            className={`mt-1 ${inputClass}`}
            value={value.style}
            onChange={(e) => onChange({ ...value, style: e.target.value })}
          />
        </div>
        <div>
          <label className={labelClass}>Độ dài mỗi cảnh</label>
          <input
            className={`mt-1 ${inputClass}`}
            value={value.durationHint}
            onChange={(e) => onChange({ ...value, durationHint: e.target.value })}
          />
        </div>
      </div>

      <div>
        <div className="flex items-center justify-between">
          <label className={labelClass}>Tập phim</label>
          <button type="button" onClick={addEpisode} className={addButtonClass}>
            + Thêm tập
          </button>
        </div>
        <div className="mt-1.5 space-y-3">
          {value.episodes.map((ep, ei) => (
            <div key={ei} className="space-y-1.5 rounded-md border border-slate-100 p-3">
              <div className="flex items-center gap-1.5">
                <p className="shrink-0 text-xs font-semibold text-violet-700">Tập {ep.episodeNumber}</p>
                <input
                  className={inputClass}
                  placeholder="Tiêu đề tập"
                  value={ep.title}
                  onChange={(e) => updateEpisode(ei, { title: e.target.value })}
                />
                <button type="button" onClick={() => removeEpisode(ei)} className={removeButtonClass}>
                  Xoá tập
                </button>
              </div>
              <textarea
                className={textareaClass}
                rows={2}
                placeholder="Tóm tắt cốt truyện tập này"
                value={ep.plotSummary}
                onChange={(e) => updateEpisode(ei, { plotSummary: e.target.value })}
              />

              <div className="space-y-2">
                {(ep.scenes ?? []).map((scene, si) => (
                  <div key={si} className="space-y-1.5 rounded border border-slate-100 bg-slate-50/60 p-2">
                    <div className="flex items-center justify-between gap-2">
                      <p className="text-[11px] font-medium text-slate-500">Cảnh {scene.sceneNumber}</p>
                      <button
                        type="button"
                        onClick={() => removeScene(ei, si)}
                        className={removeButtonClass}
                      >
                        Xoá cảnh
                      </button>
                    </div>
                    <div className="flex gap-1.5">
                      <input
                        className={inputClass}
                        placeholder="Bối cảnh"
                        value={scene.setting}
                        onChange={(e) => updateScene(ei, si, { setting: e.target.value })}
                      />
                      <select
                        className={`${inputClass} w-36 shrink-0`}
                        value={scene.shotType}
                        onChange={(e) => updateScene(ei, si, { shotType: e.target.value })}
                      >
                        <option value="">— Loại khung —</option>
                        {SHOT_TYPES.map((t) => (
                          <option key={t} value={t}>
                            {t}
                          </option>
                        ))}
                      </select>
                    </div>
                    <input
                      className={inputClass}
                      placeholder="Nhân vật trong cảnh (cách nhau bởi dấu phẩy)"
                      value={scene.characters.join(', ')}
                      onChange={(e) => updateScene(ei, si, { characters: splitList(e.target.value) })}
                    />
                    <textarea
                      className={textareaClass}
                      rows={1}
                      placeholder="Hành động / thoại chính (tiếng Việt)"
                      value={scene.action}
                      onChange={(e) => updateScene(ei, si, { action: e.target.value })}
                    />
                    <textarea
                      className={monoTextareaClass}
                      rows={2}
                      placeholder="Prompt cho AI video (tiếng Anh)"
                      value={scene.prompt}
                      onChange={(e) => updateScene(ei, si, { prompt: e.target.value })}
                    />
                  </div>
                ))}
                <button type="button" onClick={() => addScene(ei)} className={addButtonClass}>
                  + Thêm cảnh
                </button>
              </div>

              <textarea
                className={monoTextareaClass}
                rows={1}
                placeholder="Negative prompt cả tập (tiếng Anh, tuỳ chọn)"
                value={ep.negativePrompt ?? ''}
                onChange={(e) => updateEpisode(ei, { negativePrompt: e.target.value })}
              />
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
