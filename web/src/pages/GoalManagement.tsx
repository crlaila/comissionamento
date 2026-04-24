import React, { useState, useEffect } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useApi } from '../hooks/useApi'
import './GoalManagement.css'

interface Goal {
  id: number
  rep_id: number
  rep_name?: string
  period_id: number
  acquisition_target: number
  renewal_target: number
  commission_value: number
}

type GoalsResponse = Goal[] | { goals: Goal[] }

const emptyForm = {
  repId: '',
  acquisitionTarget: '',
  renewalTarget: '',
  commissionValue: '',
}

const formatBRL = (centavos: number): string =>
  (centavos / 100).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })

export const GoalManagement: React.FC = () => {
  const { user, accessToken } = useAuth()
  const [periodId, setPeriodId] = useState<string>('')
  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [formData, setFormData] = useState(emptyForm)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const [reloadKey, setReloadKey] = useState(0)

  const queryString = periodId ? `?period_id=${periodId}&_r=${reloadKey}` : ''
  const { data: goalsData, isLoading } = useApi<GoalsResponse>(
    `/api/goals${queryString}`,
    { skip: !periodId },
  )

  const goals: Goal[] = Array.isArray(goalsData)
    ? goalsData
    : goalsData?.goals ?? []

  useEffect(() => {
    if (!showForm) {
      setEditingId(null)
      setFormData(emptyForm)
    }
  }, [showForm])

  if (user?.role !== 'manager' && user?.role !== 'admin') {
    return <div className="error">Acesso negado. Apenas gerenciadores podem gerenciar metas.</div>
  }

  const startEdit = (goal: Goal) => {
    setEditingId(goal.id)
    setFormData({
      repId: String(goal.rep_id),
      acquisitionTarget: String(goal.acquisition_target),
      renewalTarget: String(goal.renewal_target),
      commissionValue: String(goal.commission_value),
    })
    setShowForm(true)
    setMessage(null)
  }

  const resetForm = () => {
    setFormData(emptyForm)
    setEditingId(null)
    setShowForm(false)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setMessage(null)

    if (
      !formData.repId ||
      !formData.acquisitionTarget ||
      !formData.renewalTarget ||
      !formData.commissionValue
    ) {
      setMessage({ type: 'error', text: 'Todos os campos são obrigatórios' })
      return
    }

    const isEdit = editingId !== null
    const url = isEdit ? `/api/goals/${editingId}` : '/api/goals'
    const method = isEdit ? 'PUT' : 'POST'
    const body = isEdit
      ? {
          acquisition_target: parseInt(formData.acquisitionTarget),
          renewal_target: parseInt(formData.renewalTarget),
          commission_value: parseInt(formData.commissionValue),
        }
      : {
          rep_id: parseInt(formData.repId),
          period_id: parseInt(periodId),
          acquisition_target: parseInt(formData.acquisitionTarget),
          renewal_target: parseInt(formData.renewalTarget),
          commission_value: parseInt(formData.commissionValue),
        }

    try {
      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${accessToken ?? ''}`,
        },
        body: JSON.stringify(body),
      })

      if (!response.ok) {
        throw new Error(isEdit ? 'Falha ao atualizar meta' : 'Falha ao criar meta')
      }

      setMessage({
        type: 'success',
        text: isEdit ? 'Meta atualizada com sucesso!' : 'Meta criada com sucesso!',
      })
      resetForm()
      setReloadKey((k) => k + 1)
    } catch (err) {
      setMessage({
        type: 'error',
        text: err instanceof Error ? err.message : 'Erro ao salvar meta',
      })
    }
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
          onClick={() => {
            if (showForm) {
              resetForm()
            } else {
              setEditingId(null)
              setFormData(emptyForm)
              setShowForm(true)
            }
          }}
          disabled={!periodId}
        >
          {showForm ? '✕ Cancelar' : '+ Nova Meta'}
        </button>
      </div>

      {message && (
        <div className={`message ${message.type}`}>{message.text}</div>
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
              disabled={editingId !== null}
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
                onChange={(e) =>
                  setFormData({ ...formData, acquisitionTarget: e.target.value })
                }
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
                onChange={(e) =>
                  setFormData({ ...formData, renewalTarget: e.target.value })
                }
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
              onChange={(e) =>
                setFormData({ ...formData, commissionValue: e.target.value })
              }
              placeholder="0"
              required
            />
          </div>

          <button type="submit" className="submit-button">
            {editingId !== null ? 'Salvar Alterações' : 'Criar Meta'}
          </button>
        </form>
      )}

      {periodId && (
        <section className="goals-list">
          <h2>Metas do Período</h2>
          {isLoading ? (
            <p className="loading">Carregando metas...</p>
          ) : goals.length > 0 ? (
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
                {goals.map((goal) => (
                  <tr key={goal.id}>
                    <td>{goal.rep_name ?? `Rep #${goal.rep_id}`}</td>
                    <td>{goal.acquisition_target}</td>
                    <td>{goal.renewal_target}</td>
                    <td>{formatBRL(goal.commission_value)}</td>
                    <td>
                      <button className="edit-button" onClick={() => startEdit(goal)}>
                        Editar
                      </button>
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
