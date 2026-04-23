package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"comissionamento/internal/model"
	"comissionamento/internal/service"
)

// Test: GET /api/users lists all users
func TestListUsers(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	userHandler := NewUserHandler(authService, mockRepo)

	// Create test users
	for i := 1; i <= 3; i++ {
		user := &model.User{
			Email:  "user" + string(rune('0'+i)) + "@example.com",
			Name:   "User " + string(rune('0'+i)),
			Role:   model.RoleRep,
			Active: true,
		}
		mockRepo.Create(context.Background(), user, "hash")
	}

	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	userHandler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var users []*model.User
	json.NewDecoder(w.Body).Decode(&users)

	if len(users) != 3 {
		t.Fatalf("Expected 3 users, got %d", len(users))
	}
}

// Test: POST /api/users creates a new user
func TestCreateUserSuccess(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	userHandler := NewUserHandler(authService, mockRepo)

	body, _ := json.Marshal(CreateUserRequest{
		Email:    "newuser@example.com",
		Name:     "New User",
		Password: "password123456",
		Role:     model.RoleRep,
	})

	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userHandler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", w.Code)
	}

	var user model.User
	json.NewDecoder(w.Body).Decode(&user)

	if user.Email != "newuser@example.com" {
		t.Fatalf("Expected email newuser@example.com, got %s", user.Email)
	}
}

// Test: POST /api/users with missing email
func TestCreateUserMissingEmail(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	userHandler := NewUserHandler(authService, mockRepo)

	body, _ := json.Marshal(CreateUserRequest{
		Email:    "",
		Name:     "New User",
		Password: "password123456",
		Role:     model.RoleRep,
	})

	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userHandler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// Test: POST /api/users with invalid email format
func TestCreateUserInvalidEmail(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	userHandler := NewUserHandler(authService, mockRepo)

	body, _ := json.Marshal(CreateUserRequest{
		Email:    "invalid-email",
		Name:     "New User",
		Password: "password123456",
		Role:     model.RoleRep,
	})

	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userHandler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// Test: POST /api/users with short password
func TestCreateUserShortPassword(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	userHandler := NewUserHandler(authService, mockRepo)

	body, _ := json.Marshal(CreateUserRequest{
		Email:    "user@example.com",
		Name:     "New User",
		Password: "short",
		Role:     model.RoleRep,
	})

	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userHandler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// Test: POST /api/users with invalid request body
func TestCreateUserInvalidBody(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	userHandler := NewUserHandler(authService, mockRepo)

	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userHandler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// Test: PUT /api/users/{id} updates a user
func TestUpdateUserSuccess(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	userHandler := NewUserHandler(authService, mockRepo)

	// Create a user first
	user := &model.User{
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   model.RoleRep,
		Active: true,
	}
	mockRepo.Create(context.Background(), user, "hash")

	body, _ := json.Marshal(UpdateUserRequest{
		Name:   "Updated User",
		Role:   model.RoleManager,
		Active: true,
	})

	req := httptest.NewRequest("PUT", "/api/users/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userHandler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var updatedUser model.User
	json.NewDecoder(w.Body).Decode(&updatedUser)

	if updatedUser.Name != "Updated User" {
		t.Fatalf("Expected name Updated User, got %s", updatedUser.Name)
	}

	if updatedUser.Role != model.RoleManager {
		t.Fatalf("Expected role manager, got %s", updatedUser.Role)
	}
}

// Test: PUT /api/users/{id} with invalid ID
func TestUpdateUserInvalidID(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	userHandler := NewUserHandler(authService, mockRepo)

	body, _ := json.Marshal(UpdateUserRequest{
		Name:   "Updated User",
		Role:   model.RoleManager,
		Active: true,
	})

	req := httptest.NewRequest("PUT", "/api/users/invalid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userHandler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// Test: PUT /api/users/{id} with non-existent user
func TestUpdateUserNotFound(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	userHandler := NewUserHandler(authService, mockRepo)

	body, _ := json.Marshal(UpdateUserRequest{
		Name:   "Updated User",
		Role:   model.RoleManager,
		Active: true,
	})

	req := httptest.NewRequest("PUT", "/api/users/999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userHandler.Update(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", w.Code)
	}
}

// Test: POST /api/users with invalid request method
func TestCreateUserInvalidMethod(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	userHandler := NewUserHandler(authService, mockRepo)

	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	userHandler.Create(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status 405, got %d", w.Code)
	}
}

// Test: PUT /api/users/{id} with invalid request body
func TestUpdateUserInvalidBody(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := service.NewAuthService("test-secret", mockRepo)
	userHandler := NewUserHandler(authService, mockRepo)

	// Create a user first
	user := &model.User{
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   model.RoleRep,
		Active: true,
	}
	mockRepo.Create(context.Background(), user, "hash")

	req := httptest.NewRequest("PUT", "/api/users/1", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userHandler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}
