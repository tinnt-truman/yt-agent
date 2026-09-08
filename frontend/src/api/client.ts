import type {
  Analysis,
  ChannelAnalytics,
  ConnectedChannel,
  ScriptJob,
  ScriptRequest,
  Settings,
  TrendingInsight,
  TrendingReport,
  UpdateSettingsRequest,
  VideoCategory,
  VideoPrompt,
  VideoPromptRequest,
  VideoPromptSeries,
  VideoPromptSeriesRequest,
} from '../types'
import { getStoredPassword, notifyUnauthorized } from './auth-token'

// Empty by default so relative '/api/...' calls go through Vite's dev proxy
// (see vite.config.ts) or same-origin in production. Set VITE_API_BASE_URL
// at build time when the frontend and backend are deployed as separate
// Vercel projects on different domains.
const API_BASE = import.meta.env.VITE_API_BASE_URL ?? ''

function authHeaders(): HeadersInit {
  const password = getStoredPassword()
  return password ? { Authorization: `Bearer ${password}` } : {}
}

async function handle<T>(res: Response): Promise<T> {
  if (!res.ok) {
    if (res.status === 401) {
      notifyUnauthorized()
    }
    let message = `Request failed (${res.status})`
    try {
      const body = await res.json()
      if (body?.error) message = body.error
    } catch {
      // ignore parse failure, use default message
    }
    throw new Error(message)
  }
  return res.json() as Promise<T>
}

export async function login(password: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  })
  if (!res.ok) {
    let message = 'Sai mật khẩu'
    try {
      const body = await res.json()
      if (body?.error) message = body.error
    } catch {
      // ignore parse failure, use default message
    }
    throw new Error(message)
  }
}

export async function createAnalysis(url: string): Promise<Analysis> {
  const res = await fetch(`${API_BASE}/api/analyses`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify({ url }),
  })
  return handle(res)
}

export async function getAnalysis(id: string): Promise<Analysis> {
  const res = await fetch(`${API_BASE}/api/analyses/${id}`, { headers: authHeaders() })
  return handle(res)
}

// stepAnalysis pushes a job forward by exactly one pipeline stage and
// returns its updated state. The backend has no background worker (it must
// also run on serverless hosts), so the frontend drives progress by calling
// this on every poll tick while a job is still in progress.
export async function stepAnalysis(id: string): Promise<Analysis> {
  const res = await fetch(`${API_BASE}/api/analyses/${id}/step`, {
    method: 'POST',
    headers: authHeaders(),
  })
  return handle(res)
}

export async function listAnalyses(): Promise<Analysis[]> {
  const res = await fetch(`${API_BASE}/api/analyses`, { headers: authHeaders() })
  return handle(res)
}

export async function deleteAnalysis(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/analyses/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  await handle(res)
}

export async function getSettings(): Promise<Settings> {
  const res = await fetch(`${API_BASE}/api/settings`, { headers: authHeaders() })
  return handle(res)
}

export async function updateSettings(patch: UpdateSettingsRequest): Promise<Settings> {
  const res = await fetch(`${API_BASE}/api/settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(patch),
  })
  return handle(res)
}

export async function getTrending(
  region: string,
  category?: string,
  max = 25,
): Promise<TrendingReport> {
  const params = new URLSearchParams({ region, max: String(max) })
  if (category) params.set('category', category)
  const res = await fetch(`${API_BASE}/api/trending?${params}`, { headers: authHeaders() })
  return handle(res)
}

export async function getVideoCategories(region: string): Promise<VideoCategory[]> {
  const params = new URLSearchParams({ region })
  const res = await fetch(`${API_BASE}/api/trending/categories?${params}`, {
    headers: authHeaders(),
  })
  return handle(res)
}

export async function getTrendingInsight(report: TrendingReport): Promise<TrendingInsight> {
  const res = await fetch(`${API_BASE}/api/trending/insight`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(report),
  })
  return handle(res)
}

// getGoogleAuthUrl asks the backend for a Google consent URL (signed,
// short-lived state). The caller must navigate the whole browser tab there
// (window.location.href = url) — a fetch() can't complete an OAuth consent
// redirect, and a plain <a href> would skip our Authorization header, so
// this two-step (authenticated fetch for the URL, then a raw navigation) is
// the flow.
export async function getGoogleAuthUrl(): Promise<string> {
  const res = await fetch(`${API_BASE}/api/oauth/google/url`, { headers: authHeaders() })
  const data = await handle<{ url: string }>(res)
  return data.url
}

export async function listConnectedChannels(): Promise<ConnectedChannel[]> {
  const res = await fetch(`${API_BASE}/api/channels`, { headers: authHeaders() })
  return handle(res)
}

export async function disconnectChannel(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/channels/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  await handle(res)
}

export async function getChannelAnalytics(id: string): Promise<ChannelAnalytics> {
  const res = await fetch(`${API_BASE}/api/channels/${id}/analytics`, { headers: authHeaders() })
  return handle(res)
}

export async function generateVideoPrompt(req: VideoPromptRequest): Promise<VideoPrompt> {
  const res = await fetch(`${API_BASE}/api/videos/prompt`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(req),
  })
  return handle(res)
}

// generateVideoPromptSeries writes an original multi-episode story with a
// text-to-video prompt per episode — for a longer serialized story told
// across several separately-generated videos, unlike generateVideoPrompt's
// single short clip (still used internally by the script modal's "open
// Kling/Google Flow" action, which only needs one clip-length prompt).
export async function generateVideoPromptSeries(
  req: VideoPromptSeriesRequest,
): Promise<VideoPromptSeries> {
  const res = await fetch(`${API_BASE}/api/videos/prompt-series`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(req),
  })
  return handle(res)
}

// createScriptJob queues a script-generation job and returns it immediately
// at "pending" — like createAnalysis, the caller drives it forward via
// stepScriptJob on a poll loop (no backend background worker; see
// stepAnalysis for why).
export async function createScriptJob(req: ScriptRequest): Promise<ScriptJob> {
  const res = await fetch(`${API_BASE}/api/scripts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(req),
  })
  return handle(res)
}

export async function stepScriptJob(id: string): Promise<ScriptJob> {
  const res = await fetch(`${API_BASE}/api/scripts/${id}/step`, {
    method: 'POST',
    headers: authHeaders(),
  })
  return handle(res)
}

export async function listScriptJobs(): Promise<ScriptJob[]> {
  const res = await fetch(`${API_BASE}/api/scripts`, { headers: authHeaders() })
  return handle(res)
}

export async function deleteScriptJob(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/scripts/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  await handle(res)
}
