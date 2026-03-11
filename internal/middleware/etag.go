package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

// ETag adds ETag response headers and handles conditional GET requests.
// When a client sends If-None-Match matching the current ETag, it returns
// 304 Not Modified with no body, saving bandwidth on unchanged resources.
func ETag(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		ew := &etagWriter{
			ResponseWriter: w,
			buf:            make([]byte, 0, 512),
		}

		next.ServeHTTP(ew, r)

		if ew.status >= 200 && ew.status < 300 && len(ew.buf) > 0 {
			hash := sha256.Sum256(ew.buf)
			etag := `"` + hex.EncodeToString(hash[:8]) + `"`

			if r.Header.Get("If-None-Match") == etag {
				w.Header().Set("ETag", etag)
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("ETag", etag)
		}

		if !ew.wroteHeader {
			w.WriteHeader(ew.status)
		}
		_, _ = w.Write(ew.buf)
	})
}

type etagWriter struct {
	http.ResponseWriter
	buf         []byte
	status      int
	wroteHeader bool
}

func (w *etagWriter) WriteHeader(code int) {
	w.status = code
}

func (w *etagWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.buf = append(w.buf, b...)
	return len(b), nil
}
