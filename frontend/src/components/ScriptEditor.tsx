import type { Script, ScriptCharacter, ScriptDialogueLine, ScriptScene, ScriptShot } from '../types'

// normalizeScriptForEdit upgrades a legacy scene (flat visual/voiceover
// string, from scripts generated before the shots/dialogue array shape
// existed) into that current shape, so the editor only ever has to deal
// with one shape. Saving afterwards always writes the current schema — the
// legacy fields are dropped once a scene passes through the editor once.
export function normalizeScriptForEdit(script: Script): Script {
  return {
    ...script,
    characters: script.characters ?? [],
    scenes: script.scenes.map((scene) => ({
      ...scene,
      characters: scene.characters ?? [],
      shots:
        scene.shots && scene.shots.length > 0
          ? scene.shots
          : scene.visual
            ? [{ shotType: '', description: scene.visual }]
            : [],
      dialogue:
        scene.dialogue && scene.dialogue.length > 0
          ? scene.dialogue
          : scene.voiceover
            ? [{ character: 'Voice-over', line: scene.voiceover }]
            : [],
      visual: undefined,
      voiceover: undefined,
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
const labelClass = 'text-xs font-medium text-slate-500'
const removeButtonClass =
  'shrink-0 rounded-md border border-red-200 px-2 py-1 text-[11px] font-medium text-red-600 transition hover:bg-red-50'
const addButtonClass =
  'rounded-md border border-dashed border-slate-300 px-2.5 py-1.5 text-xs font-medium text-slate-500 transition hover:border-slate-400 hover:text-slate-700'

const emptyCharacter: ScriptCharacter = {
  name: '',
  role: '',
  appearance: '',
  coreTags: [],
  personalInfo: '',
  personality: '',
}

// Fully controlled structured editor for a Script — every field (title,
// hook, cast, per-scene shots/dialogue, CTA) gets its own input, with
// add/remove for the repeating pieces (characters, scenes, shots, dialogue
// lines). Used identically from ScriptsPage's detail panel and ScriptModal.
export function ScriptEditor({
  value,
  onChange,
}: {
  value: Script
  onChange: (next: Script) => void
}) {
  const characters = value.characters ?? []

  function updateCharacter(index: number, patch: Partial<ScriptCharacter>) {
    const next = [...characters]
    next[index] = { ...next[index], ...patch }
    onChange({ ...value, characters: next })
  }
  function addCharacter() {
    onChange({ ...value, characters: [...characters, { ...emptyCharacter }] })
  }
  function removeCharacter(index: number) {
    onChange({ ...value, characters: characters.filter((_, i) => i !== index) })
  }

  function updateScene(index: number, patch: Partial<ScriptScene>) {
    const scenes = [...value.scenes]
    scenes[index] = { ...scenes[index], ...patch }
    onChange({ ...value, scenes })
  }
  function addScene() {
    const scenes: ScriptScene[] = [
      ...value.scenes,
      {
        sceneNumber: value.scenes.length + 1,
        timecode: '',
        setting: '',
        characters: [],
        shots: [],
        dialogue: [],
        cutaway: '',
      },
    ]
    onChange({ ...value, scenes })
  }
  function removeScene(index: number) {
    const scenes = value.scenes.filter((_, i) => i !== index).map((s, i) => ({ ...s, sceneNumber: i + 1 }))
    onChange({ ...value, scenes })
  }

  function updateShot(sceneIndex: number, shotIndex: number, patch: Partial<ScriptShot>) {
    const scenes = [...value.scenes]
    const shots = [...(scenes[sceneIndex].shots ?? [])]
    shots[shotIndex] = { ...shots[shotIndex], ...patch }
    scenes[sceneIndex] = { ...scenes[sceneIndex], shots }
    onChange({ ...value, scenes })
  }
  function addShot(sceneIndex: number) {
    const scenes = [...value.scenes]
    scenes[sceneIndex] = {
      ...scenes[sceneIndex],
      shots: [...(scenes[sceneIndex].shots ?? []), { shotType: '', description: '' }],
    }
    onChange({ ...value, scenes })
  }
  function removeShot(sceneIndex: number, shotIndex: number) {
    const scenes = [...value.scenes]
    scenes[sceneIndex] = {
      ...scenes[sceneIndex],
      shots: (scenes[sceneIndex].shots ?? []).filter((_, i) => i !== shotIndex),
    }
    onChange({ ...value, scenes })
  }

  function updateDialogue(sceneIndex: number, lineIndex: number, patch: Partial<ScriptDialogueLine>) {
    const scenes = [...value.scenes]
    const dialogue = [...(scenes[sceneIndex].dialogue ?? [])]
    dialogue[lineIndex] = { ...dialogue[lineIndex], ...patch }
    scenes[sceneIndex] = { ...scenes[sceneIndex], dialogue }
    onChange({ ...value, scenes })
  }
  function addDialogue(sceneIndex: number) {
    const scenes = [...value.scenes]
    scenes[sceneIndex] = {
      ...scenes[sceneIndex],
      dialogue: [...(scenes[sceneIndex].dialogue ?? []), { character: '', direction: '', line: '' }],
    }
    onChange({ ...value, scenes })
  }
  function removeDialogue(sceneIndex: number, lineIndex: number) {
    const scenes = [...value.scenes]
    scenes[sceneIndex] = {
      ...scenes[sceneIndex],
      dialogue: (scenes[sceneIndex].dialogue ?? []).filter((_, i) => i !== lineIndex),
    }
    onChange({ ...value, scenes })
  }

  return (
    <div className="space-y-4">
      <div>
        <label className={labelClass}>Tiêu đề</label>
        <input
          className={`mt-1 ${inputClass}`}
          value={value.title ?? ''}
          onChange={(e) => onChange({ ...value, title: e.target.value })}
        />
      </div>

      <div>
        <label className={labelClass}>Hook mở đầu</label>
        <textarea
          className={`mt-1 ${textareaClass}`}
          rows={2}
          value={value.hook}
          onChange={(e) => onChange({ ...value, hook: e.target.value })}
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
          {characters.map((char, i) => (
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
                placeholder="Thông tin cá nhân (tuổi, thân phận...)"
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
                className={textareaClass}
                rows={2}
                placeholder="Ngoại hình / trang phục"
                value={char.appearance}
                onChange={(e) => updateCharacter(i, { appearance: e.target.value })}
              />
            </div>
          ))}
        </div>
      </div>

      <div>
        <div className="flex items-center justify-between">
          <label className={labelClass}>Cảnh</label>
          <button type="button" onClick={addScene} className={addButtonClass}>
            + Thêm cảnh
          </button>
        </div>
        <div className="mt-1.5 space-y-3">
          {value.scenes.map((scene, si) => (
            <div key={si} className="space-y-1.5 rounded-md border border-slate-100 p-3">
              <div className="flex items-center justify-between gap-2">
                <p className="text-xs font-semibold text-indigo-700">Cảnh {scene.sceneNumber}</p>
                <button type="button" onClick={() => removeScene(si)} className={removeButtonClass}>
                  Xoá cảnh
                </button>
              </div>
              <div className="flex gap-1.5">
                <input
                  className={`${inputClass} w-32 shrink-0`}
                  placeholder="Timecode"
                  value={scene.timecode}
                  onChange={(e) => updateScene(si, { timecode: e.target.value })}
                />
                <input
                  className={inputClass}
                  placeholder="Nhân vật trong cảnh (cách nhau bởi dấu phẩy)"
                  value={(scene.characters ?? []).join(', ')}
                  onChange={(e) => updateScene(si, { characters: splitList(e.target.value) })}
                />
              </div>
              <input
                className={inputClass}
                placeholder="Bối cảnh (vd: Sáng · Ngoại cảnh · Quán cà phê)"
                value={scene.setting}
                onChange={(e) => updateScene(si, { setting: e.target.value })}
              />

              <div>
                <div className="flex items-center justify-between">
                  <p className="text-[11px] font-medium text-slate-500">Khung hình</p>
                  <button type="button" onClick={() => addShot(si)} className={addButtonClass}>
                    + Khung hình
                  </button>
                </div>
                <div className="mt-1 space-y-1.5">
                  {(scene.shots ?? []).map((shot, shi) => (
                    <div key={shi} className="flex items-start gap-1.5">
                      <input
                        className={`${inputClass} w-28 shrink-0`}
                        placeholder="Loại khung"
                        value={shot.shotType}
                        onChange={(e) => updateShot(si, shi, { shotType: e.target.value })}
                      />
                      <textarea
                        className={`${textareaClass} flex-1`}
                        rows={1}
                        placeholder="Mô tả hình ảnh"
                        value={shot.description}
                        onChange={(e) => updateShot(si, shi, { description: e.target.value })}
                      />
                      <button type="button" onClick={() => removeShot(si, shi)} className={removeButtonClass}>
                        Xoá
                      </button>
                    </div>
                  ))}
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between">
                  <p className="text-[11px] font-medium text-slate-500">Lời thoại</p>
                  <button type="button" onClick={() => addDialogue(si)} className={addButtonClass}>
                    + Lời thoại
                  </button>
                </div>
                <div className="mt-1 space-y-1.5">
                  {(scene.dialogue ?? []).map((line, li) => (
                    <div key={li} className="flex items-start gap-1.5">
                      <input
                        className={`${inputClass} w-24 shrink-0`}
                        placeholder="Nhân vật"
                        value={line.character}
                        onChange={(e) => updateDialogue(si, li, { character: e.target.value })}
                      />
                      <input
                        className={`${inputClass} w-28 shrink-0`}
                        placeholder="Chỉ dẫn diễn xuất"
                        value={line.direction ?? ''}
                        onChange={(e) => updateDialogue(si, li, { direction: e.target.value })}
                      />
                      <textarea
                        className={`${textareaClass} flex-1`}
                        rows={1}
                        placeholder="Câu thoại"
                        value={line.line}
                        onChange={(e) => updateDialogue(si, li, { line: e.target.value })}
                      />
                      <button type="button" onClick={() => removeDialogue(si, li)} className={removeButtonClass}>
                        Xoá
                      </button>
                    </div>
                  ))}
                </div>
              </div>

              <textarea
                className={textareaClass}
                rows={1}
                placeholder="Không gian trống (tuỳ chọn)"
                value={scene.cutaway ?? ''}
                onChange={(e) => updateScene(si, { cutaway: e.target.value })}
              />
            </div>
          ))}
        </div>
      </div>

      <div>
        <label className={labelClass}>Call-to-action kết thúc</label>
        <textarea
          className={`mt-1 ${textareaClass}`}
          rows={2}
          value={value.callToAction}
          onChange={(e) => onChange({ ...value, callToAction: e.target.value })}
        />
      </div>
    </div>
  )
}
