import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { EventDetail } from './EventDetail'
import * as ApiModule from '../hooks/useApi'

const mockNavigate = vi.fn()

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

vi.mock('../hooks/useApi', () => ({
  useApi: vi.fn(),
}))

const renderAt = (eventId: string) =>
  render(
    <MemoryRouter initialEntries={[`/events/${eventId}`]}>
      <Routes>
        <Route path="/events/:eventId" element={<EventDetail />} />
      </Routes>
    </MemoryRouter>,
  )

describe('EventDetail', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('mostra loading enquanto carrega', () => {
    vi.mocked(ApiModule.useApi).mockReturnValue({
      data: null,
      isLoading: true,
      error: null,
    })
    renderAt('10')
    expect(screen.getByText(/Carregando evento/)).toBeInTheDocument()
  })

  it('mostra erro e botão Voltar quando a API falha', () => {
    vi.mocked(ApiModule.useApi).mockReturnValue({
      data: null,
      isLoading: false,
      error: 'boom',
    })
    renderAt('10')
    expect(screen.getByText(/Erro ao carregar evento/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /Voltar/ }))
    expect(mockNavigate).toHaveBeenCalledWith(-1)
  })

  it('renderiza evento de aquisição com label "Aquisição"', () => {
    vi.mocked(ApiModule.useApi).mockReturnValue({
      data: {
        id: 10,
        hinova_id: 'HV-42',
        rep_id: 5,
        rep_name: 'Ana',
        event_type: 'acquisition',
        member_name: 'Cliente X',
        event_date: '2026-04-01T00:00:00Z',
        synced_at: '2026-04-02T00:00:00Z',
        created_at: '2026-04-02T00:00:00Z',
      },
      isLoading: false,
      error: null,
    })
    renderAt('10')
    expect(screen.getByText('Cliente X')).toBeInTheDocument()
    expect(screen.getByText(/Aquisição/)).toBeInTheDocument()
    expect(screen.getByText('HV-42')).toBeInTheDocument()
    expect(screen.getByText('Ana')).toBeInTheDocument()
  })

  it('renderiza evento de renovação com label "Renovação"', () => {
    vi.mocked(ApiModule.useApi).mockReturnValue({
      data: {
        id: 11,
        hinova_id: 'HV-43',
        rep_id: 6,
        rep_name: 'Bruno',
        event_type: 'renewal',
        member_name: 'Cliente Y',
        event_date: '2026-03-10T00:00:00Z',
        synced_at: '2026-03-11T00:00:00Z',
        created_at: '2026-03-11T00:00:00Z',
      },
      isLoading: false,
      error: null,
    })
    renderAt('11')
    expect(screen.getByText(/Renovação/)).toBeInTheDocument()
  })

  it('navega para o dashboard do rep ao clicar no botão', () => {
    vi.mocked(ApiModule.useApi).mockReturnValue({
      data: {
        id: 12,
        hinova_id: 'HV-44',
        rep_id: 7,
        rep_name: 'Carla',
        event_type: 'acquisition',
        member_name: 'Cliente Z',
        event_date: '2026-02-20T00:00:00Z',
        synced_at: '2026-02-21T00:00:00Z',
        created_at: '2026-02-21T00:00:00Z',
      },
      isLoading: false,
      error: null,
    })
    renderAt('12')
    fireEvent.click(screen.getByRole('button', { name: /Ver Dashboard do Rep/ }))
    expect(mockNavigate).toHaveBeenCalledWith('/rep/7')
  })
})
