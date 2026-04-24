package model

import "time"

type EventType string

const (
	EventTypeAcquisition EventType = "acquisition"
	EventTypeRenewal     EventType = "renewal"
)

type MemberEvent struct {
	ID        int64     `json:"id"`
	HinovaID  string    `json:"hinova_id"`
	RepID     int64     `json:"rep_id"`
	EventType EventType `json:"event_type"`
	MemberName string   `json:"member_name"`
	EventDate time.Time `json:"event_date"`
	SyncedAt  time.Time `json:"synced_at"`
	CreatedAt time.Time `json:"created_at"`
}
