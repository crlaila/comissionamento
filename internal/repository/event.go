package repository

import (
	"context"
	"errors"
	"fmt"

	"comissionamento/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MemberEventRepository struct {
	pool *pgxpool.Pool
}

func NewMemberEventRepository(pool *pgxpool.Pool) *MemberEventRepository {
	return &MemberEventRepository{pool: pool}
}

// Upsert inserts or updates a member event with deduplication by hinova_id
func (r *MemberEventRepository) Upsert(ctx context.Context, event *model.MemberEvent) error {
	query := `
		INSERT INTO member_events (hinova_id, rep_id, event_type, member_name, event_date, synced_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (hinova_id) DO NOTHING
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(ctx, query,
		event.HinovaID,
		event.RepID,
		event.EventType,
		event.MemberName,
		event.EventDate,
		event.SyncedAt,
	).Scan(&event.ID, &event.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Event already exists, fetch it
			return r.getByHinovaID(ctx, event.HinovaID, event)
		}
		return fmt.Errorf("failed to upsert member event: %w", err)
	}

	return nil
}

func (r *MemberEventRepository) getByHinovaID(ctx context.Context, hinovaID string, event *model.MemberEvent) error {
	query := `
		SELECT id, hinova_id, rep_id, event_type, member_name, event_date, synced_at, created_at
		FROM member_events
		WHERE hinova_id = $1
	`

	err := r.pool.QueryRow(ctx, query, hinovaID).Scan(
		&event.ID,
		&event.HinovaID,
		&event.RepID,
		&event.EventType,
		&event.MemberName,
		&event.EventDate,
		&event.SyncedAt,
		&event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to query event by hinova_id: %w", err)
	}

	return nil
}

func (r *MemberEventRepository) GetByID(ctx context.Context, id int64) (*model.MemberEvent, error) {
	query := `
		SELECT id, hinova_id, rep_id, event_type, member_name, event_date, synced_at, created_at
		FROM member_events
		WHERE id = $1
	`

	var event model.MemberEvent
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&event.ID,
		&event.HinovaID,
		&event.RepID,
		&event.EventType,
		&event.MemberName,
		&event.EventDate,
		&event.SyncedAt,
		&event.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query event by id: %w", err)
	}

	return &event, nil
}

func (r *MemberEventRepository) ListByRepAndPeriod(ctx context.Context, repID int64, startDate, endDate interface{}) ([]*model.MemberEvent, error) {
	query := `
		SELECT id, hinova_id, rep_id, event_type, member_name, event_date, synced_at, created_at
		FROM member_events
		WHERE rep_id = $1 AND event_date >= $2 AND event_date <= $3
		ORDER BY event_date DESC
	`

	rows, err := r.pool.Query(ctx, query, repID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by rep and period: %w", err)
	}
	defer rows.Close()

	var events []*model.MemberEvent
	for rows.Next() {
		var event model.MemberEvent
		err := rows.Scan(
			&event.ID,
			&event.HinovaID,
			&event.RepID,
			&event.EventType,
			&event.MemberName,
			&event.EventDate,
			&event.SyncedAt,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

func (r *MemberEventRepository) CountByRepAndType(ctx context.Context, repID int64, eventType model.EventType, startDate, endDate interface{}) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM member_events
		WHERE rep_id = $1 AND event_type = $2 AND event_date >= $3 AND event_date <= $4
	`

	var count int
	err := r.pool.QueryRow(ctx, query, repID, eventType, startDate, endDate).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}

	return count, nil
}

func (r *MemberEventRepository) ListByRep(ctx context.Context, repID int64) ([]*model.MemberEvent, error) {
	query := `
		SELECT id, hinova_id, rep_id, event_type, member_name, event_date, synced_at, created_at
		FROM member_events
		WHERE rep_id = $1
		ORDER BY event_date DESC
	`

	rows, err := r.pool.Query(ctx, query, repID)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by rep: %w", err)
	}
	defer rows.Close()

	var events []*model.MemberEvent
	for rows.Next() {
		var event model.MemberEvent
		err := rows.Scan(
			&event.ID,
			&event.HinovaID,
			&event.RepID,
			&event.EventType,
			&event.MemberName,
			&event.EventDate,
			&event.SyncedAt,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}
