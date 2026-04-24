package service

import (
	"context"
	"testing"
	"time"

	"comissionamento/internal/model"
)

// Mock repositories for testing

type MockGoalRepository struct {
	goals map[int64][]*model.Goal // periodID -> goals
}

func NewMockGoalRepository() *MockGoalRepository {
	return &MockGoalRepository{
		goals: make(map[int64][]*model.Goal),
	}
}

func (m *MockGoalRepository) GetByRepAndPeriod(ctx context.Context, repID, periodID int64) (*model.Goal, error) {
	for _, goal := range m.goals[periodID] {
		if goal.RepID == repID {
			return goal, nil
		}
	}
	return nil, nil
}

func (m *MockGoalRepository) ListByPeriod(ctx context.Context, periodID int64) ([]*model.Goal, error) {
	return m.goals[periodID], nil
}

func (m *MockGoalRepository) AddGoal(goal *model.Goal) {
	m.goals[goal.PeriodID] = append(m.goals[goal.PeriodID], goal)
}

type MockPeriodRepository struct {
	periods    map[int64]*model.Period
	periodList []*model.Period // for List() method
}

func NewMockPeriodRepository() *MockPeriodRepository {
	return &MockPeriodRepository{
		periods:    make(map[int64]*model.Period),
		periodList: []*model.Period{},
	}
}

func (m *MockPeriodRepository) GetByID(ctx context.Context, id int64) (*model.Period, error) {
	return m.periods[id], nil
}

func (m *MockPeriodRepository) List(ctx context.Context) ([]*model.Period, error) {
	return m.periodList, nil
}

func (m *MockPeriodRepository) AddPeriod(period *model.Period) {
	m.periods[period.ID] = period
	m.periodList = append(m.periodList, period)
}

type MockStatementRepository struct {
	statements map[int64]*model.Statement // periodID -> repID -> statement
	nextID     int64
}

func NewMockStatementRepository() *MockStatementRepository {
	return &MockStatementRepository{
		statements: make(map[int64]*model.Statement),
		nextID:     1,
	}
}

func (m *MockStatementRepository) Create(ctx context.Context, statement *model.Statement) error {
	statement.ID = m.nextID
	statement.CreatedAt = time.Now()
	statement.UpdatedAt = time.Now()
	key := statement.PeriodID*10000 + statement.RepID
	m.statements[key] = statement
	m.nextID++
	return nil
}

func (m *MockStatementRepository) GetByRepAndPeriod(ctx context.Context, repID, periodID int64) (*model.Statement, error) {
	key := periodID*10000 + repID
	return m.statements[key], nil
}

func (m *MockStatementRepository) GetByID(ctx context.Context, id int64) (*model.Statement, error) {
	for _, stmt := range m.statements {
		if stmt.ID == id {
			return stmt, nil
		}
	}
	return nil, nil
}

func (m *MockStatementRepository) ListByPeriod(ctx context.Context, periodID int64) ([]*model.Statement, error) {
	var stmts []*model.Statement
	for _, stmt := range m.statements {
		if stmt.PeriodID == periodID {
			stmts = append(stmts, stmt)
		}
	}
	return stmts, nil
}

func (m *MockStatementRepository) UpdateStatus(ctx context.Context, id int64, status model.StatementStatus, approverID *int64, reason *string) error {
	for _, stmt := range m.statements {
		if stmt.ID == id {
			stmt.Status = status
			stmt.ApprovedBy = approverID
			if status == model.StatementStatusApproved {
				now := time.Now()
				stmt.ApprovedAt = &now
			}
			stmt.RejectionReason = reason
			return nil
		}
	}
	return nil
}

type MockCommissionEventRepository struct {
	events map[int64][]*model.MemberEvent // repID -> events
}

func NewMockCommissionEventRepository() *MockCommissionEventRepository {
	return &MockCommissionEventRepository{
		events: make(map[int64][]*model.MemberEvent),
	}
}

func (m *MockCommissionEventRepository) CountByRepAndType(ctx context.Context, repID int64, eventType model.EventType, startDate, endDate interface{}) (int, error) {
	count := 0
	for _, event := range m.events[repID] {
		if event.EventType == eventType {
			count++
		}
	}
	return count, nil
}

func (m *MockCommissionEventRepository) ListByRepAndPeriod(ctx context.Context, repID int64, startDate, endDate interface{}) ([]*model.MemberEvent, error) {
	return m.events[repID], nil
}

