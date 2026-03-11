package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devaloi/restgo/internal/auth"
	"github.com/devaloi/restgo/internal/domain"
	"github.com/devaloi/restgo/internal/middleware"
	"github.com/devaloi/restgo/internal/repository"
	"github.com/devaloi/restgo/internal/service"
)

func newUserTestHandler() (*UserHandler, *repository.MockUserRepository, *auth.JWTService) {
	repo := repository.NewMockUserRepository()
	jwt := auth.New(testSecret, time.Hour)
	svc := service.NewUserService(repo, jwt)
	return NewUserHandler(svc), repo, jwt
}

func registerTestUser(t *testing.T, h *UserHandler) (userID, token string) {
	t.Helper()

	body, _ := json.Marshal(domain.CreateUserRequest{
		Email:    "alice@example.com",
		Password: "securepass123",
		Name:     "Alice",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp envelope
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]any)
	user := data["user"].(map[string]any)
	return user["id"].(string), data["token"].(string)
}

func TestUserHandler_Register_Success(t *testing.T) {
	h, _, _ := newUserTestHandler()

	body, _ := json.Marshal(domain.CreateUserRequest{
		Email:    "alice@example.com",
		Password: "securepass123",
		Name:     "Alice",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp envelope
	json.NewDecoder(rec.Body).Decode(&resp)
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data map, got %T", resp.Data)
	}
	if data["token"] == nil || data["token"] == "" {
		t.Error("expected token in response")
	}
	user := data["user"].(map[string]any)
	if user["email"] != "alice@example.com" {
		t.Errorf("email = %q, want %q", user["email"], "alice@example.com")
	}
}

func TestUserHandler_Register_InvalidJSON(t *testing.T) {
	h, _, _ := newUserTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()
	h.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_Register_ValidationError(t *testing.T) {
	h, _, _ := newUserTestHandler()

	body, _ := json.Marshal(domain.CreateUserRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Register(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestUserHandler_Register_DuplicateEmail(t *testing.T) {
	h, _, _ := newUserTestHandler()

	body, _ := json.Marshal(domain.CreateUserRequest{
		Email: "alice@example.com", Password: "securepass123", Name: "Alice",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first register: status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	h.Register(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("duplicate register: status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestUserHandler_Login_Success(t *testing.T) {
	h, _, _ := newUserTestHandler()
	registerTestUser(t, h)

	body, _ := json.Marshal(domain.LoginRequest{
		Email:    "alice@example.com",
		Password: "securepass123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp envelope
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]any)
	if data["token"] == nil || data["token"] == "" {
		t.Error("expected token in login response")
	}
}

func TestUserHandler_Login_InvalidJSON(t *testing.T) {
	h, _, _ := newUserTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte("{bad")))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_Login_WrongPassword(t *testing.T) {
	h, _, _ := newUserTestHandler()
	registerTestUser(t, h)

	body, _ := json.Marshal(domain.LoginRequest{
		Email:    "alice@example.com",
		Password: "wrongpassword",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestUserHandler_Login_UserNotFound(t *testing.T) {
	h, _, _ := newUserTestHandler()

	body, _ := json.Marshal(domain.LoginRequest{
		Email:    "nobody@example.com",
		Password: "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestUserHandler_Login_ValidationError(t *testing.T) {
	h, _, _ := newUserTestHandler()

	body, _ := json.Marshal(domain.LoginRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestUserHandler_GetProfile_Success(t *testing.T) {
	h, _, jwt := newUserTestHandler()
	userID, token := registerTestUser(t, h)

	mux := http.NewServeMux()
	authMW := middleware.Auth(jwt)
	mux.Handle("GET /api/users/me", authMW(http.HandlerFunc(h.GetProfile)))

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp envelope
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]any)
	if data["id"] != userID {
		t.Errorf("id = %q, want %q", data["id"], userID)
	}
}

func TestUserHandler_GetProfile_Unauthorized(t *testing.T) {
	h, _, jwt := newUserTestHandler()

	mux := http.NewServeMux()
	authMW := middleware.Auth(jwt)
	mux.Handle("GET /api/users/me", authMW(http.HandlerFunc(h.GetProfile)))

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandleServiceError_Mapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not found", domain.ErrNotFound, http.StatusNotFound},
		{"conflict", domain.ErrConflict, http.StatusConflict},
		{"unauthorized", domain.ErrUnauthorized, http.StatusUnauthorized},
		{"forbidden", domain.ErrForbidden, http.StatusForbidden},
		{"validation", &domain.ValidationErrors{
			Errors: []domain.ValidationError{{Field: "x", Message: "bad"}},
		}, http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handleServiceError(rec, tt.err)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestResponseHelpers(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		rec := httptest.NewRecorder()
		JSON(rec, http.StatusOK, map[string]string{"key": "value"})

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}
	})

	t.Run("Error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		Error(rec, http.StatusBadRequest, "bad input")

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var resp errorBody
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Error.Message != "bad input" {
			t.Errorf("message = %q, want %q", resp.Error.Message, "bad input")
		}
	})

	t.Run("ValidationErr", func(t *testing.T) {
		rec := httptest.NewRecorder()
		verr := &domain.ValidationErrors{
			Errors: []domain.ValidationError{{Field: "email", Message: "required"}},
		}
		ValidationErr(rec, verr)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
		}

		var resp errorBody
		json.NewDecoder(rec.Body).Decode(&resp)
		if len(resp.Error.Details) != 1 {
			t.Errorf("details count = %d, want 1", len(resp.Error.Details))
		}
	})

	t.Run("Paginated", func(t *testing.T) {
		rec := httptest.NewRecorder()
		Paginated(rec, []string{"a", "b"}, domain.PaginationMeta{Page: 1, PerPage: 10, Total: 2, TotalPages: 1})

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var resp domain.PaginatedResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Meta.Total != 2 {
			t.Errorf("total = %d, want 2", resp.Meta.Total)
		}
	})
}

// Ensure decodeJSON returns false and 400 for non-JSON bodies.
func TestDecodeJSON_InvalidBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("not json")))

	var target domain.CreateArticleRequest
	ok := decodeJSON(rec, req, &target)

	if ok {
		t.Error("expected false for invalid JSON")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// Unused import guard — context is used in newArticleTestHandler seed calls.
var _ = context.Background
