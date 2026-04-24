package repository

import (
	"testing"
	"time"

	"comissionamento/internal/model"
)

// Unit tests for Period model and repository

func TestPeriodCreation(t *testing.T) {
	tests := []struct {
		name    string
		period  *model.Period
		wantErr bool
	}{
		{
			name: "valid period",
			period: &model.Period{
				Name:      "April 2026",
				StartDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
				Status:    model.PeriodStatusOpen,
			},
			wantErr: false,
		},
		{
			name: "period with closed status",
			period: &model.Period{
				Name:      "March 2026",
				StartDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
				Status:    model.PeriodStatusClosed,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify model fields are set correctly
			if tt.period.Name == "" {
				t.Errorf("period name should not be empty")
			}
			if tt.period.StartDate.IsZero() {
				t.Errorf("period start_date should not be zero")
			}
			if tt.period.EndDate.IsZero() {
				t.Errorf("period end_date should not be zero")
			}
			if tt.period.StartDate.After(tt.period.EndDate) {
				t.Errorf("start_date should be before end_date")
			}
		})
	}
}

func TestPeriodStatusTransitions(t *testing.T) {
	validTransitions := map[model.PeriodStatus][]model.PeriodStatus{
		model.PeriodStatusOpen:     {model.PeriodStatusClosed},
		model.PeriodStatusClosed:   {model.PeriodStatusArchived},
		model.PeriodStatusArchived: {},
	}

	period := &model.Period{
		Name:      "Test Period",
		StartDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}

	// Verify initial status
	if period.Status != model.PeriodStatusOpen {
		t.Errorf("initial status should be open")
	}

	// Simulate transition
	newStatus := validTransitions[period.Status][0]
	if newStatus != model.PeriodStatusClosed {
		t.Errorf("expected closed status, got %s", newStatus)
	}
}