func (m *MockCommissionEventRepository) AddEvent(event *model.MemberEvent) {
	m.events[event.RepID] = append(m.events[event.RepID], event)
}

// Helper function to add a user directly to MockUserRepository
func addUserToMock(m *MockUserRepository, user *model.User) int64 {
	user.ID = m.nextID
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[m.nextID] = user
	m.nextID++
	return user.ID
}

// Test cases

func TestCalculateAttainment(t *testing.T) {
	tests := []struct {
		name      string
		acqActual float64
		acqGoal   float64
		renActual float64
		renGoal   float64
		expected  float64
	}{
		{
			name:      "0% attainment",
			acqActual: 0,
			acqGoal:   10,
			renActual: 0,
			renGoal:   10,
			expected:  0.0,
		},
		{
			name:      "50% attainment",
			acqActual: 5,
			acqGoal:   10,
			renActual: 5,
			renGoal:   10,
			expected:  0.5,
		},
		{
			name:      "100% attainment",
			acqActual: 10,
			acqGoal:   10,
			renActual: 10,
			renGoal:   10,
			expected:  1.0,
		},
		{
			name:      "over 100% attainment (capped)",
			acqActual: 20,
			acqGoal:   10,
			renActual: 20,
			renGoal:   10,
			expected:  1.0,
		},
		{
			name:      "mixed attainment (80% acq + 60% renew = 70%)",
			acqActual: 8,
			acqGoal:   10,
			renActual: 6,
			renGoal:   10,
			expected:  0.7,
		},
		{
			name:      "zero goals",
			acqActual: 5,
			acqGoal:   0,
			renActual: 5,
			renGoal:   0,
			expected:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAttainment(tt.acqActual, tt.acqGoal, tt.renActual, tt.renGoal)
			if result != tt.expected {
				t.Errorf("expected %.2f, got %.2f", tt.expected, result)
			}
		})
	}
}

func TestCalculateCommission(t *testing.T) {
	tests := []struct {
		name             string
		attainmentPct    float64
		commissionValue  int64
		expected         int64
	}{
		{
			name:            "0% attainment = 0 commission",
			attainmentPct:   0.0,
			commissionValue: 100000,
			expected:        0,
		},
		{
			name:            "50% attainment = 50% commission",
			attainmentPct:   0.5,
			commissionValue: 100000,
			expected:        50000,
		},
		{
			name:            "100% attainment = 100% commission",
			attainmentPct:   1.0,
			commissionValue: 100000,
			expected:        100000,
		},
		{
			name:            "75% attainment",
			attainmentPct:   0.75,
			commissionValue: 100000,
			expected:        75000,
		},
		{
			name:            "commission in centavos without rounding errors",
			attainmentPct:   0.33,
			commissionValue: 10000,
			expected:        3300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateCommission(tt.attainmentPct, tt.commissionValue)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetRepDashboard(t *testing.T) {
	ctx := context.Background()

	// Setup mock repositories
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	// Add test data
	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	periodRepo.AddPeriod(period)

	goal := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000, // 1000 BRL in centavos
	}
	goalRepo.AddGoal(goal)

	// Add events: 5 acquisitions and 5 renewals (50% attainment)
	for i := 0; i < 5; i++ {
		eventRepo.AddEvent(&model.MemberEvent{
			RepID:     100,
			EventType: model.EventTypeAcquisition,
		})
		eventRepo.AddEvent(&model.MemberEvent{
			RepID:     100,
			EventType: model.EventTypeRenewal,
		})
	}

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	dashboard, err := service.GetRepDashboard(ctx, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dashboard.RepID != 100 {
		t.Errorf("expected rep_id 100, got %d", dashboard.RepID)
	}

	if dashboard.AttainmentPct != 0.5 {
		t.Errorf("expected 50%% attainment, got %.2f", dashboard.AttainmentPct)
	}

	if dashboard.CommissionEarned != 50000 {
		t.Errorf("expected 50000 centavos earned, got %d", dashboard.CommissionEarned)
	}

	if dashboard.AcquisitionActual != 5 {
		t.Errorf("expected 5 acquisitions, got %d", dashboard.AcquisitionActual)
	}

	if dashboard.RenewalActual != 5 {
		t.Errorf("expected 5 renewals, got %d", dashboard.RenewalActual)
	}
}

func TestGetRepDashboardZeroAttainment(t *testing.T) {
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	periodRepo.AddPeriod(period)

	goal := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goalRepo.AddGoal(goal)

	// No events = 0% attainment
	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	dashboard, err := service.GetRepDashboard(ctx, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dashboard.AttainmentPct != 0.0 {
		t.Errorf("expected 0%% attainment, got %.2f", dashboard.AttainmentPct)
	}

	if dashboard.CommissionEarned != 0 {
		t.Errorf("expected 0 commission for 0%% attainment, got %d", dashboard.CommissionEarned)
	}
}

func TestGetTeamDashboard(t *testing.T) {
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	// Add period
	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	periodRepo.AddPeriod(period)

	// Add manager and direct reports
	managerID := int64(10)
	manager := &model.User{
		ID:   managerID,
		Name: "John Manager",
		Role: model.RoleManager,
	}
	userRepo.users[managerID] = manager

	rep1 := &model.User{
		ID:        100,
		Name:      "Rep 1",
		Role:      model.RoleRep,
		ManagerID: &managerID,
	}
	rep2 := &model.User{
		ID:        101,
		Name:      "Rep 2",
		Role:      model.RoleRep,
		ManagerID: &managerID,
	}
	userRepo.users[100] = rep1
	userRepo.users[101] = rep2

	// Add goals for both reps
	goal1 := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goal2 := &model.Goal{
		ID:                2,
		RepID:             101,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goalRepo.AddGoal(goal1)
	goalRepo.AddGoal(goal2)

	// Add events for rep1 (100% attainment)
	for i := 0; i < 10; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeAcquisition})
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeRenewal})
	}

	// Add events for rep2 (50% attainment)
	for i := 0; i < 5; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 101, EventType: model.EventTypeAcquisition})
		eventRepo.AddEvent(&model.MemberEvent{RepID: 101, EventType: model.EventTypeRenewal})
	}

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	dashboard, err := service.GetTeamDashboard(ctx, managerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dashboard.ManagerID != managerID {
		t.Errorf("expected manager_id %d, got %d", managerID, dashboard.ManagerID)
	}

	if len(dashboard.DirectReports) != 2 {
		t.Errorf("expected 2 direct reports, got %d", len(dashboard.DirectReports))
	}

	// Total commission should be 100000 + 50000 = 150000
	if dashboard.TotalCommission != 150000 {
		t.Errorf("expected total commission 150000, got %d", dashboard.TotalCommission)
	}
}

