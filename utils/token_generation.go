package utils

import (
	"math/rand"
	"strings"
	"time"
)

// HexKeyGenerator generate refresh token
func HexKeyGenerator(nb int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyz"
	sb := strings.Builder{}
	sb.Grow(nb)
	for ; nb > 0; nb-- {
		sb.WriteByte(letterBytes[rand.Intn(len(letterBytes)-1)])
	}
	return sb.String()
}
