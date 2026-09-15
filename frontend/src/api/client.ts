import type {
  Analysis,
  ChannelAnalytics,
  ConnectedChannel,
  MusicPromptJob,
  MusicPromptRequest,
  Script,
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
  VideoPromptSeriesJob,
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

// createVideoPromptSeriesJob queues a multi-episode-story job and returns it
// immediately at "pending" — like createScriptJob, the caller drives it
// forward via stepVideoPromptSeriesJob on a poll loop (no backend background
// worker; see stepAnalysis for why). Unlike generateVideoPrompt's single
// short clip prompt (still used internally by the script modal's "open
// Kling/Google Flow" action), this is a longer serialized story told across
// several separately-generated videos.
export async function createVideoPromptSeriesJob(
  req: VideoPromptSeriesRequest,
): Promise<VideoPromptSeriesJob> {
  const res = await fetch(`${API_BASE}/api/videos/prompt-series`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(req),
  })
  return handle(res)
}

export async function stepVideoPromptSeriesJob(id: string): Promise<VideoPromptSeriesJob> {
  const res = await fetch(`${API_BASE}/api/videos/prompt-series/${id}/step`, {
    method: 'POST',
    headers: authHeaders(),
  })
  return handle(res)
}

export async function listVideoPromptSeriesJobs(): Promise<VideoPromptSeriesJob[]> {
  const res = await fetch(`${API_BASE}/api/videos/prompt-series`, { headers: authHeaders() })
  return handle(res)
}

export async function deleteVideoPromptSeriesJob(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/videos/prompt-series/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  await handle(res)
}

// retryVideoPromptSeriesJob puts a job back at "pending" in place (same id,
// error cleared) rather than creating a new one — works for a "done" job
// (re-roll the result) as well as a "failed" one. The caller then resumes
// the normal stepVideoPromptSeriesJob poll loop to (re)run generation.
export async function retryVideoPromptSeriesJob(id: string): Promise<VideoPromptSeriesJob> {
  const res = await fetch(`${API_BASE}/api/videos/prompt-series/${id}/retry`, {
    method: 'POST',
    headers: authHeaders(),
  })
  return handle(res)
}

// reviseVideoPromptSeriesJob queues a human-written note against an
// already-generated series — puts the job back at "pending" (like retry),
// but the next stepVideoPromptSeriesJob call asks the AI to edit the
// existing series per the note instead of generating a fresh one.
export async function reviseVideoPromptSeriesJob(
  id: string,
  feedback: string,
): Promise<VideoPromptSeriesJob> {
  const res = await fetch(`${API_BASE}/api/videos/prompt-series/${id}/revise`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify({ feedback }),
  })
  return handle(res)
}

// updateVideoPromptSeriesJob saves a series a human edited directly in the
// UI — synchronous, no AI call involved.
export async function updateVideoPromptSeriesJob(
  id: string,
  series: VideoPromptSeries,
): Promise<VideoPromptSeriesJob> {
  const res = await fetch(`${API_BASE}/api/videos/prompt-series/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(series),
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

// retryScriptJob puts a script job back at "pending" in place (same id,
// error cleared) rather than creating a new one — works for a "done" job
// (re-roll the result) as well as a "failed" one. The caller then resumes
// the normal stepScriptJob poll loop to (re)run generation.
export async function retryScriptJob(id: string): Promise<ScriptJob> {
  const res = await fetch(`${API_BASE}/api/scripts/${id}/retry`, {
    method: 'POST',
    headers: authHeaders(),
  })
  return handle(res)
}

// reviseScriptJob queues a human-written note against an already-generated
// script — puts the job back at "pending" (like retry), but the next
// stepScriptJob call asks the AI to edit the existing script per the note
// instead of generating a fresh one.
export async function reviseScriptJob(id: string, feedback: string): Promise<ScriptJob> {
  const res = await fetch(`${API_BASE}/api/scripts/${id}/revise`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify({ feedback }),
  })
  return handle(res)
}

// updateScriptJob saves a script a human edited directly in the UI —
// synchronous, no AI call involved.
export async function updateScriptJob(id: string, script: Script): Promise<ScriptJob> {
  const res = await fetch(`${API_BASE}/api/scripts/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(script),
  })
  return handle(res)
}

export async function deleteScriptJob(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/scripts/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  await handle(res)
}

// createMusicPromptJob queues a Google Flow Music prompt job and returns it
// immediately at "pending" — like createScriptJob, the caller drives it
// forward via stepMusicPromptJob on a poll loop (no backend background
// worker; see stepAnalysis for why).
export async function createMusicPromptJob(req: MusicPromptRequest): Promise<MusicPromptJob> {
  const res = await fetch(`${API_BASE}/api/music-prompts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(req),
  })
  return handle(res)
}

export async function stepMusicPromptJob(id: string): Promise<MusicPromptJob> {
  const res = await fetch(`${API_BASE}/api/music-prompts/${id}/step`, {
    method: 'POST',
    headers: authHeaders(),
  })
  return handle(res)
}

export async function listMusicPromptJobs(): Promise<MusicPromptJob[]> {
  const res = await fetch(`${API_BASE}/api/music-prompts`, { headers: authHeaders() })
  return handle(res)
}

// retryMusicPromptJob puts a failed job back at "pending" in place (same id,
// error cleared) rather than creating a new one. The caller then resumes the
// normal stepMusicPromptJob poll loop to retry generation.
export async function retryMusicPromptJob(id: string): Promise<MusicPromptJob> {
  const res = await fetch(`${API_BASE}/api/music-prompts/${id}/retry`, {
    method: 'POST',
    headers: authHeaders(),
  })
  return handle(res)
}

export async function deleteMusicPromptJob(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/music-prompts/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  await handle(res)
}
