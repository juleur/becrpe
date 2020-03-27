package model

type Token struct {
	Jwt          string `json:"jwt"`
	RefreshToken string `json:"refreshToken"`
}
