package middleware

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks HTTP request metrics for Prometheus exposition.
type Metrics struct {
	requestsTotal   sync.Map // key: "method:status" → *atomic.Int64
	requestDuration sync.Map // key: "method:path" → *durationBuckets
	inflightGauge   atomic.Int64
}

type durationBuckets struct {
	mu      sync.Mutex
	count   int64
	sum     float64
	buckets []bucketEntry
}

type bucketEntry struct {
	le    float64
	count int64
}

var defaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// NewMetrics creates a new Metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Instrument returns middleware that records request metrics.
func (m *Metrics) Instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.inflightGauge.Add(1)
		defer m.inflightGauge.Add(-1)

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		duration := time.Since(start).Seconds()

		// Count by method:status
		key := fmt.Sprintf("%s:%d", r.Method, sw.status)
		val, _ := m.requestsTotal.LoadOrStore(key, &atomic.Int64{})
		val.(*atomic.Int64).Add(1)

		// Duration by method:path
		pathKey := fmt.Sprintf("%s:%s", r.Method, normalizePath(r.URL.Path))
		durVal, _ := m.requestDuration.LoadOrStore(pathKey, newDurationBuckets())
		db := durVal.(*durationBuckets)
		db.observe(duration)
	})
}

func newDurationBuckets() *durationBuckets {
	entries := make([]bucketEntry, len(defaultBuckets))
	for i, b := range defaultBuckets {
		entries[i] = bucketEntry{le: b}
	}
	return &durationBuckets{buckets: entries}
}

func (d *durationBuckets) observe(val float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.count++
	d.sum += val
	for i := range d.buckets {
		if val <= d.buckets[i].le {
			d.buckets[i].count++
		}
	}
}

// normalizePath collapses UUID-like path segments to {id} for cardinality control.
func normalizePath(path string) string {
	segments := splitPath(path)
	for i, s := range segments {
		if looksLikeID(s) {
			segments[i] = "{id}"
		}
	}
	result := ""
	for _, s := range segments {
		result += "/" + s
	}
	if result == "" {
		return "/"
	}
	return result
}

// looksLikeID returns true for UUID-like or long hex strings that are
// likely dynamic path parameters rather than fixed route segments.
func looksLikeID(s string) bool {
	if len(s) < 8 {
		return false
	}
	for _, ch := range s {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && ch != '-' {
			return false
		}
	}
	return true
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, ch := range path {
		if ch == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// Handler returns an HTTP handler that serves Prometheus metrics.
func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		// http_requests_total
		fmt.Fprintln(w, "# HELP http_requests_total Total number of HTTP requests.")
		fmt.Fprintln(w, "# TYPE http_requests_total counter")
		var totalKeys []string
		m.requestsTotal.Range(func(key, _ any) bool {
			totalKeys = append(totalKeys, key.(string))
			return true
		})
		sort.Strings(totalKeys)
		for _, key := range totalKeys {
			val, _ := m.requestsTotal.Load(key)
			method, status := parseMethodStatus(key)
			fmt.Fprintf(w, "http_requests_total{method=%q,status=%q} %d\n", method, status, val.(*atomic.Int64).Load())
		}

		// http_request_duration_seconds
		fmt.Fprintln(w, "# HELP http_request_duration_seconds HTTP request duration in seconds.")
		fmt.Fprintln(w, "# TYPE http_request_duration_seconds histogram")
		var durKeys []string
		m.requestDuration.Range(func(key, _ any) bool {
			durKeys = append(durKeys, key.(string))
			return true
		})
		sort.Strings(durKeys)
		for _, key := range durKeys {
			val, _ := m.requestDuration.Load(key)
			db := val.(*durationBuckets)
			method, path := parseMethodStatus(key)
			db.mu.Lock()
			for _, b := range db.buckets {
				fmt.Fprintf(w, "http_request_duration_seconds_bucket{method=%q,path=%q,le=\"%.3f\"} %d\n",
					method, path, b.le, b.count)
			}
			fmt.Fprintf(w, "http_request_duration_seconds_bucket{method=%q,path=%q,le=\"+Inf\"} %d\n",
				method, path, db.count)
			fmt.Fprintf(w, "http_request_duration_seconds_sum{method=%q,path=%q} %.6f\n", method, path, db.sum)
			fmt.Fprintf(w, "http_request_duration_seconds_count{method=%q,path=%q} %d\n", method, path, db.count)
			db.mu.Unlock()
		}

		// http_requests_inflight
		fmt.Fprintln(w, "# HELP http_requests_inflight Current number of in-flight requests.")
		fmt.Fprintln(w, "# TYPE http_requests_inflight gauge")
		fmt.Fprintf(w, "http_requests_inflight %d\n", m.inflightGauge.Load())
	}
}

func parseMethodStatus(key string) (string, string) {
	for i, ch := range key {
		if ch == ':' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}
