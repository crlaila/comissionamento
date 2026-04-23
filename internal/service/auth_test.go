package service

import (
	"context"
	"testing"
	"time"

	"comissionamento/internal/model"
)

// MockUserRepository implements UserRepository for testing
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
		return ErrInvalidCredentials
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
		return ErrInvalidCredentials
	}
	return nil
}

func (m *MockUserRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	m.refreshTokens[token] = true
	return nil
}

// Test: bcrypt hashing
func TestHashPassword(t *testing.T) {
	authService := NewAuthService("secret", NewMockUserRepository())

	password := "mypassword123"
	hash, err := authService.HashPassword(password)

	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash should not be empty")
	}

	// Verify password against hash
	if !authService.VerifyPassword(hash, password) {
		t.Fatal("VerifyPassword should return true for correct password")
	}

	// Verify wrong password
	if authService.VerifyPassword(hash, "wrongpassword") {
		t.Fatal("VerifyPassword should return false for wrong password")
	}
}

// Test: JWT generation
func TestGenerateAccessToken(t *testing.T) {
	authService := NewAuthService("my-secret-key", NewMockUserRepository())

	user := &model.User{
		ID:    1,
		Email: "test@example.com",
		Name:  "Test User",
		Role:  model.RoleRep,
	}

	token, err := authService.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("Token should not be empty")
	}

	// Validate token
	claims, err := authService.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != user.ID {
		t.Fatalf("Expected UserID %d, got %d", user.ID, claims.UserID)
	}

	if claims.Role != user.Role {
		t.Fatalf("Expected Role %s, got %s", user.Role, claims.Role)
	}
}

// Test: JWT validation with expired token
func TestValidateTokenExpired(t *testing.T) {
	authService := &AuthService{
		jwtSecret:  "test-secret",
		userRepo:   NewMockUserRepository(),
		tokenTTL:   -1 * time.Minute, // Negative to make it expired
		refreshTTL: 7 * 24 * time.Hour,
	}

	user := &model.User{
		ID:    1,
		Email: "test@example.com",
		Role:  model.RoleAdmin,
	}

	token, err := authService.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// Wait a moment to ensure token is definitely expired
	time.Sleep(100 * time.Millisecond)

	_, err = authService.ValidateToken(token)
	if err == nil {
		t.Fatal("ValidateToken should fail for expired token")
	}
}

// Test: JWT validation with invalid signature
func TestValidateTokenInvalidSignature(t *testing.T) {
	authService := NewAuthService("secret1", NewMockUserRepository())

	user := &model.User{
		ID:    1,
		Email: "test@example.com",
		Role:  model.RoleAdmin,
	}

	token, _ := authService.GenerateAccessToken(user)

	// Try to validate with different secret
	authService.jwtSecret = "secret2"
	_, err := authService.ValidateToken(token)
	if err == nil {
		t.Fatal("ValidateToken should fail for token with invalid signature")
	}
}

// Test: Login with invalid credentials
func TestLoginInvalidPassword(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService("secret", mockRepo)

	// Create a test user
	password := "correctpassword"
	hash, _ := authService.HashPassword(password)

	user := &model.User{
		ID:     1,
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   model.RoleRep,
		Active: true,
	}

	mockRepo.Create(context.Background(), user, hash)

	// Try to login with wrong password
	_, err := authService.Login(context.Background(), "test@example.com", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Fatalf("Expected ErrInvalidCredentials, got %v", err)
	}
}

// Test: Login with non-existent email
func TestLoginNonExistentEmail(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService("secret", mockRepo)

	_, err := authService.Login(context.Background(), "nonexistent@example.com", "password")
	if err != ErrInvalidCredentials {
		t.Fatalf("Expected ErrInvalidCredentials, got %v", err)
	}
}

// Test: Successful login returns token pair
func TestSuccessfulLogin(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService("secret", mockRepo)

	password := "mypassword123"
	hash, _ := authService.HashPassword(password)

	user := &model.User{
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   model.RoleRep,
		Active: true,
	}

	mockRepo.Create(context.Background(), user, hash)

	tokenPair, err := authService.Login(context.Background(), "test@example.com", password)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Fatal("AccessToken should not be empty")
	}

	if tokenPair.RefreshToken == "" {
		t.Fatal("RefreshToken should not be empty")
	}

	if tokenPair.ExpiresIn != 900 {
		t.Fatalf("Expected ExpiresIn 900, got %d", tokenPair.ExpiresIn)
	}
}

// Test: Refresh access token with valid refresh token
func TestRefreshAccessTokenWithUserID(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService("secret", mockRepo)

	password := "mypassword123"
	hash, _ := authService.HashPassword(password)

	user := &model.User{
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   model.RoleRep,
		Active: true,
	}

	mockRepo.Create(context.Background(), user, hash)

	tokenPair, _ := authService.Login(context.Background(), "test@example.com", password)

	// Refresh the token
	newAccessToken, err := authService.RefreshAccessTokenWithUserID(context.Background(), user.ID, tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshAccessTokenWithUserID failed: %v", err)
	}

	if newAccessToken == "" {
		t.Fatal("New access token should not be empty")
	}

	// Validate the new access token
	claims, err := authService.ValidateToken(newAccessToken)
	if err != nil {
		t.Fatalf("New access token should be valid: %v", err)
	}

	if claims.UserID != user.ID {
		t.Fatalf("Expected UserID %d, got %d", user.ID, claims.UserID)
	}
}

// Test: Refresh with invalid refresh token
func TestRefreshAccessTokenInvalid(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService("secret", mockRepo)

	password := "mypassword123"
	hash, _ := authService.HashPassword(password)

	user := &model.User{
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   model.RoleRep,
		Active: true,
	}

	mockRepo.Create(context.Background(), user, hash)

	_, err := authService.RefreshAccessTokenWithUserID(context.Background(), user.ID, "invalid-token")
	if err == nil {
		t.Fatal("RefreshAccessTokenWithUserID should fail with invalid token")
	}
}

// Test: Logout revokes refresh token
func TestLogout(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService("secret", mockRepo)

	password := "mypassword123"
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
	err := authService.Logout(context.Background(), tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	// Token should be revoked
	revoked, ok := mockRepo.refreshTokens[tokenPair.RefreshToken]
	if !ok || !revoked {
		t.Fatal("Refresh token should be revoked")
	}
}

// Test: GenerateRefreshToken creates a valid token
func TestGenerateRefreshToken(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService("secret", mockRepo)

	token, expiresAt, err := authService.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("Token should not be empty")
	}

	if expiresAt.IsZero() {
		t.Fatal("ExpiresAt should not be zero")
	}

	if expiresAt.Before(time.Now()) {
		t.Fatal("ExpiresAt should be in the future")
	}
}

// Test: Login with inactive user
func TestLoginInactiveUser(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService("secret", mockRepo)

	password := "mypassword123"
	hash, _ := authService.HashPassword(password)

	user := &model.User{
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   model.RoleRep,
		Active: false, // Inactive
	}

	mockRepo.Create(context.Background(), user, hash)

	_, err := authService.Login(context.Background(), "test@example.com", password)
	if err != ErrInvalidCredentials {
		t.Fatalf("Expected ErrInvalidCredentials for inactive user, got %v", err)
	}
}
