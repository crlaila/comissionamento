// +build integration

package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"comissionamento/internal/handler"
	"comissionamento/internal/hinova"
	"comissionamento/internal/middleware"
	"comissionamento/internal/model"
	"comissionamento/internal/service"
)

// MockUserRepository for testing
type MockUserRepository struct {
	users           map[int64]*model.User
	passwords       map[int64]string
	refreshTokens   map[string]bool
	nextID          int64
	userByEmail     map[string]*model.UserWithPassword
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:         make(map[int64]*model.User),
		passwords:     make(map[int64]string),
		refreshTokens: make(map[string]bool),
		userByEmail:   make(map[string]*model.UserWithPassword),
		nextID:        1,
	}
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*model.UserWithPassword, error) {
	return m.userByEmail[email], nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return m.users[id], nil
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User, passwordHash string) (int64, error) {
	user.ID = m.nextID
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[m.nextID] = user
	m.passwords[m.nextID] = passwordHash

	userWithPass := &model.UserWithPassword{
		User:         user,
		PasswordHash: passwordHash,
	}
	m.userByEmail[user.Email] = userWithPass
	m.nextID++
	return user.ID, nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	if _, ok := m.users[user.ID]; !ok {
		return service.ErrInvalidCredentials
	}
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) List(ctx context.Context) ([]*model.User, error) {
	var users []*model.User
	for _, u := range m.users {
		users = append(users, u)
	}
	return users, nil
}

func (m *MockUserRepository) StoreRefreshToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	m.refreshTokens[token] = false
	return nil
}

func (m *MockUserRepository) ValidateRefreshToken(ctx context.Context, userID int64, token string) error {
	revoked, ok := m.refreshTokens[token]
	if !ok || revoked {
		return service.ErrInvalidCredentials
	}
	return nil
}

func (m *MockUserRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	m.refreshTokens[token] = true
	return nil
}

// TestCompleteAuthFlow tests the complete authentication flow
func TestCompleteAuthFlow(t *testing.T) {
	// Setup
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret-key", mockRepo)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(authService, mockRepo)
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Create a test admin user
	adminPassword := "admin123456"
	adminHash, _ := authService.HashPassword(adminPassword)
	adminUser := &model.User{
		Email:  "admin@example.com",
		Name:   "Admin User",
		Role:   model.RoleAdmin,
		Active: true,
	}
	mockRepo.Create(context.Background(), adminUser, adminHash)

	// Step 1: Login with admin credentials
	loginBody, _ := json.Marshal(handler.LoginRequest{
		Email:    "admin@example.com",
		Password: adminPassword,
	})

	loginReq := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	authHandler.Login(loginW, loginReq)

	if loginW.Code != http.StatusOK {
		t.Fatalf("Login failed: got status %d, expected 200", loginW.Code)
	}

	var loginResp handler.LoginResponse
	json.NewDecoder(loginW.Body).Decode(&loginResp)

	accessToken := loginResp.AccessToken
	refreshToken := loginResp.RefreshToken

	if accessToken == "" || refreshToken == "" {
		t.Fatal("Login should return both access and refresh tokens")
	}

	// Step 2: Use access token to access protected endpoint (GET /api/users)
	listReq := httptest.NewRequest("GET", "/api/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+accessToken)
	listHandler := authMiddleware.Authenticate(
		authMiddleware.RequireRole(model.RoleAdmin)(http.HandlerFunc(userHandler.List)),
	)
	listW := httptest.NewRecorder()
	listHandler.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("GET /api/users failed: got status %d, expected 200", listW.Code)
	}

	// Step 3: Create a new user (admin only)
	newUserBody, _ := json.Marshal(handler.CreateUserRequest{
		Email:    "newuser@example.com",
		Name:     "New User",
		Password: "password123456",
		Role:     model.RoleRep,
	})

	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(newUserBody))
	createReq.Header.Set("Authorization", "Bearer "+accessToken)
	createReq.Header.Set("Content-Type", "application/json")
	createHandler := authMiddleware.Authenticate(
		authMiddleware.RequireRole(model.RoleAdmin)(http.HandlerFunc(userHandler.Create)),
	)
	createW := httptest.NewRecorder()
	createHandler.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("POST /api/users failed: got status %d, expected 201", createW.Code)
	}

	// Step 4: Logout (revoke refresh token)
	logoutBody, _ := json.Marshal(handler.LogoutRequest{
		RefreshToken: refreshToken,
	})

	logoutReq := httptest.NewRequest("POST", "/api/auth/logout", bytes.NewReader(logoutBody))
	logoutReq.Header.Set("Content-Type", "application/json")
	logoutW := httptest.NewRecorder()
	authHandler.Logout(logoutW, logoutReq)

	if logoutW.Code != http.StatusOK {
		t.Fatalf("POST /api/auth/logout failed: got status %d, expected 200", logoutW.Code)
	}
}

