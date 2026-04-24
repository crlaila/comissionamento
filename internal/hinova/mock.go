package hinova

import (
	"context"
	"time"

	"comissionamento/internal/model"
)

// MockClient implements HinovaClient for testing and development
type MockClient struct {
	Events []*model.MemberEvent
	Error  error
}

func NewMockClient() *MockClient {
	return &MockClient{
		Events: make([]*model.MemberEvent, 0),
	}
}

func (mc *MockClient) FetchMemberEvents(ctx context.Context, since time.Time) ([]*model.MemberEvent, error) {
	if mc.Error != nil {
		return nil, mc.Error
	}

	// Filter events by since timestamp
	var filtered []*model.MemberEvent
	for _, event := range mc.Events {
		if event.EventDate.After(since) || event.EventDate.Equal(since) {
			filtered = append(filtered, event)
		}
	}

	return filtered, nil
}

// AddEvent adds an event to the mock client
func (mc *MockClient) AddEvent(event *model.MemberEvent) {
	mc.Events = append(mc.Events, event)
}

// SetError sets an error to be returned on next call
func (mc *MockClient) SetError(err error) {
	mc.Error = err
}

// Clear resets the mock client
func (mc *MockClient) Clear() {
	mc.Events = make([]*model.MemberEvent, 0)
	mc.Error = nil
}
