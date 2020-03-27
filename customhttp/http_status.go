package customhttp

import (
	"net/http"
)

// more accurate than 401 (Unauthorized)
const StatusTokenExpired = 498

var statusText = map[int]string{
	StatusTokenExpired: "Token Expired",
}

func StatusText(code int) string {
	if code == StatusTokenExpired {
		return statusText[code]
	}
	return http.StatusText(code)
}