func TestGenerateStatements(t *testing.T) {
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusClosed,
	}
	periodRepo.AddPeriod(period)

	goal := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goalRepo.AddGoal(goal)

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	err := service.GenerateStatements(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that statement was created
	stmt, err := statementRepo.GetByRepAndPeriod(ctx, 100, 1)
	if err != nil {
		t.Fatalf("failed to get statement: %v", err)
	}

	if stmt == nil {
		t.Fatal("statement should have been created")
	}

	if stmt.Status != model.StatementStatusPendingApproval {
		t.Errorf("expected status pending_approval, got %s", stmt.Status)
	}

	if stmt.TotalAmount != 0 {
		t.Errorf("expected 0 commission (no events), got %d", stmt.TotalAmount)
	}
}

func TestApproveStatement(t *testing.T) {
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	// Create a statement
	stmt := &model.Statement{
		RepID:         100,
		PeriodID:      1,
		TotalAmount:   50000,
		AttainmentPct: 0.5,
		Status:        model.StatementStatusPendingApproval,
	}
	statementRepo.Create(ctx, stmt)

	approverID := int64(999)
	err := service.ApproveStatement(ctx, stmt.ID, approverID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that statement was approved
	approved, _ := statementRepo.GetByRepAndPeriod(ctx, 100, 1)
	if approved.Status != model.StatementStatusApproved {
		t.Errorf("expected status approved, got %s", approved.Status)
	}

	if approved.ApprovedBy == nil || *approved.ApprovedBy != approverID {
		t.Errorf("expected approved_by %d", approverID)
	}
}

func TestCalculateForPeriod(t *testing.T) {
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusClosed,
	}
	periodRepo.AddPeriod(period)

	// Add multiple goals
	goal1 := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goal2 := &model.Goal{
		ID:                2,
		RepID:             101,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goalRepo.AddGoal(goal1)
	goalRepo.AddGoal(goal2)

	// Add events for rep1
	for i := 0; i < 10; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeAcquisition})
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeRenewal})
	}

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	err := service.CalculateForPeriod(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that statements were created for both reps
	stmt1, _ := statementRepo.GetByRepAndPeriod(ctx, 100, 1)
	if stmt1 == nil {
		t.Fatal("statement for rep 100 should have been created")
	}

	// Rep 100 has 100% attainment, so should get full commission
	if stmt1.TotalAmount != 100000 {
		t.Errorf("expected 100000 centavos, got %d", stmt1.TotalAmount)
	}

	if stmt1.AttainmentPct != 1.0 {
		t.Errorf("expected 100%% attainment, got %.2f", stmt1.AttainmentPct)
	}

	// Rep 101 has 0% attainment
	stmt2, _ := statementRepo.GetByRepAndPeriod(ctx, 101, 1)
	if stmt2 == nil {
		t.Fatal("statement for rep 101 should have been created")
	}

	if stmt2.TotalAmount != 0 {
		t.Errorf("expected 0 centavos, got %d", stmt2.TotalAmount)
	}
}

