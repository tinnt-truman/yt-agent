import type { Analysis, Settings, UpdateSettingsRequest } from '../types'

// Empty by default so relative '/api/...' calls go through Vite's dev proxy
// (see vite.config.ts) or same-origin in production. Set VITE_API_BASE_URL
// at build time when the frontend and backend are deployed as separate
// Vercel projects on different domains.
const API_BASE = import.meta.env.VITE_API_BASE_URL ?? ''

async function handle<T>(res: Response): Promise<T> {
  if (!res.ok) {
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

export async function createAnalysis(url: string): Promise<Analysis> {
  const res = await fetch(`${API_BASE}/api/analyses`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url }),
  })
  return handle(res)
}

export async function getAnalysis(id: string): Promise<Analysis> {
  const res = await fetch(`${API_BASE}/api/analyses/${id}`)
  return handle(res)
}

// stepAnalysis pushes a job forward by exactly one pipeline stage and
// returns its updated state. The backend has no background worker (it must
// also run on serverless hosts), so the frontend drives progress by calling
// this on every poll tick while a job is still in progress.
export async function stepAnalysis(id: string): Promise<Analysis> {
  const res = await fetch(`${API_BASE}/api/analyses/${id}/step`, { method: 'POST' })
  return handle(res)
}

export async function listAnalyses(): Promise<Analysis[]> {
  const res = await fetch(`${API_BASE}/api/analyses`)
  return handle(res)
}

export async function getSettings(): Promise<Settings> {
  const res = await fetch(`${API_BASE}/api/settings`)
  return handle(res)
}

export async function updateSettings(patch: UpdateSettingsRequest): Promise<Settings> {
  const res = await fetch(`${API_BASE}/api/settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  })
  return handle(res)
}