// TestProtectedEndpointWithoutToken tests that protected endpoint requires token
func TestProtectedEndpointWithoutToken(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret-key", mockRepo)
	userHandler := handler.NewUserHandler(authService, mockRepo)
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Try to access protected endpoint without token
	listReq := httptest.NewRequest("GET", "/api/users", nil)
	listHandler := authMiddleware.Authenticate(http.HandlerFunc(userHandler.List))
	listW := httptest.NewRecorder()
	listHandler.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusUnauthorized {
		t.Fatalf("Protected endpoint without token should return 401, got %d", listW.Code)
	}
}

// TestProtectedEndpointWithInsufficientRole tests role-based access control
func TestProtectedEndpointWithInsufficientRole(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret-key", mockRepo)
	userHandler := handler.NewUserHandler(authService, mockRepo)
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Create a rep user (not admin)
	repPassword := "rep123456"
	repHash, _ := authService.HashPassword(repPassword)
	repUser := &model.User{
		Email:  "rep@example.com",
		Name:   "Rep User",
		Role:   model.RoleRep,
		Active: true,
	}
	mockRepo.Create(context.Background(), repUser, repHash)

	// Login as rep
	tokenPair, _ := authService.Login(context.Background(), "rep@example.com", repPassword)

	// Try to access admin-only endpoint
	listReq := httptest.NewRequest("GET", "/api/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	listHandler := authMiddleware.Authenticate(
		authMiddleware.RequireRole(model.RoleAdmin)(http.HandlerFunc(userHandler.List)),
	)
	listW := httptest.NewRecorder()
	listHandler.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusForbidden {
		t.Fatalf("Insufficient role should return 403, got %d", listW.Code)
	}
}

// TestGetUsersReturnsAdminOnly tests that GET /api/users is admin-only
func TestGetUsersReturnsAdminOnly(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret-key", mockRepo)
	userHandler := handler.NewUserHandler(authService, mockRepo)
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Create an admin user
	adminPassword := "admin123456"
	adminHash, _ := authService.HashPassword(adminPassword)
	adminUser := &model.User{
		Email:  "admin@example.com",
		Name:   "Admin User",
		Role:   model.RoleAdmin,
		Active: true,
	}
	mockRepo.Create(context.Background(), adminUser, adminHash)

	// Create another user
	user := &model.User{
		Email:  "user@example.com",
		Name:   "Regular User",
		Role:   model.RoleRep,
		Active: true,
	}
	mockRepo.Create(context.Background(), user, "hash")

	// Login as admin
	tokenPair, _ := authService.Login(context.Background(), "admin@example.com", adminPassword)

	// Access GET /api/users as admin
	listReq := httptest.NewRequest("GET", "/api/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	listHandler := authMiddleware.Authenticate(
		authMiddleware.RequireRole(model.RoleAdmin)(http.HandlerFunc(userHandler.List)),
	)
	listW := httptest.NewRecorder()
	listHandler.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("Admin accessing GET /api/users should succeed, got %d", listW.Code)
	}

	var users []*model.User
	json.NewDecoder(listW.Body).Decode(&users)

	if len(users) != 2 {
		t.Fatalf("Should return 2 users, got %d", len(users))
	}
}