func TestRepDashboardWithPendingCommission(t *testing.T) {
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	periodRepo.AddPeriod(period)

	goal := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goalRepo.AddGoal(goal)

	// Add 100% attainment
	for i := 0; i < 10; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeAcquisition})
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeRenewal})
	}

	// Pre-create a statement (pending approval)
	stmt := &model.Statement{
		RepID:         100,
		PeriodID:      1,
		TotalAmount:   100000,
		AttainmentPct: 1.0,
		Status:        model.StatementStatusPendingApproval,
	}
	statementRepo.Create(ctx, stmt)

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	dashboard, err := service.GetRepDashboard(ctx, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Commission earned should be 100000 (100% of 100000)
	if dashboard.CommissionEarned != 100000 {
		t.Errorf("expected 100000 earned, got %d", dashboard.CommissionEarned)
	}

	// Commission pending should also be 100000 (from statement)
	if dashboard.CommissionPending != 100000 {
		t.Errorf("expected 100000 pending, got %d", dashboard.CommissionPending)
	}
}

func TestTeamDashboardNoDirectReports(t *testing.T) {
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	periodRepo.AddPeriod(period)

	// Manager with no direct reports
	managerID := int64(10)
	manager := &model.User{
		ID:   managerID,
		Name: "John Manager",
		Role: model.RoleManager,
	}
	userRepo.users[managerID] = manager

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	dashboard, err := service.GetTeamDashboard(ctx, managerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dashboard.DirectReports) != 0 {
		t.Errorf("expected 0 direct reports, got %d", len(dashboard.DirectReports))
	}

	if dashboard.TotalCommission != 0 {
		t.Errorf("expected 0 total commission, got %d", dashboard.TotalCommission)
	}
}

func TestOverAttainment(t *testing.T) {
	// Test that over-attainment is capped at 100% for commission
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	periodRepo.AddPeriod(period)

	goal := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goalRepo.AddGoal(goal)

	// Add 200% events (way over goal)
	for i := 0; i < 20; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeAcquisition})
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeRenewal})
	}

	ctx := context.Background()
	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	dashboard, err := service.GetRepDashboard(ctx, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Commission should be capped at 100% of commission value
	if dashboard.CommissionEarned > 100000 {
		t.Errorf("over-attainment commission should be capped at 100000, got %d", dashboard.CommissionEarned)
	}

	// Attainment % shown should be capped at 100%
	if dashboard.AttainmentPct > 1.0 {
		t.Errorf("attainment pct should be capped at 1.0, got %.2f", dashboard.AttainmentPct)
	}
}

func TestPartialAcquisitionAttainment(t *testing.T) {
	// Test mixed attainment: acquisition partial, renewal complete
	ctx := context.Background()
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	periodRepo.AddPeriod(period)

	goal := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goalRepo.AddGoal(goal)

	// 5 acquisitions (50%) + 10 renewals (100%) = 75%
	for i := 0; i < 5; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeAcquisition})
	}
	for i := 0; i < 10; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeRenewal})
	}

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	dashboard, err := service.GetRepDashboard(ctx, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dashboard.AttainmentPct != 0.75 {
		t.Errorf("expected 75%% attainment, got %.2f", dashboard.AttainmentPct)
	}

	// Commission should be 75000 (75% of 100000)
	if dashboard.CommissionEarned != 75000 {
		t.Errorf("expected 75000 commission, got %d", dashboard.CommissionEarned)
	}
}

