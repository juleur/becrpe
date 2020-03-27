package model

import (
	"time"
)

type Video struct {
	ID        int       `json:"id,omitempty" db:"id,omitempty"`
	Path      string    `json:"path,omitempty" db:"path,omitempty"`
	Duration  string    `json:"duration,omitempty" db:"duration,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty" db:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" db:"updated_at,omitempty"`
	SessionID int       `db:"session_id,omitempty"`
	UserID    int
}
