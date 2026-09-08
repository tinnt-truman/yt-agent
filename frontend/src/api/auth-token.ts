// Shared app password storage + a lightweight pub/sub for "you got logged
// out" (401 from any API call). Kept separate from AuthContext so client.ts
// (plain functions, no React) can read/clear it without importing React.
const STORAGE_KEY = 'yt-agent-password'

export function getStoredPassword(): string | null {
  try {
    return localStorage.getItem(STORAGE_KEY)
  } catch {
    return null
  }
}

export function setStoredPassword(password: string) {
  try {
    localStorage.setItem(STORAGE_KEY, password)
  } catch {
    // Private browsing / storage disabled — session just won't persist across reloads.
  }
}

export function clearStoredPassword() {
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    // ignore
  }
}

type UnauthorizedListener = () => void
let unauthorizedListener: UnauthorizedListener | null = null

export function onUnauthorized(listener: UnauthorizedListener) {
  unauthorizedListener = listener
}

export function notifyUnauthorized() {
  clearStoredPassword()
  unauthorizedListener?.()
}