func TestGenerateStatementsWithExistingStatements(t *testing.T) {
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusClosed,
	}
	periodRepo.AddPeriod(period)

	goal := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goalRepo.AddGoal(goal)

	// Pre-create a statement
	existingStmt := &model.Statement{
		RepID:         100,
		PeriodID:      1,
		TotalAmount:   50000,
		AttainmentPct: 0.5,
		Status:        model.StatementStatusApproved,
	}
	statementRepo.Create(ctx, existingStmt)

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	// GenerateStatements should skip existing statements
	err := service.GenerateStatements(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify statement still exists with old values
	stmt, _ := statementRepo.GetByRepAndPeriod(ctx, 100, 1)
	if stmt.TotalAmount != 50000 {
		t.Errorf("existing statement should not be overwritten, got %d", stmt.TotalAmount)
	}
	if stmt.Status != model.StatementStatusApproved {
		t.Errorf("existing statement status should not be changed, got %s", stmt.Status)
	}
}

func TestCalculateForPeriodNoGoals(t *testing.T) {
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusClosed,
	}
	periodRepo.AddPeriod(period)

	// No goals for this period
	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	err := service.CalculateForPeriod(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should complete without creating any statements
	stmts, _ := statementRepo.ListByPeriod(ctx, 1)
	if len(stmts) != 0 {
		t.Errorf("expected no statements, got %d", len(stmts))
	}
}

func TestGetRepDashboardNilRecentEvents(t *testing.T) {
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()

	// EventRepository that returns nil instead of empty slice
	eventRepo := &MockCommissionEventRepository{
		events: make(map[int64][]*model.MemberEvent),
	}

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	periodRepo.AddPeriod(period)

	goal := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goalRepo.AddGoal(goal)

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	dashboard, err := service.GetRepDashboard(ctx, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// RecentEvents should not be nil for JSON serialization
	if dashboard.RecentEvents == nil {
		t.Fatal("RecentEvents should not be nil")
	}

	if len(dashboard.RecentEvents) != 0 {
		t.Errorf("expected 0 recent events, got %d", len(dashboard.RecentEvents))
	}
}

func TestPartialRenewalAttainment(t *testing.T) {
	// Test with only partial renewal attainment
	ctx := context.Background()
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	periodRepo.AddPeriod(period)

	goal := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goalRepo.AddGoal(goal)

	// 0 acquisitions (0%) + 3 renewals (30%) = 15%
	for i := 0; i < 3; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeRenewal})
	}

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	dashboard, err := service.GetRepDashboard(ctx, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dashboard.AttainmentPct != 0.15 {
		t.Errorf("expected 15%% attainment, got %.2f", dashboard.AttainmentPct)
	}

	// Commission should be 15000 (15% of 100000)
	if dashboard.CommissionEarned != 15000 {
		t.Errorf("expected 15000 commission, got %d", dashboard.CommissionEarned)
	}
}

func TestCalculateRepCommissionNoEvents(t *testing.T) {
	ctx := context.Background()
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusClosed,
	}
	periodRepo.AddPeriod(period)

	goal := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goalRepo.AddGoal(goal)

	// No events
	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	err := service.CalculateForPeriod(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stmt, _ := statementRepo.GetByRepAndPeriod(ctx, 100, 1)
	if stmt == nil {
		t.Fatal("statement should be created even with no events")
	}

	if stmt.TotalAmount != 0 {
		t.Errorf("expected 0 commission with no events, got %d", stmt.TotalAmount)
	}

	if stmt.AttainmentPct != 0.0 {
		t.Errorf("expected 0%% attainment, got %.2f", stmt.AttainmentPct)
	}
}

