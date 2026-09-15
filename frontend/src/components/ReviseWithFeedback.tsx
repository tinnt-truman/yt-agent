import { useState } from 'react'

// Shared "leave a note, AI revises in place" control — used on both the
// script and video-prompt-series detail views (ScriptsPage/ScriptModal,
// VideoPromptSeriesPage/VideoPromptModal). Deliberately generic: it only
// knows how to collect feedback text and hand it to the caller, which owns
// the actual revise API call and polling.
export function ReviseWithFeedback({
  onSubmit,
  busy,
  disabled = false,
}: {
  onSubmit: (feedback: string) => void | Promise<void>
  busy: boolean
  disabled?: boolean
}) {
  const [feedback, setFeedback] = useState('')

  async function handleSubmit() {
    const trimmed = feedback.trim()
    if (!trimmed) return
    await onSubmit(trimmed)
    setFeedback('')
  }

  return (
    <div className="space-y-1.5">
      <textarea
        value={feedback}
        onChange={(e) => setFeedback(e.target.value)}
        disabled={busy || disabled}
        placeholder={'Góp ý cho AI, vd: "Cảnh 3 dài quá, rút gọn lại", "đổi giọng nhân vật A cho hài hước hơn"...'}
        rows={2}
        className="w-full rounded-md border border-slate-200 px-2.5 py-1.5 text-xs text-slate-700 disabled:opacity-50"
      />
      <button
        type="button"
        onClick={handleSubmit}
        disabled={busy || disabled || !feedback.trim()}
        className="rounded-md bg-slate-800 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-slate-900 disabled:opacity-50"
      >
        {busy ? 'Đang gửi cho AI...' : 'Gửi góp ý cho AI tạo lại'}
      </button>
    </div>
  )
}
