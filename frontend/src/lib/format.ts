import type { ScriptScene } from '../types'

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

// buildSceneAIContent flattens one scene (setting, cast, shots, dialogue,
// cutaway) into a plain-text block meant to be pasted straight into an AI
// image/video generator's prompt field (Midjourney, Kling, Sora, ...) —
// everything the AI needs to render that scene, in one copy.
export function buildSceneAIContent(scene: ScriptScene): string {
  const lines: string[] = []
  lines.push(scene.sceneNumber ? `Cảnh ${scene.sceneNumber} · ${scene.timecode}` : scene.timecode)
  if (scene.setting) lines.push(scene.setting)
  if (scene.characters && scene.characters.length > 0) {
    lines.push(`Nhân vật: ${scene.characters.join(', ')}`)
  }

  if (scene.shots && scene.shots.length > 0) {
    for (const shot of scene.shots) {
      lines.push(`${shot.shotType}: ${shot.description}`)
    }
  } else if (scene.visual) {
    lines.push(`Hình ảnh: ${scene.visual}`)
  }

  if (scene.dialogue && scene.dialogue.length > 0) {
    for (const d of scene.dialogue) {
      const label = d.direction ? `${d.character} (${stripOuterParens(d.direction)})` : d.character
      lines.push(`${label}: ${d.line}`)
    }
  } else if (scene.voiceover) {
    lines.push(`Lời thoại: ${scene.voiceover}`)
  }

  if (scene.cutaway) lines.push(`Không gian trống: ${scene.cutaway}`)

  return lines.join('\n')
}
