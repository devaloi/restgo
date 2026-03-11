package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/devaloi/restgo/internal/domain"
	"github.com/devaloi/restgo/internal/middleware"
	"github.com/devaloi/restgo/internal/service"
)

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	svc *service.UserService
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

type authResponse struct {
	User  *domain.User `json:"user"`
	Token string       `json:"token"`
}

// Register handles POST /api/auth/register.
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	user, token, err := h.svc.Register(r.Context(), req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	JSON(w, http.StatusCreated, authResponse{User: user, Token: token})
}

// Login handles POST /api/auth/login.
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	user, token, err := h.svc.Login(r.Context(), req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, authResponse{User: user, Token: token})
}

// GetProfile handles GET /api/users/me.
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.UserFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.svc.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, user)
}

func handleServiceError(w http.ResponseWriter, err error) {
	var verr *domain.ValidationErrors
	if errors.As(err, &verr) {
		ValidationErr(w, verr)
		return
	}

	switch {
	case errors.Is(err, domain.ErrConflict):
		Error(w, http.StatusConflict, "resource already exists")
	case errors.Is(err, domain.ErrNotFound):
		Error(w, http.StatusNotFound, "resource not found")
	case errors.Is(err, domain.ErrUnauthorized):
		Error(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, domain.ErrForbidden):
		Error(w, http.StatusForbidden, "forbidden")
	default:
		slog.Error("unhandled service error", "error", err)
		Error(w, http.StatusInternalServerError, "internal server error")
	}
}
