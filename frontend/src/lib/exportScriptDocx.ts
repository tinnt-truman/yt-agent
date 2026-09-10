import { Document, HeadingLevel, Packer, Paragraph, TextRun } from 'docx'
import { stripOuterParens } from './format'
import type { Script } from '../types'

export async function exportScriptToDocx(title: string, script: Script): Promise<void> {
  const durationLabel = script.durationFormat === 'short' ? 'Short (60 giây)' : 'Video dài (5-10 phút)'

  const children: Paragraph[] = [
    new Paragraph({ text: title, heading: HeadingLevel.TITLE }),
    new Paragraph({ text: durationLabel, spacing: { after: 200 } }),
    new Paragraph({ text: 'Hook mở đầu', heading: HeadingLevel.HEADING_2 }),
    new Paragraph({ text: script.hook, spacing: { after: 200 } }),
  ]

  if (script.characters && script.characters.length > 0) {
    children.push(new Paragraph({ text: 'Nhân vật', heading: HeadingLevel.HEADING_2 }))
    for (const char of script.characters) {
      children.push(
        new Paragraph({
          children: [
            new TextRun({ text: `${char.name} `, bold: true }),
            new TextRun({ text: `(${char.role})` }),
          ],
        }),
        new Paragraph({ text: char.personalInfo }),
        new Paragraph({ text: char.personality }),
        new Paragraph({ text: char.appearance, spacing: { after: 200 } }),
      )
    }
  }

  for (const scene of script.scenes) {
    const sceneTitle = scene.sceneNumber ? `Cảnh ${scene.sceneNumber} · ${scene.timecode}` : scene.timecode
    children.push(new Paragraph({ text: sceneTitle, heading: HeadingLevel.HEADING_2 }))
    if (scene.setting) children.push(new Paragraph({ text: scene.setting }))
    if (scene.characters && scene.characters.length > 0) {
      children.push(new Paragraph({ text: `Nhân vật: ${scene.characters.join(', ')}` }))
    }

    if (scene.shots && scene.shots.length > 0) {
      for (const shot of scene.shots) {
        children.push(
          new Paragraph({
            children: [
              new TextRun({ text: `${shot.shotType}: `, bold: true }),
              new TextRun(shot.description),
            ],
          }),
        )
      }
    } else if (scene.visual) {
      children.push(
        new Paragraph({
          children: [new TextRun({ text: 'Hình ảnh: ', bold: true }), new TextRun(scene.visual)],
        }),
      )
    }

    if (scene.dialogue && scene.dialogue.length > 0) {
      for (const line of scene.dialogue) {
        const label = line.direction
          ? `${line.character} (${stripOuterParens(line.direction)}): `
          : `${line.character}: `
        children.push(
          new Paragraph({
            children: [new TextRun({ text: label, bold: true }), new TextRun(line.line)],
          }),
        )
      }
    } else if (scene.voiceover) {
      children.push(
        new Paragraph({
          children: [new TextRun({ text: 'Lời thoại: ', bold: true }), new TextRun(scene.voiceover)],
        }),
      )
    }

    if (scene.cutaway) {
      children.push(
        new Paragraph({
          children: [new TextRun({ text: 'Không gian trống: ', italics: true }), new TextRun({ text: scene.cutaway, italics: true })],
          spacing: { after: 200 },
        }),
      )
    } else {
      children.push(new Paragraph({ text: '', spacing: { after: 200 } }))
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
