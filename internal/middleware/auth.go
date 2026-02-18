package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/devaloi/restgo/internal/auth"
)

type claimsKey struct{}

// Auth returns middleware that validates a Bearer token and injects claims
// into the request context. Returns 401 if the token is missing or invalid.
func Auth(jwt *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"message":"missing or invalid authorization header"}}`))
				return
			}

			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := jwt.Validate(token)
			if err != nil {
				slog.Warn("auth token validation failed", "error", err, "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"message":"invalid or expired token"}}`))
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext extracts authenticated user claims from the request context.
func UserFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsKey{}).(*auth.Claims)
	return claims, ok
}
