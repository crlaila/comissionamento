package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"comissionamento/internal/model"
)

// MockAuditRepository for testing
type MockAuditRepository struct {
	logs map[int64]*model.AuditLog
}

func (m *MockAuditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	if m.logs == nil {
		m.logs = make(map[int64]*model.AuditLog)
	}
	log.ID = int64(len(m.logs) + 1)
	log.CreatedAt = time.Now()
	m.logs[log.ID] = log
	return nil
}

func (m *MockAuditRepository) ListByEntity(ctx context.Context, entityType string, entityID int64) ([]*model.AuditLog, error) {
	var logs []*model.AuditLog
	for _, log := range m.logs {
		if log.EntityType == entityType && log.EntityID == entityID {
			logs = append(logs, log)
		}
	}
	return logs, nil
}

func (m *MockAuditRepository) ListByUser(ctx context.Context, userID int64) ([]*model.AuditLog, error) {
	var logs []*model.AuditLog
	for _, log := range m.logs {
		if log.UserID != nil && *log.UserID == userID {
			logs = append(logs, log)
		}
	}
	return logs, nil
}

type MockStatementRepository struct {
	statements map[int64]*model.Statement
}

func NewMockStatementRepository() *MockStatementRepository {
	return &MockStatementRepository{
		statements: make(map[int64]*model.Statement),
	}
}

func (m *MockStatementRepository) Create(ctx context.Context, statement *model.Statement) error {
	statement.ID = int64(len(m.statements) + 1)
	statement.CreatedAt = time.Now()
	statement.UpdatedAt = time.Now()
	m.statements[statement.ID] = statement
	return nil
}

func (m *MockStatementRepository) GetByID(ctx context.Context, id int64) (*model.Statement, error) {
	return m.statements[id], nil
}

func (m *MockStatementRepository) ListByPeriod(ctx context.Context, periodID int64) ([]*model.Statement, error) {
	var statements []*model.Statement
	for _, s := range m.statements {
		if s.PeriodID == periodID {
			statements = append(statements, s)
		}
	}
	return statements, nil
}

func (m *MockStatementRepository) GetByRepAndPeriod(ctx context.Context, repID, periodID int64) (*model.Statement, error) {
	for _, s := range m.statements {
		if s.RepID == repID && s.PeriodID == periodID {
			return s, nil
		}
	}
	return nil, nil
}

func (m *MockStatementRepository) UpdateStatus(ctx context.Context, id int64, status model.StatementStatus, approverID *int64, reason *string) error {
	if stmt, ok := m.statements[id]; ok {
		stmt.Status = status
		if status == model.StatementStatusApproved {
			now := time.Now()
			stmt.ApprovedAt = &now
			stmt.ApprovedBy = approverID
		}
		if reason != nil {
			stmt.RejectionReason = reason
		}
		stmt.UpdatedAt = time.Now()
	}
	return nil
}


