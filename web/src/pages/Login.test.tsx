import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BrowserRouter } from 'react-router-dom'
import { Login } from './Login'
import * as AuthModule from '../contexts/AuthContext'

vi.mock('../contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}))

describe('Login Page', () => {
  let mockLogin: ReturnType<typeof vi.fn>

  beforeEach(() => {
    mockLogin = vi.fn() as any
    vi.mocked(AuthModule.useAuth).mockReturnValue({
      login: mockLogin as any,
      logout: vi.fn() as any,
      refreshToken: vi.fn() as any,
      user: null,
      accessToken: null,
      isLoading: false,
      isAuthenticated: false,
    })
  })

  const renderLogin = () => {
    return render(
      <BrowserRouter>
        <Login />
      </BrowserRouter>,
    )
  }

  it('renderiza o formulário de login', () => {
    renderLogin()
    expect(screen.getByText('Comissionamento')).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
    expect(screen.getByLabelText('Senha')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Entrar/i })).toBeInTheDocument()
  })

  it('valida campo email obrigatório', async () => {
    const user = userEvent.setup()
    renderLogin()

    const submitButton = screen.getByRole('button', { name: /Entrar/i })
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByText('Email é obrigatório')).toBeInTheDocument()
    })
  })

  it('valida campo senha obrigatório', async () => {
    const user = userEvent.setup()
    renderLogin()

    const emailInput = screen.getByLabelText('Email')
    await user.type(emailInput, 'test@example.com')

    const submitButton = screen.getByRole('button', { name: /Entrar/i })
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByText('Senha é obrigatória')).toBeInTheDocument()
    })
  })

  it('mostra mensagem de erro quando login falha', async () => {
    const user = userEvent.setup()
    mockLogin.mockRejectedValueOnce(new Error('Login failed'))

    renderLogin()

    const emailInput = screen.getByLabelText('Email')
    const passwordInput = screen.getByLabelText('Senha')
    const submitButton = screen.getByRole('button', { name: /Entrar/i })

    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'password123')
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByText('Login failed')).toBeInTheDocument()
    })
  })

  it('desabilita inputs durante o login', async () => {
    const user = userEvent.setup()
    mockLogin.mockImplementationOnce(
      () => new Promise(() => {}), // Never resolves
    )

    renderLogin()

    const emailInput = screen.getByLabelText('Email')
    const passwordInput = screen.getByLabelText('Senha')
    const submitButton = screen.getByRole('button', { name: /Entrar/i })

    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'password123')
    await user.click(submitButton)

    expect(emailInput).toBeDisabled()
    expect(passwordInput).toBeDisabled()
    expect(submitButton).toBeDisabled()
  })
})
