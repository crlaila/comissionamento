package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"comissionamento/internal/model"
)

func TestGetCommissionDetail_Success(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	// Setup test data
	period := &model.Period{
		ID:        1,
		Name:      "Jan 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
		CreatedAt: time.Now(),
	}
	periodRepo.periods[1] = period

	user := &model.User{
		ID:        1,
		Email:     "rep1@test.com",
		Name:      "Rep 1",
		Role:      model.RoleRep,
		Active:    true,
		CreatedAt: time.Now(),
	}
	userRepo.users[1] = user

	stmt := &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   50000,
		AttainmentPct: 0.95,
		Status:        model.StatementStatusApproved,
	}
	stmtRepo.statements[1] = stmt

	// Test as finance user
	req := httptest.NewRequest("GET", "/api/reports/commission-detail?rep_id=1&period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetCommissionDetail(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var report CommissionDetailReport
	json.NewDecoder(w.Body).Decode(&report)
	if report.CommissionAmount != 50000 || report.AttainmentPct != 0.95 {
		t.Errorf("unexpected report values")
	}
}

func TestGetCommissionDetail_RepAccessesOwn(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	period := &model.Period{
		ID:        1,
		Name:      "Jan 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
		CreatedAt: time.Now(),
	}
	periodRepo.periods[1] = period

	user := &model.User{
		ID:        1,
		Email:     "rep1@test.com",
		Name:      "Rep 1",
		Role:      model.RoleRep,
		Active:    true,
		CreatedAt: time.Now(),
	}
	userRepo.users[1] = user

	stmt := &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   50000,
		AttainmentPct: 0.95,
		Status:        model.StatementStatusApproved,
	}
	stmtRepo.statements[1] = stmt

	// Rep accessing their own data - should succeed
	req := httptest.NewRequest("GET", "/api/reports/commission-detail?rep_id=1&period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleRep,
	))

	w := httptest.NewRecorder()
	handler.GetCommissionDetail(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGetTeamSummary_Finance(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	period := &model.Period{
		ID:        1,
		Name:      "Jan 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
		CreatedAt: time.Now(),
	}
	periodRepo.periods[1] = period

	userRepo.users[1] = &model.User{
		ID:        1,
		Email:     "rep1@test.com",
		Name:      "Rep 1",
		Role:      model.RoleRep,
		Active:    true,
		CreatedAt: time.Now(),
	}

	userRepo.users[2] = &model.User{
		ID:        2,
		Email:     "rep2@test.com",
		Name:      "Rep 2",
		Role:      model.RoleRep,
		Active:    true,
		CreatedAt: time.Now(),
	}

	stmtRepo.statements[1] = &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   50000,
		AttainmentPct: 0.95,
		Status:        model.StatementStatusApproved,
	}

	stmtRepo.statements[2] = &model.Statement{
		ID:            2,
		RepID:         2,
		PeriodID:      1,
		TotalAmount:   60000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusApproved,
	}

	req := httptest.NewRequest("GET", "/api/reports/team-summary?period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetTeamSummary(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var report TeamSummaryReport
	json.NewDecoder(w.Body).Decode(&report)
	if report.TotalCommission != 110000 || report.RepsCount != 2 {
		t.Errorf("unexpected summary values: total=%d, count=%d", report.TotalCommission, report.RepsCount)
	}
}

func TestGetLiability_FinanceOnly(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	// Test non-finance user
	req := httptest.NewRequest("GET", "/api/reports/liability?period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleRep,
	))

	w := httptest.NewRecorder()
	handler.GetLiability(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestGetLiability_Calculations(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	period := &model.Period{
		ID:        1,
		Name:      "Jan 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
		CreatedAt: time.Now(),
	}
	periodRepo.periods[1] = period

	// Draft statement
	stmtRepo.statements[1] = &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   30000,
		AttainmentPct: 0.8,
		Status:        model.StatementStatusDraft,
	}

	// Pending approval statement
	stmtRepo.statements[2] = &model.Statement{
		ID:            2,
		RepID:         2,
		PeriodID:      1,
		TotalAmount:   40000,
		AttainmentPct: 0.9,
		Status:        model.StatementStatusPendingApproval,
	}

	// Approved statement
	stmtRepo.statements[3] = &model.Statement{
		ID:            3,
		RepID:         3,
		PeriodID:      1,
		TotalAmount:   50000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusApproved,
	}

	req := httptest.NewRequest("GET", "/api/reports/liability?period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetLiability(w, req)

	var report LiabilityReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.TotalLiability != 120000 {
		t.Errorf("expected total liability 120000, got %d", report.TotalLiability)
	}

	if report.ApprovedLiability != 50000 {
		t.Errorf("expected approved liability 50000, got %d", report.ApprovedLiability)
	}

	if report.PendingApprovalLiability != 40000 {
		t.Errorf("expected pending approval liability 40000, got %d", report.PendingApprovalLiability)
	}
}

func TestGetExport_CommissionDetail(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	period := &model.Period{
		ID:        1,
		Name:      "Jan 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
		CreatedAt: time.Now(),
	}
	periodRepo.periods[1] = period

	userRepo.users[1] = &model.User{
		ID:        1,
		Email:     "rep1@test.com",
		Name:      "Rep 1",
		Role:      model.RoleRep,
		Active:    true,
		CreatedAt: time.Now(),
	}

	stmtRepo.statements[1] = &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   50000,
		AttainmentPct: 0.95,
		Status:        model.StatementStatusApproved,
	}

	req := httptest.NewRequest("GET", "/api/reports/export?type=commission-detail&period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetExport(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify CSV content
	reader := csv.NewReader(strings.NewReader(w.Body.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Errorf("failed to read CSV: %v", err)
	}

	// Should have header + 1 data row
	if len(records) != 2 {
		t.Errorf("expected 2 rows (header + data), got %d", len(records))
	}

	// Check header
	expectedHeaders := []string{"Rep ID", "Rep Name", "Period", "Attainment %", "Commission (BRL)", "Status"}
	for i, header := range expectedHeaders {
		if records[0][i] != header {
			t.Errorf("unexpected header at index %d: %s", i, records[0][i])
		}
	}
}

func TestGetExport_RequiresFinance(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	req := httptest.NewRequest("GET", "/api/reports/export?type=commission-detail&period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleRep,
	))

	w := httptest.NewRecorder()
	handler.GetExport(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestGetExport_MissingParameters(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	req := httptest.NewRequest("GET", "/api/reports/export", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetExport(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetExport_TeamSummary(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	period := &model.Period{
		ID:        1,
		Name:      "Jan 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
		CreatedAt: time.Now(),
	}
	periodRepo.periods[1] = period

	stmtRepo.statements[1] = &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   50000,
		AttainmentPct: 0.95,
		Status:        model.StatementStatusApproved,
	}

	req := httptest.NewRequest("GET", "/api/reports/export?type=team-summary&period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetExport(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGetExport_Liability(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	period := &model.Period{
		ID:        1,
		Name:      "Jan 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
		CreatedAt: time.Now(),
	}
	periodRepo.periods[1] = period

	stmtRepo.statements[1] = &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   50000,
		AttainmentPct: 0.95,
		Status:        model.StatementStatusApproved,
	}

	req := httptest.NewRequest("GET", "/api/reports/export?type=liability&period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetExport(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGetCommissionDetail_StatemnetNotFound(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	req := httptest.NewRequest("GET", "/api/reports/commission-detail?rep_id=1&period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetCommissionDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGetCommissionDetail_InvalidQueryParams(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	// Missing rep_id
	req := httptest.NewRequest("GET", "/api/reports/commission-detail?period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetCommissionDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetCommissionDetail_InvalidRepID(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	req := httptest.NewRequest("GET", "/api/reports/commission-detail?rep_id=invalid&period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetCommissionDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetTeamSummary_MissingPeriodID(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	req := httptest.NewRequest("GET", "/api/reports/team-summary", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetTeamSummary(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetTeamSummary_RepCannotAccess(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	req := httptest.NewRequest("GET", "/api/reports/team-summary?period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleRep,
	))

	w := httptest.NewRecorder()
	handler.GetTeamSummary(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestGetLiability_MissingPeriodID(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	req := httptest.NewRequest("GET", "/api/reports/liability", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetLiability(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetExport_InvalidType(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	period := &model.Period{
		ID:        1,
		Name:      "Jan 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
		CreatedAt: time.Now(),
	}
	periodRepo.periods[1] = period

	req := httptest.NewRequest("GET", "/api/reports/export?type=invalid&period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetExport(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetTeamSummary_ManagerSeesOwnTeam(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	goalRepo := NewMockGoalRepository()
	eventRepo := NewMockEventRepository()

	handler := NewReportHandler(stmtRepo, periodRepo, userRepo, goalRepo, eventRepo)

	period := &model.Period{
		ID:        1,
		Name:      "Jan 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
		CreatedAt: time.Now(),
	}
	periodRepo.periods[1] = period

	managerID := int64(100)
	userRepo.users[1] = &model.User{
		ID:        1,
		Email:     "rep1@test.com",
		Name:      "Rep 1",
		Role:      model.RoleRep,
		ManagerID: &managerID,
		Active:    true,
		CreatedAt: time.Now(),
	}

	stmtRepo.statements[1] = &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   50000,
		AttainmentPct: 0.95,
		Status:        model.StatementStatusApproved,
	}

	req := httptest.NewRequest("GET", "/api/reports/team-summary?period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", managerID), "user_role", model.RoleManager,
	))

	w := httptest.NewRecorder()
	handler.GetTeamSummary(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var report TeamSummaryReport
	json.NewDecoder(w.Body).Decode(&report)
	if report.RepsCount != 1 {
		t.Errorf("expected 1 rep, got %d", report.RepsCount)
	}
}

