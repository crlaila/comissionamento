package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"comissionamento/internal/model"
)

type GoalRepositoryInterface interface {
	Create(ctx context.Context, goal *model.Goal) error
	ListByPeriod(ctx context.Context, periodID int64) ([]*model.Goal, error)
	Update(ctx context.Context, goal *model.Goal) error
}

type GoalHandler struct {
	goalRepo   GoalRepositoryInterface
	periodRepo PeriodRepositoryInterface
}

func NewGoalHandler(goalRepo GoalRepositoryInterface, periodRepo PeriodRepositoryInterface) *GoalHandler {
	return &GoalHandler{
		goalRepo:   goalRepo,
		periodRepo: periodRepo,
	}
}

type GoalResponse struct {
	ID                int64  `json:"id"`
	RepID             int64  `json:"rep_id"`
	PeriodID          int64  `json:"period_id"`
	AcquisitionTarget int    `json:"acquisition_target"`
	RenewalTarget     int    `json:"renewal_target"`
	CommissionValue   int64  `json:"commission_value"` // in centavos
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

type CreateGoalRequest struct {
	RepID             int64 `json:"rep_id"`
	PeriodID          int64 `json:"period_id"`
	AcquisitionTarget int   `json:"acquisition_target"`
	RenewalTarget     int   `json:"renewal_target"`
	CommissionValue   int64 `json:"commission_value"`
}

type UpdateGoalRequest struct {
	AcquisitionTarget int   `json:"acquisition_target"`
	RenewalTarget     int   `json:"renewal_target"`
	CommissionValue   int64 `json:"commission_value"`
}

// ListGoals handles GET /api/goals?period_id=X
// Returns goals for a period, filtered by role
func (h *GoalHandler) ListGoals(w http.ResponseWriter, r *http.Request) {
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

	periodIDStr := r.URL.Query().Get("period_id")
	if periodIDStr == "" {
		http.Error(w, "period_id is required", http.StatusBadRequest)
		return
	}

	periodID, err := strconv.ParseInt(periodIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid period_id", http.StatusBadRequest)
		return
	}

	goals, err := h.goalRepo.ListByPeriod(r.Context(), periodID)
	if err != nil {
		slog.Error("failed to list goals", "period_id", periodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Filter by role
	var filteredGoals []*model.Goal
	for _, goal := range goals {
		if userRole == model.RoleRep {
			// Reps only see their own goals
			if goal.RepID != userID {
				continue
			}
		} else if userRole == model.RoleManager {
			// Managers see goals for their direct reports - not implemented here
			// For MVP, managers can see all goals in the period
		} else if userRole == model.RoleFinance || userRole == model.RoleAdmin {
			// Finance and admins see all goals
		}
		filteredGoals = append(filteredGoals, goal)
	}

	response := make([]GoalResponse, len(filteredGoals))
	for i, goal := range filteredGoals {
		response[i] = GoalResponse{
			ID:                goal.ID,
			RepID:             goal.RepID,
			PeriodID:          goal.PeriodID,
			AcquisitionTarget: goal.AcquisitionTarget,
			RenewalTarget:     goal.RenewalTarget,
			CommissionValue:   goal.CommissionValue,
			CreatedAt:         goal.CreatedAt.String(),
			UpdatedAt:         goal.UpdatedAt.String(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CreateGoal handles POST /api/goals
// Creates a goal (manager/admin only)
func (h *GoalHandler) CreateGoal(w http.ResponseWriter, r *http.Request) {
	userRole, ok := r.Context().Value("user_role").(model.UserRole)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Only managers and admins can create goals
	if userRole != model.RoleManager && userRole != model.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req CreateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RepID == 0 || req.PeriodID == 0 {
		http.Error(w, "rep_id and period_id are required", http.StatusBadRequest)
		return
	}

	// Check if period is open
	period, err := h.periodRepo.GetByID(r.Context(), req.PeriodID)
	if err != nil {
		slog.Error("failed to get period", "period_id", req.PeriodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if period == nil {
		http.Error(w, "Period not found", http.StatusNotFound)
		return
	}

	if period.Status != model.PeriodStatusOpen {
		http.Error(w, "Cannot create goal for closed period", http.StatusBadRequest)
		return
	}

	goal := &model.Goal{
		RepID:             req.RepID,
		PeriodID:          req.PeriodID,
		AcquisitionTarget: req.AcquisitionTarget,
		RenewalTarget:     req.RenewalTarget,
		CommissionValue:   req.CommissionValue,
	}

	if err := h.goalRepo.Create(r.Context(), goal); err != nil {
		slog.Error("failed to create goal", "rep_id", req.RepID, "period_id", req.PeriodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := GoalResponse{
		ID:                goal.ID,
		RepID:             goal.RepID,
		PeriodID:          goal.PeriodID,
		AcquisitionTarget: goal.AcquisitionTarget,
		RenewalTarget:     goal.RenewalTarget,
		CommissionValue:   goal.CommissionValue,
		CreatedAt:         goal.CreatedAt.String(),
		UpdatedAt:         goal.UpdatedAt.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// UpdateGoal handles PUT /api/goals/{id}
// Updates a goal (manager/admin only, only for open periods)
func (h *GoalHandler) UpdateGoal(w http.ResponseWriter, r *http.Request) {
	userRole, ok := r.Context().Value("user_role").(model.UserRole)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Only managers and admins can update goals
	if userRole != model.RoleManager && userRole != model.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	goalIDStr := r.PathValue("id")
	goalID, err := strconv.ParseInt(goalIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid goal ID", http.StatusBadRequest)
		return
	}

	var req UpdateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// For update, we'll just use a dummy Goal struct to pass to the repository
	// which will check the period status and update
	goal := &model.Goal{
		ID:                goalID,
		AcquisitionTarget: req.AcquisitionTarget,
		RenewalTarget:     req.RenewalTarget,
		CommissionValue:   req.CommissionValue,
	}

	// The Update method in repository checks if period is open
	if err := h.goalRepo.Update(r.Context(), goal); err != nil {
		slog.Error("failed to update goal", "goal_id", goalID, "error", err)

		// Check if error is due to closed period
		if strings.Contains(err.Error(), "closed period") {
			http.Error(w, "Cannot update goal for closed period", http.StatusBadRequest)
			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(goal)
}
