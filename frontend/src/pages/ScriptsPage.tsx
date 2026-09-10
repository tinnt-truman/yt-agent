import { useEffect, useRef, useState } from 'react'
import { deleteScriptJob, listScriptJobs, stepScriptJob } from '../api/client'
import { exportScriptToDocx } from '../lib/exportScriptDocx'
import { formatDate, stripOuterParens } from '../lib/format'
import type { ScriptJob } from '../types'
import { StatusBadge } from '../components/StatusBadge'

const POLL_INTERVAL_MS = 2000

export default function ScriptsPage() {
  const [items, setItems] = useState<ScriptJob[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    listScriptJobs()
      .then(setItems)
      .catch((err) => setError(err instanceof Error ? err.message : 'Không tải được danh sách kịch bản'))
      .finally(() => setLoading(false))
  }, [])

  const selected = items.find((i) => i.id === selectedId) ?? null

  // A job left "pending" (e.g. the tab was closed mid-generation) has no
  // background worker to finish it — advance it on an interval, same as
  // ScriptModal, while it's the one currently open.
  useEffect(() => {
    if (!selected || selected.status !== 'pending') return
    let cancelled = false

    async function poll() {
      try {
        const updated = await stepScriptJob(selected!.id)
        if (cancelled) return
        setItems((prev) => prev.map((i) => (i.id === updated.id ? updated : i)))
        if (updated.status === 'pending') {
          timerRef.current = setTimeout(poll, POLL_INTERVAL_MS)
        }
      } catch {
        // Selected job just stays pending — the user can reopen it later.
      }
    }

    poll()
    return () => {
      cancelled = true
      if (timerRef.current) clearTimeout(timerRef.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected?.id])

  async function handleDelete(item: ScriptJob, e: React.MouseEvent) {
    e.stopPropagation() // don't trigger the row's own select-on-click
    if (!window.confirm(`Xoá kịch bản "${item.ideaTitle}"?`)) return
    setActionError(null)
    setDeletingId(item.id)
    try {
      await deleteScriptJob(item.id)
      setItems((prev) => prev.filter((i) => i.id !== item.id))
      if (selectedId === item.id) setSelectedId(null)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Không xoá được kịch bản')
    } finally {
      setDeletingId(null)
    }
  }

  if (loading) return <p className="text-slate-600">Đang tải...</p>
  if (error) return <p className="text-red-600">{error}</p>

  return (
    <div>
      <h1 className="text-xl font-semibold text-slate-900">Kịch bản</h1>

      {actionError && <p className="mt-3 text-sm text-red-600">{actionError}</p>}

      {items.length === 0 ? (
        <p className="mt-4 text-slate-600">
          Chưa có kịch bản nào được tạo. Vào một ý tưởng nội dung trong kết quả phân tích để tạo kịch bản mới.
        </p>
      ) : (
        <div className="mt-6 grid gap-6 md:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
          <div className="divide-y divide-slate-200 rounded-lg border border-slate-200 bg-white md:max-h-[70vh] md:overflow-y-auto">
            {items.map((item) => (
              <div
                key={item.id}
                onClick={() => setSelectedId(item.id)}
                className={`flex w-full cursor-pointer items-center justify-between gap-3 px-4 py-3 text-left transition hover:bg-slate-50 ${
                  selectedId === item.id ? 'bg-indigo-50' : ''
                }`}
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium text-slate-900">{item.ideaTitle}</p>
                  <p className="mt-0.5 text-xs text-slate-500">
                    {item.durationFormat === 'short' ? 'Short (60 giây)' : 'Video dài (5-10 phút)'} ·{' '}
                    {formatDate(item.createdAt)}
                  </p>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <StatusBadge status={item.status} />
                  <button
                    onClick={(e) => handleDelete(item, e)}
                    disabled={deletingId === item.id}
                    className="rounded-md border border-red-200 bg-white px-2 py-1 text-xs font-medium text-red-600 transition hover:bg-red-50 disabled:opacity-50"
                  >
                    {deletingId === item.id ? 'Đang xoá...' : 'Xoá'}
                  </button>
                </div>
              </div>
            ))}
          </div>

          <div className="rounded-lg border border-slate-200 bg-white p-5">
            {!selected && (
              <p className="text-sm text-slate-500">Chọn một kịch bản bên trái để xem chi tiết.</p>
            )}

            {selected && selected.status === 'pending' && (
              <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-center text-sm text-slate-600">
                <div className="mx-auto mb-2 h-5 w-5 animate-spin rounded-full border-2 border-slate-300 border-t-slate-900" />
                Đang tạo kịch bản với AI...
              </div>
            )}

            {selected && selected.status === 'failed' && (
              <p className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
                {selected.errorMessage || 'Đã có lỗi xảy ra khi tạo kịch bản.'}
              </p>
            )}

            {selected && selected.script && (
              <div className="space-y-3">
                <div>
                  <p className="text-xs font-semibold uppercase tracking-wide text-indigo-700">
                    {selected.ideaTitle}
                  </p>
                  <p className="mt-1 text-xs text-slate-500">
                    {selected.durationFormat === 'short' ? 'Short (60 giây)' : 'Video dài (5-10 phút)'} ·{' '}
                    {formatDate(selected.createdAt)}
                  </p>
                </div>

                <div>
                  <p className="text-xs font-medium text-slate-500">Hook mở đầu</p>
                  <p className="mt-1 rounded-md bg-indigo-50 p-3 text-sm leading-relaxed text-slate-800">
                    {selected.script.hook}
                  </p>
                </div>

                {selected.script.characters && selected.script.characters.length > 0 && (
                  <div>
                    <p className="text-xs font-medium text-slate-500">Nhân vật</p>
                    <div className="mt-1.5 space-y-2">
                      {selected.script.characters.map((char) => (
                        <div key={char.name} className="rounded-md border border-slate-100 p-3">
                          <div className="flex items-center justify-between gap-2">
                            <p className="text-sm font-semibold text-slate-900">{char.name}</p>
                            <span className="shrink-0 rounded-full bg-indigo-50 px-2 py-0.5 text-[10px] font-medium text-indigo-700">
                              {char.role}
                            </span>
                          </div>
                          <p className="mt-1 text-xs text-slate-600">{char.personalInfo}</p>
                          <p className="mt-1 text-xs leading-relaxed text-slate-600">{char.personality}</p>
                          {char.coreTags.length > 0 && (
                            <div className="mt-1.5 flex flex-wrap gap-1">
                              {char.coreTags.map((tag) => (
                                <span
                                  key={tag}
                                  className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] text-slate-600"
                                >
                                  {tag}
                                </span>
                              ))}
                            </div>
                          )}
                          <p className="mt-1.5 text-xs leading-relaxed text-slate-700">{char.appearance}</p>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                <div className="space-y-2">
                  {selected.script.scenes.map((scene) => (
                    <div
                      key={scene.sceneNumber ?? scene.timecode}
                      className="rounded-md border border-slate-100 p-3"
                    >
                      <p className="text-xs font-semibold text-indigo-700">
                        {scene.sceneNumber ? `Cảnh ${scene.sceneNumber} · ` : ''}
                        {scene.timecode}
                      </p>
                      {scene.setting && <p className="mt-1 text-xs text-slate-500">{scene.setting}</p>}
                      {scene.characters && scene.characters.length > 0 && (
                        <p className="mt-1 text-xs text-slate-500">
                          Nhân vật: {scene.characters.join(', ')}
                        </p>
                      )}

                      {scene.shots && scene.shots.length > 0 ? (
                        <div className="mt-1.5 space-y-1">
                          {scene.shots.map((shot, i) => (
                            <p key={i} className="text-sm text-slate-800">
                              <span className="font-medium text-slate-500">{shot.shotType}:</span>{' '}
                              {shot.description}
                            </p>
                          ))}
                        </div>
                      ) : (
                        scene.visual && (
                          <p className="mt-1.5 text-sm text-slate-800">
                            <span className="font-medium text-slate-500">Hình ảnh:</span> {scene.visual}
                          </p>
                        )
                      )}

                      {scene.dialogue && scene.dialogue.length > 0 ? (
                        <div className="mt-1.5 space-y-1">
                          {scene.dialogue.map((d, i) => (
                            <p key={i} className="text-sm text-slate-800">
                              <span className="font-medium text-slate-500">
                                {d.character}
                                {d.direction ? ` (${stripOuterParens(d.direction)})` : ''}:
                              </span>{' '}
                              {d.line}
                            </p>
                          ))}
                        </div>
                      ) : (
                        scene.voiceover && (
                          <p className="mt-1.5 text-sm text-slate-800">
                            <span className="font-medium text-slate-500">Lời thoại:</span> {scene.voiceover}
                          </p>
                        )
                      )}

                      {scene.cutaway && (
                        <p className="mt-1.5 text-xs italic leading-relaxed text-slate-400">
                          Không gian trống: {scene.cutaway}
                        </p>
                      )}
                    </div>
                  ))}
                </div>

                <div>
                  <p className="text-xs font-medium text-slate-500">Call-to-action kết thúc</p>
                  <p className="mt-1 rounded-md bg-slate-50 p-3 text-sm leading-relaxed text-slate-800">
                    {selected.script.callToAction}
                  </p>
                </div>

                <div className="border-t border-slate-100 pt-3">
                  <button
                    onClick={() => exportScriptToDocx(selected.ideaTitle, selected.script!)}
                    className="rounded-md bg-slate-100 px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:bg-slate-200"
                  >
                    Xuất file .docx
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
