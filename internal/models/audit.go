package models

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID         int64           `json:"id" db:"id"`
	UserID     *int64          `json:"user_id" db:"user_id"`
	Action     string          `json:"action" db:"action"`
	EntityType *string         `json:"entity_type" db:"entity_type"`
	EntityID   *int64          `json:"entity_id" db:"entity_id"`
	Details    json.RawMessage `json:"details" db:"details"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}
