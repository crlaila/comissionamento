package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"comissionamento/internal/model"
	"comissionamento/internal/service"
)

type ReportHandler struct {
	statementRepository service.StatementRepository
	periodRepository    service.PeriodRepository
	userRepository      service.UserRepository
	goalRepository      service.GoalRepository
	eventRepository     service.CommissionEventRepository
}

func NewReportHandler(
	statementRepository service.StatementRepository,
	periodRepository service.PeriodRepository,
	userRepository service.UserRepository,
	goalRepository service.GoalRepository,
	eventRepository service.CommissionEventRepository,
) *ReportHandler {
	return &ReportHandler{
		statementRepository: statementRepository,
		periodRepository:    periodRepository,
		userRepository:      userRepository,
		goalRepository:      goalRepository,
		eventRepository:     eventRepository,
	}
}

type CommissionDetailReport struct {
	RepID            int64   `json:"rep_id"`
	RepName          string  `json:"rep_name"`
	PeriodID         int64   `json:"period_id"`
	PeriodName       string  `json:"period_name"`
	AttainmentPct    float64 `json:"attainment_pct"`
	CommissionAmount int64   `json:"commission_amount"` // in centavos
	Status           string  `json:"status"`
}

// GetCommissionDetail handles GET /api/reports/commission-detail?rep_id=X&period_id=Y
func (h *ReportHandler) GetCommissionDetail(w http.ResponseWriter, r *http.Request) {
	userRole, ok := r.Context().Value("user_role").(model.UserRole)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	repIDStr := r.URL.Query().Get("rep_id")
	periodIDStr := r.URL.Query().Get("period_id")

	if repIDStr == "" || periodIDStr == "" {
		http.Error(w, "rep_id and period_id query parameters are required", http.StatusBadRequest)
		return
	}

	repID, err := strconv.ParseInt(repIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid rep_id", http.StatusBadRequest)
		return
	}

	periodID, err := strconv.ParseInt(periodIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid period_id", http.StatusBadRequest)
		return
	}

	// Check access: only finance/admin can access, or reps their own data
	if userRole != model.RoleFinance && userRole != model.RoleAdmin {
		if userRole == model.RoleRep {
			userID, ok := r.Context().Value("user_id").(int64)
			if !ok || userID != repID {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		} else {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	stmt, err := h.statementRepository.GetByRepAndPeriod(r.Context(), repID, periodID)
	if err != nil {
		slog.Error("failed to get statement", "rep_id", repID, "period_id", periodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if stmt == nil {
		http.Error(w, "Statement not found", http.StatusNotFound)
		return
	}

	rep, err := h.userRepository.GetByID(r.Context(), repID)
	if err != nil {
		slog.Error("failed to get rep", "rep_id", repID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	period, err := h.periodRepository.GetByID(r.Context(), periodID)
	if err != nil {
		slog.Error("failed to get period", "period_id", periodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	report := CommissionDetailReport{
		RepID:            repID,
		RepName:          rep.Name,
		PeriodID:         periodID,
		PeriodName:       period.Name,
		AttainmentPct:    stmt.AttainmentPct,
		CommissionAmount: stmt.TotalAmount,
		Status:           string(stmt.Status),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(report)
}

type TeamSummaryReport struct {
	PeriodID          int64                      `json:"period_id"`
	PeriodName        string                     `json:"period_name"`
	TotalCommission   int64                      `json:"total_commission"` // in centavos
	AverageAttainment float64                    `json:"average_attainment"`
	RepsCount         int                        `json:"reps_count"`
	RepSummaries      []TeamRepSummaryItem       `json:"rep_summaries"`
}

type TeamRepSummaryItem struct {
	RepID            int64   `json:"rep_id"`
	RepName          string  `json:"rep_name"`
	AttainmentPct    float64 `json:"attainment_pct"`
	CommissionAmount int64   `json:"commission_amount"` // in centavos
	Status           string  `json:"status"`
}

// GetTeamSummary handles GET /api/reports/team-summary?period_id=X
func (h *ReportHandler) GetTeamSummary(w http.ResponseWriter, r *http.Request) {
	userRole, ok := r.Context().Value("user_role").(model.UserRole)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if userRole != model.RoleFinance && userRole != model.RoleAdmin && userRole != model.RoleManager {
		http.Error(w, "Forbidden", http.StatusForbidden)
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

	period, err := h.periodRepository.GetByID(r.Context(), periodID)
	if err != nil {
		slog.Error("failed to get period", "period_id", periodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if period == nil {
		http.Error(w, "Period not found", http.StatusNotFound)
		return
	}

	statements, err := h.statementRepository.ListByPeriod(r.Context(), periodID)
	if err != nil {
		slog.Error("failed to list statements", "period_id", periodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// For managers, filter to their team
	if userRole == model.RoleManager {
		userID, ok := r.Context().Value("user_id").(int64)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var filtered []*model.Statement
		for _, stmt := range statements {
			repUser, err := h.userRepository.GetByID(r.Context(), stmt.RepID)
			if err == nil && repUser != nil && repUser.ManagerID != nil && *repUser.ManagerID == userID {
				filtered = append(filtered, stmt)
			}
		}
		statements = filtered
	}

	totalCommission := int64(0)
	totalAttainment := 0.0
	repSummaries := []TeamRepSummaryItem{}

	for _, stmt := range statements {
		rep, err := h.userRepository.GetByID(r.Context(), stmt.RepID)
		if err != nil {
			slog.Warn("failed to get rep", "rep_id", stmt.RepID)
			continue
		}

		repSummaries = append(repSummaries, TeamRepSummaryItem{
			RepID:            stmt.RepID,
			RepName:          rep.Name,
			AttainmentPct:    stmt.AttainmentPct,
			CommissionAmount: stmt.TotalAmount,
			Status:           string(stmt.Status),
		})

		totalCommission += stmt.TotalAmount
		totalAttainment += stmt.AttainmentPct
	}

	var avgAttainment float64
	if len(statements) > 0 {
		avgAttainment = totalAttainment / float64(len(statements))
	}

	report := TeamSummaryReport{
		PeriodID:          periodID,
		PeriodName:        period.Name,
		TotalCommission:   totalCommission,
		AverageAttainment: avgAttainment,
		RepsCount:         len(statements),
		RepSummaries:      repSummaries,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(report)
}

type LiabilityReport struct {
	PeriodID                int64 `json:"period_id"`
	PeriodName              string `json:"period_name"`
	TotalLiability          int64 `json:"total_liability"` // in centavos
	ApprovedLiability       int64 `json:"approved_liability"` // in centavos
	PendingApprovalLiability int64 `json:"pending_approval_liability"` // in centavos
}

// GetLiability handles GET /api/reports/liability?period_id=X
func (h *ReportHandler) GetLiability(w http.ResponseWriter, r *http.Request) {
	userRole, ok := r.Context().Value("user_role").(model.UserRole)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if userRole != model.RoleFinance && userRole != model.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
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

	period, err := h.periodRepository.GetByID(r.Context(), periodID)
	if err != nil {
		slog.Error("failed to get period", "period_id", periodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if period == nil {
		http.Error(w, "Period not found", http.StatusNotFound)
		return
	}

	statements, err := h.statementRepository.ListByPeriod(r.Context(), periodID)
	if err != nil {
		slog.Error("failed to list statements", "period_id", periodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	totalLiability := int64(0)
	approvedLiability := int64(0)
	pendingApprovalLiability := int64(0)

	for _, stmt := range statements {
		totalLiability += stmt.TotalAmount

		if stmt.Status == model.StatementStatusApproved || stmt.Status == model.StatementStatusPaid {
			approvedLiability += stmt.TotalAmount
		} else if stmt.Status == model.StatementStatusPendingApproval {
			pendingApprovalLiability += stmt.TotalAmount
		}
	}

	report := LiabilityReport{
		PeriodID:                 periodID,
		PeriodName:               period.Name,
		TotalLiability:           totalLiability,
		ApprovedLiability:        approvedLiability,
		PendingApprovalLiability: pendingApprovalLiability,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(report)
}

// GetExport handles GET /api/reports/export?type=X&period_id=Y
func (h *ReportHandler) GetExport(w http.ResponseWriter, r *http.Request) {
	userRole, ok := r.Context().Value("user_role").(model.UserRole)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if userRole != model.RoleFinance && userRole != model.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	exportType := r.URL.Query().Get("type")
	periodIDStr := r.URL.Query().Get("period_id")

	if exportType == "" || periodIDStr == "" {
		http.Error(w, "type and period_id query parameters are required", http.StatusBadRequest)
		return
	}

	periodID, err := strconv.ParseInt(periodIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid period_id", http.StatusBadRequest)
		return
	}

	period, err := h.periodRepository.GetByID(r.Context(), periodID)
	if err != nil {
		slog.Error("failed to get period", "period_id", periodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if period == nil {
		http.Error(w, "Period not found", http.StatusNotFound)
		return
	}

	statements, err := h.statementRepository.ListByPeriod(r.Context(), periodID)
	if err != nil {
		slog.Error("failed to list statements", "period_id", periodID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set CSV headers
	filename := fmt.Sprintf("commission_%s_%d.csv", strings.ToLower(exportType), time.Now().Unix())
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	switch exportType {
	case "commission-detail":
		writer.Write([]string{"Rep ID", "Rep Name", "Period", "Attainment %", "Commission (BRL)", "Status"})

		for _, stmt := range statements {
			rep, err := h.userRepository.GetByID(r.Context(), stmt.RepID)
			if err != nil {
				slog.Warn("failed to get rep", "rep_id", stmt.RepID)
				continue
			}

			commissionBRL := float64(stmt.TotalAmount) / 100.0
			writer.Write([]string{
				fmt.Sprintf("%d", stmt.RepID),
				rep.Name,
				period.Name,
				fmt.Sprintf("%.2f%%", stmt.AttainmentPct*100),
				fmt.Sprintf("R$ %.2f", commissionBRL),
				string(stmt.Status),
			})
		}

	case "team-summary":
		writer.Write([]string{"Period", "Total Commission (BRL)", "Average Attainment %", "Reps Count"})

		totalCommission := int64(0)
		totalAttainment := 0.0

		for _, stmt := range statements {
			totalCommission += stmt.TotalAmount
			totalAttainment += stmt.AttainmentPct
		}

		var avgAttainment float64
		if len(statements) > 0 {
			avgAttainment = totalAttainment / float64(len(statements))
		}

		commissionBRL := float64(totalCommission) / 100.0
		writer.Write([]string{
			period.Name,
			fmt.Sprintf("R$ %.2f", commissionBRL),
			fmt.Sprintf("%.2f%%", avgAttainment*100),
			fmt.Sprintf("%d", len(statements)),
		})

	case "liability":
		writer.Write([]string{"Period", "Total Liability (BRL)", "Approved Liability (BRL)", "Pending Approval Liability (BRL)"})

		totalLiability := int64(0)
		approvedLiability := int64(0)
		pendingApprovalLiability := int64(0)

		for _, stmt := range statements {
			totalLiability += stmt.TotalAmount

			if stmt.Status == model.StatementStatusApproved || stmt.Status == model.StatementStatusPaid {
				approvedLiability += stmt.TotalAmount
			} else if stmt.Status == model.StatementStatusPendingApproval {
				pendingApprovalLiability += stmt.TotalAmount
			}
		}

		totalLiabilityBRL := float64(totalLiability) / 100.0
		approvedLiabilityBRL := float64(approvedLiability) / 100.0
		pendingLiabilityBRL := float64(pendingApprovalLiability) / 100.0

		writer.Write([]string{
			period.Name,
			fmt.Sprintf("R$ %.2f", totalLiabilityBRL),
			fmt.Sprintf("R$ %.2f", approvedLiabilityBRL),
			fmt.Sprintf("R$ %.2f", pendingLiabilityBRL),
		})

	default:
		http.Error(w, "Invalid export type", http.StatusBadRequest)
		return
	}
}
