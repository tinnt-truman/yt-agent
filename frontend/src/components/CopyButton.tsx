import { useState } from 'react'

function CopyIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="9" y="9" width="13" height="13" rx="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </svg>
  )
}

// Reusable copy-to-clipboard button. Renders its own compact pill by
// default (className is appended on top of that so call sites can still
// tweak color/spacing), or pass `bare` to render just the icon+label with no
// button chrome, for embedding inside an already-styled container (a pill,
// a card header, ...).
export function CopyButton({
  text,
  label = 'Copy',
  bare = false,
  className = '',
}: {
  text: string
  label?: string
  bare?: boolean
  className?: string
}) {
  const [copied, setCopied] = useState(false)

  async function handleCopy(e: React.MouseEvent) {
    e.preventDefault()
    e.stopPropagation()
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard API can be denied/unavailable in some browser contexts —
      // the text is still visible on screen either way.
    }
  }

  const base = bare
    ? 'inline-flex items-center gap-1'
    : 'inline-flex items-center gap-1 rounded-md bg-slate-100 px-2 py-1 text-slate-600 hover:bg-slate-200 hover:text-slate-900'

  return (
    <button
      type="button"
      onClick={handleCopy}
      className={`${base} text-[11px] font-medium transition ${className}`}
    >
      <CopyIcon />
      {copied ? 'Đã copy!' : label}
    </button>
  )
}
