package interceptors

import (
	"context"
	"net/http"

	"github.com/juleur/ecrpe/utils"
)

type UserIPAddressContextKey struct {
	name string
}

var userIPAddressCtxKey = &UserIPAddressContextKey{"userIPAddress"}

type IPAddress struct {
	string
}

// GetIPAddress decodes the share session cookie and packs the session into context
func GetIPAddress() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			IPAddress := IPAddress{utils.PrettifyIP(r.RemoteAddr)}
			ctx := context.WithValue(r.Context(), userIPAddressCtxKey, IPAddress)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ForIPAddress finds the user from the context. REQUIRES Middleware to have run.
func ForIPAddress(ctx context.Context) string {
	IP := ctx.Value(userIPAddressCtxKey).(IPAddress)
	return IP.string
}
