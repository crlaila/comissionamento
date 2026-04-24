package model

import "time"

type PeriodStatus string

const (
	PeriodStatusOpen     PeriodStatus = "open"
	PeriodStatusClosed   PeriodStatus = "closed"
	PeriodStatusArchived PeriodStatus = "archived"
)

type Period struct {
	ID        int64         `json:"id"`
	Name      string        `json:"name"`
	StartDate time.Time     `json:"start_date"`
	EndDate   time.Time     `json:"end_date"`
	Status    PeriodStatus  `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}
