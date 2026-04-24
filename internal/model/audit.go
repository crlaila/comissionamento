package model

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID         int64               `json:"id"`
	UserID     *int64              `json:"user_id"`
	Action     string              `json:"action"`
	EntityType string              `json:"entity_type"`
	EntityID   int64               `json:"entity_id"`
	Details    json.RawMessage     `json:"details"`
	CreatedAt  time.Time           `json:"created_at"`
}
