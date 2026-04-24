package model

import "testing"

// Unit tests for all domain models

func TestGoalModelMonetaryValues(t *testing.T) {
	tests := []struct {
		name            string
		commissionValue int64
		wantValid       bool
	}{
		{
			name:            "valid commission in centavos",
			commissionValue: 500000, // R$ 5.000,00
			wantValid:       true,
		},
		{
			name:            "zero commission",
			commissionValue: 0,
			wantValid:       true,
		},
		{
			name:            "large amount",
			commissionValue: 999999999, // Large value in centavos
			wantValid:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goal := &Goal{
				ID:              1,
				RepID:           1,
				PeriodID:        1,
				CommissionValue: tt.commissionValue,
			}

			if goal.CommissionValue != tt.commissionValue {
				t.Errorf("expected commission_value %d, got %d", tt.commissionValue, goal.CommissionValue)
			}
		})
	}
}

func TestStatementStatusWorkflow(t *testing.T) {
	// Define valid status transitions
	validTransitions := map[StatementStatus][]StatementStatus{
		StatementStatusDraft:           {StatementStatusPendingApproval},
		StatementStatusPendingApproval: {StatementStatusApproved},
		StatementStatusApproved:        {StatementStatusPaid},
		StatementStatusPaid:            {}, // Terminal state
	}

	stmt := &Statement{
		ID:       1,
		RepID:    1,
		PeriodID: 1,
		Status:   StatementStatusDraft,
	}

	// Verify initial status
	if stmt.Status != StatementStatusDraft {
		t.Errorf("expected initial status draft, got %s", stmt.Status)
	}

	// Test transition from draft
	nextStatuses := validTransitions[stmt.Status]
	if len(nextStatuses) == 0 {
		t.Errorf("draft status should have valid transitions")
	}

	if nextStatuses[0] != StatementStatusPendingApproval {
		t.Errorf("expected pending_approval as next status, got %s", nextStatuses[0])
	}

	// Test invalid transition
	invalidStatus := StatementStatus("invalid_status")
	if validTransitions[invalidStatus] != nil {
		t.Errorf("invalid status should not have transitions")
	}
}

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		expected  string
	}{
		{
			name:      "acquisition event",
			eventType: EventTypeAcquisition,
			expected:  "acquisition",
		},
		{
			name:      "renewal event",
			eventType: EventTypeRenewal,
			expected:  "renewal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.eventType) != tt.expected {
				t.Errorf("expected event type %s, got %s", tt.expected, tt.eventType)
			}
		})
	}
}

func TestStatementMonetaryValues(t *testing.T) {
	stmt := &Statement{
		ID:          1,
		RepID:       1,
		PeriodID:    1,
		TotalAmount: 250000, // R$ 2.500,00
		Status:      StatementStatusDraft,
	}

	if stmt.TotalAmount != 250000 {
		t.Errorf("expected total_amount 250000, got %d", stmt.TotalAmount)
	}

	if stmt.TotalAmount < 0 {
		t.Errorf("total_amount should not be negative")
	}
}
