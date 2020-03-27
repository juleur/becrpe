package model

import (
	"database/sql"
	"time"
)

type User struct {
	ID           int            `json:"id,omitempty" db:"id,omitempty"`
	Username     string         `json:"username,omitempty" db:"username,omitempty"`
	Fullname     sql.NullString `json:"fullname,omitempty" db:"fullname,omitempty"`
	Email        string         `json:"email,omitempty" db:"email,omitempty"`
	EncryptedPWD string         `db:"encrypted_pwd,omitempty"`
	IsTeacher    bool           `json:"isTeacher,omitempty" db:"is_teacher,omitempty"`
	CreatedAt    time.Time      `json:"createdAt,omitempty" db:"created_at,omitempty"`
	UpdatedAt    sql.NullTime   `json:"updatedAt,omitempty" db:"updated_at,omitempty"`
}

//sql.NullString
//sql.NullTime
