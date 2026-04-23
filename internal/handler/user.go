package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"comissionamento/internal/model"
	"comissionamento/internal/service"
)

type UserHandler struct {
	authService *service.AuthService
	userRepo    service.UserRepository
}

func NewUserHandler(authService *service.AuthService, userRepo service.UserRepository) *UserHandler {
	return &UserHandler{
		authService: authService,
		userRepo:    userRepo,
	}
}

type CreateUserRequest struct {
	Email    string      `json:"email"`
	Name     string      `json:"name"`
	Password string      `json:"password"`
	Role     model.UserRole `json:"role"`
	ManagerID *int64     `json:"manager_id"`
}

type UpdateUserRequest struct {
	Name      string      `json:"name"`
	Role      model.UserRole `json:"role"`
	ManagerID *int64     `json:"manager_id"`
	Active    bool       `json:"active"`
}

// List handles GET /api/users
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	users, err := h.userRepo.List(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// Create handles POST /api/users
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Email == "" || req.Name == "" || req.Password == "" {
		http.Error(w, "Email, name, and password are required", http.StatusBadRequest)
		return
	}

	// Validate email format
	if !isValidEmail(req.Email) {
		http.Error(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	// Validate password length
	if len(req.Password) < 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Hash password
	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	user := &model.User{
		Email:     req.Email,
		Name:      req.Name,
		Role:      req.Role,
		ManagerID: req.ManagerID,
		Active:    true,
	}

	userID, err := h.userRepo.Create(r.Context(), user, passwordHash)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusBadRequest)
		return
	}

	user.ID = userID
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// Update handles PUT /api/users/{id}
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid request path", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing user
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update fields
	user.Name = req.Name
	user.Role = req.Role
	user.ManagerID = req.ManagerID
	user.Active = req.Active

	if err := h.userRepo.Update(r.Context(), user); err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// isValidEmail checks if email format is valid
func isValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}
