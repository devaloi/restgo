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

// RateLimitConfig holds rate limiter parameters.
type RateLimitConfig struct {
	Limit int // requests per minute
	Burst int // max burst size (bucket capacity)
}

// RateLimit returns middleware that limits requests per client IP using a
// token bucket algorithm. limit is the maximum requests per minute.
// burst controls the maximum bucket capacity (defaults to limit if zero).
func RateLimit(limit, burst int) func(http.Handler) http.Handler {
	if burst <= 0 {
		burst = limit
	}

	var clients sync.Map
	rate := float64(limit) / domain.SecondsPerMinute

	// Periodically clean up stale entries to prevent memory leaks.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			clients.Range(func(key, val any) bool {
				b := val.(*bucket)
				mu := getMutex(key.(string))
				mu.Lock()
				if now.Sub(b.lastCheck) > 10*time.Minute {
					clients.Delete(key)
					ipMutexes.Delete(key)
				}
				mu.Unlock()
				return true
			})
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip == "" {
				ip = r.RemoteAddr
			}

			now := time.Now()
			val, _ := clients.LoadOrStore(ip, &bucket{
				tokens:    float64(burst),
				lastCheck: now,
			})
			b := val.(*bucket)

			mu := getMutex(ip)
			mu.Lock()

			elapsed := now.Sub(b.lastCheck).Seconds()
			b.tokens += elapsed * rate
			if b.tokens > float64(burst) {
				b.tokens = float64(burst)
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
