import type { Analysis, Settings, UpdateSettingsRequest } from '../types'

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

export async function createAnalysis(url: string): Promise<{ id: string; status: string }> {
  const res = await fetch('/api/analyses', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url }),
  })
  return handle(res)
}

export async function getAnalysis(id: string): Promise<Analysis> {
  const res = await fetch(`/api/analyses/${id}`)
  return handle(res)
}

export async function listAnalyses(): Promise<Analysis[]> {
  const res = await fetch('/api/analyses')
  return handle(res)
}

export async function getSettings(): Promise<Settings> {
  const res = await fetch('/api/settings')
  return handle(res)
}

export async function updateSettings(patch: UpdateSettingsRequest): Promise<Settings> {
  const res = await fetch('/api/settings', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  })
  return handle(res)
}
