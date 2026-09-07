import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import { getSettings } from '../api/client'
import type { Settings } from '../types'

interface SettingsContextValue {
  settings: Settings | null
  loading: boolean
  refresh: () => Promise<void>
}

const SettingsContext = createContext<SettingsContextValue | undefined>(undefined)

export function SettingsProvider({ children }: { children: React.ReactNode }) {
  const [settings, setSettings] = useState<Settings | null>(null)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      const s = await getSettings()
      setSettings(s)
    } catch {
      setSettings(null)
    }
  }, [])

  useEffect(() => {
    refresh().finally(() => setLoading(false))
  }, [refresh])

  return (
    <SettingsContext.Provider value={{ settings, loading, refresh }}>
      {children}
    </SettingsContext.Provider>
  )
}

export function useSettingsContext(): SettingsContextValue {
  const ctx = useContext(SettingsContext)
  if (!ctx) throw new Error('useSettingsContext must be used within SettingsProvider')
  return ctx
}
