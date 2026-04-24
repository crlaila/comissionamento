import React from 'react'
import { useApi } from '../hooks/useApi'
import { SyncStatus } from '../components/SyncStatus'
import './Dashboard.css'

interface OrgDashboardData {
  period_name: string
  total_commission_liability: number
  pending_approvals_count: number
  total_reps: number
  active_period_status: string
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

export const OrgDashboard: React.FC = () => {
  const { data: dashboardData, isLoading: dashboardLoading } =
    useApi<OrgDashboardData>('/api/dashboard/org')
  const { data: syncData } =
    useApi<SyncStatusData>('/api/sync/status')

  if (dashboardLoading) {
    return <div className="loading">Carregando dashboard da organização...</div>
  }

  if (!dashboardData) {
    return <div className="error">Erro ao carregar dados da organização</div>
  }

  return (
    <div className="dashboard org-dashboard">
      <header className="dashboard-header">
        <h1>Dashboard Financeiro</h1>
        <p className="period-name">{dashboardData.period_name}</p>
      </header>

      <div className="dashboard-grid">
        <section className="dashboard-section liability-section">
          <h2>Liability de Comissões</h2>
          <div className="liability-card">
            <p className="label">Total de Comissões Devidas</p>
            <p className="amount">{formatBRL(dashboardData.total_commission_liability)}</p>
            <p className="subtitle">{dashboardData.total_reps} representantes ativos</p>
          </div>
        </section>

        <section className="dashboard-section approvals-section">
          <h2>Aprovações Pendentes</h2>
          <div className="approval-card">
            <p className="approval-count">{dashboardData.pending_approvals_count}</p>
            <p className="approval-label">Declarações aguardando aprovação</p>
            {dashboardData.pending_approvals_count > 0 && (
              <a href="/approvals" className="approval-link">
                Revisar aprovações →
              </a>
            )}
          </div>
        </section>

        <section className="dashboard-section period-status-section">
          <h2>Status do Período</h2>
          <div className="status-card">
            <p className="label">Período Atual</p>
            <p className={`status ${dashboardData.active_period_status.toLowerCase()}`}>
              {dashboardData.active_period_status}
            </p>
          </div>
        </section>
      </div>

      <footer className="dashboard-footer">
        <SyncStatus lastSyncTime={syncData?.last_synced_at} />
      </footer>
    </div>
  )
}
