import React, { useState } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useApi } from '../hooks/useApi'
import './GoalManagement.css'

interface Goal {
  id: number
  rep_id: number
  rep_name: string
  period_id: number
  acquisition_target: number
  renewal_target: number
  commission_value: number
}

interface GoalsData {
  goals: Goal[]
}

export const GoalManagement: React.FC = () => {
  const { user } = useAuth()
  const [periodId, setPeriodId] = useState<string>('')
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({
    repId: '',
    acquisitionTarget: '',
    renewalTarget: '',
    commissionValue: '',
  })
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  const queryString = periodId ? `?period_id=${periodId}` : ''
  const { data: goalsData, isLoading } = useApi<GoalsData>(
    `/api/goals${queryString}`,
    { skip: !periodId },
  )

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setMessage(null)

    if (!formData.repId || !formData.acquisitionTarget || !formData.renewalTarget || !formData.commissionValue) {
      setMessage({ type: 'error', text: 'Todos os campos são obrigatórios' })
      return
    }

    try {
      const response = await fetch('/api/goals', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${useAuth().accessToken}`,
        },
        body: JSON.stringify({
          rep_id: parseInt(formData.repId),
          period_id: parseInt(periodId),
          acquisition_target: parseInt(formData.acquisitionTarget),
          renewal_target: parseInt(formData.renewalTarget),
          commission_value: parseInt(formData.commissionValue),
        }),
      })

      if (!response.ok) {
        throw new Error('Falha ao criar meta')
      }

      setMessage({ type: 'success', text: 'Meta criada com sucesso!' })
      setFormData({ repId: '', acquisitionTarget: '', renewalTarget: '', commissionValue: '' })
      setShowForm(false)
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Erro ao criar meta' })
    }
  }

  if (user?.role !== 'manager' && user?.role !== 'admin') {
    return <div className="error">Acesso negado. Apenas gerenciadores podem gerenciar metas.</div>
  }

  return (
    <div className="goal-management">
      <header className="goal-header">
        <h1>Gerenciamento de Metas</h1>
        <p>Configure as metas de aquisição e renovação para sua equipe</p>
      </header>

      <div className="goal-controls">
        <div className="period-selector">
          <label htmlFor="period">Período:</label>
          <input
            id="period"
            type="number"
            value={periodId}
            onChange={(e) => setPeriodId(e.target.value)}
            placeholder="ID do período"
          />
        </div>

        <button
          className="add-goal-button"
          onClick={() => setShowForm(!showForm)}
          disabled={!periodId}
        >
          {showForm ? '✕ Cancelar' : '+ Nova Meta'}
        </button>
      </div>

      {message && (
        <div className={`message ${message.type}`}>
          {message.text}
        </div>
      )}

      {showForm && (
        <form className="goal-form" onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="repId">Representante:</label>
            <input
              id="repId"
              type="number"
              value={formData.repId}
              onChange={(e) => setFormData({ ...formData, repId: e.target.value })}
              placeholder="ID do representante"
              required
            />
          </div>

          <div className="form-row">
            <div className="form-group">
              <label htmlFor="acquisitionTarget">Meta de Aquisição:</label>
              <input
                id="acquisitionTarget"
                type="number"
                value={formData.acquisitionTarget}
                onChange={(e) => setFormData({ ...formData, acquisitionTarget: e.target.value })}
                placeholder="0"
                required
              />
            </div>

            <div className="form-group">
              <label htmlFor="renewalTarget">Meta de Renovação:</label>
              <input
                id="renewalTarget"
                type="number"
                value={formData.renewalTarget}
                onChange={(e) => setFormData({ ...formData, renewalTarget: e.target.value })}
                placeholder="0"
                required
              />
            </div>
          </div>

          <div className="form-group">
            <label htmlFor="commissionValue">Valor de Comissão (em centavos):</label>
            <input
              id="commissionValue"
              type="number"
              value={formData.commissionValue}
              onChange={(e) => setFormData({ ...formData, commissionValue: e.target.value })}
              placeholder="0"
              required
            />
          </div>

          <button type="submit" className="submit-button">Criar Meta</button>
        </form>
      )}

      {periodId && (
        <section className="goals-list">
          <h2>Metas do Período</h2>
          {isLoading ? (
            <p className="loading">Carregando metas...</p>
          ) : goalsData?.goals && goalsData.goals.length > 0 ? (
            <table className="goals-table">
              <thead>
                <tr>
                  <th>Representante</th>
                  <th>Aquisições</th>
                  <th>Renovações</th>
                  <th>Comissão (R$)</th>
                  <th>Ações</th>
                </tr>
              </thead>
              <tbody>
                {goalsData.goals.map((goal) => (
                  <tr key={goal.id}>
                    <td>{goal.rep_name}</td>
                    <td>{goal.acquisition_target}</td>
                    <td>{goal.renewal_target}</td>
                    <td>{(goal.commission_value / 100).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}</td>
                    <td>
                      <button className="edit-button">Editar</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="no-goals">Nenhuma meta configurada para este período</p>
          )}
        </section>
      )}
    </div>
  )
}
