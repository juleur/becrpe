package model

import "github.com/gbrlsnchs/jwt/v3"

// CustomPayload struct
type CustomPayload struct {
	jwt.Payload
	Username string `json:"username"`
	UserID   int    `json:"userId"`
	Teacher  bool   `json:"teacher"`
}
