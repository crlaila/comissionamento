import React from 'react'
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

export const TeamDashboard: React.FC = () => {
  const navigate = useNavigate()
  const { data: dashboardData, isLoading: dashboardLoading } =
    useApi<TeamDashboardData>('/api/dashboard/team')
  const { data: syncData } =
    useApi<SyncStatusData>('/api/sync/status')

  if (dashboardLoading) {
    return <div className="loading">Carregando dashboard do time...</div>
  }

  if (!dashboardData || !dashboardData.team_members) {
    return <div className="error">Erro ao carregar dados do time</div>
  }

  const totalEarned = dashboardData.team_members.reduce((sum, m) => sum + m.commission_earned, 0)
  const totalPending = dashboardData.team_members.reduce((sum, m) => sum + m.commission_pending, 0)

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
