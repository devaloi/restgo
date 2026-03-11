package middleware

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/devaloi/restgo/internal/domain"
)

// staleEntryTTL is how long an IP's bucket can be idle before cleanup removes it.
const staleEntryTTL = 10 * time.Minute

// cleanupInterval is the minimum time between successive cleanup sweeps.
const cleanupInterval = 5 * time.Minute

type bucket struct {
	tokens    float64
	lastCheck time.Time
}

// rateLimiter holds shared state for a rate limiter instance, including
// periodic cleanup of stale per-IP entries to prevent unbounded memory growth.
type rateLimiter struct {
	clients   sync.Map // IP → *bucket
	mutexes   sync.Map // IP → *sync.Mutex
	rate      float64
	limit     int
	lastClean time.Time
	cleanMu   sync.Mutex
}

// RateLimit returns middleware that limits requests per client IP using a
// token bucket algorithm. limit is the maximum requests per minute.
// Stale entries are periodically evicted to prevent memory leaks.
func RateLimit(limit int) func(http.Handler) http.Handler {
	rl := &rateLimiter{
		rate:      float64(limit) / domain.SecondsPerMinute,
		limit:     limit,
		lastClean: time.Now(),
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip == "" {
				ip = r.RemoteAddr
			}

			now := time.Now()
			val, _ := rl.clients.LoadOrStore(ip, &bucket{
				tokens:    float64(limit),
				lastCheck: now,
			})
			b := val.(*bucket)

			mu := rl.getMutex(ip)
			mu.Lock()

			elapsed := now.Sub(b.lastCheck).Seconds()
			b.tokens += elapsed * rl.rate
			if b.tokens > float64(limit) {
				b.tokens = float64(limit)
			}
			b.lastCheck = now

			if b.tokens < 1 {
				mu.Unlock()
				retryAfter := int((1 - b.tokens) / rl.rate)
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				http.Error(w, `{"error":{"message":"rate limit exceeded"}}`, http.StatusTooManyRequests)
				rl.maybeCleanup(now)
				return
			}

			b.tokens--
			mu.Unlock()

			rl.maybeCleanup(now)
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *rateLimiter) getMutex(ip string) *sync.Mutex {
	val, _ := rl.mutexes.LoadOrStore(ip, &sync.Mutex{})
	return val.(*sync.Mutex)
}

// maybeCleanup removes stale IP entries if enough time has passed since
// the last sweep. This prevents unbounded memory growth from transient clients.
func (rl *rateLimiter) maybeCleanup(now time.Time) {
	if now.Sub(rl.lastClean) < cleanupInterval {
		return
	}

	if !rl.cleanMu.TryLock() {
		return
	}
	defer rl.cleanMu.Unlock()

	// Double-check after acquiring lock
	if now.Sub(rl.lastClean) < cleanupInterval {
		return
	}

	rl.clients.Range(func(key, value any) bool {
		b := value.(*bucket)
		if now.Sub(b.lastCheck) > staleEntryTTL {
			rl.clients.Delete(key)
			rl.mutexes.Delete(key)
		}
		return true
	})

	rl.lastClean = now
}
