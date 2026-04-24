import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { RepDashboard } from './RepDashboard'
import * as AuthModule from '../contexts/AuthContext'
import * as ApiModule from '../hooks/useApi'

vi.mock('../contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}))

vi.mock('../hooks/useApi', () => ({
  useApi: vi.fn(),
}))

describe('RepDashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    vi.mocked(AuthModule.useAuth).mockReturnValue({
      user: {
        id: 1,
        email: 'rep@example.com',
        name: 'John Rep',
        role: 'rep',
      },
      logout: vi.fn() as any,
      login: vi.fn() as any,
      refreshToken: vi.fn() as any,
      accessToken: 'token',
      isLoading: false,
      isAuthenticated: true,
    })
  })

  it('mostra loading enquanto carrega dados', () => {
    vi.mocked(ApiModule.useApi).mockReturnValue({
      data: null,
      isLoading: true,
      error: null,
    })

    render(
      <BrowserRouter>
        <RepDashboard />
      </BrowserRouter>,
    )

    expect(screen.getByText(/Carregando dashboard/)).toBeInTheDocument()
  })

  it('renderiza dashboard com dados do rep', async () => {
    const dashboardData = {
      rep_id: 1,
      period_name: 'Período 1',
      acquisition_goal: 10,
      acquisition_actual: 8,
      renewal_goal: 5,
      renewal_actual: 4,
      attainment_pct: 80,
      commission_earned: 50000,
      commission_pending: 10000,
      recent_events: [],
    }

    vi.mocked(ApiModule.useApi)
      .mockReturnValueOnce({
        data: dashboardData,
        isLoading: false,
        error: null,
      })
      .mockReturnValueOnce({
        data: { last_synced_at: Date.now() },
        isLoading: false,
        error: null,
      })

    render(
      <BrowserRouter>
        <RepDashboard />
      </BrowserRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText(/John Rep/)).toBeInTheDocument()
      expect(screen.getByText('Período 1')).toBeInTheDocument()
    })
  })

  it('mostra comissões formatadas em BRL', async () => {
    const dashboardData = {
      rep_id: 1,
      period_name: 'Período 1',
      acquisition_goal: 10,
      acquisition_actual: 8,
      renewal_goal: 5,
      renewal_actual: 4,
      attainment_pct: 80,
      commission_earned: 50000, // 500 BRL
      commission_pending: 10000,
      recent_events: [],
    }

    vi.mocked(ApiModule.useApi)
      .mockReturnValueOnce({
        data: dashboardData,
        isLoading: false,
        error: null,
      })
      .mockReturnValueOnce({
        data: null,
        isLoading: false,
        error: null,
      })

    render(
      <BrowserRouter>
        <RepDashboard />
      </BrowserRouter>,
    )

    await waitFor(() => {
      // Find amount elements with currency format
      const amounts = screen.getAllByText(/R\$/)
      expect(amounts.length).toBeGreaterThan(0)
    })
  })

  it('renderiza barras de progresso para metas', async () => {
    const dashboardData = {
      rep_id: 1,
      period_name: 'Período 1',
      acquisition_goal: 10,
      acquisition_actual: 5,
      renewal_goal: 10,
      renewal_actual: 10,
      attainment_pct: 75,
      commission_earned: 50000,
      commission_pending: 0,
      recent_events: [],
    }

    vi.mocked(ApiModule.useApi)
      .mockReturnValueOnce({
        data: dashboardData,
        isLoading: false,
        error: null,
      })
      .mockReturnValueOnce({
        data: null,
        isLoading: false,
        error: null,
      })

    const { container } = render(
      <BrowserRouter>
        <RepDashboard />
      </BrowserRouter>,
    )

    await waitFor(() => {
      // Verify progress bars are rendered
      expect(container.querySelectorAll('.progress-bar')).toHaveLength(3)
    })
  })

  it('renderiza lista de eventos recentes', async () => {
    const dashboardData = {
      rep_id: 1,
      period_name: 'Período 1',
      acquisition_goal: 10,
      acquisition_actual: 8,
      renewal_goal: 5,
      renewal_actual: 4,
      attainment_pct: 80,
      commission_earned: 50000,
      commission_pending: 0,
      recent_events: [
        {
          id: 1,
          member_name: 'João Silva',
          event_type: 'acquisition',
          event_date: '2026-04-20',
        },
        {
          id: 2,
          member_name: 'Maria Santos',
          event_type: 'renewal',
          event_date: '2026-04-21',
        },
      ],
    }

    vi.mocked(ApiModule.useApi)
      .mockReturnValueOnce({
        data: dashboardData,
        isLoading: false,
        error: null,
      })
      .mockReturnValueOnce({
        data: null,
        isLoading: false,
        error: null,
      })

    render(
      <BrowserRouter>
        <RepDashboard />
      </BrowserRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('João Silva')).toBeInTheDocument()
      expect(screen.getByText('Maria Santos')).toBeInTheDocument()
    })
  })
})