func TestDeterministicCalculation(t *testing.T) {
	// Test that same inputs produce same results
	ctx := context.Background()

	setup := func() (*mockServiceDeps, CommissionService) {
		goalRepo := NewMockGoalRepository()
		periodRepo := NewMockPeriodRepository()
		userRepo := NewMockUserRepository()
		statementRepo := NewMockStatementRepository()
		eventRepo := NewMockCommissionEventRepository()

		period := &model.Period{
			ID:        1,
			Name:      "January 2024",
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
			Status:    model.PeriodStatusOpen,
		}
		periodRepo.AddPeriod(period)

		goal := &model.Goal{
			ID:                1,
			RepID:             100,
			PeriodID:          1,
			AcquisitionTarget: 10,
			RenewalTarget:     10,
			CommissionValue:   100000,
		}
		goalRepo.AddGoal(goal)

		// Add same events
		for i := 0; i < 7; i++ {
			eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeAcquisition})
		}
		for i := 0; i < 5; i++ {
			eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeRenewal})
		}

		service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

		return &mockServiceDeps{
			goalRepo, periodRepo, userRepo, statementRepo, eventRepo,
		}, service
	}

	// First calculation
	deps1, service1 := setup()
	dash1, err1 := service1.GetRepDashboard(ctx, 100)
	if err1 != nil {
		t.Fatalf("first calculation failed: %v", err1)
	}

	// Second calculation with exact same setup
	deps2, service2 := setup()
	dash2, err2 := service2.GetRepDashboard(ctx, 100)
	if err2 != nil {
		t.Fatalf("second calculation failed: %v", err2)
	}

	_ = deps1
	_ = deps2

	// Results should be identical
	if dash1.AttainmentPct != dash2.AttainmentPct {
		t.Errorf("attainment percentages differ: %.2f vs %.2f", dash1.AttainmentPct, dash2.AttainmentPct)
	}

	if dash1.CommissionEarned != dash2.CommissionEarned {
		t.Errorf("commission amounts differ: %d vs %d", dash1.CommissionEarned, dash2.CommissionEarned)
	}
}

// Helper struct for deterministic testing
type mockServiceDeps struct {
	goalRepo      GoalRepository
	periodRepo    PeriodRepository
	userRepo      UserRepository
	statementRepo StatementRepository
	eventRepo     CommissionEventRepository
}

