package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"comissionamento/internal/model"
	"comissionamento/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditLogRepository interface for dependency injection
type AuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
}

type StatementHandler struct {
	statementRepository service.StatementRepository
	commissionService   service.CommissionService
	auditRepository     AuditLogRepository
	periodRepository    service.PeriodRepository
	userRepository      service.UserRepository
	pool                *pgxpool.Pool
}

func NewStatementHandler(
	statementRepository service.StatementRepository,
	commissionService service.CommissionService,
	auditRepository AuditLogRepository,
	periodRepository service.PeriodRepository,
	userRepository service.UserRepository,
	pool *pgxpool.Pool,
) *StatementHandler {
	return &StatementHandler{
		statementRepository: statementRepository,
		commissionService:   commissionService,
		auditRepository:     auditRepository,
		periodRepository:    periodRepository,
		userRepository:      userRepository,
		pool:                pool,
	}
}

// GetStatements handles GET /api/statements?period_id=X
// Lists statements with role-based visibility
func (h *StatementHandler) GetStatements(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "period_id query parameter is required", http.StatusBadRequest)
		return
	}

	periodID, err := strconv.ParseInt(periodIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid period_id", http.StatusBadRequest)
		return
	}

	statements, err := h.statementRepository.ListByPeriod(r.Context(), periodID)
	if err != nil {
		slog.Error("failed to list statements", "period_id", periodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Filter by role-based visibility
	var filtered []*model.Statement
	for _, stmt := range statements {
		// Finance and admin can see all statements
		if userRole == model.RoleFinance || userRole == model.RoleAdmin {
			filtered = append(filtered, stmt)
			continue
		}

		// Reps can only see their own statements
		if userRole == model.RoleRep && stmt.RepID == userID {
			filtered = append(filtered, stmt)
			continue
		}

		// Managers can see their team's statements
		if userRole == model.RoleManager {
			repUser, err := h.userRepository.GetByID(r.Context(), stmt.RepID)
			if err == nil && repUser != nil && repUser.ManagerID != nil && *repUser.ManagerID == userID {
				filtered = append(filtered, stmt)
			}
		}
	}

	if filtered == nil {
		filtered = []*model.Statement{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(filtered)
}

// GetStatement handles GET /api/statements/{id}
// Returns statement detail with calculation breakdown
func (h *StatementHandler) GetStatement(w http.ResponseWriter, r *http.Request) {
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

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid statement id", http.StatusBadRequest)
		return
	}

	stmt, err := h.statementRepository.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get statement", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if stmt == nil {
		http.Error(w, "Statement not found", http.StatusNotFound)
		return
	}

	// Check role-based visibility
	canView := userRole == model.RoleFinance || userRole == model.RoleAdmin

	if !canView && userRole == model.RoleRep && stmt.RepID == userID {
		canView = true
	}

	if !canView && userRole == model.RoleManager {
		repUser, err := h.userRepository.GetByID(r.Context(), stmt.RepID)
		if err == nil && repUser != nil && repUser.ManagerID != nil && *repUser.ManagerID == userID {
			canView = true
		}
	}

	if !canView {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stmt)
}

type GenerateStatementsRequest struct {
	PeriodID int64 `json:"period_id"`
}

// GenerateStatements handles POST /api/statements/generate
// Creates statements for a period (finance/admin only)
func (h *StatementHandler) GenerateStatements(w http.ResponseWriter, r *http.Request) {
	userRole, ok := r.Context().Value("user_role").(model.UserRole)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if userRole != model.RoleFinance && userRole != model.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req GenerateStatementsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PeriodID == 0 {
		http.Error(w, "period_id is required", http.StatusBadRequest)
		return
	}

	if err := h.commissionService.GenerateStatements(r.Context(), req.PeriodID); err != nil {
		slog.Error("failed to generate statements", "period_id", req.PeriodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"Statements generated for period %d"}`, req.PeriodID)
}

type ApproveStatementRequest struct {
}

// ApproveStatement handles POST /api/statements/{id}/approve
// Approves a statement (finance only)
func (h *StatementHandler) ApproveStatement(w http.ResponseWriter, r *http.Request) {
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

	if userRole != model.RoleFinance && userRole != model.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid statement id", http.StatusBadRequest)
		return
	}

	stmt, err := h.statementRepository.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get statement", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if stmt == nil {
		http.Error(w, "Statement not found", http.StatusNotFound)
		return
	}

	if stmt.Status != model.StatementStatusPendingApproval {
		http.Error(w, "Statement is not in pending_approval status", http.StatusBadRequest)
		return
	}

	if err := h.commissionService.ApproveStatement(r.Context(), id, userID); err != nil {
		slog.Error("failed to approve statement", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create audit log
	auditLog := &model.AuditLog{
		UserID:     &userID,
		Action:     "approve",
		EntityType: "statement",
		EntityID:   id,
		Details:    nil,
	}
	if err := h.auditRepository.Create(r.Context(), auditLog); err != nil {
		slog.Error("failed to create audit log", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"Statement %d approved"}`, id)
}

type RejectStatementRequest struct {
	Reason string `json:"reason"`
}

// RejectStatement handles POST /api/statements/{id}/reject
// Rejects a statement with reason (finance only)
func (h *StatementHandler) RejectStatement(w http.ResponseWriter, r *http.Request) {
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

	if userRole != model.RoleFinance && userRole != model.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid statement id", http.StatusBadRequest)
		return
	}

	var req RejectStatementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Reason == "" {
		http.Error(w, "reason field is required", http.StatusBadRequest)
		return
	}

	stmt, err := h.statementRepository.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get statement", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if stmt == nil {
		http.Error(w, "Statement not found", http.StatusNotFound)
		return
	}

	if stmt.Status != model.StatementStatusPendingApproval {
		http.Error(w, "Statement is not in pending_approval status", http.StatusBadRequest)
		return
	}

	if err := h.statementRepository.UpdateStatus(r.Context(), id, model.StatementStatusDraft, nil, &req.Reason); err != nil {
		slog.Error("failed to reject statement", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create audit log
	auditLog := &model.AuditLog{
		UserID:     &userID,
		Action:     "reject",
		EntityType: "statement",
		EntityID:   id,
		Details:    nil,
	}
	if err := h.auditRepository.Create(r.Context(), auditLog); err != nil {
		slog.Error("failed to create audit log", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"Statement %d rejected"}`, id)
}
