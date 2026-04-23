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
	"comissionamento/internal/service"
)

// MockUserRepository for testing handlers
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
	m.refreshTokens[token] = false // false means not revoked
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

// Helper to setup test data
func setupTestAuthHandler(t *testing.T) (*AuthHandler, *MockUserRepository) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	authHandler := NewAuthHandler(authService)

	// Create a test user
	password := "password123"
	hash, _ := authService.HashPassword(password)
	user := &model.User{
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   model.RoleRep,
		Active: true,
	}
	mockRepo.Create(context.Background(), user, hash)

	return authHandler, mockRepo
}

// Test: POST /api/auth/login with valid credentials
func TestLoginValid(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	body, _ := json.Marshal(LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	})

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	authHandler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response LoginResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.AccessToken == "" {
		t.Fatal("AccessToken should not be empty")
	}

	if response.RefreshToken == "" {
		t.Fatal("RefreshToken should not be empty")
	}
}

// Test: POST /api/auth/login with invalid credentials
func TestLoginInvalid(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	body, _ := json.Marshal(LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	})

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	authHandler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", w.Code)
	}
}

// Test: POST /api/auth/login with missing email
func TestLoginMissingEmail(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	body, _ := json.Marshal(LoginRequest{
		Email:    "",
		Password: "password123",
	})

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	authHandler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// Test: POST /api/auth/logout invalidates refresh token
func TestLogoutRevokesToken(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	authHandler := NewAuthHandler(authService)

	// Create user and get tokens
	password := "password123"
	hash, _ := authService.HashPassword(password)
	user := &model.User{
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   model.RoleRep,
		Active: true,
	}
	mockRepo.Create(context.Background(), user, hash)

	tokenPair, _ := authService.Login(context.Background(), "test@example.com", password)

	// Logout
	logoutBody, _ := json.Marshal(LogoutRequest{
		RefreshToken: tokenPair.RefreshToken,
	})

	req := httptest.NewRequest("POST", "/api/auth/logout", bytes.NewReader(logoutBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	authHandler.Logout(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Try to validate revoked token
	err := mockRepo.ValidateRefreshToken(context.Background(), user.ID, tokenPair.RefreshToken)
	if err == nil {
		t.Fatal("ValidateRefreshToken should fail for revoked token")
	}
}

// Test: POST /api/auth/logout with missing refresh token
func TestLogoutMissingToken(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	body, _ := json.Marshal(LogoutRequest{
		RefreshToken: "",
	})

	req := httptest.NewRequest("POST", "/api/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	authHandler.Logout(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// Test: POST /api/auth/login with invalid request body
func TestLoginInvalidBody(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	authHandler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// Test: POST /api/auth/login with missing password
func TestLoginMissingPassword(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	body, _ := json.Marshal(LoginRequest{
		Email:    "test@example.com",
		Password: "",
	})

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	authHandler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// Test: POST /api/auth/login with wrong method
func TestLoginWrongMethod(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	req := httptest.NewRequest("GET", "/api/auth/login", nil)
	w := httptest.NewRecorder()
	authHandler.Login(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status 405, got %d", w.Code)
	}
}

// Test: POST /api/auth/logout with invalid request body
func TestLogoutInvalidBody(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	req := httptest.NewRequest("POST", "/api/auth/logout", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	authHandler.Logout(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// Test: POST /api/auth/logout with wrong method
func TestLogoutWrongMethod(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	req := httptest.NewRequest("GET", "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	authHandler.Logout(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status 405, got %d", w.Code)
	}
}

// Test: POST /api/auth/refresh with invalid request body
func TestRefreshInvalidBody(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	req := httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	authHandler.Refresh(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// Test: POST /api/auth/refresh with missing refresh token
func TestRefreshMissingToken(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	body, _ := json.Marshal(RefreshRequest{
		RefreshToken: "",
	})

	req := httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	authHandler.Refresh(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// Test: POST /api/auth/refresh with invalid token
func TestRefreshInvalidToken(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	body, _ := json.Marshal(RefreshRequest{
		RefreshToken: "invalid-token",
	})

	req := httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Add a user_id to the context
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1)))

	w := httptest.NewRecorder()
	authHandler.Refresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", w.Code)
	}
}

// Test: POST /api/auth/refresh with wrong method
func TestRefreshWrongMethod(t *testing.T) {
	authHandler, _ := setupTestAuthHandler(t)

	req := httptest.NewRequest("GET", "/api/auth/refresh", nil)
	w := httptest.NewRecorder()
	authHandler.Refresh(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status 405, got %d", w.Code)
	}
}

