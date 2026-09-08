import { useState } from 'react'
import { generateScript, generateVideoPrompt } from '../api/client'
import { exportScriptToDocx } from '../lib/exportScriptDocx'
import type { ContentIdea, Script, ScriptDurationFormat } from '../types'

type PlatformMode = 'video' | 'voice'

interface VideoPlatform {
  id: string
  group: string
  label: string
  url: string
  mode: PlatformMode
}

const PLATFORMS: VideoPlatform[] = [
  { id: 'kling-canvas', group: 'Kling', label: 'Canvas', url: 'https://kling.ai/canvas/', mode: 'video' },
  {
    id: 'kling-generate',
    group: 'Kling',
    label: 'Generate (ảnh → video → giọng nói)',
    url: 'https://kling.ai/app/video/new',
    mode: 'video',
  },
  { id: 'google-flow', group: 'Google', label: 'Google Flow', url: 'https://flow.google.com/', mode: 'video' },
  {
    id: 'elevenlabs-flows',
    group: 'ElevenLabs',
    label: 'Flows',
    url: 'https://elevenlabs.io/app/flows',
    mode: 'voice',
  },
  {
    id: 'elevenlabs-studio',
    group: 'ElevenLabs',
    label: 'Studio',
    url: 'https://elevenlabs.io/app/studio',
    mode: 'voice',
  },
]

// ElevenLabs is a voice platform: it gets the full narration (hook + each
// scene's voiceover line + CTA) rather than a visual text-to-video prompt.
function buildVoiceoverScript(script: Script): string {
  return [script.hook, ...script.scenes.map((s) => s.voiceover).filter(Boolean), script.callToAction].join(
    '\n\n',
  )
}

