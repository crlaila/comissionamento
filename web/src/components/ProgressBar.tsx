import React from 'react'
import './ProgressBar.css'

interface ProgressBarProps {
  current: number
  target: number
  label?: string
  showPercentage?: boolean
}

export const ProgressBar: React.FC<ProgressBarProps> = ({
  current,
  target,
  label,
  showPercentage = true,
}) => {
  const percentage = target > 0 ? (current / target) * 100 : 0
  const clampedPercentage = Math.min(percentage, 100)

  const getStatusClass = () => {
    if (percentage >= 100) return 'success'
    if (percentage >= 70) return 'warning'
    return 'danger'
  }

  return (
    <div className="progress-bar-container">
      {label && <div className="progress-label">{label}</div>}
      <div className="progress-bar-wrapper">
        <div className={`progress-bar ${getStatusClass()}`}>
          <div
            className="progress-fill"
            style={{ width: `${clampedPercentage}%` }}
          />
        </div>
        {showPercentage && (
          <div className="progress-percentage">{Math.round(percentage)}%</div>
        )}
      </div>
      <div className="progress-stats">
        <span className="progress-current">{current}</span>
        <span className="progress-slash">/</span>
        <span className="progress-target">{target}</span>
      </div>
    </div>
  )
}