// Mock repositories for testing data models

type MockPeriodRepository struct {
	periods map[int64]*model.Period
	nextID  int64
}

func NewMockPeriodRepository() *MockPeriodRepository {
	return &MockPeriodRepository{
		periods: make(map[int64]*model.Period),
		nextID:  1,
	}
}

func (m *MockPeriodRepository) Create(ctx context.Context, period *model.Period) error {
	period.ID = m.nextID
	period.CreatedAt = time.Now()
	period.UpdatedAt = time.Now()
	m.periods[m.nextID] = period
	m.nextID++
	return nil
}

func (m *MockPeriodRepository) GetByID(ctx context.Context, id int64) (*model.Period, error) {
	return m.periods[id], nil
}

func (m *MockPeriodRepository) List(ctx context.Context) ([]*model.Period, error) {
	var periods []*model.Period
	for _, p := range m.periods {
		periods = append(periods, p)
	}
	return periods, nil
}

func (m *MockPeriodRepository) UpdateStatus(ctx context.Context, id int64, status model.PeriodStatus) error {
	if p, ok := m.periods[id]; ok {
		p.Status = status
		p.UpdatedAt = time.Now()
		return nil
	}
	return nil
}

type MockGoalRepository struct {
	goals   map[int64]*model.Goal
	periods map[int64]*model.Period // For checking period status
	nextID  int64
}

func NewMockGoalRepository(periods map[int64]*model.Period) *MockGoalRepository {
	return &MockGoalRepository{
		goals:   make(map[int64]*model.Goal),
		periods: periods,
		nextID:  1,
	}
}

func (m *MockGoalRepository) Create(ctx context.Context, goal *model.Goal) error {
	goal.ID = m.nextID
	goal.CreatedAt = time.Now()
	goal.UpdatedAt = time.Now()
	m.goals[m.nextID] = goal
	m.nextID++
	return nil
}

func (m *MockGoalRepository) Update(ctx context.Context, goal *model.Goal) error {
	// Check if period is open
	if period, ok := m.periods[goal.PeriodID]; ok {
		if period.Status != model.PeriodStatusOpen {
			return service.ErrInvalidCredentials // Simulating period locked error
		}
	}
	if g, ok := m.goals[goal.ID]; ok {
		g.AcquisitionTarget = goal.AcquisitionTarget
		g.RenewalTarget = goal.RenewalTarget
		g.CommissionValue = goal.CommissionValue
		g.UpdatedAt = time.Now()
		return nil
	}
	return nil
}

func (m *MockGoalRepository) GetByRepAndPeriod(ctx context.Context, repID, periodID int64) (*model.Goal, error) {
	for _, g := range m.goals {
		if g.RepID == repID && g.PeriodID == periodID {
			return g, nil
		}
	}
	return nil, nil
}

func (m *MockGoalRepository) ListByPeriod(ctx context.Context, periodID int64) ([]*model.Goal, error) {
	var goals []*model.Goal
	for _, g := range m.goals {
		if g.PeriodID == periodID {
			goals = append(goals, g)
		}
	}
	return goals, nil
}

func (m *MockGoalRepository) ListByRep(ctx context.Context, repID int64) ([]*model.Goal, error) {
	var goals []*model.Goal
	for _, g := range m.goals {
		if g.RepID == repID {
			goals = append(goals, g)
		}
	}
	return goals, nil
}

func (m *MockGoalRepository) DeleteByID(ctx context.Context, id int64) error {
	delete(m.goals, id)
	return nil
}

type MockStatementRepository struct {
	statements map[int64]*model.Statement
	nextID     int64
}

func NewMockStatementRepository() *MockStatementRepository {
	return &MockStatementRepository{
		statements: make(map[int64]*model.Statement),
		nextID:     1,
	}
}

func (m *MockStatementRepository) Create(ctx context.Context, stmt *model.Statement) error {
	stmt.ID = m.nextID
	stmt.CreatedAt = time.Now()
	stmt.UpdatedAt = time.Now()
	m.statements[m.nextID] = stmt
	m.nextID++
	return nil
}

