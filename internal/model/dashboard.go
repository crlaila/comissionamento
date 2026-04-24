package model

type RepDashboard struct {
	RepID             int64           `json:"rep_id"`
	PeriodName        string          `json:"period_name"`
	AcquisitionGoal   int             `json:"acquisition_goal"`
	AcquisitionActual int             `json:"acquisition_actual"`
	RenewalGoal       int             `json:"renewal_goal"`
	RenewalActual     int             `json:"renewal_actual"`
	AttainmentPct     float64         `json:"attainment_pct"`
	CommissionEarned  int64           `json:"commission_earned"`   // in centavos
	CommissionPending int64           `json:"commission_pending"`  // in centavos
	RecentEvents      []*MemberEvent  `json:"recent_events"`
}

type TeamDashboard struct {
	ManagerID       int64                 `json:"manager_id"`
	PeriodName      string                `json:"period_name"`
	TotalCommission int64                 `json:"total_commission"`   // in centavos
	DirectReports   []RepDashboardSummary `json:"direct_reports"`
}

type RepDashboardSummary struct {
	RepID           int64   `json:"rep_id"`
	RepName         string  `json:"rep_name"`
	AttainmentPct   float64 `json:"attainment_pct"`
	CommissionEarned int64   `json:"commission_earned"` // in centavos
}
