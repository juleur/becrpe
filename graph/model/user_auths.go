package model

import (
	"time"
)

type UserAuth struct {
	ID           int       `json:"id,omitempty" db:"id,omitempty"`
	UserAgent    string    `json:"userAgent,omitempty" db:"user_agent,omitempty"`
	IPAddress    string    `json:"ipAddress,omitempty" db:"ip_address,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty" db:"refresh_token,omitempty"`
	DeliveredAt  time.Time `json:"deliveredAt,omitempty" db:"delivered_at,omitempty"`
	IsRevoked    bool      `json:"isRevoked,omitempty" db:"is_revoked,omitempty"`
	RevokedAt    time.Time `json:"revokedAt,omitempty" db:"revoked_at,omitempty"`
	OnLogin      bool      `json:"onLogin,omitempty" db:"on_login,omitempty"`
	OnRefresh    bool      `json:"onRefresh,omitempty" db:"on_refresh,omitempty"`
	UserID       int       `json:"userId,omitempty" db:"user_id,omitempty"`
	Username     string    `json:"username,omitempty" db:"username,omitempty"`
	IsTeacher    bool      `json:"isTeacher,omitempty" db:"is_teacher,omitempty"`
}
