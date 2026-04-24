package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"comissionamento/internal/model"
)

type MockGoalRepository struct {
	goals map[int64]*model.Goal
}

func NewMockGoalRepository() *MockGoalRepository {
	return &MockGoalRepository{
		goals: make(map[int64]*model.Goal),
	}
}

func (m *MockGoalRepository) Create(ctx context.Context, goal *model.Goal) error {
	goal.ID = int64(len(m.goals) + 1)
	goal.CreatedAt = time.Now()
	goal.UpdatedAt = time.Now()
	m.goals[goal.ID] = goal
	return nil
}

func (m *MockGoalRepository) GetByRepAndPeriod(ctx context.Context, repID, periodID int64) (*model.Goal, error) {
	for _, g := range m.goals {
		if g.RepID == repID && g.PeriodID == periodID {
			return g, nil
		}
	}
	return nil, nil
}

func (m *MockGoalRepository) ListByPeriod(ctx context.Context, periodID int64) ([]*model.Goal, error) {
	var goals []*model.Goal
	for _, g := range m.goals {
		if g.PeriodID == periodID {
			goals = append(goals, g)
		}
	}
	return goals, nil
}

func (m *MockGoalRepository) ListByRep(ctx context.Context, repID int64) ([]*model.Goal, error) {
	var goals []*model.Goal
	for _, g := range m.goals {
		if g.RepID == repID {
			goals = append(goals, g)
		}
	}
	return goals, nil
}

func (m *MockGoalRepository) Update(ctx context.Context, goal *model.Goal) error {
	if _, ok := m.goals[goal.ID]; !ok {
		return nil
	}
	goal.UpdatedAt = time.Now()
	m.goals[goal.ID] = goal
	return nil
}

func (m *MockGoalRepository) DeleteByID(ctx context.Context, id int64) error {
	delete(m.goals, id)
	return nil
}

type MockPeriodRepository struct {
	periods map[int64]*model.Period
}

func NewMockPeriodRepository() *MockPeriodRepository {
	return &MockPeriodRepository{
		periods: make(map[int64]*model.Period),
	}
}

func (m *MockPeriodRepository) Create(ctx context.Context, period *model.Period) error {
	return nil
}

func (m *MockPeriodRepository) GetByID(ctx context.Context, id int64) (*model.Period, error) {
	return m.periods[id], nil
}

func (m *MockPeriodRepository) List(ctx context.Context) ([]*model.Period, error) {
	var periods []*model.Period
	for _, p := range m.periods {
		periods = append(periods, p)
	}
	return periods, nil
}

func (m *MockPeriodRepository) Update(ctx context.Context, period *model.Period) error {
	return nil
}

func TestListGoals_RepCanSeeOwnGoals(t *testing.T) {
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()

	// Add test goals
	goalRepo.goals[1] = &model.Goal{
		ID:                1,
		RepID:             1,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     5,
		CommissionValue:   10000,
	}
	goalRepo.goals[2] = &model.Goal{
		ID:                2,
		RepID:             2,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     5,
		CommissionValue:   10000,
	}

	handler := NewGoalHandler(goalRepo, periodRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/goals?period_id=1", nil)
	ctx := context.WithValue(req.Context(), "user_id", int64(1))
	ctx = context.WithValue(ctx, "user_role", model.RoleRep)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ListGoals(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var goals []GoalResponse
	if err := json.NewDecoder(w.Body).Decode(&goals); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(goals) != 1 {
		t.Errorf("expected 1 goal, got %d", len(goals))
	}

	if goals[0].RepID != 1 {
		t.Errorf("expected rep_id 1, got %d", goals[0].RepID)
	}
}

func TestCreateGoal_OnlyManagersCanCreate(t *testing.T) {
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	periodRepo.periods[1] = &model.Period{
		ID:     1,
		Name:   "Jan 2026",
		Status: model.PeriodStatusOpen,
	}

	handler := NewGoalHandler(goalRepo, periodRepo)

	reqBody := CreateGoalRequest{
		RepID:             1,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     5,
		CommissionValue:   10000,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/goals", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), "user_id", int64(1))
	ctx = context.WithValue(ctx, "user_role", model.RoleRep)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CreateGoal(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestCreateGoal(t *testing.T) {
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	periodRepo.periods[1] = &model.Period{
		ID:     1,
		Name:   "Jan 2026",
		Status: model.PeriodStatusOpen,
	}

	handler := NewGoalHandler(goalRepo, periodRepo)

	reqBody := CreateGoalRequest{
		RepID:             1,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     5,
		CommissionValue:   10000,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/goals", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), "user_id", int64(2))
	ctx = context.WithValue(ctx, "user_role", model.RoleManager)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CreateGoal(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var goal GoalResponse
	if err := json.NewDecoder(w.Body).Decode(&goal); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if goal.RepID != 1 {
		t.Errorf("expected rep_id 1, got %d", goal.RepID)
	}

	if goal.CommissionValue != 10000 {
		t.Errorf("expected commission 10000, got %d", goal.CommissionValue)
	}
}

func TestUpdateGoal_OnlyManagersCanUpdate(t *testing.T) {
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()

	handler := NewGoalHandler(goalRepo, periodRepo)

	reqBody := UpdateGoalRequest{
		AcquisitionTarget: 15,
		RenewalTarget:     8,
		CommissionValue:   15000,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/api/goals/1", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), "user_id", int64(1))
	ctx = context.WithValue(ctx, "user_role", model.RoleRep)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.UpdateGoal(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestListGoals_MissingPeriodID(t *testing.T) {
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()

	handler := NewGoalHandler(goalRepo, periodRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/goals", nil)
	ctx := context.WithValue(req.Context(), "user_id", int64(1))
	ctx = context.WithValue(ctx, "user_role", model.RoleRep)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ListGoals(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestListGoals_InvalidPeriodID(t *testing.T) {
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()

	handler := NewGoalHandler(goalRepo, periodRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/goals?period_id=invalid", nil)
	ctx := context.WithValue(req.Context(), "user_id", int64(1))
	ctx = context.WithValue(ctx, "user_role", model.RoleRep)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ListGoals(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateGoal_PeriodNotFound(t *testing.T) {
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()

	handler := NewGoalHandler(goalRepo, periodRepo)

	reqBody := CreateGoalRequest{
		RepID:             1,
		PeriodID:          999,
		AcquisitionTarget: 10,
		RenewalTarget:     5,
		CommissionValue:   10000,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/goals", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), "user_id", int64(2))
	ctx = context.WithValue(ctx, "user_role", model.RoleManager)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CreateGoal(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCreateGoal_ClosedPeriod(t *testing.T) {
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	periodRepo.periods[1] = &model.Period{
		ID:     1,
		Name:   "Jan 2026",
		Status: model.PeriodStatusClosed,
	}

	handler := NewGoalHandler(goalRepo, periodRepo)

	reqBody := CreateGoalRequest{
		RepID:             1,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     5,
		CommissionValue:   10000,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/goals", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), "user_id", int64(2))
	ctx = context.WithValue(ctx, "user_role", model.RoleManager)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CreateGoal(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
