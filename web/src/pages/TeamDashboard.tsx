import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useApi } from '../hooks/useApi'
import { SyncStatus } from '../components/SyncStatus'
import './Dashboard.css'

interface TeamMember {
  rep_id: number
  rep_name: string
  attainment_pct: number
  commission_earned: number
  commission_pending: number
}

interface TeamDashboardData {
  manager_id: number
  period_name: string
  team_members: TeamMember[]
}

interface SyncStatusData {
  last_synced_at: number
}

interface TeamSummaryReport {
  period_id: number
  period_name: string
  total_commission: number
  average_attainment: number
  reps_count: number
}

const formatBRL = (centavos: number): string => {
  const reais = centavos / 100
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
  }).format(reais)
}

const getAttainmentClass = (percentage: number): string => {
  if (percentage >= 100) return 'success'
  if (percentage >= 70) return 'warning'
  return 'danger'
}

const formatDelta = (current: number, previous: number): string => {
  const diff = current - previous
  const sign = diff > 0 ? '+' : diff < 0 ? '' : '±'
  return `${sign}${formatBRL(diff)}`
}

const formatPctDelta = (current: number, previous: number): string => {
  const diff = current - previous
  const sign = diff > 0 ? '+' : diff < 0 ? '' : '±'
  return `${sign}${Math.round(diff)}%`
}

export const TeamDashboard: React.FC = () => {
  const navigate = useNavigate()
  const [comparePeriodId, setComparePeriodId] = useState<string>('')

  const { data: dashboardData, isLoading: dashboardLoading } =
    useApi<TeamDashboardData>('/api/dashboard/team')
  const { data: syncData } =
    useApi<SyncStatusData>('/api/sync/status')
  const { data: compareData } = useApi<TeamSummaryReport>(
    `/api/reports/team-summary?period_id=${comparePeriodId}`,
    { skip: !comparePeriodId },
  )

  if (dashboardLoading) {
    return <div className="loading">Carregando dashboard do time...</div>
  }

  if (!dashboardData || !dashboardData.team_members) {
    return <div className="error">Erro ao carregar dados do time</div>
  }

  const totalEarned = dashboardData.team_members.reduce((sum, m) => sum + m.commission_earned, 0)
  const totalPending = dashboardData.team_members.reduce((sum, m) => sum + m.commission_pending, 0)
  const avgAttainment =
    dashboardData.team_members.length > 0
      ? dashboardData.team_members.reduce((sum, m) => sum + m.attainment_pct, 0) /
        dashboardData.team_members.length
      : 0

  return (
    <div className="dashboard team-dashboard">
      <header className="dashboard-header">
        <h1>Dashboard da Equipe</h1>
        <p className="period-name">{dashboardData.period_name}</p>
      </header>

      <div className="dashboard-grid">
        <section className="dashboard-section commission-section">
          <h2>Comissões Totais</h2>
          <div className="commission-cards">
            <div className="commission-card earned">
              <h3>Total Ganho</h3>
              <p className="amount">{formatBRL(totalEarned)}</p>
            </div>
            <div className="commission-card pending">
              <h3>Total Pendente</h3>
              <p className="amount">{formatBRL(totalPending)}</p>
            </div>
          </div>
        </section>
      </div>

      <section className="dashboard-section compare-section">
        <h2>Comparar com período anterior</h2>
        <div className="compare-controls">
          <label htmlFor="compare-period">Período de comparação:</label>
          <input
            id="compare-period"
            type="number"
            value={comparePeriodId}
            onChange={(e) => setComparePeriodId(e.target.value)}
            placeholder="ID do período"
          />
        </div>
        {comparePeriodId && compareData && (
          <div className="compare-grid">
            <div className="compare-card">
              <h3>{compareData.period_name}</h3>
              <p className="compare-metric">
                Total: <strong>{formatBRL(compareData.total_commission)}</strong>
              </p>
              <p className="compare-metric">
                Atingimento: <strong>{Math.round(compareData.average_attainment)}%</strong>
              </p>
              <p className="compare-metric">
                Representantes: <strong>{compareData.reps_count}</strong>
              </p>
            </div>
            <div className="compare-card highlight">
              <h3>{dashboardData.period_name}</h3>
              <p className="compare-metric">
                Total: <strong>{formatBRL(totalEarned + totalPending)}</strong>{' '}
                <span className="compare-delta">
                  ({formatDelta(totalEarned + totalPending, compareData.total_commission)})
                </span>
              </p>
              <p className="compare-metric">
                Atingimento: <strong>{Math.round(avgAttainment)}%</strong>{' '}
                <span className="compare-delta">
                  ({formatPctDelta(avgAttainment, compareData.average_attainment)})
                </span>
              </p>
              <p className="compare-metric">
                Representantes: <strong>{dashboardData.team_members.length}</strong>
              </p>
            </div>
          </div>
        )}
      </section>

      <section className="dashboard-section team-table-section">
        <h2>Desempenho dos Representantes</h2>
        <div className="table-responsive">
          <table className="team-table">
            <thead>
              <tr>
                <th>Nome</th>
                <th>Atingimento</th>
                <th>Ganho</th>
                <th>Pendente</th>
                <th>Ação</th>
              </tr>
            </thead>
            <tbody>
              {dashboardData.team_members.map((member) => (
                <tr key={member.rep_id}>
                  <td className="member-name">{member.rep_name}</td>
                  <td className={`attainment ${getAttainmentClass(member.attainment_pct)}`}>
                    {Math.round(member.attainment_pct)}%
                  </td>
                  <td>{formatBRL(member.commission_earned)}</td>
                  <td>{formatBRL(member.commission_pending)}</td>
                  <td>
                    <button
                      className="drill-down-button"
                      onClick={() => navigate(`/rep/${member.rep_id}`)}
                    >
                      Detalhes
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <footer className="dashboard-footer">
        <SyncStatus lastSyncTime={syncData?.last_synced_at} />
      </footer>
    </div>
  )
}
