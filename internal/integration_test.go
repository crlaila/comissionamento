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
