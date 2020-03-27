package interceptors

import (
	"context"
	"net/http"
)

type UserAgentContextKey struct{ value string }

var userAgentCtxKey = &UserAgentContextKey{"userAgent"}

type UserAgent struct{ string }

// GetUserAgent
func GetUserAgent() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent := UserAgent{r.UserAgent()}
			ctx := context.WithValue(r.Context(), userAgentCtxKey, userAgent)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ForUserAgent finds the user from the context. REQUIRES Middleware to have run.
func ForUserAgent(ctx context.Context) string {
	userAgent := ctx.Value(userAgentCtxKey).(UserAgent)
	return userAgent.string
}
