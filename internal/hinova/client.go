package hinova

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"comissionamento/internal/model"
)

type HinovaClient interface {
	FetchMemberEvents(ctx context.Context, since time.Time) ([]*model.MemberEvent, error)
}

type HTTPClient struct {
	baseURL   string
	apiToken  string
	httpClient *http.Client
}

func NewHTTPClient(baseURL, apiToken string) *HTTPClient {
	return &HTTPClient{
		baseURL:   baseURL,
		apiToken:  apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type HinovaEventResponse struct {
	ID         string    `json:"id"`
	RepID      int64     `json:"seller_id"`
	EventType  string    `json:"type"` // "aquisicao" or "renovacao"
	MemberName string    `json:"member_name"`
	EventDate  string    `json:"date"` // ISO 8601 format
	CreatedAt  string    `json:"created_at"`
	UpdatedAt  string    `json:"updated_at"`
}

type HinovaListResponse struct {
	Data  []HinovaEventResponse `json:"data"`
	Error string                `json:"error"`
}

func (hc *HTTPClient) FetchMemberEvents(ctx context.Context, since time.Time) ([]*model.MemberEvent, error) {
	// Build query parameters
	sinceStr := since.UTC().Format(time.RFC3339)
	url := fmt.Sprintf("%s/events?since=%s", hc.baseURL, sinceStr)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+hc.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := hc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hinova api returned status %d: %s", resp.StatusCode, string(body))
	}

	var hinovaResp HinovaListResponse
	if err := json.Unmarshal(body, &hinovaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if hinovaResp.Error != "" {
		return nil, fmt.Errorf("hinova api error: %s", hinovaResp.Error)
	}

	// Convert Hinova response to domain model
	var events []*model.MemberEvent
	for _, event := range hinovaResp.Data {
		eventDate, err := time.Parse(time.DateOnly, event.EventDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse event date: %w", err)
		}

		var eventType model.EventType
		switch event.EventType {
		case "aquisicao":
			eventType = model.EventTypeAcquisition
		case "renovacao":
			eventType = model.EventTypeRenewal
		default:
			return nil, fmt.Errorf("unknown event type: %s", event.EventType)
		}

		memberEvent := &model.MemberEvent{
			HinovaID:  event.ID,
			RepID:     event.RepID,
			EventType: eventType,
			MemberName: event.MemberName,
			EventDate: eventDate,
			SyncedAt:  time.Now().UTC(),
		}

		events = append(events, memberEvent)
	}

	return events, nil
}
