import React from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useApi } from '../hooks/useApi'
import { ProgressBar } from '../components/ProgressBar'
import { SyncStatus } from '../components/SyncStatus'
import './Dashboard.css'

interface MemberEvent {
  id: number
  member_name: string
  event_type: 'acquisition' | 'renewal'
  event_date: string
}

interface RepDashboardData {
  rep_id: number
  period_name: string
  acquisition_goal: number
  acquisition_actual: number
  renewal_goal: number
  renewal_actual: number
  attainment_pct: number
  commission_earned: number
  commission_pending: number
  recent_events: MemberEvent[]
}

interface SyncStatusData {
  last_synced_at: number
}

const formatBRL = (centavos: number): string => {
  const reais = centavos / 100
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
  }).format(reais)
}

export const RepDashboard: React.FC = () => {
  const { user } = useAuth()
  const { data: dashboardData, isLoading: dashboardLoading } =
    useApi<RepDashboardData>('/api/dashboard/rep')
  const { data: syncData } =
    useApi<SyncStatusData>('/api/sync/status')

  if (dashboardLoading) {
    return <div className="loading">Carregando dashboard...</div>
  }

  if (!dashboardData) {
    return <div className="error">Erro ao carregar dashboard</div>
  }

  const totalGoal = dashboardData.acquisition_goal + dashboardData.renewal_goal
  const totalActual = dashboardData.acquisition_actual + dashboardData.renewal_actual

  return (
    <div className="dashboard rep-dashboard">
      <header className="dashboard-header">
        <h1>Dashboard de {user?.name}</h1>
        <p className="period-name">{dashboardData.period_name}</p>
      </header>

      <div className="dashboard-grid">
        <section className="dashboard-section commission-section">
          <h2>Comissões</h2>
          <div className="commission-cards">
            <div className="commission-card earned">
              <h3>Comissões Ganhas</h3>
              <p className="amount">{formatBRL(dashboardData.commission_earned)}</p>
            </div>
            <div className="commission-card pending">
              <h3>Pendente</h3>
              <p className="amount">{formatBRL(dashboardData.commission_pending)}</p>
            </div>
          </div>
        </section>

        <section className="dashboard-section goals-section">
          <h2>Metas</h2>
          <div className="goal-progress">
            <ProgressBar
              label="Aquisições"
              current={dashboardData.acquisition_actual}
              target={dashboardData.acquisition_goal}
            />
            <ProgressBar
              label="Renovações"
              current={dashboardData.renewal_actual}
              target={dashboardData.renewal_goal}
            />
            <ProgressBar
              label="Total"
              current={totalActual}
              target={totalGoal}
            />
          </div>
          <div className="attainment">
            <p>Atingimento: <strong>{Math.round(dashboardData.attainment_pct)}%</strong></p>
          </div>
        </section>

        <section className="dashboard-section events-section">
          <h2>Eventos Recentes</h2>
          {dashboardData.recent_events.length > 0 ? (
            <ul className="events-list">
              {dashboardData.recent_events.map((event) => (
                <li key={event.id} className={`event-item ${event.event_type}`}>
                  <span className="event-type">
                    {event.event_type === 'acquisition' ? '✓ Aquisição' : '↻ Renovação'}
                  </span>
                  <span className="event-member">{event.member_name}</span>
                  <span className="event-date">{new Date(event.event_date).toLocaleDateString('pt-BR')}</span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="no-events">Nenhum evento recente</p>
          )}
        </section>
      </div>

      <footer className="dashboard-footer">
        <SyncStatus lastSyncTime={syncData?.last_synced_at} />
      </footer>
    </div>
  )
}