export function ScriptModal({ idea, onClose }: { idea: ContentIdea; onClose: () => void }) {
  const [durationFormat, setDurationFormat] = useState<ScriptDurationFormat>(
    idea.format.toLowerCase().includes('short') ? 'short' : 'long',
  )
  const [script, setScript] = useState<Script | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [platformId, setPlatformId] = useState(PLATFORMS[0].id)
  const [platformStatus, setPlatformStatus] = useState<'idle' | 'loading' | 'done' | 'error'>('idle')
  const platform = PLATFORMS.find((p) => p.id === platformId) ?? PLATFORMS[0]

  async function handleGenerate() {
    setLoading(true)
    setError(null)
    try {
      const result = await generateScript({
        title: idea.title,
        description: idea.description,
        hook: idea.hook,
        durationFormat,
      })
      setScript(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Không tạo được kịch bản')
    } finally {
      setLoading(false)
    }
  }

  async function handleOpenPlatform() {
    if (!script) return
    setPlatformStatus('loading')
    try {
      if (platform.mode === 'voice') {
        await navigator.clipboard.writeText(buildVoiceoverScript(script))
      } else {
        const visualSummary = [script.hook, ...script.scenes.map((s) => s.visual)].join('. ')
        const prompt = await generateVideoPrompt({ title: idea.title, description: visualSummary })
        await navigator.clipboard.writeText(prompt.prompt)
      }
      window.open(platform.url, '_blank', 'noopener,noreferrer')
      setPlatformStatus('done')
      setTimeout(() => setPlatformStatus('idle'), 3000)
    } catch {
      setPlatformStatus('error')
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="max-h-[85vh] w-full max-w-xl overflow-y-auto rounded-lg bg-white p-5 shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs font-semibold uppercase tracking-wide text-indigo-700">
              Kịch bản video
            </p>
            <p className="mt-1 truncate text-sm text-slate-500">Dựa trên: {idea.title}</p>
          </div>
          <button
            onClick={onClose}
            className="shrink-0 rounded-md p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
            aria-label="Đóng"
          >
            ✕
          </button>
        </div>

        <div className="mt-4 flex gap-2">
          <button
            onClick={() => setDurationFormat('long')}
            disabled={loading}
            className={`flex-1 rounded-md border px-3 py-2 text-sm font-medium transition disabled:opacity-50 ${
              durationFormat === 'long'
                ? 'border-indigo-600 bg-indigo-50 text-indigo-700'
                : 'border-slate-200 text-slate-600 hover:bg-slate-50'
            }`}
          >
            Video dài (5-10 phút)
          </button>
          <button
            onClick={() => setDurationFormat('short')}
            disabled={loading}
            className={`flex-1 rounded-md border px-3 py-2 text-sm font-medium transition disabled:opacity-50 ${
              durationFormat === 'short'
                ? 'border-indigo-600 bg-indigo-50 text-indigo-700'
                : 'border-slate-200 text-slate-600 hover:bg-slate-50'
            }`}
          >
            Short (60 giây)
          </button>
        </div>

        <button
          onClick={handleGenerate}
          disabled={loading}
          className="mt-4 w-full rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-indigo-700 disabled:opacity-50"
        >
          {loading ? 'Đang tạo kịch bản...' : script ? 'Tạo lại kịch bản' : 'Tạo kịch bản'}
        </button>

        {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

        {script && (
          <div className="mt-4 space-y-3">
            <div>
              <p className="text-xs font-medium text-slate-500">Hook mở đầu</p>
              <p className="mt-1 rounded-md bg-indigo-50 p-3 text-sm leading-relaxed text-slate-800">
                {script.hook}
              </p>
            </div>

            <div className="space-y-2">
              {script.scenes.map((scene, i) => (
                <div key={i} className="rounded-md border border-slate-100 p-3">
                  <p className="text-xs font-semibold text-indigo-700">{scene.timecode}</p>
                  <p className="mt-1 text-sm text-slate-800">
                    <span className="font-medium text-slate-500">Hình ảnh:</span> {scene.visual}
                  </p>
                  {scene.voiceover && (
                    <p className="mt-1 text-sm text-slate-800">
                      <span className="font-medium text-slate-500">Lời thoại:</span> {scene.voiceover}
                    </p>
                  )}
                </div>
              ))}
            </div>

            <div>
              <p className="text-xs font-medium text-slate-500">Call-to-action kết thúc</p>
              <p className="mt-1 rounded-md bg-slate-50 p-3 text-sm leading-relaxed text-slate-800">
                {script.callToAction}
              </p>
            </div>

            <div className="border-t border-slate-100 pt-3">
              <button
                onClick={() => exportScriptToDocx(idea.title, script)}
                className="rounded-md bg-slate-100 px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:bg-slate-200"
              >
                Xuất file .docx
              </button>

              <div className="mt-2 flex flex-wrap items-center gap-2">
                <select
                  value={platformId}
                  onChange={(e) => setPlatformId(e.target.value)}
                  disabled={platformStatus === 'loading'}
                  className="rounded-md border border-slate-200 bg-white px-2 py-1.5 text-xs text-slate-700 disabled:opacity-50"
                >
                  {['Kling', 'Google', 'ElevenLabs'].map((group) => (
                    <optgroup key={group} label={group}>
                      {PLATFORMS.filter((p) => p.group === group).map((p) => (
                        <option key={p.id} value={p.id}>
                          {p.label}
                        </option>
                      ))}
                    </optgroup>
                  ))}
                </select>
                <button
                  onClick={handleOpenPlatform}
                  disabled={platformStatus === 'loading'}
                  className="rounded-md bg-violet-100 px-3 py-1.5 text-xs font-medium text-violet-700 transition hover:bg-violet-200 disabled:opacity-50"
                >
                  {platformStatus === 'loading'
                    ? platform.mode === 'voice'
                      ? 'Đang chuẩn bị kịch bản...'
                      : 'Đang tạo prompt...'
                    : platformStatus === 'done'
                      ? 'Đã copy — đã mở!'
                      : `Mở ${platform.group} · ${platform.label}`}
                </button>
              </div>
              {platformStatus === 'error' && (
                <p className="mt-1.5 text-xs text-red-600">Không chuẩn bị được nội dung, thử lại nhé.</p>
              )}
              <p className="mt-1.5 text-[11px] text-slate-400">
                {platform.mode === 'voice'
                  ? 'Đã copy kịch bản lồng tiếng vào clipboard — dán vào nền tảng vừa mở.'
                  : 'Đã copy prompt video vào clipboard — dán vào nền tảng vừa mở.'}
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
