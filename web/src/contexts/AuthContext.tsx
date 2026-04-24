import React, { createContext, useContext, useState, useEffect } from 'react'
import type { ReactNode } from 'react'

export interface User {
  id: number
  email: string
  name: string
  role: 'rep' | 'manager' | 'finance' | 'admin'
}

export interface AuthContextType {
  user: User | null
  accessToken: string | null
  isLoading: boolean
  isAuthenticated: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  refreshToken: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

const STORAGE_KEY_ACCESS = 'commission_access_token'
const STORAGE_KEY_REFRESH = 'commission_refresh_token'
const TOKEN_REFRESH_INTERVAL = 10 * 60 * 1000 // 10 minutes

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null)
  const [accessToken, setAccessToken] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  // Parse JWT to extract user info and expiry
  const parseToken = (token: string): { user: User; expiresAt: number } | null => {
    try {
      const parts = token.split('.')
      if (parts.length !== 3) return null
      const decoded = JSON.parse(atob(parts[1]))
      return {
        user: {
          id: decoded.sub,
          email: decoded.email,
          name: decoded.name,
          role: decoded.role,
        },
        expiresAt: decoded.exp * 1000, // Convert to milliseconds
      }
    } catch {
      return null
    }
  }

  // Initialize auth from localStorage
  useEffect(() => {
    const storedAccessToken = localStorage.getItem(STORAGE_KEY_ACCESS)
    if (storedAccessToken) {
      const parsed = parseToken(storedAccessToken)
      if (parsed && parsed.expiresAt > Date.now()) {
        setAccessToken(storedAccessToken)
        setUser(parsed.user)
      } else {
        localStorage.removeItem(STORAGE_KEY_ACCESS)
        localStorage.removeItem(STORAGE_KEY_REFRESH)
      }
    }
    setIsLoading(false)
  }, [])

  const login = async (email: string, password: string) => {
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })

    if (!response.ok) {
      throw new Error('Login failed')
    }

    const data = await response.json()
    const parsed = parseToken(data.access_token)
    if (!parsed) throw new Error('Invalid token format')

    setAccessToken(data.access_token)
    setUser(parsed.user)
    localStorage.setItem(STORAGE_KEY_ACCESS, data.access_token)
    localStorage.setItem(STORAGE_KEY_REFRESH, data.refresh_token)
  }

  const logout = async () => {
    const refreshToken = localStorage.getItem(STORAGE_KEY_REFRESH)
    if (refreshToken) {
      try {
        await fetch('/api/auth/logout', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${accessToken}`,
          },
          body: JSON.stringify({ refresh_token: refreshToken }),
        })
      } catch {
        // Ignore logout API errors, still clear local state
      }
    }

    setAccessToken(null)
    setUser(null)
    localStorage.removeItem(STORAGE_KEY_ACCESS)
    localStorage.removeItem(STORAGE_KEY_REFRESH)
  }

  const refreshToken = async () => {
    const storedRefreshToken = localStorage.getItem(STORAGE_KEY_REFRESH)
    if (!storedRefreshToken) {
      await logout()
      return
    }

    try {
      const response = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: storedRefreshToken }),
      })

      if (!response.ok) {
        await logout()
        return
      }

      const data = await response.json()
      const parsed = parseToken(data.access_token)
      if (!parsed) throw new Error('Invalid token format')

      setAccessToken(data.access_token)
      setUser(parsed.user)
      localStorage.setItem(STORAGE_KEY_ACCESS, data.access_token)
    } catch {
      await logout()
    }
  }

  // Setup token refresh interval
  useEffect(() => {
    if (!accessToken) return

    const interval = setInterval(async () => {
      await refreshToken()
    }, TOKEN_REFRESH_INTERVAL)

    return () => clearInterval(interval)
  }, [accessToken])

  const value: AuthContextType = {
    user,
    accessToken,
    isLoading,
    isAuthenticated: !!user,
    login,
    logout,
    refreshToken,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth deve ser usado dentro de um AuthProvider')
  }
  return context
}
