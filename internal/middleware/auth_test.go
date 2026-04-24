package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"comissionamento/internal/model"
	"comissionamento/internal/service"
)

// MockUserRepository for testing middleware
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

// Test: Protected endpoint without token returns 401
func TestAuthenticateNoToken(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	authMiddleware := NewAuthMiddleware(authService)

	handler := authMiddleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/protected", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", w.Code)
	}
}

// Test: Protected endpoint with invalid token returns 401
func TestAuthenticateInvalidToken(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	authMiddleware := NewAuthMiddleware(authService)

	handler := authMiddleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", w.Code)
	}
}

// Test: Protected endpoint with valid token returns 200
func TestAuthenticateValidToken(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	authMiddleware := NewAuthMiddleware(authService)

	// Create user and get token
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

	handler := authMiddleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

// Test: Role-based middleware allows matching role
func TestRequireRoleAllows(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	authMiddleware := NewAuthMiddleware(authService)

	// Create admin user
	password := "password123"
	hash, _ := authService.HashPassword(password)
	user := &model.User{
		Email:  "admin@example.com",
		Name:   "Admin User",
		Role:   model.RoleAdmin,
		Active: true,
	}
	mockRepo.Create(context.Background(), user, hash)

	tokenPair, _ := authService.Login(context.Background(), "admin@example.com", password)

	handler := authMiddleware.Authenticate(
		authMiddleware.RequireRole(model.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

// Test: Role-based middleware rejects non-matching role
func TestRequireRoleRejects(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	authMiddleware := NewAuthMiddleware(authService)

	// Create rep user (not admin)
	password := "password123"
	hash, _ := authService.HashPassword(password)
	user := &model.User{
		Email:  "rep@example.com",
		Name:   "Rep User",
		Role:   model.RoleRep,
		Active: true,
	}
	mockRepo.Create(context.Background(), user, hash)

	tokenPair, _ := authService.Login(context.Background(), "rep@example.com", password)

	handler := authMiddleware.Authenticate(
		authMiddleware.RequireRole(model.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected status 403, got %d", w.Code)
	}
}

// Test: OptionalAuth with valid token injects user context
func TestOptionalAuthWithValidToken(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	authMiddleware := NewAuthMiddleware(authService)

	// Create user and get token
	password := "password123"
	hash, _ := authService.HashPassword(password)
	user := &model.User{
		ID:    1,
		Email: "test@example.com",
		Name:  "Test User",
		Role:  model.RoleRep,
		Active: true,
	}
	mockRepo.Create(context.Background(), user, hash)

	tokenPair, _ := authService.Login(context.Background(), "test@example.com", password)

	var capturedUserID int64
	handler := authMiddleware.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = r.Context().Value("user_id").(int64)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/public", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	if capturedUserID != user.ID {
		t.Fatalf("Expected user_id %d, got %d", user.ID, capturedUserID)
	}
}

// Test: OptionalAuth without token still allows request
func TestOptionalAuthWithoutToken(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	authMiddleware := NewAuthMiddleware(authService)

	handler := authMiddleware.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/public", nil)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

// Test: OptionalAuth with invalid token still allows request
func TestOptionalAuthWithInvalidToken(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	authMiddleware := NewAuthMiddleware(authService)

	handler := authMiddleware.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/public", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}
