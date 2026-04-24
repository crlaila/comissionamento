import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { GoalManagement } from './GoalManagement'
import * as AuthModule from '../contexts/AuthContext'
import * as ApiModule from '../hooks/useApi'

vi.mock('../contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}))

vi.mock('../hooks/useApi', () => ({
  useApi: vi.fn(),
}))

const managerUser = {
  id: 1,
  email: 'manager@example.com',
  name: 'Manager',
  role: 'manager' as const,
}

const mockAuth = (overrides: Partial<ReturnType<typeof AuthModule.useAuth>> = {}) => {
  vi.mocked(AuthModule.useAuth).mockReturnValue({
    user: managerUser,
    accessToken: 'tkn-abc',
    isLoading: false,
    isAuthenticated: true,
    login: vi.fn() as any,
    logout: vi.fn() as any,
    refreshToken: vi.fn() as any,
    ...overrides,
  } as any)
}

describe('GoalManagement', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockAuth()
    vi.mocked(ApiModule.useApi).mockReturnValue({
      data: null,
      isLoading: false,
      error: null,
    })
    globalThis.fetch = vi.fn()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('nega acesso para representantes', () => {
    mockAuth({ user: { ...managerUser, role: 'rep' } as any })
    render(<GoalManagement />)
    expect(screen.getByText(/Acesso negado/)).toBeInTheDocument()
  })

  it('permite acesso para manager e admin', () => {
    render(<GoalManagement />)
    expect(screen.getByText(/Gerenciamento de Metas/)).toBeInTheDocument()
  })

  it('valida que campos obrigatórios não podem ficar vazios ao criar', async () => {
    render(<GoalManagement />)

    fireEvent.change(screen.getByLabelText(/Período/), { target: { value: '1' } })
    fireEvent.click(screen.getByRole('button', { name: /Nova Meta/ }))

    const form = screen.getByRole('button', { name: /Criar Meta/ }).closest('form')!
    fireEvent.submit(form)

    await waitFor(() => {
      expect(screen.getByText(/Todos os campos são obrigatórios/)).toBeInTheDocument()
    })
    expect(globalThis.fetch).not.toHaveBeenCalled()
  })

  it('envia POST /api/goals com payload correto e token', async () => {
    ;(globalThis.fetch as any).mockResolvedValue({ ok: true, json: async () => ({}) })
    render(<GoalManagement />)

    fireEvent.change(screen.getByLabelText(/Período/), { target: { value: '7' } })
    fireEvent.click(screen.getByRole('button', { name: /Nova Meta/ }))

    fireEvent.change(screen.getByLabelText(/Representante/), { target: { value: '3' } })
    fireEvent.change(screen.getByLabelText(/Meta de Aquisição/), { target: { value: '10' } })
    fireEvent.change(screen.getByLabelText(/Meta de Renovação/), { target: { value: '20' } })
    fireEvent.change(screen.getByLabelText(/Valor de Comissão/), { target: { value: '50000' } })

    fireEvent.click(screen.getByRole('button', { name: /Criar Meta/ }))

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/goals',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            Authorization: 'Bearer tkn-abc',
          }),
        }),
      )
    })

    const [, init] = (globalThis.fetch as any).mock.calls[0]
    expect(JSON.parse(init.body)).toEqual({
      rep_id: 3,
      period_id: 7,
      acquisition_target: 10,
      renewal_target: 20,
      commission_value: 50000,
    })

    await waitFor(() => {
      expect(screen.getByText(/Meta criada com sucesso/)).toBeInTheDocument()
    })
  })

  it('mostra feedback de erro quando POST /api/goals falha', async () => {
    ;(globalThis.fetch as any).mockResolvedValue({ ok: false, json: async () => ({}) })
    render(<GoalManagement />)

    fireEvent.change(screen.getByLabelText(/Período/), { target: { value: '1' } })
    fireEvent.click(screen.getByRole('button', { name: /Nova Meta/ }))
    fireEvent.change(screen.getByLabelText(/Representante/), { target: { value: '1' } })
    fireEvent.change(screen.getByLabelText(/Meta de Aquisição/), { target: { value: '1' } })
    fireEvent.change(screen.getByLabelText(/Meta de Renovação/), { target: { value: '1' } })
    fireEvent.change(screen.getByLabelText(/Valor de Comissão/), { target: { value: '1' } })
    fireEvent.click(screen.getByRole('button', { name: /Criar Meta/ }))

    await waitFor(() => {
      expect(screen.getByText(/Falha ao criar meta/)).toBeInTheDocument()
    })
  })

  it('renderiza tabela de metas quando a API retorna um array', async () => {
    vi.mocked(ApiModule.useApi).mockReturnValue({
      data: [
        {
          id: 1,
          rep_id: 2,
          rep_name: 'João Silva',
          period_id: 1,
          acquisition_target: 10,
          renewal_target: 5,
          commission_value: 250000,
        },
      ] as any,
      isLoading: false,
      error: null,
    })
    render(<GoalManagement />)
    fireEvent.change(screen.getByLabelText(/Período/), { target: { value: '1' } })

    await waitFor(() => {
      expect(screen.getByText('João Silva')).toBeInTheDocument()
      expect(screen.getByText(/R\$\s*2\.500,00/)).toBeInTheDocument()
    })
  })

  it('pre-preenche formulário quando clica em Editar', async () => {
    vi.mocked(ApiModule.useApi).mockReturnValue({
      data: [
        {
          id: 42,
          rep_id: 9,
          rep_name: 'Maria',
          period_id: 2,
          acquisition_target: 15,
          renewal_target: 30,
          commission_value: 100000,
        },
      ] as any,
      isLoading: false,
      error: null,
    })
    render(<GoalManagement />)
    fireEvent.change(screen.getByLabelText(/Período/), { target: { value: '2' } })

    fireEvent.click(await screen.findByRole('button', { name: /Editar/ }))

    expect((screen.getByLabelText(/Representante/) as HTMLInputElement).value).toBe('9')
    expect((screen.getByLabelText(/Meta de Aquisição/) as HTMLInputElement).value).toBe('15')
    expect((screen.getByLabelText(/Meta de Renovação/) as HTMLInputElement).value).toBe('30')
    expect((screen.getByLabelText(/Valor de Comissão/) as HTMLInputElement).value).toBe('100000')
    expect(screen.getByRole('button', { name: /Salvar Alterações/ })).toBeInTheDocument()
  })

  it('envia PUT /api/goals/{id} quando salva edição', async () => {
    ;(globalThis.fetch as any).mockResolvedValue({ ok: true, json: async () => ({}) })
    vi.mocked(ApiModule.useApi).mockReturnValue({
      data: [
        {
          id: 42,
          rep_id: 9,
          rep_name: 'Maria',
          period_id: 2,
          acquisition_target: 15,
          renewal_target: 30,
          commission_value: 100000,
        },
      ] as any,
      isLoading: false,
      error: null,
    })
    render(<GoalManagement />)
    fireEvent.change(screen.getByLabelText(/Período/), { target: { value: '2' } })

    fireEvent.click(await screen.findByRole('button', { name: /Editar/ }))
    fireEvent.change(screen.getByLabelText(/Meta de Aquisição/), { target: { value: '20' } })
    fireEvent.click(screen.getByRole('button', { name: /Salvar Alterações/ }))

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/goals/42',
        expect.objectContaining({
          method: 'PUT',
          headers: expect.objectContaining({
            Authorization: 'Bearer tkn-abc',
          }),
        }),
      )
    })

    const [, init] = (globalThis.fetch as any).mock.calls[0]
    expect(JSON.parse(init.body)).toEqual({
      acquisition_target: 20,
      renewal_target: 30,
      commission_value: 100000,
    })

    await waitFor(() => {
      expect(screen.getByText(/Meta atualizada com sucesso/)).toBeInTheDocument()
    })
  })
})