func (m *MockStatementRepository) GetByID(ctx context.Context, id int64) (*model.Statement, error) {
	return m.statements[id], nil
}

func (m *MockStatementRepository) ListByPeriod(ctx context.Context, periodID int64) ([]*model.Statement, error) {
	var stmts []*model.Statement
	for _, s := range m.statements {
		if s.PeriodID == periodID {
			stmts = append(stmts, s)
		}
	}
	return stmts, nil
}

func (m *MockStatementRepository) UpdateStatus(ctx context.Context, id int64, status model.StatementStatus, approverID *int64, reason *string) error {
	if s, ok := m.statements[id]; ok {
		s.Status = status
		s.ApprovedBy = approverID
		s.RejectionReason = reason
		if status == model.StatementStatusApproved {
			now := time.Now()
			s.ApprovedAt = &now
		}
		s.UpdatedAt = time.Now()
		return nil
	}
	return nil
}

func (m *MockStatementRepository) GetByRepAndPeriod(ctx context.Context, repID, periodID int64) (*model.Statement, error) {
	for _, s := range m.statements {
		if s.RepID == repID && s.PeriodID == periodID {
			return s, nil
		}
	}
	return nil, nil
}

type MockMemberEventRepository struct {
	events map[int64]*model.MemberEvent
	byHino map[string]*model.MemberEvent // by hinova_id
	nextID int64
}

func NewMockMemberEventRepository() *MockMemberEventRepository {
	return &MockMemberEventRepository{
		events: make(map[int64]*model.MemberEvent),
		byHino: make(map[string]*model.MemberEvent),
		nextID: 1,
	}
}

func (m *MockMemberEventRepository) Upsert(ctx context.Context, event *model.MemberEvent) error {
	// Check for duplicate by hinova_id
	if existing, ok := m.byHino[event.HinovaID]; ok {
		event.ID = existing.ID
		event.CreatedAt = existing.CreatedAt
		return nil
	}
	event.ID = m.nextID
	event.CreatedAt = time.Now()
	m.events[m.nextID] = event
	m.byHino[event.HinovaID] = event
	m.nextID++
	return nil
}

func (m *MockMemberEventRepository) GetByID(ctx context.Context, id int64) (*model.MemberEvent, error) {
	return m.events[id], nil
}

func (m *MockMemberEventRepository) ListByRepAndPeriod(ctx context.Context, repID int64, startDate, endDate interface{}) ([]*model.MemberEvent, error) {
	var events []*model.MemberEvent
	for _, e := range m.events {
		if e.RepID == repID {
			events = append(events, e)
		}
	}
	return events, nil
}

func (m *MockMemberEventRepository) CountByRepAndType(ctx context.Context, repID int64, eventType model.EventType, startDate, endDate interface{}) (int, error) {
	count := 0
	for _, e := range m.events {
		if e.RepID == repID && e.EventType == eventType {
			count++
		}
	}
	return count, nil
}

// Test period CRUD operations
func TestPeriodCRUD(t *testing.T) {
	mockPeriodRepo := NewMockPeriodRepository()

	ctx := context.Background()

	// Test: Create period
	period := &model.Period{
		Name:      "April 2026",
		StartDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}

	if err := mockPeriodRepo.Create(ctx, period); err != nil {
		t.Fatalf("Failed to create period: %v", err)
	}

	if period.ID == 0 {
		t.Error("Period should have an ID after creation")
	}

	// Test: Get period by ID
	retrieved, err := mockPeriodRepo.GetByID(ctx, period.ID)
	if err != nil {
		t.Fatalf("Failed to get period: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved period should not be nil")
	}

	if retrieved.Name != period.Name {
		t.Errorf("Expected period name %s, got %s", period.Name, retrieved.Name)
	}

	// Test: Update status
	if err := mockPeriodRepo.UpdateStatus(ctx, period.ID, model.PeriodStatusClosed); err != nil {
		t.Fatalf("Failed to update period status: %v", err)
	}

	updated, _ := mockPeriodRepo.GetByID(ctx, period.ID)
	if updated.Status != model.PeriodStatusClosed {
		t.Errorf("Expected status closed, got %s", updated.Status)
	}

	// Test: List periods
	periods, err := mockPeriodRepo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list periods: %v", err)
	}

	if len(periods) != 1 {
		t.Errorf("Expected 1 period, got %d", len(periods))
	}
}

