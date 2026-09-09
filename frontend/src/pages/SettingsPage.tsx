import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { updateSettings } from '../api/client'
import { useSettingsContext } from '../context/SettingsContext'

type AIProvider = 'deepseek' | 'openrouter' | '9router'

const MODEL_GROUPS: { provider: AIProvider; label: string; models: { value: string; label: string }[] }[] = [
  {
    provider: 'deepseek',
    label: 'DeepSeek',
    models: [
      { value: 'deepseek-v4-pro', label: 'DeepSeek V4 Pro (chất lượng cao hơn, khuyến nghị)' },
      { value: 'deepseek-v4-flash', label: 'DeepSeek V4 Flash (nhanh hơn, rẻ hơn)' },
    ],
  },
  {
    provider: 'openrouter',
    label: 'OpenRouter — miễn phí (danh sách xoay vòng)',
    models: [
      { value: 'openrouter/free', label: 'Auto-router (tự chọn model free còn trống)' },
      { value: 'nvidia/nemotron-3-ultra-550b-a55b:free', label: 'Nemotron 3 Ultra (free, ctx ~1M)' },
      { value: 'nvidia/nemotron-3.5-lightning:free', label: 'Nemotron 3.5 Lightning (free, ctx ~262K)' },
      { value: 'inclusionai/ling-3.0-flash-fin:free', label: 'Ling 3.0 Flash Fin (free)' },
      { value: 'minimax/minimax-m3:free', label: 'MiniMax M3 (free)' },
      { value: 'minimax/minimax-m2.7:free', label: 'MiniMax M2.7 (free)' },
      { value: 'z-ai/glm-5.2:free', label: 'GLM 5.2 (free)' },
    ],
  },
]

function providerOf(model: string): AIProvider {
  for (const g of MODEL_GROUPS) {
    if (g.models.some((m) => m.value === model)) return g.provider
  }
  if (model.includes('/') || model.endsWith(':free')) return 'openrouter'
  return 'deepseek'
}

const KNOWN_MODELS = new Set(MODEL_GROUPS.flatMap((g) => g.models.map((m) => m.value)))