func TestGetRepDashboardWithAllEventTypes(t *testing.T) {
	// Test GetRepDashboard with both acquisition and renewal events
	ctx := context.Background()
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	periodRepo.AddPeriod(period)

	goal := &model.Goal{
		ID:                1,
		RepID:             100,
		PeriodID:          1,
		AcquisitionTarget: 10,
		RenewalTarget:     10,
		CommissionValue:   100000,
	}
	goalRepo.AddGoal(goal)

	// Add actual event objects to test ListByRepAndPeriod
	for i := 0; i < 8; i++ {
		eventRepo.AddEvent(&model.MemberEvent{
			RepID:     100,
			EventType: model.EventTypeAcquisition,
			MemberName: "Member " + string(rune(i)),
		})
	}
	for i := 0; i < 4; i++ {
		eventRepo.AddEvent(&model.MemberEvent{
			RepID:     100,
			EventType: model.EventTypeRenewal,
			MemberName: "Renewal " + string(rune(i)),
		})
	}

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	dashboard, err := service.GetRepDashboard(ctx, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 80% acquisition + 40% renewal = 60% attainment
	// Allow for floating point precision issues
	expectedAttainment := 0.6
	epsilon := 0.001
	if dashboard.AttainmentPct < expectedAttainment-epsilon || dashboard.AttainmentPct > expectedAttainment+epsilon {
		t.Errorf("expected ~%.2f attainment, got %.2f", expectedAttainment, dashboard.AttainmentPct)
	}

	// RecentEvents should be populated
	if len(dashboard.RecentEvents) != 12 {
		t.Errorf("expected 12 recent events, got %d", len(dashboard.RecentEvents))
	}
}

func TestTeamDashboardPartialGoalCoverage(t *testing.T) {
	// Test when only some team members have goals
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	periodRepo.AddPeriod(period)

	// Manager and 3 direct reports
	managerID := int64(10)
	manager := &model.User{
		ID:   managerID,
		Name: "John Manager",
		Role: model.RoleManager,
	}
	userRepo.users[managerID] = manager

	rep1 := &model.User{ID: 100, Name: "Rep 1", Role: model.RoleRep, ManagerID: &managerID}
	rep2 := &model.User{ID: 101, Name: "Rep 2", Role: model.RoleRep, ManagerID: &managerID}
	rep3 := &model.User{ID: 102, Name: "Rep 3", Role: model.RoleRep, ManagerID: &managerID}
	userRepo.users[100] = rep1
	userRepo.users[101] = rep2
	userRepo.users[102] = rep3

	// Only 2 goals for rep1 and rep2
	goal1 := &model.Goal{RepID: 100, PeriodID: 1, AcquisitionTarget: 10, RenewalTarget: 10, CommissionValue: 100000}
	goal2 := &model.Goal{RepID: 101, PeriodID: 1, AcquisitionTarget: 10, RenewalTarget: 10, CommissionValue: 100000}
	goalRepo.AddGoal(goal1)
	goalRepo.AddGoal(goal2)

	// Events only for rep1
	for i := 0; i < 10; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeAcquisition})
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeRenewal})
	}

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	dashboard, err := service.GetTeamDashboard(ctx, managerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have 2 direct reports (rep1 and rep2 have goals, rep3 doesn't)
	if len(dashboard.DirectReports) != 2 {
		t.Errorf("expected 2 direct reports, got %d", len(dashboard.DirectReports))
	}

	// Total should be 100000 (rep1 100% + rep2 0%)
	if dashboard.TotalCommission != 100000 {
		t.Errorf("expected 100000 total commission, got %d", dashboard.TotalCommission)
	}
}

func TestCalculateForPeriodMultipleReps(t *testing.T) {
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusClosed,
	}
	periodRepo.AddPeriod(period)

	// 3 different reps with different attainment levels
	rep1Goal := &model.Goal{ID: 1, RepID: 100, PeriodID: 1, AcquisitionTarget: 10, RenewalTarget: 10, CommissionValue: 100000}
	rep2Goal := &model.Goal{ID: 2, RepID: 101, PeriodID: 1, AcquisitionTarget: 10, RenewalTarget: 10, CommissionValue: 100000}
	rep3Goal := &model.Goal{ID: 3, RepID: 102, PeriodID: 1, AcquisitionTarget: 10, RenewalTarget: 10, CommissionValue: 100000}
	goalRepo.AddGoal(rep1Goal)
	goalRepo.AddGoal(rep2Goal)
	goalRepo.AddGoal(rep3Goal)

	// Rep1: 100% attainment
	for i := 0; i < 10; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeAcquisition})
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeRenewal})
	}

	// Rep2: 50% attainment
	for i := 0; i < 5; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 101, EventType: model.EventTypeAcquisition})
		eventRepo.AddEvent(&model.MemberEvent{RepID: 101, EventType: model.EventTypeRenewal})
	}

	// Rep3: 0% attainment

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	err := service.CalculateForPeriod(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all 3 statements were created
	stmt1, _ := statementRepo.GetByRepAndPeriod(ctx, 100, 1)
	stmt2, _ := statementRepo.GetByRepAndPeriod(ctx, 101, 1)
	stmt3, _ := statementRepo.GetByRepAndPeriod(ctx, 102, 1)

	if stmt1.TotalAmount != 100000 {
		t.Errorf("rep1 commission: expected 100000, got %d", stmt1.TotalAmount)
	}

	if stmt2.TotalAmount != 50000 {
		t.Errorf("rep2 commission: expected 50000, got %d", stmt2.TotalAmount)
	}

	if stmt3.TotalAmount != 0 {
		t.Errorf("rep3 commission: expected 0, got %d", stmt3.TotalAmount)
	}
}

func TestGenerateStatementsAllReps(t *testing.T) {
	// Test GenerateStatements generates for all reps with goals
	ctx := context.Background()
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusClosed,
	}
	periodRepo.AddPeriod(period)

	// Multiple goals
	for i := 0; i < 3; i++ {
		repID := int64(100 + i)
		goal := &model.Goal{
			ID:                int64(i + 1),
			RepID:             repID,
			PeriodID:          1,
			AcquisitionTarget: 10,
			RenewalTarget:     10,
			CommissionValue:   100000,
		}
		goalRepo.AddGoal(goal)

		// Events only for rep 100
		if i == 0 {
			for j := 0; j < 5; j++ {
				eventRepo.AddEvent(&model.MemberEvent{RepID: repID, EventType: model.EventTypeAcquisition})
			}
		}
	}

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	err := service.GenerateStatements(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All 3 statements should be created with pending_approval status
	for i := 0; i < 3; i++ {
		repID := int64(100 + i)
		stmt, _ := statementRepo.GetByRepAndPeriod(ctx, repID, 1)
		if stmt == nil {
			t.Errorf("statement for rep %d should have been created", repID)
			continue
		}

		if stmt.Status != model.StatementStatusPendingApproval {
			t.Errorf("rep %d statement status should be pending_approval, got %s", repID, stmt.Status)
		}
	}
}

