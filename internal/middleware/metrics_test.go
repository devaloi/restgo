package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsInstrument(t *testing.T) {
	m := NewMetrics()
	handler := m.Instrument(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make a few requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/articles", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Check metrics output
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "http_requests_total") {
		t.Error("metrics output should contain http_requests_total")
	}
	if !strings.Contains(body, `method="GET"`) {
		t.Error("metrics output should contain method label")
	}
	if !strings.Contains(body, `status="200"`) {
		t.Error("metrics output should contain status label")
	}
	if !strings.Contains(body, "http_request_duration_seconds") {
		t.Error("metrics output should contain http_request_duration_seconds")
	}
	if !strings.Contains(body, "http_requests_inflight") {
		t.Error("metrics output should contain http_requests_inflight")
	}
	if !strings.Contains(body, " 3") {
		t.Error("metrics output should show count of 3 for requests total")
	}
}

func TestMetricsNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/health", "/health"},
		{"/api/articles", "/api/articles"},
		{"/api/articles/550e8400-e29b-41d4-a716-446655440000", "/api/articles/{id}"},
		{"/api/auth/register", "/api/auth/register"},
		{"/", "/"},
	}

	for _, tt := range tests {
		got := normalizePath(tt.input)
		if got != tt.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
