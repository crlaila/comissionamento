package model

import "time"

type StatementStatus string

const (
	StatementStatusDraft            StatementStatus = "draft"
	StatementStatusPendingApproval  StatementStatus = "pending_approval"
	StatementStatusApproved         StatementStatus = "approved"
	StatementStatusPaid             StatementStatus = "paid"
)

type Statement struct {
	ID              int64            `json:"id"`
	RepID           int64            `json:"rep_id"`
	PeriodID        int64            `json:"period_id"`
	TotalAmount     int64            `json:"total_amount"` // in centavos
	AttainmentPct   float64          `json:"attainment_pct"`
	Status          StatementStatus  `json:"status"`
	ApprovedBy      *int64           `json:"approved_by"`
	ApprovedAt      *time.Time       `json:"approved_at"`
	RejectionReason *string          `json:"rejection_reason"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}
