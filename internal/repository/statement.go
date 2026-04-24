package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"comissionamento/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatementRepository struct {
	pool *pgxpool.Pool
}

func NewStatementRepository(pool *pgxpool.Pool) *StatementRepository {
	return &StatementRepository{pool: pool}
}

func (r *StatementRepository) Create(ctx context.Context, statement *model.Statement) error {
	query := `
		INSERT INTO statements (rep_id, period_id, total_amount, attainment_pct, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		statement.RepID,
		statement.PeriodID,
		statement.TotalAmount,
		statement.AttainmentPct,
		statement.Status,
	).Scan(&statement.ID, &statement.CreatedAt, &statement.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create statement: %w", err)
	}

	return nil
}

func (r *StatementRepository) GetByID(ctx context.Context, id int64) (*model.Statement, error) {
	query := `
		SELECT id, rep_id, period_id, total_amount, attainment_pct, status, approved_by, approved_at, rejection_reason, created_at, updated_at
		FROM statements
		WHERE id = $1
	`

	var statement model.Statement
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&statement.ID,
		&statement.RepID,
		&statement.PeriodID,
		&statement.TotalAmount,
		&statement.AttainmentPct,
		&statement.Status,
		&statement.ApprovedBy,
		&statement.ApprovedAt,
		&statement.RejectionReason,
		&statement.CreatedAt,
		&statement.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query statement by id: %w", err)
	}

	return &statement, nil
}

func (r *StatementRepository) ListByPeriod(ctx context.Context, periodID int64) ([]*model.Statement, error) {
	query := `
		SELECT id, rep_id, period_id, total_amount, attainment_pct, status, approved_by, approved_at, rejection_reason, created_at, updated_at
		FROM statements
		WHERE period_id = $1
		ORDER BY rep_id
	`

	rows, err := r.pool.Query(ctx, query, periodID)
	if err != nil {
		return nil, fmt.Errorf("failed to query statements by period: %w", err)
	}
	defer rows.Close()

	var statements []*model.Statement
	for rows.Next() {
		var stmt model.Statement
		err := rows.Scan(
			&stmt.ID,
			&stmt.RepID,
			&stmt.PeriodID,
			&stmt.TotalAmount,
			&stmt.AttainmentPct,
			&stmt.Status,
			&stmt.ApprovedBy,
			&stmt.ApprovedAt,
			&stmt.RejectionReason,
			&stmt.CreatedAt,
			&stmt.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan statement: %w", err)
		}
		statements = append(statements, &stmt)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating statements: %w", err)
	}

	return statements, nil
}

func (r *StatementRepository) UpdateStatus(ctx context.Context, id int64, status model.StatementStatus, approverID *int64, reason *string) error {
	var approvedAt *time.Time
	if status == model.StatementStatusApproved {
		now := time.Now()
		approvedAt = &now
	}

	query := `
		UPDATE statements
		SET status = $1, approved_by = $2, approved_at = $3, rejection_reason = $4, updated_at = now()
		WHERE id = $5
	`

	result, err := r.pool.Exec(ctx, query, status, approverID, approvedAt, reason, id)
	if err != nil {
		return fmt.Errorf("failed to update statement status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("statement not found: %d", id)
	}

	return nil
}

func (r *StatementRepository) GetByRepAndPeriod(ctx context.Context, repID, periodID int64) (*model.Statement, error) {
	query := `
		SELECT id, rep_id, period_id, total_amount, attainment_pct, status, approved_by, approved_at, rejection_reason, created_at, updated_at
		FROM statements
		WHERE rep_id = $1 AND period_id = $2
	`

	var statement model.Statement
	err := r.pool.QueryRow(ctx, query, repID, periodID).Scan(
		&statement.ID,
		&statement.RepID,
		&statement.PeriodID,
		&statement.TotalAmount,
		&statement.AttainmentPct,
		&statement.Status,
		&statement.ApprovedBy,
		&statement.ApprovedAt,
		&statement.RejectionReason,
		&statement.CreatedAt,
		&statement.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query statement: %w", err)
	}

	return &statement, nil
}
