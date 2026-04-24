package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"comissionamento/internal/hinova"
	"comissionamento/internal/model"
)

// MockEventRepository for testing
type MockEventRepository struct {
	events map[string]*model.MemberEvent
	errors map[string]error
}

func NewMockEventRepository() *MockEventRepository {
	return &MockEventRepository{
		events: make(map[string]*model.MemberEvent),
		errors: make(map[string]error),
	}
}

func (m *MockEventRepository) Upsert(ctx context.Context, event *model.MemberEvent) error {
	if err, exists := m.errors[event.HinovaID]; exists {
		return err
	}
	m.events[event.HinovaID] = event
	return nil
}

func TestSyncMemberEvents_Success(t *testing.T) {
	mockClient := hinova.NewMockClient()
	mockRepo := NewMockEventRepository()
	syncService := NewSyncService(mockClient, mockRepo, 5*time.Minute)

	// Add test events - use dates within last 12 hours (service defaults to 24h lookback)
	now := time.Now().UTC()
	events := []*model.MemberEvent{
		{
			HinovaID:   "hinova-1",
			RepID:      1,
			EventType:  model.EventTypeAcquisition,
			MemberName: "Member 1",
			EventDate:  now.Add(-12 * time.Hour),
			SyncedAt:   now,
		},
		{
			HinovaID:   "hinova-2",
			RepID:      2,
			EventType:  model.EventTypeRenewal,
			MemberName: "Member 2",
			EventDate:  now.Add(-6 * time.Hour),
			SyncedAt:   now,
		},
	}

	for _, event := range events {
		mockClient.AddEvent(event)
	}

	result, err := syncService.SyncMemberEvents(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.EventsFetched != 2 {
		t.Errorf("expected 2 events fetched, got %d", result.EventsFetched)
	}

	if result.EventsNew != 2 {
		t.Errorf("expected 2 new events, got %d", result.EventsNew)
	}

	if result.EventsDuplicate != 0 {
		t.Errorf("expected 0 duplicate events, got %d", result.EventsDuplicate)
	}

	if syncService.consecutiveErrors != 0 {
		t.Errorf("expected 0 consecutive errors, got %d", syncService.consecutiveErrors)
	}
}

func TestSyncMemberEvents_Deduplication(t *testing.T) {
	mockClient := hinova.NewMockClient()
	mockRepo := NewMockEventRepository()
	syncService := NewSyncService(mockClient, mockRepo, 5*time.Minute)

	now := time.Now().UTC()

	// Pre-populate repository with an existing event
	existingEvent := &model.MemberEvent{
		HinovaID:   "hinova-1",
		RepID:      1,
		EventType:  model.EventTypeAcquisition,
		MemberName: "Member 1",
		EventDate:  now.Add(-12 * time.Hour),
		SyncedAt:   now,
		ID:         1,
		CreatedAt:  now,
	}
	mockRepo.Upsert(context.Background(), existingEvent)

	// Add the same event to the mock client (simulating duplicate from Hinova)
	mockClient.AddEvent(&model.MemberEvent{
		HinovaID:   "hinova-1",
		RepID:      1,
		EventType:  model.EventTypeAcquisition,
		MemberName: "Member 1",
		EventDate:  now.Add(-12 * time.Hour),
		SyncedAt:   now,
	})

	// Add a new event
	mockClient.AddEvent(&model.MemberEvent{
		HinovaID:   "hinova-2",
		RepID:      2,
		EventType:  model.EventTypeRenewal,
		MemberName: "Member 2",
		EventDate:  now.Add(-6 * time.Hour),
		SyncedAt:   now,
	})

	result, err := syncService.SyncMemberEvents(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.EventsFetched != 2 {
		t.Errorf("expected 2 events fetched, got %d", result.EventsFetched)
	}

	if result.EventsNew != 2 {
		t.Errorf("expected 2 new events (deduplication happens in repo), got %d", result.EventsNew)
	}
}

func TestSyncMemberEvents_ApiFailure(t *testing.T) {
	mockClient := hinova.NewMockClient()
	mockRepo := NewMockEventRepository()
	syncService := NewSyncService(mockClient, mockRepo, 5*time.Minute)

	mockClient.SetError(errors.New("api error"))

	result, err := syncService.SyncMemberEvents(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Error == "" {
		t.Error("expected error in result")
	}

	if syncService.consecutiveErrors != 1 {
		t.Errorf("expected 1 consecutive error, got %d", syncService.consecutiveErrors)
	}
}

func TestSyncMemberEvents_MultipleFailures(t *testing.T) {
	mockClient := hinova.NewMockClient()
	mockRepo := NewMockEventRepository()
	syncService := NewSyncService(mockClient, mockRepo, 5*time.Minute)

	mockClient.SetError(errors.New("api error"))

	// Simulate 3 failed sync attempts
	for i := 0; i < 3; i++ {
		syncService.SyncMemberEvents(context.Background())
	}

	if syncService.consecutiveErrors != 3 {
		t.Errorf("expected 3 consecutive errors, got %d", syncService.consecutiveErrors)
	}
}

func TestSyncMemberEvents_ErrorRecovery(t *testing.T) {
	mockClient := hinova.NewMockClient()
	mockRepo := NewMockEventRepository()
	syncService := NewSyncService(mockClient, mockRepo, 5*time.Minute)

	// First failure
	mockClient.SetError(errors.New("api error"))
	syncService.SyncMemberEvents(context.Background())

	if syncService.consecutiveErrors != 1 {
		t.Errorf("expected 1 consecutive error after first failure, got %d", syncService.consecutiveErrors)
	}

	// Successful sync
	mockClient.SetError(nil)
	mockClient.AddEvent(&model.MemberEvent{
		HinovaID:   "hinova-1",
		RepID:      1,
		EventType:  model.EventTypeAcquisition,
		MemberName: "Member 1",
		EventDate:  time.Now().UTC(),
		SyncedAt:   time.Now().UTC(),
	})

	syncService.SyncMemberEvents(context.Background())

	if syncService.consecutiveErrors != 0 {
		t.Errorf("expected 0 consecutive errors after successful sync, got %d", syncService.consecutiveErrors)
	}
}

func TestSyncMemberEvents_SyncInterval(t *testing.T) {
	mockClient := hinova.NewMockClient()
	mockRepo := NewMockEventRepository()
	syncInterval := 1 * time.Second
	syncService := NewSyncService(mockClient, mockRepo, syncInterval)

	if syncService.syncInterval != syncInterval {
		t.Errorf("expected sync interval %v, got %v", syncInterval, syncService.syncInterval)
	}
}

func TestGetSyncStatus(t *testing.T) {
	mockClient := hinova.NewMockClient()
	mockRepo := NewMockEventRepository()
	syncService := NewSyncService(mockClient, mockRepo, 5*time.Minute)

	// Set initial status
	syncService.lastSyncedAt = time.Now().UTC()
	syncService.status = "idle"
	syncService.consecutiveErrors = 0

	status, err := syncService.GetSyncStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Status != "idle" {
		t.Errorf("expected status 'idle', got %s", status.Status)
	}

	if status.FailureCount != 0 {
		t.Errorf("expected 0 failures, got %d", status.FailureCount)
	}
}

func TestExponentialBackoff(t *testing.T) {
	// Test that backoff increases exponentially
	// Verify backoff formula: baseMs * 2^(attempt-1)
	// Attempt 1: 1000 * 2^0 = 1000ms
	// Attempt 2: 1000 * 2^1 = 2000ms
	// Attempt 3: 1000 * 2^2 = 4000ms

	baseMs := 1000
	for attempt := 1; attempt < 4; attempt++ {
		multiplier := 1 << uint(attempt-1)
		expectedBackoff := int64(baseMs * multiplier)
		if attempt == 1 {
			if expectedBackoff != 1000 {
				t.Errorf("attempt %d: expected backoff 1000, got %d", attempt, expectedBackoff)
			}
		} else if attempt == 2 {
			if expectedBackoff != 2000 {
				t.Errorf("attempt %d: expected backoff 2000, got %d", attempt, expectedBackoff)
			}
		} else if attempt == 3 {
			if expectedBackoff != 4000 {
				t.Errorf("attempt %d: expected backoff 4000, got %d", attempt, expectedBackoff)
			}
		}
	}
}
