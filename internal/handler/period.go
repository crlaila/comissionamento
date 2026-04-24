package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"comissionamento/internal/model"
	"comissionamento/internal/repository"
)

type PeriodHandler struct {
	periodRepo *repository.PeriodRepository
}

func NewPeriodHandler(periodRepo *repository.PeriodRepository) *PeriodHandler {
	return &PeriodHandler{
		periodRepo: periodRepo,
	}
}

type CreatePeriodRequest struct {
	Name      string    `json:"name"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	Status    string    `json:"status"`
}

type UpdatePeriodStatusRequest struct {
	Status string `json:"status"`
}

// List handles GET /api/periods
func (h *PeriodHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	periods, err := h.periodRepo.List(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(periods)
}

// Create handles POST /api/periods
func (h *PeriodHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreatePeriodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse dates (format: YYYY-MM-DD)
	const dateFormat = "2006-01-02"
	startDate, err := time.Parse(dateFormat, req.StartDate)
	if err != nil {
		http.Error(w, "Invalid start_date format (use YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse(dateFormat, req.EndDate)
	if err != nil {
		http.Error(w, "Invalid end_date format (use YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	status := model.PeriodStatusOpen
	if req.Status != "" {
		status = model.PeriodStatus(req.Status)
	}

	period := &model.Period{
		Name:      req.Name,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    status,
	}

	if err := h.periodRepo.Create(r.Context(), period); err != nil {
		http.Error(w, "Failed to create period", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(period)
}

// Update handles PUT /api/periods/{id}
func (h *PeriodHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid request path", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(pathParts[3], 10, 64)
	if err != nil {
		http.Error(w, "Invalid period ID", http.StatusBadRequest)
		return
	}

	var req UpdatePeriodStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	status := model.PeriodStatus(req.Status)

	if err := h.periodRepo.UpdateStatus(r.Context(), id, status); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Period not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update period", http.StatusInternalServerError)
		}
		return
	}

	// Fetch and return updated period
	period, err := h.periodRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to fetch period", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(period)
}

