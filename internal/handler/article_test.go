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

const testSecret = "test-secret-for-handler-tests"

func newArticleTestHandler() (*ArticleHandler, *repository.MockArticleRepository, *auth.JWTService) {
	repo := repository.NewMockArticleRepository()
	svc := service.NewArticleService(repo)
	jwt := auth.New(testSecret, time.Hour)
	return NewArticleHandler(svc), repo, jwt
}

// setupAuthMux wires a handler behind the auth middleware so the handler's
// middleware.UserFromContext call works correctly.
func setupAuthMux(jwt *auth.JWTService, method, pattern string, handlerFunc http.HandlerFunc) *http.ServeMux {
	mux := http.NewServeMux()
	authMW := middleware.Auth(jwt)
	mux.Handle(method+" "+pattern, authMW(handlerFunc))
	return mux
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestArticleHandler_Create_Success(t *testing.T) {
	h, _, jwt := newArticleTestHandler()
	token, _ := jwt.Generate("user-1", "alice@example.com")

	mux := setupAuthMux(jwt, "POST", "/api/articles", h.Create)

	body := mustMarshal(t, domain.CreateArticleRequest{Title: "My Article", Body: "Content here"})
	req := httptest.NewRequest(http.MethodPost, "/api/articles", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp envelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data map, got %T", resp.Data)
	}
	if data["title"] != "My Article" {
		t.Errorf("title = %q, want %q", data["title"], "My Article")
	}
	if data["author_id"] != "user-1" {
		t.Errorf("author_id = %q, want %q", data["author_id"], "user-1")
	}
}

func TestArticleHandler_Create_Unauthorized(t *testing.T) {
	h, _, jwt := newArticleTestHandler()

	mux := setupAuthMux(jwt, "POST", "/api/articles", h.Create)

	body := mustMarshal(t, domain.CreateArticleRequest{Title: "X", Body: "Y"})
	req := httptest.NewRequest(http.MethodPost, "/api/articles", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestArticleHandler_Create_InvalidBody(t *testing.T) {
	h, _, jwt := newArticleTestHandler()
	token, _ := jwt.Generate("user-1", "alice@example.com")

	mux := setupAuthMux(jwt, "POST", "/api/articles", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/api/articles", bytes.NewReader([]byte("not json")))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestArticleHandler_Create_ValidationError(t *testing.T) {
	h, _, jwt := newArticleTestHandler()
	token, _ := jwt.Generate("user-1", "alice@example.com")

	mux := setupAuthMux(jwt, "POST", "/api/articles", h.Create)

	body := mustMarshal(t, domain.CreateArticleRequest{Title: "", Body: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/articles", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestArticleHandler_GetByID_Success(t *testing.T) {
	h, repo, _ := newArticleTestHandler()

	repo.Create(context.Background(), &domain.Article{
		ID: "art-1", Title: "Test", Body: "Body", AuthorID: "user-1",
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/articles/{id}", h.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/articles/art-1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp envelope
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]any)
	if data["id"] != "art-1" {
		t.Errorf("id = %q, want %q", data["id"], "art-1")
	}
}

func TestArticleHandler_GetByID_NotFound(t *testing.T) {
	h, _, _ := newArticleTestHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/articles/{id}", h.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/articles/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestArticleHandler_List_Success(t *testing.T) {
	h, repo, _ := newArticleTestHandler()

	for i := 0; i < 3; i++ {
		repo.Create(context.Background(), &domain.Article{
			ID: "art-" + string(rune('a'+i)), Title: "Article", Body: "Body", AuthorID: "user-1",
		})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/articles", h.List)

	req := httptest.NewRequest(http.MethodGet, "/api/articles?page=1&per_page=10", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp domain.PaginatedResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Meta.Total != 3 {
		t.Errorf("total = %d, want 3", resp.Meta.Total)
	}
}

func TestArticleHandler_List_DefaultPagination(t *testing.T) {
	h, _, _ := newArticleTestHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/articles", h.List)

	req := httptest.NewRequest(http.MethodGet, "/api/articles", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp domain.PaginatedResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Meta.Page != 1 {
		t.Errorf("page = %d, want 1", resp.Meta.Page)
	}
	if resp.Meta.PerPage != domain.DefaultPageSize {
		t.Errorf("per_page = %d, want %d", resp.Meta.PerPage, domain.DefaultPageSize)
	}
}

func TestArticleHandler_Update_Success(t *testing.T) {
	h, repo, jwt := newArticleTestHandler()
	token, _ := jwt.Generate("user-1", "alice@example.com")

	repo.Create(context.Background(), &domain.Article{
		ID: "art-1", Title: "Old Title", Body: "Old Body", AuthorID: "user-1",
	})

	mux := setupAuthMux(jwt, "PUT", "/api/articles/{id}", h.Update)

	body := mustMarshal(t, domain.UpdateArticleRequest{Title: "New Title"})
	req := httptest.NewRequest(http.MethodPut, "/api/articles/art-1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp envelope
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]any)
	if data["title"] != "New Title" {
		t.Errorf("title = %q, want %q", data["title"], "New Title")
	}
}

func TestArticleHandler_Update_Forbidden(t *testing.T) {
	h, repo, jwt := newArticleTestHandler()
	token, _ := jwt.Generate("user-2", "bob@example.com")

	repo.Create(context.Background(), &domain.Article{
		ID: "art-1", Title: "Title", Body: "Body", AuthorID: "user-1",
	})

	mux := setupAuthMux(jwt, "PUT", "/api/articles/{id}", h.Update)

	body := mustMarshal(t, domain.UpdateArticleRequest{Title: "Hijacked"})
	req := httptest.NewRequest(http.MethodPut, "/api/articles/art-1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestArticleHandler_Delete_Success(t *testing.T) {
	h, repo, jwt := newArticleTestHandler()
	token, _ := jwt.Generate("user-1", "alice@example.com")

	repo.Create(context.Background(), &domain.Article{
		ID: "art-1", Title: "Title", Body: "Body", AuthorID: "user-1",
	})

	mux := setupAuthMux(jwt, "DELETE", "/api/articles/{id}", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/articles/art-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestArticleHandler_Delete_NotFound(t *testing.T) {
	h, _, jwt := newArticleTestHandler()
	token, _ := jwt.Generate("user-1", "alice@example.com")

	mux := setupAuthMux(jwt, "DELETE", "/api/articles/{id}", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/articles/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestArticleHandler_Delete_Forbidden(t *testing.T) {
	h, repo, jwt := newArticleTestHandler()
	token, _ := jwt.Generate("user-2", "bob@example.com")

	repo.Create(context.Background(), &domain.Article{
		ID: "art-1", Title: "Title", Body: "Body", AuthorID: "user-1",
	})

	mux := setupAuthMux(jwt, "DELETE", "/api/articles/{id}", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/articles/art-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
