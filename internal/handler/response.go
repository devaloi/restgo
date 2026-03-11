package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/devaloi/restgo/internal/domain"
)

type envelope struct {
	Data any `json:"data"`
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Message string                   `json:"message"`
	Details []domain.ValidationError `json:"details,omitempty"`
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope{Data: data}); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// Error writes a JSON error response.
func Error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorBody{
		Error: errorDetail{Message: message},
	}); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

// ValidationErr writes a JSON validation error response.
func ValidationErr(w http.ResponseWriter, verr *domain.ValidationErrors) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := json.NewEncoder(w).Encode(errorBody{
		Error: errorDetail{
			Message: "validation failed",
			Details: verr.Errors,
		},
	}); err != nil {
		slog.Error("failed to encode validation error response", "error", err)
	}
}

// Paginated writes a JSON paginated response.
func Paginated(w http.ResponseWriter, data any, meta domain.PaginationMeta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(domain.PaginatedResponse{
		Data: data,
		Meta: meta,
	}); err != nil {
		slog.Error("failed to encode paginated response", "error", err)
	}
}

// decodeJSON reads and decodes a JSON request body into v.
// Returns false and writes an error response if the content type is wrong
// or decoding fails (including body-too-large from MaxBytesReader).
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	ct := r.Header.Get("Content-Type")
	if ct != "" && ct != "application/json" {
		Error(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return false
	}

	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		if err.Error() == "http: request body too large" {
			Error(w, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		Error(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}
