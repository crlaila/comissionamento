import React from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useApi } from '../hooks/useApi'
import './EventDetail.css'

interface MemberEvent {
  id: number
  hinova_id: string
  rep_id: number
  rep_name: string
  event_type: 'acquisition' | 'renewal'
  member_name: string
  event_date: string
  synced_at: string
  created_at: string
}

export const EventDetail: React.FC = () => {
  const { eventId } = useParams<{ eventId: string }>()
  const navigate = useNavigate()
  const { data: event, isLoading, error } = useApi<MemberEvent>(
    `/api/events/${eventId}`,
    { skip: !eventId },
  )

  if (isLoading) {
    return <div className="loading">Carregando evento...</div>
  }

  if (error || !event) {
    return (
      <div className="error-container">
        <div className="error">Erro ao carregar evento</div>
        <button onClick={() => navigate(-1)}>Voltar</button>
      </div>
    )
  }

  const eventTypeLabel = event.event_type === 'acquisition' ? 'Aquisição' : 'Renovação'
  const eventTypeIcon = event.event_type === 'acquisition' ? '✓' : '↻'

  return (
    <div className="event-detail">
      <button className="back-button" onClick={() => navigate(-1)}>← Voltar</button>

      <div className="event-header">
        <h1>{event.member_name}</h1>
        <p className="event-type-label">
          <span className={`event-badge ${event.event_type}`}>
            {eventTypeIcon} {eventTypeLabel}
          </span>
        </p>
      </div>

      <div className="event-card">
        <div className="event-info-row">
          <span className="label">Representante:</span>
          <span className="value">{event.rep_name}</span>
        </div>
        <div className="event-info-row">
          <span className="label">ID Hinova:</span>
          <span className="value">{event.hinova_id}</span>
        </div>
        <div className="event-info-row">
          <span className="label">Data do Evento:</span>
          <span className="value">{new Date(event.event_date).toLocaleDateString('pt-BR', {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
          })}</span>
        </div>
        <div className="event-info-row">
          <span className="label">Sincronizado em:</span>
          <span className="value">{new Date(event.synced_at).toLocaleString('pt-BR')}</span>
        </div>
        <div className="event-info-row">
          <span className="label">Criado em:</span>
          <span className="value">{new Date(event.created_at).toLocaleString('pt-BR')}</span>
        </div>
      </div>

      <div className="event-actions">
        <button
          className="action-button"
          onClick={() => navigate(`/rep/${event.rep_id}`)}
        >
          Ver Dashboard do Rep
        </button>
      </div>
    </div>
  )
}
