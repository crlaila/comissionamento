package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"comissionamento/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// GetByEmail retrieves a user by email with password hash
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.UserWithPassword, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	query := `
		SELECT id, email, name, role::text, manager_id, active, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := model.UserWithPassword{User: &model.User{}}
	var managerId *int64

	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Role,
		&managerId,
		&user.Active,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}

	user.ManagerID = managerId

	return &user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("database not initialized: UserRepository or pool is nil")
	}

	query := `
		SELECT id, email, name, role::text, manager_id, active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user model.User
	var managerId *int64

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Role,
		&managerId,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query user by id: %w", err)
	}

	user.ManagerID = managerId
	return &user, nil
}

// Create inserts a new user
func (r *UserRepository) Create(ctx context.Context, user *model.User, passwordHash string) (int64, error) {
	if r == nil || r.pool == nil {
		return 0, fmt.Errorf("database not initialized: UserRepository or pool is nil")
	}

	query := `
		INSERT INTO users (email, name, role, manager_id, active, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id
	`

	var id int64
	err := r.pool.QueryRow(ctx, query,
		user.Email,
		user.Name,
		user.Role,
		user.ManagerID,
		true,
		passwordHash,
	).Scan(&id)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				return 0, fmt.Errorf("user with email %s already exists", user.Email)
			}
		}
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	return id, nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("database not initialized: UserRepository or pool is nil")
	}

	query := `
		UPDATE users
		SET name = $1, role = $2, manager_id = $3, active = $4, updated_at = NOW()
		WHERE id = $5
	`

	result, err := r.pool.Exec(ctx, query,
		user.Name,
		user.Role,
		user.ManagerID,
		user.Active,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// List retrieves all users
func (r *UserRepository) List(ctx context.Context) ([]*model.User, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("database not initialized: UserRepository or pool is nil")
	}

	query := `
		SELECT id, email, name, role::text, manager_id, active, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var user model.User
		var managerId *int64

		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.Role,
			&managerId,
			&user.Active,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		user.ManagerID = managerId
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// StoreRefreshToken stores a refresh token in the database
func (r *UserRepository) StoreRefreshToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("database not initialized: UserRepository or pool is nil")
	}

	query := `
		INSERT INTO refresh_tokens (user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, NOW())
	`

	_, err := r.pool.Exec(ctx, query, userID, token, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

// ValidateRefreshToken checks if a refresh token is valid and not revoked
func (r *UserRepository) ValidateRefreshToken(ctx context.Context, userID int64, token string) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("database not initialized: UserRepository or pool is nil")
	}

	query := `
		SELECT id FROM refresh_tokens
		WHERE user_id = $1 AND token = $2 AND expires_at > NOW() AND revoked_at IS NULL
	`

	var id int64
	err := r.pool.QueryRow(ctx, query, userID, token).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("refresh token not found or expired")
		}
		return fmt.Errorf("failed to validate refresh token: %w", err)
	}

	return nil
}

// RevokeRefreshToken revokes a refresh token
func (r *UserRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("database not initialized: UserRepository or pool is nil")
	}

	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token = $1
	`

	_, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}
