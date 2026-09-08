import { useEffect, useRef, useState } from 'react'
import { createScriptJob, generateVideoPrompt, listScriptJobs, stepScriptJob } from '../api/client'
import { exportScriptToDocx } from '../lib/exportScriptDocx'
import { formatDate } from '../lib/format'
import type { ContentIdea, Script, ScriptDurationFormat, ScriptJob } from '../types'
import { StatusBadge } from './StatusBadge'

const POLL_INTERVAL_MS = 2000

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
  const [job, setJob] = useState<ScriptJob | null>(null)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [history, setHistory] = useState<ScriptJob[]>([])
  const [historyLoading, setHistoryLoading] = useState(true)
  const [platformId, setPlatformId] = useState(PLATFORMS[0].id)
  const [platformStatus, setPlatformStatus] = useState<'idle' | 'loading' | 'done' | 'error'>('idle')
  const platform = PLATFORMS.find((p) => p.id === platformId) ?? PLATFORMS[0]
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // The generation history — every script ever generated, newest first, so
  // past ones can be reopened instead of regenerated. Loaded once on open;
  // refreshed locally as jobs are created/advanced below.
  useEffect(() => {
    let cancelled = false
    listScriptJobs()
      .then((items) => {
        if (!cancelled) setHistory(items)
      })
      .catch(() => {
        // History is a convenience list — a failed fetch shouldn't block
        // generating a new script, so this fails silently.
      })
      .finally(() => {
        if (!cancelled) setHistoryLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Queued generation: the backend has no background worker (same reason as
  // analyses — see stepAnalysis), so this drives the job forward by calling
  // Step on an interval while it's still pending.
  useEffect(() => {
    if (!job || job.status !== 'pending') return
    let cancelled = false

    async function poll() {
      try {
        const updated = await stepScriptJob(job!.id)
        if (cancelled) return
        setJob(updated)
        setHistory((prev) => prev.map((h) => (h.id === updated.id ? updated : h)))
        if (updated.status === 'pending') {
          timerRef.current = setTimeout(poll, POLL_INTERVAL_MS)
        }
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Không tạo được kịch bản')
      }
    }

    poll()
    return () => {
      cancelled = true
      if (timerRef.current) clearTimeout(timerRef.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [job?.id])

  async function handleGenerate() {
    setCreating(true)
    setError(null)
    try {
      const created = await createScriptJob({
        title: idea.title,
        description: idea.description,
        hook: idea.hook,
        durationFormat,
      })
      setJob(created)
      setHistory((prev) => [created, ...prev])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Không tạo được kịch bản')
    } finally {
      setCreating(false)
    }
  }

  function handleSelectHistory(item: ScriptJob) {
    setError(null)
    setJob(item)
  }

  async function handleOpenPlatform() {
    const script = job?.script
    if (!script || !job) return
    setPlatformStatus('loading')
    try {
      if (platform.mode === 'voice') {
        await navigator.clipboard.writeText(buildVoiceoverScript(script))
      } else {
        const visualSummary = [script.hook, ...script.scenes.map((s) => s.visual)].join('. ')
        const prompt = await generateVideoPrompt({ title: job.ideaTitle, description: visualSummary })
        await navigator.clipboard.writeText(prompt.prompt)
      }
      window.open(platform.url, '_blank', 'noopener,noreferrer')
      setPlatformStatus('done')
      setTimeout(() => setPlatformStatus('idle'), 3000)
    } catch {
      setPlatformStatus('error')
    }
  }

  const script = job?.script

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
            disabled={creating}
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
            disabled={creating}
            className={`flex-1 rounded-md border px-3 py-2 text-sm font-medium transition disabled:opacity-50 ${
              durationFormat === 'short'
                ? 'border-indigo-600 bg-indigo-50 text-indigo-700'
                : 'border-slate-200 text-slate-600 hover:bg-slate-50'
            }`}
          >
            Short (60 giây)
          </button>
        </div>

        <div className="mt-3">
          <label className="text-xs font-medium text-slate-500" htmlFor="script-platform">
            Nền tảng AI tạo video/giọng nói
          </label>
          <select
            id="script-platform"
            value={platformId}
            onChange={(e) => setPlatformId(e.target.value)}
            className="mt-1 w-full rounded-md border border-slate-200 bg-white px-2 py-2 text-sm text-slate-700"
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
        </div>

        <button
          onClick={handleGenerate}
          disabled={creating}
          className="mt-4 w-full rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-indigo-700 disabled:opacity-50"
        >
          {creating ? 'Đang xếp hàng...' : 'Tạo kịch bản mới'}
        </button>

        {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

        {job && job.status === 'pending' && (
          <div className="mt-4 rounded-lg border border-slate-200 bg-slate-50 p-4 text-center text-sm text-slate-600">
            <div className="mx-auto mb-2 h-5 w-5 animate-spin rounded-full border-2 border-slate-300 border-t-slate-900" />
            Đang tạo kịch bản với AI...
          </div>
        )}

        {job && job.status === 'failed' && (
          <p className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
            {job.errorMessage || 'Đã có lỗi xảy ra khi tạo kịch bản.'}
          </p>
        )}

        {job && script && (
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
                onClick={() => exportScriptToDocx(job.ideaTitle, script)}
                className="rounded-md bg-slate-100 px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:bg-slate-200"
              >
                Xuất file .docx
              </button>

              <div className="mt-2">
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

        <div className="mt-5 border-t border-slate-100 pt-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Kịch bản đã tạo
          </p>
          {historyLoading ? (
            <p className="mt-2 text-xs text-slate-400">Đang tải...</p>
          ) : history.length === 0 ? (
            <p className="mt-2 text-xs text-slate-400">Chưa có kịch bản nào được tạo.</p>
          ) : (
            <div className="mt-2 max-h-48 space-y-1.5 overflow-y-auto pr-1">
              {history.map((item) => (
                <button
                  key={item.id}
                  onClick={() => handleSelectHistory(item)}
                  className={`flex w-full items-center justify-between gap-2 rounded-md border px-2.5 py-1.5 text-left text-xs transition ${
                    job?.id === item.id
                      ? 'border-indigo-300 bg-indigo-50'
                      : 'border-slate-100 hover:bg-slate-50'
                  }`}
                >
                  <span className="min-w-0 flex-1 truncate text-slate-700">{item.ideaTitle}</span>
                  <span className="shrink-0 text-slate-400">
                    {item.durationFormat === 'short' ? 'Short' : 'Dài'} · {formatDate(item.createdAt)}
                  </span>
                  <StatusBadge status={item.status} />
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
