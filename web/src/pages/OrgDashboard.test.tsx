import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { OrgDashboard } from './OrgDashboard'
import * as ApiModule from '../hooks/useApi'

vi.mock('../hooks/useApi', () => ({
  useApi: vi.fn(),
}))

describe('OrgDashboard', () => {
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
        <OrgDashboard />
      </BrowserRouter>,
    )

    expect(screen.getByText(/Carregando dashboard/)).toBeInTheDocument()
  })

  it('renderiza dashboard da organização com dados corretos', async () => {
    const orgData = {
      period_name: 'Período 1',
      total_commission_liability: 500000,
      pending_approvals_count: 5,
      total_reps: 10,
      active_period_status: 'open',
    }

    vi.mocked(ApiModule.useApi)
      .mockReturnValueOnce({
        data: orgData,
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
        <OrgDashboard />
      </BrowserRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('Período 1')).toBeInTheDocument()
      expect(screen.getByText('5')).toBeInTheDocument()
    })
  })

  it('mostra total de liability de comissões', async () => {
    const orgData = {
      period_name: 'Período 1',
      total_commission_liability: 100000, // 1000 BRL
      pending_approvals_count: 0,
      total_reps: 5,
      active_period_status: 'open',
    }

    vi.mocked(ApiModule.useApi)
      .mockReturnValueOnce({
        data: orgData,
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
        <OrgDashboard />
      </BrowserRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText(/R\$/)).toBeInTheDocument()
    })
  })

  it('mostra contagem de aprovações pendentes', async () => {
    const orgData = {
      period_name: 'Período 1',
      total_commission_liability: 500000,
      pending_approvals_count: 3,
      total_reps: 10,
      active_period_status: 'open',
    }

    vi.mocked(ApiModule.useApi)
      .mockReturnValueOnce({
        data: orgData,
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
        <OrgDashboard />
      </BrowserRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('3')).toBeInTheDocument()
      expect(screen.getByText(/Declarações aguardando aprovação/)).toBeInTheDocument()
    })
  })

  it('mostra status do período ativo', async () => {
    const orgData = {
      period_name: 'Período 1',
      total_commission_liability: 500000,
      pending_approvals_count: 0,
      total_reps: 10,
      active_period_status: 'open',
    }

    vi.mocked(ApiModule.useApi)
      .mockReturnValueOnce({
        data: orgData,
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
        <OrgDashboard />
      </BrowserRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('open')).toBeInTheDocument()
    })
  })
})
