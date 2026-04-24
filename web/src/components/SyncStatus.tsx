import React, { useEffect, useState } from 'react'
import './SyncStatus.css'

interface SyncStatusProps {
  lastSyncTime?: number // Timestamp em milissegundos
}

export const SyncStatus: React.FC<SyncStatusProps> = ({ lastSyncTime }) => {
  const [minutesAgo, setMinutesAgo] = useState<number | null>(null)

  useEffect(() => {
    const updateMinutesAgo = () => {
      if (!lastSyncTime) return

      const now = Date.now()
      const minutes = Math.floor((now - lastSyncTime) / 60000)
      setMinutesAgo(minutes)
    }

    updateMinutesAgo()
    const interval = setInterval(updateMinutesAgo, 60000) // Update every minute

    return () => clearInterval(interval)
  }, [lastSyncTime])

  if (!lastSyncTime || minutesAgo === null) {
    return <div className="sync-status pending">Sincronizando...</div>
  }

  const getStatus = () => {
    if (minutesAgo === 0) return 'just now'
    if (minutesAgo === 1) return '1 minuto atrás'
    if (minutesAgo < 60) return `${minutesAgo} minutos atrás`
    const hoursAgo = Math.floor(minutesAgo / 60)
    if (hoursAgo === 1) return '1 hora atrás'
    return `${hoursAgo} horas atrás`
  }

  return (
    <div className="sync-status">
      <span className="sync-indicator" />
      Última sincronização: <span className="sync-time">{getStatus()}</span>
    </div>
  )
}