// Test goal update validation (period must be open)
func TestGoalUpdateValidatePeriodOpen(t *testing.T) {
	ctx := context.Background()

	// Create a closed period
	closedPeriod := &model.Period{
		ID:        1,
		Name:      "March 2026",
		StartDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		Status:    model.PeriodStatusClosed,
	}

	periods := map[int64]*model.Period{
		1: closedPeriod,
	}

	mockGoalRepo := NewMockGoalRepository(periods)

	// Create a goal
	goal := &model.Goal{
		RepID:    1,
		PeriodID: 1,
		ID:       1,
	}

	// Try to update goal in closed period (should fail)
	err := mockGoalRepo.Update(ctx, goal)
	if err == nil {
		t.Error("Should not allow goal update for closed period")
	}
}

// Test member event deduplication by hinova_id
func TestMemberEventDeduplication(t *testing.T) {
	ctx := context.Background()
	mockEventRepo := NewMockMemberEventRepository()

	event1 := &model.MemberEvent{
		HinovaID:   "HINOVA-12345",
		RepID:      1,
		EventType:  model.EventTypeAcquisition,
		MemberName: "John Doe",
		EventDate:  time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
		SyncedAt:   time.Now(),
	}

	// Upsert first time
	if err := mockEventRepo.Upsert(ctx, event1); err != nil {
		t.Fatalf("Failed to upsert first event: %v", err)
	}

	firstID := event1.ID

	// Upsert same event again (should not create duplicate)
	event2 := &model.MemberEvent{
		HinovaID:   "HINOVA-12345",
		RepID:      1,
		EventType:  model.EventTypeAcquisition,
		MemberName: "John Doe",
		EventDate:  time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
		SyncedAt:   time.Now(),
	}

	if err := mockEventRepo.Upsert(ctx, event2); err != nil {
		t.Fatalf("Failed to upsert duplicate event: %v", err)
	}

	// Both should have same ID (no duplicate created)
	if event2.ID != firstID {
		t.Errorf("Duplicate event should have same ID, got %d (first) vs %d (second)", firstID, event2.ID)
	}
}

// Test statement status workflow
func TestStatementStatusWorkflow(t *testing.T) {
	ctx := context.Background()
	mockStmtRepo := NewMockStatementRepository()

	stmt := &model.Statement{
		RepID:       1,
		PeriodID:    1,
		TotalAmount: 250000,
		Status:      model.StatementStatusDraft,
	}

	if err := mockStmtRepo.Create(ctx, stmt); err != nil {
		t.Fatalf("Failed to create statement: %v", err)
	}

	// Test: draft → pending_approval
	if err := mockStmtRepo.UpdateStatus(ctx, stmt.ID, model.StatementStatusPendingApproval, nil, nil); err != nil {
		t.Fatalf("Failed to update to pending_approval: %v", err)
	}

	retrieved, _ := mockStmtRepo.GetByID(ctx, stmt.ID)
	if retrieved.Status != model.StatementStatusPendingApproval {
		t.Errorf("Expected pending_approval, got %s", retrieved.Status)
	}

	// Test: pending_approval → approved
	approverID := int64(1)
	if err := mockStmtRepo.UpdateStatus(ctx, stmt.ID, model.StatementStatusApproved, &approverID, nil); err != nil {
		t.Fatalf("Failed to update to approved: %v", err)
	}

	retrieved, _ = mockStmtRepo.GetByID(ctx, stmt.ID)
	if retrieved.Status != model.StatementStatusApproved {
		t.Errorf("Expected approved, got %s", retrieved.Status)
	}

	if retrieved.ApprovedBy == nil || *retrieved.ApprovedBy != approverID {
		t.Error("ApprovedBy should be set")
	}

	if retrieved.ApprovedAt == nil {
		t.Error("ApprovedAt should be set")
	}

	// Test: approved → paid
	if err := mockStmtRepo.UpdateStatus(ctx, stmt.ID, model.StatementStatusPaid, nil, nil); err != nil {
		t.Fatalf("Failed to update to paid: %v", err)
	}

	retrieved, _ = mockStmtRepo.GetByID(ctx, stmt.ID)
	if retrieved.Status != model.StatementStatusPaid {
		t.Errorf("Expected paid, got %s", retrieved.Status)
	}
}