func TestGetStatements_Finance(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	auditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	handler := NewStatementHandler(stmtRepo, commService, auditRepo, periodRepo, userRepo, nil)

	// Add test statements
	stmt1 := &model.Statement{
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   10000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusPendingApproval,
	}
	stmt1.ID = 1
	stmtRepo.statements[1] = stmt1

	req := httptest.NewRequest("GET", "/api/statements?period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetStatements(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var statements []*model.Statement
	json.NewDecoder(w.Body).Decode(&statements)
	if len(statements) != 1 {
		t.Errorf("expected 1 statement, got %d", len(statements))
	}
}

func TestGetStatement_RoleBasedVisibility(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	auditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	// Add test user
	userRepo.users[1] = &model.User{
		ID:        1,
		Email:     "rep1@test.com",
		Name:      "Rep 1",
		Role:      model.RoleRep,
		Active:    true,
		CreatedAt: time.Now(),
	}

	handler := NewStatementHandler(stmtRepo, commService, auditRepo, periodRepo, userRepo, nil)

	// Add test statement
	stmt := &model.Statement{
		ID:            1,
		RepID:         2,
		PeriodID:      1,
		TotalAmount:   10000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusPendingApproval,
	}
	stmtRepo.statements[1] = stmt

	// Test rep accessing another rep's statement - should be forbidden
	req := httptest.NewRequest("GET", "/api/statements/1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleRep,
	))
	req.SetPathValue("id", "1")

	w := httptest.NewRecorder()
	handler.GetStatement(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestApproveStatement_RequiresFinance(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	auditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	handler := NewStatementHandler(stmtRepo, commService, auditRepo, periodRepo, userRepo, nil)

	// Add test statement
	stmt := &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   10000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusPendingApproval,
	}
	stmtRepo.statements[1] = stmt

	// Test non-finance user trying to approve
	req := httptest.NewRequest("POST", "/api/statements/1/approve", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleRep,
	))
	req.SetPathValue("id", "1")

	w := httptest.NewRecorder()
	handler.ApproveStatement(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestApproveStatement_FailsNotPendingApproval(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	auditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	handler := NewStatementHandler(stmtRepo, commService, auditRepo, periodRepo, userRepo, nil)

	// Add test statement in draft status
	stmt := &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   10000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusDraft,
	}
	stmtRepo.statements[1] = stmt

	// Try to approve a draft statement
	req := httptest.NewRequest("POST", "/api/statements/1/approve", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))
	req.SetPathValue("id", "1")

	w := httptest.NewRecorder()
	handler.ApproveStatement(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestRejectStatement_RequiresReason(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	auditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	handler := NewStatementHandler(stmtRepo, commService, auditRepo, periodRepo, userRepo, nil)

	// Add test statement
	stmt := &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   10000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusPendingApproval,
	}
	stmtRepo.statements[1] = stmt

	// Try to reject without reason
	reqBody := RejectStatementRequest{Reason: ""}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/statements/1/reject", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))
	req.SetPathValue("id", "1")

	w := httptest.NewRecorder()
	handler.RejectStatement(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGenerateStatements_RequiresFinance(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	auditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	handler := NewStatementHandler(stmtRepo, commService, auditRepo, periodRepo, userRepo, nil)

	// Try to generate as non-finance user
	reqBody := GenerateStatementsRequest{PeriodID: 1}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/statements/generate", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleRep,
	))

	w := httptest.NewRecorder()
	handler.GenerateStatements(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestGetStatements_RepsCanOnlySeeOwn(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	auditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	handler := NewStatementHandler(stmtRepo, commService, auditRepo, periodRepo, userRepo, nil)

	// Add statements from different reps
	stmt1 := &model.Statement{
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   10000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusPendingApproval,
	}
	stmt1.ID = 1
	stmtRepo.statements[1] = stmt1

	stmt2 := &model.Statement{
		RepID:         2,
		PeriodID:      1,
		TotalAmount:   15000,
		AttainmentPct: 0.8,
		Status:        model.StatementStatusPendingApproval,
	}
	stmt2.ID = 2
	stmtRepo.statements[2] = stmt2

	// Rep 1 requests statements - should only see their own
	req := httptest.NewRequest("GET", "/api/statements?period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleRep,
	))

	w := httptest.NewRecorder()
	handler.GetStatements(w, req)

	var statements []*model.Statement
	json.NewDecoder(w.Body).Decode(&statements)
	if len(statements) != 1 || statements[0].RepID != 1 {
		t.Errorf("rep should only see their own statement")
	}
}

func TestGetStatements_MissingPeriodID(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	auditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	handler := NewStatementHandler(stmtRepo, commService, auditRepo, periodRepo, userRepo, nil)

	req := httptest.NewRequest("GET", "/api/statements", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GetStatements(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetStatement_NotFound(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	auditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	handler := NewStatementHandler(stmtRepo, commService, auditRepo, periodRepo, userRepo, nil)

	req := httptest.NewRequest("GET", "/api/statements/999", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))
	req.SetPathValue("id", "999")

	w := httptest.NewRecorder()
	handler.GetStatement(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGetStatement_InvalidID(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	auditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	handler := NewStatementHandler(stmtRepo, commService, auditRepo, periodRepo, userRepo, nil)

	req := httptest.NewRequest("GET", "/api/statements/invalid", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))
	req.SetPathValue("id", "invalid")

	w := httptest.NewRecorder()
	handler.GetStatement(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetStatement_FinanceCanSeeAll(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	auditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	userRepo.users[2] = &model.User{
		ID:        2,
		Email:     "rep2@test.com",
		Name:      "Rep 2",
		Role:      model.RoleRep,
		Active:    true,
		CreatedAt: time.Now(),
	}

	handler := NewStatementHandler(stmtRepo, commService, auditRepo, periodRepo, userRepo, nil)

	stmt := &model.Statement{
		ID:            1,
		RepID:         2,
		PeriodID:      1,
		TotalAmount:   10000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusPendingApproval,
	}
	stmtRepo.statements[1] = stmt

	// Finance user can see any rep's statement
	req := httptest.NewRequest("GET", "/api/statements/1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))
	req.SetPathValue("id", "1")

	w := httptest.NewRecorder()
	handler.GetStatement(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestApproveStatement_Success(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	// Create a mock audit repository wrapper that doesn't require a pool
	mockAuditRepo := &MockAuditRepository{}

	handler := NewStatementHandler(stmtRepo, commService, mockAuditRepo, periodRepo, userRepo, nil)

	stmt := &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   10000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusPendingApproval,
	}
	stmtRepo.statements[1] = stmt

	req := httptest.NewRequest("POST", "/api/statements/1/approve", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))
	req.SetPathValue("id", "1")

	w := httptest.NewRecorder()
	handler.ApproveStatement(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRejectStatement_Success(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	mockAuditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	handler := NewStatementHandler(stmtRepo, commService, mockAuditRepo, periodRepo, userRepo, nil)

	stmt := &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   10000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusPendingApproval,
	}
	stmtRepo.statements[1] = stmt

	reqBody := RejectStatementRequest{Reason: "Commission calculation error"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/statements/1/reject", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))
	req.SetPathValue("id", "1")

	w := httptest.NewRecorder()
	handler.RejectStatement(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGenerateStatements_Success(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	mockAuditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	handler := NewStatementHandler(stmtRepo, commService, mockAuditRepo, periodRepo, userRepo, nil)

	reqBody := GenerateStatementsRequest{PeriodID: 1}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/statements/generate", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GenerateStatements(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGenerateStatements_MissingPeriodID(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	mockAuditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

	handler := NewStatementHandler(stmtRepo, commService, mockAuditRepo, periodRepo, userRepo, nil)

	reqBody := GenerateStatementsRequest{PeriodID: 0}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/statements/generate", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", int64(1)), "user_role", model.RoleFinance,
	))

	w := httptest.NewRecorder()
	handler.GenerateStatements(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetStatements_ManagerSeesTeam(t *testing.T) {
	stmtRepo := NewMockStatementRepository()
	commService := &MockCommissionService{}
	auditRepo := &MockAuditRepository{}
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()

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

	handler := NewStatementHandler(stmtRepo, commService, auditRepo, periodRepo, userRepo, nil)

	stmt := &model.Statement{
		ID:            1,
		RepID:         1,
		PeriodID:      1,
		TotalAmount:   10000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusPendingApproval,
	}
	stmtRepo.statements[1] = stmt

	req := httptest.NewRequest("GET", "/api/statements?period_id=1", nil)
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "user_id", managerID), "user_role", model.RoleManager,
	))

	w := httptest.NewRecorder()
	handler.GetStatements(w, req)

	var statements []*model.Statement
	json.NewDecoder(w.Body).Decode(&statements)
	if len(statements) != 1 {
		t.Errorf("manager should see their team's statements")
	}
}
