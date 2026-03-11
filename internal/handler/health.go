package handler

import (
	"context"
	"net/http"
	"time"
)

// DBPinger is implemented by *sql.DB and used for health checks.
type DBPinger interface {
	PingContext(ctx context.Context) error
}

// HealthHandler handles the health check endpoint.
type HealthHandler struct {
	db DBPinger
}

// NewHealthHandler creates a HealthHandler. db may be nil when running
// with in-memory repositories.
func NewHealthHandler(db DBPinger) *HealthHandler {
	return &HealthHandler{db: db}
}

type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// Check handles GET /health.
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	checks := map[string]string{}

	if h.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := h.db.PingContext(ctx); err != nil {
			status = "degraded"
			checks["database"] = "unavailable"
		} else {
			checks["database"] = "ok"
		}
	} else {
		checks["database"] = "not_configured"
	}

	code := http.StatusOK
	if status != "ok" {
		code = http.StatusServiceUnavailable
	}

	JSON(w, code, healthResponse{
		Status: status,
		Checks: checks,
	})
}
