package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_CleanupStaleEntries(t *testing.T) {
	rl := &rateLimiter{
		rate:      10.0 / 60.0,
		limit:     10,
		lastClean: time.Now().Add(-cleanupInterval - time.Second),
	}

	now := time.Now()

	// Add a fresh entry and a stale entry
	rl.clients.Store("fresh-ip", &bucket{tokens: 10, lastCheck: now})
	rl.mutexes.Store("fresh-ip", &struct{}{})

	staleTime := now.Add(-staleEntryTTL - time.Second)
	rl.clients.Store("stale-ip", &bucket{tokens: 5, lastCheck: staleTime})
	rl.mutexes.Store("stale-ip", &struct{}{})

	rl.maybeCleanup(now)

	// Fresh entry should still exist
	if _, ok := rl.clients.Load("fresh-ip"); !ok {
		t.Error("fresh entry should not be evicted")
	}

	// Stale entry should be removed
	if _, ok := rl.clients.Load("stale-ip"); ok {
		t.Error("stale entry should be evicted")
	}
	if _, ok := rl.mutexes.Load("stale-ip"); ok {
		t.Error("stale mutex should be evicted")
	}
}

func TestRateLimiter_SkipsCleanupWhenRecent(t *testing.T) {
	rl := &rateLimiter{
		rate:      10.0 / 60.0,
		limit:     10,
		lastClean: time.Now(), // just cleaned
	}

	now := time.Now()

	staleTime := now.Add(-staleEntryTTL - time.Second)
	rl.clients.Store("stale-ip", &bucket{tokens: 5, lastCheck: staleTime})

	rl.maybeCleanup(now)

	// Entry should NOT be evicted because cleanup interval hasn't elapsed
	if _, ok := rl.clients.Load("stale-ip"); !ok {
		t.Error("cleanup should be skipped — interval not elapsed")
	}
}

func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	handler := RateLimit(1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First IP uses its token
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "10.0.0.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("IP1 first request: got %d, want %d", rec1.Code, http.StatusOK)
	}

	// Second IP should still have tokens
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "10.0.0.2:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("IP2 first request: got %d, want %d", rec2.Code, http.StatusOK)
	}

	// First IP should be rate limited
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "10.0.0.1:1234"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusTooManyRequests {
		t.Errorf("IP1 second request: got %d, want %d", rec3.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimiter_FallbackRemoteAddr(t *testing.T) {
	handler := RateLimit(1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// RemoteAddr without port
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got %d, want %d", rec.Code, http.StatusOK)
	}
}
