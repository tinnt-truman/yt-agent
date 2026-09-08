import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import { login as apiLogin } from '../api/client'
import { getStoredPassword, setStoredPassword, onUnauthorized, clearStoredPassword } from '../api/auth-token'

interface AuthContextValue {
  isAuthenticated: boolean
  login: (password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(() => getStoredPassword() !== null)

  const logout = useCallback(() => {
    clearStoredPassword()
    setIsAuthenticated(false)
  }, [])

  useEffect(() => {
    // Any API call that comes back 401 (e.g. the stored password was wrong
    // all along, or the server's APP_PASSWORD changed) drops us back to the
    // login screen instead of leaving the app stuck showing errors.
    onUnauthorized(logout)
  }, [logout])

  const login = useCallback(async (password: string) => {
    await apiLogin(password)
    setStoredPassword(password)
    setIsAuthenticated(true)
  }, [])

  return (
    <AuthContext.Provider value={{ isAuthenticated, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