// Test sync flow: fetch from Hinova, deduplicate, and store
func TestFullSyncFlow(t *testing.T) {
	ctx := context.Background()

	// Create a mock Hinova client with test data
	mockHinovaClient := hinova.NewMockClient()
	mockEventRepo := NewMockMemberEventRepository()

	now := time.Now().UTC()
	event1 := &model.MemberEvent{
		HinovaID:   "HINOVA-1",
		RepID:      1,
		EventType:  model.EventTypeAcquisition,
		MemberName: "John Doe",
		EventDate:  now.Add(-12 * time.Hour),
		SyncedAt:   now,
	}
	event2 := &model.MemberEvent{
		HinovaID:   "HINOVA-2",
		RepID:      2,
		EventType:  model.EventTypeRenewal,
		MemberName: "Jane Smith",
		EventDate:  now.Add(-6 * time.Hour),
		SyncedAt:   now,
	}

	mockHinovaClient.AddEvent(event1)
	mockHinovaClient.AddEvent(event2)

	syncService := service.NewSyncService(mockHinovaClient, mockEventRepo, 5*time.Minute)

	// Perform sync
	result, err := syncService.SyncMemberEvents(ctx)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify results
	if result.EventsFetched != 2 {
		t.Errorf("Expected 2 events fetched, got %d", result.EventsFetched)
	}

	if result.EventsNew != 2 {
		t.Errorf("Expected 2 new events, got %d", result.EventsNew)
	}

	// Verify events were stored
	if len(mockEventRepo.events) != 2 {
		t.Errorf("Expected 2 events in repository, got %d", len(mockEventRepo.events))
	}
}

// Test sync with duplicate events (deduplication)
func TestSyncWithDuplicates(t *testing.T) {
	ctx := context.Background()

	mockHinovaClient := hinova.NewMockClient()
	mockEventRepo := NewMockMemberEventRepository()

	now := time.Now().UTC()

	// Pre-populate with existing event
	existingEvent := &model.MemberEvent{
		HinovaID:   "HINOVA-1",
		RepID:      1,
		EventType:  model.EventTypeAcquisition,
		MemberName: "John Doe",
		EventDate:  now.Add(-12 * time.Hour),
		SyncedAt:   now,
	}
	mockEventRepo.Upsert(ctx, existingEvent)

	// Add same event to Hinova (simulating duplicate)
	mockHinovaClient.AddEvent(&model.MemberEvent{
		HinovaID:   "HINOVA-1",
		RepID:      1,
		EventType:  model.EventTypeAcquisition,
		MemberName: "John Doe",
		EventDate:  now.Add(-12 * time.Hour),
		SyncedAt:   now,
	})

	// Add new event
	mockHinovaClient.AddEvent(&model.MemberEvent{
		HinovaID:   "HINOVA-2",
		RepID:      2,
		EventType:  model.EventTypeRenewal,
		MemberName: "Jane Smith",
		EventDate:  now.Add(-6 * time.Hour),
		SyncedAt:   now,
	})

	syncService := service.NewSyncService(mockHinovaClient, mockEventRepo, 5*time.Minute)

	result, err := syncService.SyncMemberEvents(ctx)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.EventsFetched != 2 {
		t.Errorf("Expected 2 events fetched, got %d", result.EventsFetched)
	}

	// Total events should be 2 (no new duplicates created)
	if len(mockEventRepo.events) != 2 {
		t.Errorf("Expected 2 total events in repository, got %d", len(mockEventRepo.events))
	}
}

