package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"comissionamento/internal/model"
	"comissionamento/internal/service"
)

type DashboardHandler struct {
	commissionService service.CommissionService
}

func NewDashboardHandler(commissionService service.CommissionService) *DashboardHandler {
	return &DashboardHandler{
		commissionService: commissionService,
	}
}

// GetRepDashboard handles GET /api/dashboard/rep
// Returns the current authenticated rep's dashboard
func (h *DashboardHandler) GetRepDashboard(w http.ResponseWriter, r *http.Request) {
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

	// Reps can only see their own dashboard
	if userRole == model.RoleRep {
		// OK - reps can access their own dashboard
	}

	dashboard, err := h.commissionService.GetRepDashboard(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get rep dashboard", "user_id", userID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dashboard)
}

// GetTeamDashboard handles GET /api/dashboard/team
// Returns team overview for a manager
func (h *DashboardHandler) GetTeamDashboard(w http.ResponseWriter, r *http.Request) {
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

	// Only managers and admins can see team dashboard
	if userRole != model.RoleManager && userRole != model.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	dashboard, err := h.commissionService.GetTeamDashboard(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get team dashboard", "manager_id", userID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dashboard)
}

// GetOrgDashboard handles GET /api/dashboard/org
// Returns organization-wide commission summary for finance users
func (h *DashboardHandler) GetOrgDashboard(w http.ResponseWriter, r *http.Request) {
	userRole, ok := r.Context().Value("user_role").(model.UserRole)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Only finance and admins can see org dashboard
	if userRole != model.RoleFinance && userRole != model.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	dashboard, err := h.commissionService.GetOrgDashboard(r.Context())
	if err != nil {
		slog.Error("failed to get org dashboard", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dashboard)
}
