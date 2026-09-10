const compactFormatter = new Intl.NumberFormat('vi-VN', { notation: 'compact' })

export function formatCompact(n: number): string {
  return compactFormatter.format(n)
}

export function formatPercent(n: number, digits = 1): string {
  return `${n.toFixed(digits)}%`
}

export function formatDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString('vi-VN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

export function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.round(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

// stripOuterParens removes one layer of enclosing "(...)" — the AI is
// instructed not to include them in an acting direction, but is not always
// reliable, and the UI wraps the value in its own parens when displaying it.
export function stripOuterParens(s: string): string {
  const trimmed = s.trim()
  return trimmed.startsWith('(') && trimmed.endsWith(')') ? trimmed.slice(1, -1).trim() : trimmed
}
