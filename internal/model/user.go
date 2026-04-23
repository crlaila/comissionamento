package model

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleManager UserRole = "manager"
	RoleRep     UserRole = "rep"
	RoleFinance UserRole = "finance"
)

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      UserRole  `json:"role"`
	ManagerID *int64    `json:"manager_id"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserWithPassword struct {
	*User
	PasswordHash string `json:"-"`
}

type JWTClaims struct {
	UserID int64    `json:"user_id"`
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Role   UserRole `json:"role"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type RefreshToken struct {
	ID        int64     `json:"-"`
	UserID    int64     `json:"-"`
	Token     string    `json:"-"`
	ExpiresAt time.Time `json:"-"`
	CreatedAt time.Time `json:"-"`
}
