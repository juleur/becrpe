package interceptors

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"github.com/juleur/ecrpe/customhttp"
	"github.com/juleur/ecrpe/graph/model"
)

// JWTContextKey struct
type JWTContextKey struct {
	name string
}

var userJWTCtxKey = &JWTContextKey{"userJWT"}

// HttpErrorResponse
type HttpErrorResponse struct {
	Message    string
	StatusCode int
	StatusText string
}

// User struct
type User struct {
	Username          string
	UserID            int
	IsAuth            bool
	HttpErrorResponse HttpErrorResponse
}

// JWTCheck decodes the share session cookie and packs the session into context
func JWTCheck(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userJWT := r.Header.Get("Authorization")
			// Check if header has bearer jwt
			if userJWT == "" {
				user := User{HttpErrorResponse: HttpErrorResponse{
					Message:    "Oops, une erreur est survenue, veuillez vous réauthentifier",
					StatusCode: http.StatusUnauthorized,
					StatusText: http.StatusText(http.StatusUnauthorized),
				}}
				ctx := context.WithValue(r.Context(), userJWTCtxKey, &user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			pl := model.CustomPayload{}
			signature := jwt.NewHS512([]byte(secretKey))
			// Validating alg
			if _, err := jwt.Verify([]byte(strings.Split(userJWT, " ")[1]), signature, &pl, jwt.ValidateHeader); err != nil {
				user := User{HttpErrorResponse: HttpErrorResponse{
					Message:    "Oops, une erreur est survenue, veuillez vous réauthentifier",
					StatusCode: http.StatusUnauthorized,
					StatusText: http.StatusText(http.StatusUnauthorized),
				}}
				ctx := context.WithValue(r.Context(), userJWTCtxKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			expValidator := jwt.ExpirationTimeValidator(time.Now())
			issuerValidator := jwt.IssuerValidator("https://rf.ecrpe.fr")
			validatePayload := jwt.ValidatePayload(&pl.Payload, issuerValidator, expValidator)
			// Split "bearer" from JWT
			// Validating claims
			if _, err := jwt.Verify([]byte(strings.Split(userJWT, " ")[1]), signature, &pl, validatePayload); err != nil {
				switch err {
				case jwt.ErrExpValidation:
					user := User{
						UserID: pl.UserID,
						HttpErrorResponse: HttpErrorResponse{
							StatusCode: customhttp.StatusTokenExpired,
							StatusText: customhttp.StatusText(customhttp.StatusTokenExpired),
						},
					}
					ctx := context.WithValue(r.Context(), userJWTCtxKey, user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				default:
					user := User{HttpErrorResponse: HttpErrorResponse{
						Message:    "Oops, une erreur est survenue, veuillez vous réauthentifier",
						StatusCode: http.StatusUnauthorized,
						StatusText: http.StatusText(http.StatusUnauthorized),
					}}
					ctx := context.WithValue(r.Context(), userJWTCtxKey, user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
			user := User{Username: pl.Username, UserID: pl.UserID, IsAuth: true}
			ctx := context.WithValue(r.Context(), userJWTCtxKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ForUserContext finds the user from the context. REQUIRES Middleware to have run.
func ForUserContext(ctx context.Context) User {
	raw, _ := ctx.Value(userJWTCtxKey).(User)
	return raw
}
