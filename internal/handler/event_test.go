package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"comissionamento/internal/model"
)

type MockEventRepository struct {
	events map[int64]*model.MemberEvent
}

func NewMockEventRepository() *MockEventRepository {
	return &MockEventRepository{
		events: make(map[int64]*model.MemberEvent),
	}
}

func (m *MockEventRepository) Upsert(ctx context.Context, event *model.MemberEvent) error {
	return nil
}

func (m *MockEventRepository) GetByID(ctx context.Context, id int64) (*model.MemberEvent, error) {
	return m.events[id], nil
}

func (m *MockEventRepository) ListByRepAndPeriod(ctx context.Context, repID int64, startDate, endDate interface{}) ([]*model.MemberEvent, error) {
	var events []*model.MemberEvent
	for _, e := range m.events {
		if e.RepID == repID {
			events = append(events, e)
		}
	}
	return events, nil
}

func (m *MockEventRepository) CountByRepAndType(ctx context.Context, repID int64, eventType model.EventType, startDate, endDate interface{}) (int, error) {
	count := 0
	for _, e := range m.events {
		if e.RepID == repID && e.EventType == eventType {
			count++
		}
	}
	return count, nil
}

func (m *MockEventRepository) ListByRep(ctx context.Context, repID int64) ([]*model.MemberEvent, error) {
	var events []*model.MemberEvent
	for _, e := range m.events {
		if e.RepID == repID {
			events = append(events, e)
		}
	}
	return events, nil
}

func TestListEvents_RepCanSeeOwnEvents(t *testing.T) {
	eventRepo := NewMockEventRepository()
	userRepo := NewMockUserRepository()
	periodRepo := NewMockPeriodRepository()

	eventRepo.events[1] = &model.MemberEvent{
		ID:        1,
		HinovaID:  "h1",
		RepID:     1,
		EventType: model.EventTypeAcquisition,
		MemberName: "John Doe",
		EventDate: time.Now(),
		SyncedAt:  time.Now(),
		CreatedAt: time.Now(),
	}

	eventRepo.events[2] = &model.MemberEvent{
		ID:        2,
		HinovaID:  "h2",
		RepID:     2,
		EventType: model.EventTypeRenewal,
		MemberName: "Jane Smith",
		EventDate: time.Now(),
		SyncedAt:  time.Now(),
		CreatedAt: time.Now(),
	}

	periodRepo.periods[1] = &model.Period{
		ID:        1,
		Name:      "Jan 2026",
		StartDate: time.Now().AddDate(0, 0, -30),
		EndDate:   time.Now(),
		Status:    model.PeriodStatusOpen,
	}

	handler := NewEventHandler(eventRepo, userRepo, periodRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/events?period_id=1", nil)
	ctx := context.WithValue(req.Context(), "user_id", int64(1))
	ctx = context.WithValue(ctx, "user_role", model.RoleRep)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ListEvents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var events []EventResponse
	if err := json.NewDecoder(w.Body).Decode(&events); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	if events[0].RepID != 1 {
		t.Errorf("expected rep_id 1, got %d", events[0].RepID)
	}
}

func TestListEvents_RequiresPeriodForFinance(t *testing.T) {
	eventRepo := NewMockEventRepository()
	userRepo := NewMockUserRepository()
	periodRepo := NewMockPeriodRepository()

	handler := NewEventHandler(eventRepo, userRepo, periodRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	ctx := context.WithValue(req.Context(), "user_id", int64(3))
	ctx = context.WithValue(ctx, "user_role", model.RoleFinance)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ListEvents(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Note: GetEvent tests require actual router context to properly parse path parameters
// These are covered in integration tests instead of unit tests
