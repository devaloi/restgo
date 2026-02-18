package middleware

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/devaloi/restgo/internal/domain"
)

type bucket struct {
	tokens    float64
	lastCheck time.Time
}

// RateLimit returns middleware that limits requests per client IP using a
// token bucket algorithm. limit is the maximum requests per minute.
func RateLimit(limit int) func(http.Handler) http.Handler {
	var clients sync.Map
	// Convert per-minute limit to a per-second refill rate. Each second that
	// elapses adds (limit / 60) tokens back to the bucket, up to the maximum.
	rate := float64(limit) / domain.SecondsPerMinute

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip == "" {
				ip = r.RemoteAddr
			}

			now := time.Now()
			val, _ := clients.LoadOrStore(ip, &bucket{
				tokens:    float64(limit),
				lastCheck: now,
			})
			b := val.(*bucket)

			// Synchronize per-bucket access via simple CAS-style with sync.Map
			// For correctness under -race, we use a mutex embedded approach.
			// Since sync.Map values are pointers, we guard with a global lock per IP.
			mu := getMutex(ip)
			mu.Lock()

			elapsed := now.Sub(b.lastCheck).Seconds()
			b.tokens += elapsed * rate
			if b.tokens > float64(limit) {
				b.tokens = float64(limit)
			}
			b.lastCheck = now

			if b.tokens < 1 {
				mu.Unlock()
				retryAfter := int((1 - b.tokens) / rate)
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				http.Error(w, `{"error":{"message":"rate limit exceeded"}}`, http.StatusTooManyRequests)
				return
			}

			b.tokens--
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

var (
	ipMutexes sync.Map
)

func getMutex(ip string) *sync.Mutex {
	val, _ := ipMutexes.LoadOrStore(ip, &sync.Mutex{})
	return val.(*sync.Mutex)
}
