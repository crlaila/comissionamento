import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { ProtectedRoute } from './ProtectedRoute'
import * as AuthModule from '../contexts/AuthContext'

vi.mock('../contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}))

describe('ProtectedRoute', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('mostra loading enquanto verifica autenticação', () => {
    vi.mocked(AuthModule.useAuth).mockReturnValue({
      isLoading: true,
      isAuthenticated: false,
      user: null,
      accessToken: null,
      login: vi.fn(),
      logout: vi.fn(),
      refreshToken: vi.fn(),
    })

    render(
      <BrowserRouter>
        <ProtectedRoute>
          <div>Content</div>
        </ProtectedRoute>
      </BrowserRouter>,
    )

    expect(screen.getByText('Carregando...')).toBeInTheDocument()
  })

  it('redireciona para login quando não autenticado', () => {
    vi.mocked(AuthModule.useAuth).mockReturnValue({
      isLoading: false,
      isAuthenticated: false,
      user: null,
      accessToken: null,
      login: vi.fn(),
      logout: vi.fn(),
      refreshToken: vi.fn(),
    })

    render(
      <BrowserRouter>
        <ProtectedRoute>
          <div>Content</div>
        </ProtectedRoute>
      </BrowserRouter>,
    )

    // When redirected, the content should not be shown
    expect(screen.queryByText('Content')).not.toBeInTheDocument()
  })

  it('renderiza children quando autenticado', () => {
    vi.mocked(AuthModule.useAuth).mockReturnValue({
      isLoading: false,
      isAuthenticated: true,
      user: {
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        role: 'rep',
      },
      accessToken: 'token',
      login: vi.fn(),
      logout: vi.fn(),
      refreshToken: vi.fn(),
    })

    render(
      <BrowserRouter>
        <ProtectedRoute>
          <div>Protected Content</div>
        </ProtectedRoute>
      </BrowserRouter>,
    )

    expect(screen.getByText('Protected Content')).toBeInTheDocument()
  })

  it('redireciona quando role não corresponde', () => {
    vi.mocked(AuthModule.useAuth).mockReturnValue({
      isLoading: false,
      isAuthenticated: true,
      user: {
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        role: 'rep',
      },
      accessToken: 'token',
      login: vi.fn(),
      logout: vi.fn(),
      refreshToken: vi.fn(),
    })

    render(
      <BrowserRouter>
        <ProtectedRoute requiredRole="manager">
          <div>Manager Content</div>
        </ProtectedRoute>
      </BrowserRouter>,
    )

    expect(screen.queryByText('Manager Content')).not.toBeInTheDocument()
  })

  it('renderiza quando role corresponde', () => {
    vi.mocked(AuthModule.useAuth).mockReturnValue({
      isLoading: false,
      isAuthenticated: true,
      user: {
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        role: 'manager',
      },
      accessToken: 'token',
      login: vi.fn(),
      logout: vi.fn(),
      refreshToken: vi.fn(),
    })

    render(
      <BrowserRouter>
        <ProtectedRoute requiredRole="manager">
          <div>Manager Content</div>
        </ProtectedRoute>
      </BrowserRouter>,
    )

    expect(screen.getByText('Manager Content')).toBeInTheDocument()
  })
})