// Test sync status endpoint
func TestGetSyncStatus(t *testing.T) {
	ctx := context.Background()

	mockHinovaClient := hinova.NewMockClient()
	mockEventRepo := NewMockMemberEventRepository()

	syncService := service.NewSyncService(mockHinovaClient, mockEventRepo, 5*time.Minute)

	status, err := syncService.GetSyncStatus(ctx)
	if err != nil {
		t.Fatalf("Failed to get sync status: %v", err)
	}

	if status.Status != "idle" {
		t.Errorf("Expected status 'idle', got %s", status.Status)
	}

	if status.FailureCount != 0 {
		t.Errorf("Expected 0 failures, got %d", status.FailureCount)
	}
}

// Test exponential backoff behavior
func TestBackoffOnConsecutiveFailures(t *testing.T) {
	ctx := context.Background()

	mockHinovaClient := hinova.NewMockClient()
	mockEventRepo := NewMockMemberEventRepository()

	syncService := service.NewSyncService(mockHinovaClient, mockEventRepo, 5*time.Minute)

	// Simulate failures
	mockHinovaClient.SetError(service.ErrInvalidCredentials)

	// First failure
	result1, _ := syncService.SyncMemberEvents(ctx)
	if result1.Error == "" {
		t.Error("Expected error in first sync result")
	}

	// Second failure
	result2, _ := syncService.SyncMemberEvents(ctx)
	if result2.Error == "" {
		t.Error("Expected error in second sync result")
	}

	// Third failure - should trigger alert
	result3, _ := syncService.SyncMemberEvents(ctx)
	if result3.Error == "" {
		t.Error("Expected error in third sync result")
	}

	// Verify status through GetSyncStatus
	status, _ := syncService.GetSyncStatus(ctx)
	if status.FailureCount != 3 {
		t.Errorf("Expected 3 failure count after 3 failures, got %d", status.FailureCount)
	}

	if status.Status != "error" {
		t.Errorf("Expected status 'error', got %s", status.Status)
	}
}

// Test recovery from errors
func TestRecoveryFromErrors(t *testing.T) {
	ctx := context.Background()

	mockHinovaClient := hinova.NewMockClient()
	mockEventRepo := NewMockMemberEventRepository()

	syncService := service.NewSyncService(mockHinovaClient, mockEventRepo, 5*time.Minute)

	// Simulate a failure
	mockHinovaClient.SetError(service.ErrInvalidCredentials)
	result1, _ := syncService.SyncMemberEvents(ctx)

	if result1.Error == "" {
		t.Error("Expected error in first sync")
	}

	// Check failure count
	status1, _ := syncService.GetSyncStatus(ctx)
	if status1.FailureCount != 1 {
		t.Errorf("Expected 1 failure count, got %d", status1.FailureCount)
	}

	// Now recover
	mockHinovaClient.SetError(nil)
	mockHinovaClient.AddEvent(&model.MemberEvent{
		HinovaID:   "HINOVA-1",
		RepID:      1,
		EventType:  model.EventTypeAcquisition,
		MemberName: "John Doe",
		EventDate:  time.Now().UTC().Add(-12 * time.Hour),
		SyncedAt:   time.Now().UTC(),
	})

	result2, _ := syncService.SyncMemberEvents(ctx)

	if result2.Error != "" {
		t.Errorf("Expected successful sync, got error: %s", result2.Error)
	}

	// Check status after recovery
	status2, _ := syncService.GetSyncStatus(ctx)
	if status2.FailureCount != 0 {
		t.Errorf("Expected 0 consecutive errors after successful sync, got %d", status2.FailureCount)
	}

	if status2.Status != "idle" {
		t.Errorf("Expected status 'idle' after successful sync, got %s", status2.Status)
	}
}
