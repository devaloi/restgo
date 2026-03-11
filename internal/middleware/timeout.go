package middleware

import (
	"context"
	"net/http"
	"time"
)

// Timeout returns middleware that sets a context deadline on each request.
// If the handler does not complete within the given duration, downstream
// code observing the context (e.g. database queries) will be cancelled.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
