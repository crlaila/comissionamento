package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"comissionamento/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTokenExpired       = errors.New("token expired")
	ErrInvalidToken       = errors.New("invalid token")
)

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*model.UserWithPassword, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
	Create(ctx context.Context, user *model.User, passwordHash string) (int64, error)
	Update(ctx context.Context, user *model.User) error
	List(ctx context.Context) ([]*model.User, error)
	StoreRefreshToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error
	ValidateRefreshToken(ctx context.Context, userID int64, token string) error
	RevokeRefreshToken(ctx context.Context, token string) error
}

type AuthService struct {
	jwtSecret  string
	userRepo   UserRepository
	tokenTTL   time.Duration
	refreshTTL time.Duration
}

func NewAuthService(jwtSecret string, userRepo UserRepository) *AuthService {
	return &AuthService{
		jwtSecret:  jwtSecret,
		userRepo:   userRepo,
		tokenTTL:   15 * time.Minute,
		refreshTTL: 7 * 24 * time.Hour,
	}
}

// HashPassword hashes a password using bcrypt with cost 12
func (as *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func (as *AuthService) VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Login authenticates a user and returns a token pair
func (as *AuthService) Login(ctx context.Context, email, password string) (*model.TokenPair, error) {
	user, err := as.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if user == nil || !user.Active {
		return nil, ErrInvalidCredentials
	}

	if !as.VerifyPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := as.GenerateAccessToken(user.User)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, expiresAt, err := as.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	if err := as.userRepo.StoreRefreshToken(ctx, user.ID, refreshToken, expiresAt); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(as.tokenTTL.Seconds()),
	}, nil
}

// GenerateAccessToken creates a new access token
func (as *AuthService) GenerateAccessToken(user *model.User) (string, error) {
	now := time.Now()
	expiresAt := now.Add(as.tokenTTL)

	claims := &model.JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(as.jwtSecret))
}

// GenerateRefreshToken creates a new refresh token
func (as *AuthService) GenerateRefreshToken() (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(as.refreshTTL)

	claims := jwt.MapClaims{
		"exp": expiresAt.Unix(),
		"iat": now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(as.jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// RefreshAccessToken exchanges a refresh token for a new access token
func (as *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (string, error) {
	// Parse and validate refresh token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(as.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}

	// The refresh token doesn't contain user info, we need to validate it from the database
	// and extract user info separately. This is validated via the request context
	// which should contain the user info from the middleware.

	return "", ErrInvalidToken
}

// RefreshAccessTokenWithUserID exchanges a refresh token for a new access token
func (as *AuthService) RefreshAccessTokenWithUserID(ctx context.Context, userID int64, refreshToken string) (string, error) {
	// Validate refresh token signature
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(as.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}

	// Validate refresh token is stored and not revoked
	if err := as.userRepo.ValidateRefreshToken(ctx, userID, refreshToken); err != nil {
		return "", fmt.Errorf("refresh token validation failed: %w", err)
	}

	// Get fresh user data
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil || !user.Active {
		return "", ErrInvalidCredentials
	}

	// Generate new access token
	return as.GenerateAccessToken(user)
}

// ValidateToken validates and parses a JWT token
func (as *AuthService) ValidateToken(tokenString string) (*model.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(as.jwtSecret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*model.JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// Logout revokes a refresh token
func (as *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return as.userRepo.RevokeRefreshToken(ctx, refreshToken)
}
