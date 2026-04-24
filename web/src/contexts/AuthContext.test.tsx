import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { AuthProvider, useAuth } from './AuthContext'

// Mock fetch
const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

// Create a valid JWT token for testing
const createMockToken = (expiresIn: number = 3600) => {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(
    JSON.stringify({
      sub: 1,
      email: 'test@example.com',
      name: 'Test User',
      role: 'rep',
      exp: Math.floor(Date.now() / 1000) + expiresIn,
    }),
  )
  const signature = 'mock_signature'
  return `${header}.${payload}.${signature}`
}

const TestComponent = ({ onAuth }: { onAuth?: (auth: any) => void }) => {
  const auth = useAuth()
  onAuth?.(auth)
  return <div>{auth.isAuthenticated ? 'Authenticated' : 'Not Authenticated'}</div>
}

describe('AuthContext', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    mockFetch.mockClear()
  })

  afterEach(() => {
    localStorage.clear()
  })

  it('inicia com usuário não autenticado', () => {
    let authValue: any
    render(
      <AuthProvider>
        <TestComponent onAuth={(auth) => (authValue = auth)} />
      </AuthProvider>,
    )

    expect(screen.getByText('Not Authenticated')).toBeInTheDocument()
    expect(authValue.user).toBeNull()
    expect(authValue.accessToken).toBeNull()
  })

  it('realiza login com sucesso', async () => {
    const token = createMockToken()
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ access_token: token, refresh_token: 'refresh_token' }),
    })

    let authValue: any
    render(
      <AuthProvider>
        <TestComponent onAuth={(auth) => (authValue = auth)} />
      </AuthProvider>,
    )

    await authValue.login('test@example.com', 'password123')

    await waitFor(() => {
      expect(authValue.isAuthenticated).toBe(true)
      expect(authValue.user?.email).toBe('test@example.com')
      expect(authValue.user?.role).toBe('rep')
    })
  })

  it('armazena tokens no localStorage após login', async () => {
    const token = createMockToken()
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ access_token: token, refresh_token: 'refresh_token' }),
    })

    let authValue: any
    render(
      <AuthProvider>
        <TestComponent onAuth={(auth) => (authValue = auth)} />
      </AuthProvider>,
    )

    await authValue.login('test@example.com', 'password123')

    await waitFor(() => {
      expect(localStorage.getItem('commission_access_token')).toBe(token)
      expect(localStorage.getItem('commission_refresh_token')).toBe('refresh_token')
    })
  })

  it('recupera usuário do localStorage ao inicializar', async () => {
    const token = createMockToken()
    localStorage.setItem('commission_access_token', token)

    let authValue: any
    render(
      <AuthProvider>
        <TestComponent onAuth={(auth) => (authValue = auth)} />
      </AuthProvider>,
    )

    await waitFor(() => {
      expect(authValue.isAuthenticated).toBe(true)
      expect(authValue.user?.email).toBe('test@example.com')
    })
  })

  it('logout limpa tokens e usuário', async () => {
    const token = createMockToken()
    localStorage.setItem('commission_access_token', token)
    localStorage.setItem('commission_refresh_token', 'refresh_token')

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    })

    let authValue: any
    render(
      <AuthProvider>
        <TestComponent onAuth={(auth) => (authValue = auth)} />
      </AuthProvider>,
    )

    await authValue.logout()

    await waitFor(() => {
      expect(authValue.isAuthenticated).toBe(false)
      expect(authValue.user).toBeNull()
      expect(localStorage.getItem('commission_access_token')).toBeNull()
      expect(localStorage.getItem('commission_refresh_token')).toBeNull()
    })
  })

  it('rejeita tokens expirados ao inicializar', async () => {
    const expiredToken = createMockToken(-3600) // Expired 1 hour ago
    localStorage.setItem('commission_access_token', expiredToken)

    let authValue: any
    render(
      <AuthProvider>
        <TestComponent onAuth={(auth) => (authValue = auth)} />
      </AuthProvider>,
    )

    await waitFor(() => {
      expect(authValue.isAuthenticated).toBe(false)
      expect(localStorage.getItem('commission_access_token')).toBeNull()
    })
  })

  it('trata erro de login', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
    })

    let authValue: any
    render(
      <AuthProvider>
        <TestComponent onAuth={(auth) => (authValue = auth)} />
      </AuthProvider>,
    )

    await expect(authValue.login('test@example.com', 'wrong')).rejects.toThrow('Login failed')
  })
})
