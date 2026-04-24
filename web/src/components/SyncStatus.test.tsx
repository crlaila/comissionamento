import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SyncStatus } from './SyncStatus'

describe('SyncStatus', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('mostra "Sincronizando..." quando não há lastSyncTime', () => {
    render(<SyncStatus />)
    expect(screen.getByText(/Sincronizando/)).toBeInTheDocument()
  })

  it('mostra "just now" para sincronização há menos de 1 minuto', () => {
    const now = Date.now()
    vi.setSystemTime(now)

    render(<SyncStatus lastSyncTime={now} />)
    expect(screen.getByText(/just now/)).toBeInTheDocument()
  })

  it('mostra "1 minuto atrás" para 1 minuto', () => {
    const now = Date.now()
    const oneMinuteAgo = now - 60000
    vi.setSystemTime(now)

    render(<SyncStatus lastSyncTime={oneMinuteAgo} />)
    expect(screen.getByText(/1 minuto atrás/)).toBeInTheDocument()
  })

  it('mostra "X minutos atrás" para múltiplos minutos', () => {
    const now = Date.now()
    const thirtyMinutesAgo = now - 30 * 60000
    vi.setSystemTime(now)

    render(<SyncStatus lastSyncTime={thirtyMinutesAgo} />)
    expect(screen.getByText(/30 minutos atrás/)).toBeInTheDocument()
  })

  it('mostra "1 hora atrás" para 1 hora', () => {
    const now = Date.now()
    const oneHourAgo = now - 60 * 60000
    vi.setSystemTime(now)

    render(<SyncStatus lastSyncTime={oneHourAgo} />)
    expect(screen.getByText(/1 hora atrás/)).toBeInTheDocument()
  })

  it('mostra "X horas atrás" para múltiplas horas', () => {
    const now = Date.now()
    const twoHoursAgo = now - 2 * 60 * 60000
    vi.setSystemTime(now)

    render(<SyncStatus lastSyncTime={twoHoursAgo} />)
    expect(screen.getByText(/2 horas atrás/)).toBeInTheDocument()
  })
})
