package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"comissionamento/internal/model"
)

type MockCommissionService struct {
	repDashboardFunc  func(ctx context.Context, repID int64) (*model.RepDashboard, error)
	teamDashboardFunc func(ctx context.Context, managerID int64) (*model.TeamDashboard, error)
	orgDashboardFunc  func(ctx context.Context) (*model.OrgDashboard, error)
}

func (m *MockCommissionService) CalculateForPeriod(ctx context.Context, periodID int64) error {
	return nil
}

func (m *MockCommissionService) GetRepDashboard(ctx context.Context, repID int64) (*model.RepDashboard, error) {
	if m.repDashboardFunc != nil {
		return m.repDashboardFunc(ctx, repID)
	}
	return &model.RepDashboard{}, nil
}

func (m *MockCommissionService) GetTeamDashboard(ctx context.Context, managerID int64) (*model.TeamDashboard, error) {
	if m.teamDashboardFunc != nil {
		return m.teamDashboardFunc(ctx, managerID)
	}
	return &model.TeamDashboard{}, nil
}

func (m *MockCommissionService) GetOrgDashboard(ctx context.Context) (*model.OrgDashboard, error) {
	if m.orgDashboardFunc != nil {
		return m.orgDashboardFunc(ctx)
	}
	return &model.OrgDashboard{}, nil
}

func (m *MockCommissionService) GenerateStatements(ctx context.Context, periodID int64) error {
	return nil
}

func (m *MockCommissionService) ApproveStatement(ctx context.Context, statementID int64, approverID int64) error {
	return nil
}

func TestGetRepDashboard(t *testing.T) {
	mockService := &MockCommissionService{
		repDashboardFunc: func(ctx context.Context, repID int64) (*model.RepDashboard, error) {
			return &model.RepDashboard{
				RepID:            repID,
				PeriodName:       "Jan 2026",
				AcquisitionGoal:  10,
				AcquisitionActual: 8,
				RenewalGoal:      5,
				RenewalActual:    4,
				AttainmentPct:    0.9,
				CommissionEarned: 50000,
			}, nil
		},
	}

	handler := NewDashboardHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/rep", nil)
	ctx := context.WithValue(req.Context(), "user_id", int64(1))
	ctx = context.WithValue(ctx, "user_role", model.RoleRep)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetRepDashboard(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var dashboard model.RepDashboard
	if err := json.NewDecoder(w.Body).Decode(&dashboard); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if dashboard.RepID != 1 {
		t.Errorf("expected rep_id 1, got %d", dashboard.RepID)
	}

	if dashboard.AttainmentPct != 0.9 {
		t.Errorf("expected attainment 0.9, got %f", dashboard.AttainmentPct)
	}
}

func TestGetTeamDashboard_OnlyManagersCanAccess(t *testing.T) {
	mockService := &MockCommissionService{}
	handler := NewDashboardHandler(mockService)

	// Test that reps cannot access team dashboard
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/team", nil)
	ctx := context.WithValue(req.Context(), "user_id", int64(1))
	ctx = context.WithValue(ctx, "user_role", model.RoleRep)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetTeamDashboard(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestGetTeamDashboard(t *testing.T) {
	mockService := &MockCommissionService{
		teamDashboardFunc: func(ctx context.Context, managerID int64) (*model.TeamDashboard, error) {
			return &model.TeamDashboard{
				ManagerID:       managerID,
				PeriodName:      "Jan 2026",
				TotalCommission: 100000,
				DirectReports: []model.RepDashboardSummary{
					{
						RepID:           1,
						RepName:         "John Doe",
						AttainmentPct:   0.8,
						CommissionEarned: 50000,
					},
				},
			}, nil
		},
	}

	handler := NewDashboardHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/team", nil)
	ctx := context.WithValue(req.Context(), "user_id", int64(2))
	ctx = context.WithValue(ctx, "user_role", model.RoleManager)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetTeamDashboard(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var dashboard model.TeamDashboard
	if err := json.NewDecoder(w.Body).Decode(&dashboard); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if dashboard.ManagerID != 2 {
		t.Errorf("expected manager_id 2, got %d", dashboard.ManagerID)
	}

	if dashboard.TotalCommission != 100000 {
		t.Errorf("expected total commission 100000, got %d", dashboard.TotalCommission)
	}
}

func TestGetOrgDashboard_OnlyFinanceCanAccess(t *testing.T) {
	mockService := &MockCommissionService{}
	handler := NewDashboardHandler(mockService)

	// Test that reps cannot access org dashboard
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/org", nil)
	ctx := context.WithValue(req.Context(), "user_id", int64(1))
	ctx = context.WithValue(ctx, "user_role", model.RoleRep)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetOrgDashboard(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestGetOrgDashboard(t *testing.T) {
	mockService := &MockCommissionService{
		orgDashboardFunc: func(ctx context.Context) (*model.OrgDashboard, error) {
			return &model.OrgDashboard{
				PeriodName:            "Jan 2026",
				TotalCommissionAmount: 500000,
				TotalReps:             5,
				AverageAttainmentPct:  0.85,
				PeriodStatus:          "open",
			}, nil
		},
	}

	handler := NewDashboardHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/org", nil)
	ctx := context.WithValue(req.Context(), "user_id", int64(3))
	ctx = context.WithValue(ctx, "user_role", model.RoleFinance)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetOrgDashboard(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var dashboard model.OrgDashboard
	if err := json.NewDecoder(w.Body).Decode(&dashboard); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if dashboard.TotalCommissionAmount != 500000 {
		t.Errorf("expected total commission 500000, got %d", dashboard.TotalCommissionAmount)
	}

	if dashboard.TotalReps != 5 {
		t.Errorf("expected 5 reps, got %d", dashboard.TotalReps)
	}
}
