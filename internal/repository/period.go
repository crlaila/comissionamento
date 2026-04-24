package repository

import (
	"context"
	"errors"
	"fmt"

	"comissionamento/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PeriodRepository struct {
	pool *pgxpool.Pool
}

func NewPeriodRepository(pool *pgxpool.Pool) *PeriodRepository {
	return &PeriodRepository{pool: pool}
}

func (r *PeriodRepository) Create(ctx context.Context, period *model.Period) error {
	query := `
		INSERT INTO periods (name, start_date, end_date, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		period.Name,
		period.StartDate,
		period.EndDate,
		period.Status,
	).Scan(&period.ID, &period.CreatedAt, &period.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create period: %w", err)
	}

	return nil
}

func (r *PeriodRepository) GetByID(ctx context.Context, id int64) (*model.Period, error) {
	query := `
		SELECT id, name, start_date, end_date, status, created_at, updated_at
		FROM periods
		WHERE id = $1
	`

	var period model.Period
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&period.ID,
		&period.Name,
		&period.StartDate,
		&period.EndDate,
		&period.Status,
		&period.CreatedAt,
		&period.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query period by id: %w", err)
	}

	return &period, nil
}

func (r *PeriodRepository) List(ctx context.Context) ([]*model.Period, error) {
	query := `
		SELECT id, name, start_date, end_date, status, created_at, updated_at
		FROM periods
		ORDER BY start_date DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query periods: %w", err)
	}
	defer rows.Close()

	var periods []*model.Period
	for rows.Next() {
		var period model.Period
		err := rows.Scan(
			&period.ID,
			&period.Name,
			&period.StartDate,
			&period.EndDate,
			&period.Status,
			&period.CreatedAt,
			&period.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan period: %w", err)
		}
		periods = append(periods, &period)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating periods: %w", err)
	}

	return periods, nil
}

func (r *PeriodRepository) UpdateStatus(ctx context.Context, id int64, status model.PeriodStatus) error {
	query := `
		UPDATE periods
		SET status = $1, updated_at = now()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update period status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("period not found: %d", id)
	}

	return nil
}
