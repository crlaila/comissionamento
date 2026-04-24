package model

import "time"

type Goal struct {
	ID                 int64     `json:"id"`
	RepID              int64     `json:"rep_id"`
	PeriodID           int64     `json:"period_id"`
	AcquisitionTarget  int       `json:"acquisition_target"`
	RenewalTarget      int       `json:"renewal_target"`
	CommissionValue    int64     `json:"commission_value"` // in centavos
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
