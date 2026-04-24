import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { TeamDashboard } from './TeamDashboard'
import * as ApiModule from '../hooks/useApi'

vi.mock('../hooks/useApi', () => ({
  useApi: vi.fn(),
}))

describe('TeamDashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('mostra loading enquanto carrega dados', () => {
    vi.mocked(ApiModule.useApi).mockReturnValue({
      data: null,
      isLoading: true,
      error: null,
    })

    render(
      <BrowserRouter>
        <TeamDashboard />
      </BrowserRouter>,
    )

    expect(screen.getByText(/Carregando dashboard do time/)).toBeInTheDocument()
  })

  it('renderiza tabela com todos os membros do time', async () => {
    const teamData = {
      manager_id: 1,
      period_name: 'Período 1',
      team_members: [
        {
          rep_id: 1,
          rep_name: 'João',
          attainment_pct: 100,
          commission_earned: 50000,
          commission_pending: 0,
        },
        {
          rep_id: 2,
          rep_name: 'Maria',
          attainment_pct: 75,
          commission_earned: 40000,
          commission_pending: 5000,
        },
      ],
    }

    vi.mocked(ApiModule.useApi)
      .mockReturnValueOnce({
        data: teamData,
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
        <TeamDashboard />
      </BrowserRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('João')).toBeInTheDocument()
      expect(screen.getByText('Maria')).toBeInTheDocument()
    })
  })

  it('renderiza coluna de atingimento com percentagem', async () => {
    const teamData = {
      manager_id: 1,
      period_name: 'Período 1',
      team_members: [
        {
          rep_id: 1,
          rep_name: 'João',
          attainment_pct: 85,
          commission_earned: 50000,
          commission_pending: 0,
        },
      ],
    }

    vi.mocked(ApiModule.useApi)
      .mockReturnValueOnce({
        data: teamData,
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
        <TeamDashboard />
      </BrowserRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('85%')).toBeInTheDocument()
    })
  })

  it('renderiza botão de drill-down para cada rep', async () => {
    const teamData = {
      manager_id: 1,
      period_name: 'Período 1',
      team_members: [
        {
          rep_id: 1,
          rep_name: 'João',
          attainment_pct: 100,
          commission_earned: 50000,
          commission_pending: 0,
        },
      ],
    }

    vi.mocked(ApiModule.useApi)
      .mockReturnValueOnce({
        data: teamData,
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
        <TeamDashboard />
      </BrowserRouter>,
    )

    await waitFor(() => {
      const buttons = screen.getAllByText('Detalhes')
      expect(buttons.length).toBeGreaterThan(0)
    })
  })

  it('mostra comparação quando manager informa período de comparação', async () => {
    const teamData = {
      manager_id: 1,
      period_name: 'Abril 2026',
      team_members: [
        {
          rep_id: 1,
          rep_name: 'João',
          attainment_pct: 80,
          commission_earned: 60000,
          commission_pending: 0,
        },
      ],
    }
    const compareData = {
      period_id: 42,
      period_name: 'Março 2026',
      total_commission: 40000,
      average_attainment: 60,
      reps_count: 1,
      rep_summaries: [],
    }

    vi.mocked(ApiModule.useApi).mockImplementation((url: string) => {
      if (url === '/api/dashboard/team') {
        return { data: teamData, isLoading: false, error: null } as any
      }
      if (url === '/api/sync/status') {
        return { data: null, isLoading: false, error: null } as any
      }
      if (url.startsWith('/api/reports/team-summary?period_id=42')) {
        return { data: compareData, isLoading: false, error: null } as any
      }
      return { data: null, isLoading: false, error: null } as any
    })

    render(
      <BrowserRouter>
        <TeamDashboard />
      </BrowserRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText(/Comparar com período/)).toBeInTheDocument()
    })

    fireEvent.change(screen.getByLabelText(/Período de comparação/), {
      target: { value: '42' },
    })

    await waitFor(() => {
      expect(screen.getByText('Março 2026')).toBeInTheDocument()
      expect(screen.getByText(/R\$\s*400,00/)).toBeInTheDocument()
      expect(screen.getByText(/60%/)).toBeInTheDocument()
    })
  })
})
