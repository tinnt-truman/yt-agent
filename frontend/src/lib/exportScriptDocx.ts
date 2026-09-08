import { Document, HeadingLevel, Packer, Paragraph, TextRun } from 'docx'
import type { Script } from '../types'

export async function exportScriptToDocx(title: string, script: Script): Promise<void> {
  const durationLabel = script.durationFormat === 'short' ? 'Short (60 giây)' : 'Video dài (5-10 phút)'

  const children: Paragraph[] = [
    new Paragraph({ text: title, heading: HeadingLevel.TITLE }),
    new Paragraph({ text: durationLabel, spacing: { after: 200 } }),
    new Paragraph({ text: 'Hook mở đầu', heading: HeadingLevel.HEADING_2 }),
    new Paragraph({ text: script.hook, spacing: { after: 200 } }),
  ]

  for (const scene of script.scenes) {
    children.push(
      new Paragraph({ text: scene.timecode, heading: HeadingLevel.HEADING_2 }),
      new Paragraph({
        children: [new TextRun({ text: 'Hình ảnh: ', bold: true }), new TextRun(scene.visual)],
      }),
    )
    if (scene.voiceover) {
      children.push(
        new Paragraph({
          children: [new TextRun({ text: 'Lời thoại: ', bold: true }), new TextRun(scene.voiceover)],
          spacing: { after: 200 },
        }),
      )
    }
  }

  children.push(
    new Paragraph({ text: 'Call-to-action', heading: HeadingLevel.HEADING_2 }),
    new Paragraph({ text: script.callToAction }),
  )

  const doc = new Document({ sections: [{ children }] })
  const blob = await Packer.toBlob(doc)

  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${sanitizeFilename(title)}.docx`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

function sanitizeFilename(name: string): string {
  return name.replace(/[\\/:*?"<>|]/g, '').trim() || 'kich-ban'
}