func TestGetTeamDashboardEmptyPeriodList(t *testing.T) {
	ctx := context.Background()

	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	// No periods
	managerID := int64(10)
	manager := &model.User{
		ID:   managerID,
		Name: "John Manager",
		Role: model.RoleManager,
	}
	userRepo.users[managerID] = manager

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	_, err := service.GetTeamDashboard(ctx, managerID)
	if err == nil {
		t.Fatal("should return error when no periods exist")
	}
}

func TestTeamDashboardMixedAttainment(t *testing.T) {
	// Test team dashboard with mixed attainment levels for team members
	ctx := context.Background()
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	periodRepo.AddPeriod(period)

	managerID := int64(10)
	manager := &model.User{ID: managerID, Name: "Manager", Role: model.RoleManager}
	userRepo.users[managerID] = manager

	// Create 2 reps with different attainment
	rep1 := &model.User{ID: 100, Name: "High Performer", Role: model.RoleRep, ManagerID: &managerID}
	rep2 := &model.User{ID: 101, Name: "Low Performer", Role: model.RoleRep, ManagerID: &managerID}
	userRepo.users[100] = rep1
	userRepo.users[101] = rep2

	goal1 := &model.Goal{RepID: 100, PeriodID: 1, AcquisitionTarget: 10, RenewalTarget: 10, CommissionValue: 100000}
	goal2 := &model.Goal{RepID: 101, PeriodID: 1, AcquisitionTarget: 10, RenewalTarget: 10, CommissionValue: 100000}
	goalRepo.AddGoal(goal1)
	goalRepo.AddGoal(goal2)

	// Rep1: 90% attainment
	for i := 0; i < 9; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeAcquisition})
		eventRepo.AddEvent(&model.MemberEvent{RepID: 100, EventType: model.EventTypeRenewal})
	}

	// Rep2: 20% attainment
	for i := 0; i < 2; i++ {
		eventRepo.AddEvent(&model.MemberEvent{RepID: 101, EventType: model.EventTypeAcquisition})
	}

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	dashboard, err := service.GetTeamDashboard(ctx, managerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dashboard.DirectReports) != 2 {
		t.Errorf("expected 2 direct reports, got %d", len(dashboard.DirectReports))
	}

	// Rep 1 should be first with higher attainment
	if dashboard.DirectReports[0].CommissionEarned < dashboard.DirectReports[1].CommissionEarned {
		t.Errorf("rep 1 should have higher commission (%.2f) than rep 2 (%.2f)",
			float64(dashboard.DirectReports[0].CommissionEarned)/1000,
			float64(dashboard.DirectReports[1].CommissionEarned)/1000)
	}

	// Total should be sum of both (90000 + 10000)
	expectedTotal := int64(100000) // 90000 + 10000
	if dashboard.TotalCommission != expectedTotal {
		t.Errorf("expected total %d, got %d", expectedTotal, dashboard.TotalCommission)
	}
}

func TestCalculateForPeriodSkipsExistingStatements(t *testing.T) {
	// Test that CalculateForPeriod skips existing statements
	ctx := context.Background()
	goalRepo := NewMockGoalRepository()
	periodRepo := NewMockPeriodRepository()
	userRepo := NewMockUserRepository()
	statementRepo := NewMockStatementRepository()
	eventRepo := NewMockCommissionEventRepository()

	period := &model.Period{
		ID:        1,
		Name:      "January 2024",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusClosed,
	}
	periodRepo.AddPeriod(period)

	goal := &model.Goal{RepID: 100, PeriodID: 1, AcquisitionTarget: 10, RenewalTarget: 10, CommissionValue: 100000}
	goalRepo.AddGoal(goal)

	// Create existing statement
	existingStmt := &model.Statement{
		RepID:         100,
		PeriodID:      1,
		TotalAmount:   25000,
		AttainmentPct: 0.25,
		Status:        model.StatementStatusApproved,
	}
	statementRepo.Create(ctx, existingStmt)

	service := NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, nil)

	err := service.CalculateForPeriod(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Existing statement should not be overwritten
	stmt, _ := statementRepo.GetByRepAndPeriod(ctx, 100, 1)
	if stmt.TotalAmount != 25000 {
		t.Errorf("existing statement should not be modified, got %d", stmt.TotalAmount)
	}

	if stmt.Status != model.StatementStatusApproved {
		t.Errorf("existing statement status should not be changed, got %s", stmt.Status)
	}
}
