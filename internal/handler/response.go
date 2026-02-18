package handler

import (
	"encoding/json"
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
	Errors  []domain.ValidationError `json:"errors,omitempty"`
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope{Data: data})
}

// Error writes a JSON error response.
func Error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorBody{
		Error: errorDetail{Message: message},
	})
}

// ValidationErr writes a JSON validation error response.
func ValidationErr(w http.ResponseWriter, verr *domain.ValidationErrors) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(errorBody{
		Error: errorDetail{
			Message: "validation error",
			Errors:  verr.Errors,
		},
	})
}

// Paginated writes a JSON paginated response.
func Paginated(w http.ResponseWriter, data any, meta domain.PaginationMeta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(domain.PaginatedResponse{
		Data: data,
		Meta: meta,
	})
}