export default function SettingsPage() {
  const { settings, loading, refresh } = useSettingsContext()
  const [youtubeApiKey, setYoutubeApiKey] = useState('')
  const [deepseekApiKey, setDeepseekApiKey] = useState('')
  const [openrouterApiKey, setOpenrouterApiKey] = useState('')
  const [ninerouterApiKey, setNinerouterApiKey] = useState('')
  const [aiProvider, setAiProvider] = useState<AIProvider>('deepseek')
  const [aiModel, setAiModel] = useState('deepseek-v4-pro')
  const [customModel, setCustomModel] = useState('')
  const [maxVideos, setMaxVideos] = useState(50)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const navigate = useNavigate()

  useEffect(() => {
    if (settings) {
      setAiProvider(settings.aiProvider || providerOf(settings.aiModel || ''))
      const m = settings.aiModel || 'deepseek-v4-pro'
      setAiModel(KNOWN_MODELS.has(m) ? m : 'custom')
      setCustomModel(KNOWN_MODELS.has(m) ? '' : m)
      setMaxVideos(settings.maxVideos || 50)
    }
  }, [settings])

  function handleModelChange(value: string) {
    setAiModel(value)
    if (value !== 'custom') {
      setAiProvider(providerOf(value))
      setCustomModel('')
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const model = aiModel === 'custom' ? customModel.trim() : aiModel
    if (aiModel === 'custom' && !model) {
      setError('Nhập model ID custom (vd: nvidia/nemotron-3-ultra-550b-a55b:free)')
      return
    }
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      const updated = await updateSettings({
        youtubeApiKey: youtubeApiKey || undefined,
        deepseekApiKey: deepseekApiKey || undefined,
        openrouterApiKey: openrouterApiKey || undefined,
        ninerouterApiKey: ninerouterApiKey || undefined,
        // Use the provider the user actually selected in the dropdown, not
        // a guess from the model ID's shape — a custom 9Router model ID
        // ("cc/claude-opus-4-7") looks just like an OpenRouter one
        // ("author/slug"), so re-deriving it here would silently save the
        // wrong provider.
        aiProvider,
        aiModel: model,
        maxVideos,
      })
      await refresh()
      setYoutubeApiKey('')
      setDeepseekApiKey('')
      setOpenrouterApiKey('')
      setNinerouterApiKey('')
      setSaved(true)
      if (updated.configured) {
        setTimeout(() => navigate('/'), 800)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Không lưu được cấu hình')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <p className="text-slate-600">Đang tải...</p>

  return (
    <div className="mx-auto max-w-xl">
      <h1 className="text-xl font-semibold text-slate-900">Cài đặt</h1>
      <p className="mt-1 text-sm text-slate-600">
        Cấu hình API key và tham số phân tích. Key được lưu trong cơ sở dữ liệu của bạn, không
        chia sẻ đi đâu khác.
      </p>

      {settings && !settings.configured && (
        <p className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
          Bạn cần nhập đủ YouTube API key và API key của AI provider đang chọn trước khi tạo phân tích.
        </p>
      )}

      <form onSubmit={handleSubmit} className="mt-6 space-y-5">
        <Field
          label="YouTube API Key"
          hint={
            settings?.youtubeApiKeySet
              ? `Đã lưu (${settings.youtubeApiKeyPreview}). Để trống nếu không muốn đổi.`
              : 'Tạo tại Google Cloud Console → APIs & Services → Credentials.'
          }
        >
          <input
            type="password"
            value={youtubeApiKey}
            onChange={(e) => setYoutubeApiKey(e.target.value)}
            placeholder={settings?.youtubeApiKeySet ? '••••••••' : 'AIza...'}
            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500"
          />
        </Field>

        <Field label="AI Provider">
          <select
            value={aiProvider}
            onChange={(e) => {
              const p = e.target.value as AIProvider
              setAiProvider(p)
              if (p === '9router') {
                // 9Router has no fixed preset list — models are whatever
                // upstream providers you've connected in its own dashboard —
                // so there's nothing to pick from but the custom ID field.
                setAiModel('custom')
                return
              }
              const first = MODEL_GROUPS.find((g) => g.provider === p)?.models[0]
              if (first && providerOf(aiModel === 'custom' ? customModel : aiModel) !== p) {
                setAiModel(first.value)
                setCustomModel('')
              }
            }}
            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500"
          >
            <option value="deepseek">DeepSeek (trả phí, ổn định)</option>
            <option value="openrouter">OpenRouter (có model miễn phí)</option>
            <option value="9router">9Router (tự host local, gộp nhiều provider)</option>
          </select>
        </Field>

        {aiProvider === 'deepseek' && (
          <Field
            label="DeepSeek API Key"
            hint={
              settings?.deepseekApiKeySet
                ? `Đã lưu (${settings.deepseekApiKeyPreview}). Để trống nếu không muốn đổi.`
                : 'Tạo tại platform.deepseek.com (mục API Keys).'
            }
          >
            <input
              type="password"
              value={deepseekApiKey}
              onChange={(e) => setDeepseekApiKey(e.target.value)}
              placeholder={settings?.deepseekApiKeySet ? '••••••••' : 'sk-...'}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500"
            />
          </Field>
        )}

        {aiProvider === 'openrouter' && (
          <Field
            label="OpenRouter API Key"
            hint={
              settings?.openrouterApiKeySet
                ? `Đã lưu (${settings.openrouterApiKeyPreview}). Để trống nếu không muốn đổi.`
                : 'Tạo tại openrouter.ai/keys (không cần thẻ cho model :free). Giới hạn free ~20 req/phút, ~200 req/ngày.'
            }
          >
            <input
              type="password"
              value={openrouterApiKey}
              onChange={(e) => setOpenrouterApiKey(e.target.value)}
              placeholder={settings?.openrouterApiKeySet ? '••••••••' : 'sk-or-...'}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500"
            />
          </Field>
        )}

        {aiProvider === '9router' && (
          <Field
            label="9Router API Key"
            hint={
              settings?.ninerouterApiKeySet
                ? `Đã lưu (${settings.ninerouterApiKeyPreview}). Để trống nếu không muốn đổi.`
                : 'Cài "npm install -g 9router", chạy "9router", mở dashboard tại localhost:20128 để lấy key và kết nối provider. YT-Agent gọi vào http://localhost:20128/v1 — cần chạy 9Router trên cùng máy với backend này.'
            }
          >
            <input
              type="password"
              value={ninerouterApiKey}
              onChange={(e) => setNinerouterApiKey(e.target.value)}
              placeholder={settings?.ninerouterApiKeySet ? '••••••••' : 'dán key từ dashboard 9Router'}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500"
            />
          </Field>
        )}

        {aiProvider === '9router' ? (
          <Field
            label="Model ID (9Router)"
            hint='Dạng "provider/model" theo dashboard 9Router của bạn, vd: cc/claude-opus-4-7, kr/claude-sonnet-4.5, glm/glm-5.1'
          >
            <input
              type="text"
              value={customModel}
              onChange={(e) => setCustomModel(e.target.value)}
              placeholder="provider/model"
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500"
            />
          </Field>
        ) : (
          <>
            <Field label="Model AI">
              <select
                value={aiModel}
                onChange={(e) => handleModelChange(e.target.value)}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500"
              >
                {MODEL_GROUPS.map((g) => (
                  <optgroup key={g.provider} label={g.label}>
                    {g.models.map((m) => (
                      <option key={m.value} value={m.value}>
                        {m.label}
                      </option>
                    ))}
                  </optgroup>
                ))}
                <option value="custom">Custom model ID (dán từ openrouter.ai/collections/free-models)...</option>
              </select>
            </Field>

            {aiModel === 'custom' && (
              <Field
                label="Model ID custom"
                hint='Dạng "author/slug" hoặc "...:free", vd: nvidia/nemotron-3-ultra-550b-a55b:free'
              >
                <input
                  type="text"
                  value={customModel}
                  onChange={(e) => setCustomModel(e.target.value)}
                  placeholder="author/slug[:free]"
                  className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500"
                />
              </Field>
            )}
          </>
        )}

        <Field label="Số video tối đa lấy mẫu mỗi kênh" hint="Càng nhiều càng chính xác nhưng tốn quota YouTube API hơn.">
          <input
            type="number"
            min={5}
            max={200}
            value={maxVideos}
            onChange={(e) => setMaxVideos(Number(e.target.value))}
            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500"
          />
        </Field>

        {error && <p className="text-sm text-red-600">{error}</p>}
        {saved && <p className="text-sm text-emerald-600">Đã lưu cấu hình.</p>}

        <button
          type="submit"
          disabled={saving}
          className="rounded-lg bg-slate-900 px-6 py-2.5 text-sm font-medium text-white transition hover:bg-slate-700 disabled:opacity-50"
        >
          {saving ? 'Đang lưu...' : 'Lưu cấu hình'}
        </button>
      </form>
    </div>
  )
}

function Field({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <div>
      <label className="block text-sm font-medium text-slate-900">{label}</label>
      {children}
      {hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>}
    </div>
  )
}
