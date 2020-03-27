package model

import (
	"time"
)

type ClassPaper struct {
	ID        int       `json:"id,omitempty" db:"id,omitempty"`
	Title     string    `json:"title,omitempty" db:"title,omitempty"`
	Path      string    `json:"path,omitempty" db:"path,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty" db:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" db:"updated_at,omitempty"`
	SessionID int       `db:"session_id,omitempty"`
}
