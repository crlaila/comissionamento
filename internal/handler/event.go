package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"comissionamento/internal/model"
	"github.com/jackc/pgx/v5"
)

type MemberEventRepositoryInterface interface {
	GetByID(ctx context.Context, id int64) (*model.MemberEvent, error)
	ListByRepAndPeriod(ctx context.Context, repID int64, startDate, endDate interface{}) ([]*model.MemberEvent, error)
	ListByRep(ctx context.Context, repID int64) ([]*model.MemberEvent, error)
}

type UserRepositoryInterface interface {
	GetByID(ctx context.Context, id int64) (*model.User, error)
}

type PeriodRepositoryInterface interface {
	GetByID(ctx context.Context, id int64) (*model.Period, error)
}

type EventHandler struct {
	eventRepo  MemberEventRepositoryInterface
	userRepo   UserRepositoryInterface
	periodRepo PeriodRepositoryInterface
}

func NewEventHandler(eventRepo MemberEventRepositoryInterface, userRepo UserRepositoryInterface, periodRepo PeriodRepositoryInterface) *EventHandler {
	return &EventHandler{
		eventRepo:  eventRepo,
		userRepo:   userRepo,
		periodRepo: periodRepo,
	}
}

type EventResponse struct {
	ID        int64  `json:"id"`
	HinovaID  string `json:"hinova_id"`
	RepID     int64  `json:"rep_id"`
	EventType string `json:"event_type"`
	MemberName string `json:"member_name"`
	EventDate string `json:"event_date"`
	SyncedAt  string `json:"synced_at"`
	CreatedAt string `json:"created_at"`
}

// ListEvents handles GET /api/events?rep_id=X&period_id=Y
// Returns member events, filtered by role
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userRole, ok := r.Context().Value("user_role").(model.UserRole)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	repIDStr := r.URL.Query().Get("rep_id")
	periodIDStr := r.URL.Query().Get("period_id")

	// Get period to determine date range
	var period *model.Period
	if periodIDStr != "" {
		periodID, err := strconv.ParseInt(periodIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid period_id", http.StatusBadRequest)
			return
		}

		period, err = h.periodRepo.GetByID(r.Context(), periodID)
		if err != nil {
			slog.Error("failed to get period", "period_id", periodID, "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if period == nil {
			http.Error(w, "Period not found", http.StatusNotFound)
			return
		}
	}

	// Determine which rep to filter by based on role
	var filterRepID int64
	if repIDStr != "" {
		repID, err := strconv.ParseInt(repIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid rep_id", http.StatusBadRequest)
			return
		}
		filterRepID = repID
	}

	// Role-based filtering
	if userRole == model.RoleRep {
		// Reps can only see their own events
		filterRepID = userID
	} else if userRole == model.RoleManager {
		// Managers can see events for their direct reports
		// For MVP, we'll require explicit rep_id parameter
		if filterRepID == 0 {
			http.Error(w, "rep_id is required for managers", http.StatusBadRequest)
			return
		}
	} else if userRole == model.RoleFinance || userRole == model.RoleAdmin {
		// Finance and admins can see all events if rep_id is specified
		if filterRepID == 0 {
			http.Error(w, "rep_id is required", http.StatusBadRequest)
			return
		}
	}

	// List events by rep and period
	var events []*model.MemberEvent
	var err error

	if period != nil {
		events, err = h.eventRepo.ListByRepAndPeriod(r.Context(), filterRepID, period.StartDate, period.EndDate)
	} else {
		// If no period specified, list all events for rep
		events, err = h.eventRepo.ListByRep(r.Context(), filterRepID)
	}

	if err != nil {
		slog.Error("failed to list events", "rep_id", filterRepID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]EventResponse, len(events))
	for i, event := range events {
		response[i] = EventResponse{
			ID:        event.ID,
			HinovaID:  event.HinovaID,
			RepID:     event.RepID,
			EventType: string(event.EventType),
			MemberName: event.MemberName,
			EventDate: event.EventDate.String(),
			SyncedAt:  event.SyncedAt.String(),
			CreatedAt: event.CreatedAt.String(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetEvent handles GET /api/events/{id}
// Returns event detail with commission attribution
func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userRole, ok := r.Context().Value("user_role").(model.UserRole)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	event, err := h.eventRepo.GetByID(r.Context(), eventID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get event", "event_id", eventID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if event == nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Role-based access check
	if userRole == model.RoleRep {
		// Reps can only see their own events
		if event.RepID != userID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	} else if userRole == model.RoleManager {
		// Managers can see events for their direct reports
		// For MVP, we allow access if they can view the rep's data
		_ = userID // not checking manager relationship for MVP
	}
	// Finance and admins can see all events

	response := EventResponse{
		ID:        event.ID,
		HinovaID:  event.HinovaID,
		RepID:     event.RepID,
		EventType: string(event.EventType),
		MemberName: event.MemberName,
		EventDate: event.EventDate.String(),
		SyncedAt:  event.SyncedAt.String(),
		CreatedAt: event.CreatedAt.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
