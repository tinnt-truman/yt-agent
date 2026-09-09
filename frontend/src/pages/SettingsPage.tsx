import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { updateSettings } from '../api/client'
import { useSettingsContext } from '../context/SettingsContext'

type AIProvider = 'deepseek' | 'zen'

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
    provider: 'zen',
    label: 'OpenCode Zen — miễn phí (9/2026, có thể hết hạn)',
    models: [
      { value: 'big-pickle', label: 'Big Pickle (free, ctx ~200K)' },
      { value: 'mimo-v2.5-free', label: 'MiMo-V2.5 Free (free, ctx ~200K)' },
      { value: 'ling-3.0-flash-fin-free', label: 'Ling 3.0 Flash Fin Free (free)' },
      { value: 'nemotron-3-ultra-free', label: 'Nemotron 3 Ultra Free (free, ctx ~1M)' },
      { value: 'nemotron-3.5-lightning-free', label: 'Nemotron 3.5 Lightning Free (free, ctx ~262K)' },
      { value: 'muse-spark-1.2-contributor-free', label: 'Muse Spark 1.2 Contributor Free (free, ctx ~1M)' },
      { value: 'muse-spark-1.3-contributor-free', label: 'Muse Spark 1.3 Contributor Free (free, ctx ~1M)' },
    ],
  },
]

function providerOf(model: string): AIProvider {
  for (const g of MODEL_GROUPS) {
    if (g.models.some((m) => m.value === model)) return g.provider
  }
  return 'deepseek'
}

export default function SettingsPage() {
  const { settings, loading, refresh } = useSettingsContext()
  const [youtubeApiKey, setYoutubeApiKey] = useState('')
  const [deepseekApiKey, setDeepseekApiKey] = useState('')
  const [opencodeApiKey, setOpencodeApiKey] = useState('')
  const [aiProvider, setAiProvider] = useState<AIProvider>('deepseek')
  const [aiModel, setAiModel] = useState('deepseek-v4-pro')
  const [maxVideos, setMaxVideos] = useState(50)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const navigate = useNavigate()

  useEffect(() => {
    if (settings) {
      setAiProvider(settings.aiProvider || providerOf(settings.aiModel || ''))
      setAiModel(settings.aiModel || 'deepseek-v4-pro')
      setMaxVideos(settings.maxVideos || 50)
    }
  }, [settings])

  function handleModelChange(value: string) {
    setAiModel(value)
    setAiProvider(providerOf(value))
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      const updated = await updateSettings({
        youtubeApiKey: youtubeApiKey || undefined,
        deepseekApiKey: deepseekApiKey || undefined,
        opencodeApiKey: opencodeApiKey || undefined,
        aiProvider,
        aiModel,
        maxVideos,
      })
      await refresh()
      setYoutubeApiKey('')
      setDeepseekApiKey('')
      setOpencodeApiKey('')
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
              const first = MODEL_GROUPS.find((g) => g.provider === p)?.models[0]
              if (first && providerOf(aiModel) !== p) setAiModel(first.value)
            }}
            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500"
          >
            <option value="deepseek">DeepSeek (trả phí, ổn định)</option>
            <option value="zen">OpenCode Zen (có model miễn phí)</option>
          </select>
        </Field>

        {aiProvider === 'deepseek' ? (
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
        ) : (
          <Field
            label="OpenCode Zen API Key"
            hint={
              settings?.opencodeApiKeySet
                ? `Đã lưu (${settings.opencodeApiKeyPreview}). Để trống nếu không muốn đổi.`
                : 'Đăng nhập tại opencode.ai/auth rồi copy API key. Model free dùng để feedback nên đừng gửi dữ liệu nhạy cảm.'
            }
          >
            <input
              type="password"
              value={opencodeApiKey}
              onChange={(e) => setOpencodeApiKey(e.target.value)}
              placeholder={settings?.opencodeApiKeySet ? '••••••••' : 'opencode-...'}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500"
            />
          </Field>
        )}

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
          </select>
        </Field>

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
