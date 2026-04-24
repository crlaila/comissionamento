package repository

import (
	"context"
	"errors"
	"fmt"

	"comissionamento/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GoalRepository struct {
	pool *pgxpool.Pool
}

func NewGoalRepository(pool *pgxpool.Pool) *GoalRepository {
	return &GoalRepository{pool: pool}
}

func (r *GoalRepository) Create(ctx context.Context, goal *model.Goal) error {
	query := `
		INSERT INTO goals (rep_id, period_id, acquisition_target, renewal_target, commission_value)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		goal.RepID,
		goal.PeriodID,
		goal.AcquisitionTarget,
		goal.RenewalTarget,
		goal.CommissionValue,
	).Scan(&goal.ID, &goal.CreatedAt, &goal.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create goal: %w", err)
	}

	return nil
}

func (r *GoalRepository) GetByRepAndPeriod(ctx context.Context, repID, periodID int64) (*model.Goal, error) {
	query := `
		SELECT id, rep_id, period_id, acquisition_target, renewal_target, commission_value, created_at, updated_at
		FROM goals
		WHERE rep_id = $1 AND period_id = $2
	`

	var goal model.Goal
	err := r.pool.QueryRow(ctx, query, repID, periodID).Scan(
		&goal.ID,
		&goal.RepID,
		&goal.PeriodID,
		&goal.AcquisitionTarget,
		&goal.RenewalTarget,
		&goal.CommissionValue,
		&goal.CreatedAt,
		&goal.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query goal: %w", err)
	}

	return &goal, nil
}

func (r *GoalRepository) ListByPeriod(ctx context.Context, periodID int64) ([]*model.Goal, error) {
	query := `
		SELECT id, rep_id, period_id, acquisition_target, renewal_target, commission_value, created_at, updated_at
		FROM goals
		WHERE period_id = $1
		ORDER BY rep_id
	`

	rows, err := r.pool.Query(ctx, query, periodID)
	if err != nil {
		return nil, fmt.Errorf("failed to query goals by period: %w", err)
	}
	defer rows.Close()

	var goals []*model.Goal
	for rows.Next() {
		var goal model.Goal
		err := rows.Scan(
			&goal.ID,
			&goal.RepID,
			&goal.PeriodID,
			&goal.AcquisitionTarget,
			&goal.RenewalTarget,
			&goal.CommissionValue,
			&goal.CreatedAt,
			&goal.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan goal: %w", err)
		}
		goals = append(goals, &goal)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating goals: %w", err)
	}

	return goals, nil
}

func (r *GoalRepository) ListByRep(ctx context.Context, repID int64) ([]*model.Goal, error) {
	query := `
		SELECT id, rep_id, period_id, acquisition_target, renewal_target, commission_value, created_at, updated_at
		FROM goals
		WHERE rep_id = $1
		ORDER BY period_id DESC
	`

	rows, err := r.pool.Query(ctx, query, repID)
	if err != nil {
		return nil, fmt.Errorf("failed to query goals by rep: %w", err)
	}
	defer rows.Close()

	var goals []*model.Goal
	for rows.Next() {
		var goal model.Goal
		err := rows.Scan(
			&goal.ID,
			&goal.RepID,
			&goal.PeriodID,
			&goal.AcquisitionTarget,
			&goal.RenewalTarget,
			&goal.CommissionValue,
			&goal.CreatedAt,
			&goal.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan goal: %w", err)
		}
		goals = append(goals, &goal)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating goals: %w", err)
	}

	return goals, nil
}

// Update updates a goal only if the period is open
func (r *GoalRepository) Update(ctx context.Context, goal *model.Goal) error {
	// Check if period is open
	periodQuery := `SELECT status FROM periods WHERE id = $1`
	var status model.PeriodStatus
	err := r.pool.QueryRow(ctx, periodQuery, goal.PeriodID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("period not found: %d", goal.PeriodID)
		}
		return fmt.Errorf("failed to check period status: %w", err)
	}

	if status != model.PeriodStatusOpen {
		return fmt.Errorf("cannot update goal for closed period: status=%s", status)
	}

	query := `
		UPDATE goals
		SET acquisition_target = $1, renewal_target = $2, commission_value = $3, updated_at = now()
		WHERE id = $4
	`

	result, err := r.pool.Exec(ctx, query,
		goal.AcquisitionTarget,
		goal.RenewalTarget,
		goal.CommissionValue,
		goal.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update goal: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("goal not found: %d", goal.ID)
	}

	return nil
}

func (r *GoalRepository) DeleteByID(ctx context.Context, id int64) error {
	query := `DELETE FROM goals WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete goal: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("goal not found: %d", id)
	}

	return nil
}
