import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { Layout } from './Layout'
import * as AuthModule from '../contexts/AuthContext'

vi.mock('../contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}))

describe('Layout', () => {
  let mockLogout: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    mockLogout = vi.fn() as any
  })

  const renderLayout = (role: string = 'rep') => {
    vi.mocked(AuthModule.useAuth).mockReturnValue({
      user: {
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        role: role as any,
      },
      logout: mockLogout as any,
      login: vi.fn() as any,
      refreshToken: vi.fn() as any,
      accessToken: 'token',
      isLoading: false,
      isAuthenticated: true,
    })


    return render(
      <BrowserRouter>
        <Layout>
          <div>Test Content</div>
        </Layout>
      </BrowserRouter>,
    )
  }

  it('renderiza o conteúdo', () => {
    renderLayout()
    expect(screen.getByText('Test Content')).toBeInTheDocument()
  })

  it('mostra nome e role do usuário', () => {
    renderLayout()
    expect(screen.getByText('Test User')).toBeInTheDocument()
    expect(screen.getByText('Representante')).toBeInTheDocument()
  })

  it('mostra item Dashboard para todos', () => {
    renderLayout()
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
  })

  it('mostra item Equipe apenas para managers', () => {
    renderLayout('rep')
    expect(screen.queryByText('Equipe')).not.toBeInTheDocument()

    renderLayout('manager')
    expect(screen.getByText('Equipe')).toBeInTheDocument()
  })

  it('mostra itens de Relatórios e Aprovações para finance', () => {
    renderLayout('finance')
    expect(screen.getByText('Relatórios')).toBeInTheDocument()
    expect(screen.getByText('Aprovações')).toBeInTheDocument()
  })

  it('mostra item Configurações apenas para admin', () => {
    renderLayout('rep')
    expect(screen.queryByText('Configurações')).not.toBeInTheDocument()

    renderLayout('admin')
    expect(screen.getByText('Configurações')).toBeInTheDocument()
  })

  it('logout funciona corretamente', async () => {
    renderLayout()
    const logoutButton = screen.getByRole('button', { name: 'Sair' })
    fireEvent.click(logoutButton)

    await waitFor(() => {
      expect(mockLogout).toHaveBeenCalled()
    })
  })

  it('toggle sidebar em mobile', async () => {
    renderLayout()
    const toggleButton = screen.getByRole('button', { name: 'Toggle sidebar' })

    const sidebar = screen.getByText('Dashboard').closest('.layout-sidebar')
    expect(sidebar).not.toHaveClass('open')

    fireEvent.click(toggleButton)
    await waitFor(() => {
      expect(sidebar).toHaveClass('open')
    })
  })
})
