package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devaloi/restgo/internal/config"
	"github.com/devaloi/restgo/internal/repository"
)

func setupServer(t *testing.T) (*httptest.Server, *repository.MockUserRepository, *repository.MockArticleRepository) {
	t.Helper()
	userRepo := repository.NewMockUserRepository()
	articleRepo := repository.NewMockArticleRepository()
	cfg := &config.Config{
		JWT:  config.JWTConfig{Secret: "test-secret", Expiry: time.Hour},
		CORS: config.CORSConfig{Origins: "*"},
		Rate: config.RateConfig{Limit: 1000},
	}
	h := New(cfg, userRepo, articleRepo)
	return httptest.NewServer(h), userRepo, articleRepo
}

func registerUser(t *testing.T, srv *httptest.Server, email, password, name string) (string, string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"email": email, "password": password, "name": name,
	})
	resp, err := http.Post(srv.URL+"/api/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result struct {
		Data struct {
			User  struct{ ID string } `json:"user"`
			Token string              `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Data.User.ID, result.Data.Token
}

// --- Registration Tests ---

func TestRegister(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{
		"email": "test@example.com", "password": "password123", "name": "Test User",
	})
	resp, err := http.Post(srv.URL+"/api/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result struct {
		Data struct {
			User  map[string]any `json:"user"`
			Token string         `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Data.Token == "" {
		t.Error("token should not be empty")
	}
	if result.Data.User["email"] != "test@example.com" {
		t.Errorf("email = %v, want test@example.com", result.Data.User["email"])
	}
}

func TestRegisterDuplicate(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	registerUser(t, srv, "dup@example.com", "password123", "User1")

	body, _ := json.Marshal(map[string]string{
		"email": "dup@example.com", "password": "password123", "name": "User2",
	})
	resp, err := http.Post(srv.URL+"/api/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestRegisterValidation(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"email": "", "password": "", "name": ""})
	resp, err := http.Post(srv.URL+"/api/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnprocessableEntity)
	}
}

// --- Login Tests ---

func TestLogin(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	registerUser(t, srv, "login@example.com", "password123", "Test")

	body, _ := json.Marshal(map[string]string{
		"email": "login@example.com", "password": "password123",
	})
	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Data.Token == "" {
		t.Error("token should not be empty")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	registerUser(t, srv, "wrong@example.com", "password123", "Test")

	body, _ := json.Marshal(map[string]string{
		"email": "wrong@example.com", "password": "wrongpassword",
	})
	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestLoginNonexistentUser(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{
		"email": "nonexistent@example.com", "password": "password123",
	})
	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

// --- Profile Tests ---

func TestGetProfile(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token := registerUser(t, srv, "profile@example.com", "password123", "Profile User")

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result struct {
		Data struct {
			Email string `json:"email"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Data.Email != "profile@example.com" {
		t.Errorf("email = %q, want %q", result.Data.Email, "profile@example.com")
	}
}

func TestGetProfileNoAuth(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/users/me", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestGetProfileExpiredToken(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/users/me", nil)
	req.Header.Set("Authorization", "Bearer expired.token.value")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

// --- Article CRUD Tests ---

func createArticle(t *testing.T, srv *httptest.Server, token, title, body string) string {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{"title": title, "body": body})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/articles", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create article failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create article status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result struct {
		Data struct{ ID string } `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Data.ID
}

func TestCreateArticle(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token := registerUser(t, srv, "author@example.com", "password123", "Author")

	payload, _ := json.Marshal(map[string]string{
		"title": "Test Article", "body": "Article body content",
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/articles", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestCreateArticleNoAuth(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	payload, _ := json.Marshal(map[string]string{
		"title": "Test", "body": "Body",
	})
	resp, err := http.Post(srv.URL+"/api/articles", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestGetArticle(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token := registerUser(t, srv, "reader@example.com", "password123", "Reader")
	id := createArticle(t, srv, token, "Read Me", "Content here")

	resp, err := http.Get(srv.URL + "/api/articles/" + id)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result struct {
		Data struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Data.Title != "Read Me" {
		t.Errorf("title = %q, want %q", result.Data.Title, "Read Me")
	}
}

func TestGetArticleNotFound(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/articles/nonexistent-id")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestUpdateArticle(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token := registerUser(t, srv, "editor@example.com", "password123", "Editor")
	id := createArticle(t, srv, token, "Original", "Original body")

	payload, _ := json.Marshal(map[string]string{
		"title": "Updated", "body": "Updated body",
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/articles/"+id, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result struct {
		Data struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Data.Title != "Updated" {
		t.Errorf("title = %q, want %q", result.Data.Title, "Updated")
	}
}

func TestUpdateArticleNotOwner(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token1 := registerUser(t, srv, "owner@example.com", "password123", "Owner")
	_, token2 := registerUser(t, srv, "other@example.com", "password123", "Other")
	id := createArticle(t, srv, token1, "Mine", "My article")

	payload, _ := json.Marshal(map[string]string{
		"title": "Hacked", "body": "Hacked body",
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/articles/"+id, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token2)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestDeleteArticle(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token := registerUser(t, srv, "deleter@example.com", "password123", "Deleter")
	id := createArticle(t, srv, token, "Delete Me", "To be deleted")

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/articles/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	// Verify it's gone
	resp2, err := http.Get(srv.URL + "/api/articles/" + id)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("deleted article status = %d, want %d", resp2.StatusCode, http.StatusNotFound)
	}
}

func TestDeleteArticleNotOwner(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token1 := registerUser(t, srv, "deleteowner@example.com", "password123", "Owner")
	_, token2 := registerUser(t, srv, "deleteother@example.com", "password123", "Other")
	id := createArticle(t, srv, token1, "Protected", "Can't delete")

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/articles/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

// --- List / Pagination Tests ---

func TestListArticles(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token := registerUser(t, srv, "lister@example.com", "password123", "Lister")
	for i := 0; i < 5; i++ {
		createArticle(t, srv, token, fmt.Sprintf("Article %d", i), "Body")
	}

	resp, err := http.Get(srv.URL + "/api/articles?page=1&per_page=3")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result struct {
		Data []map[string]any `json:"data"`
		Meta struct {
			Page       int `json:"page"`
			PerPage    int `json:"per_page"`
			Total      int `json:"total"`
			TotalPages int `json:"total_pages"`
		} `json:"meta"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Data) != 3 {
		t.Errorf("data length = %d, want 3", len(result.Data))
	}
	if result.Meta.Total != 5 {
		t.Errorf("total = %d, want 5", result.Meta.Total)
	}
	if result.Meta.TotalPages != 2 {
		t.Errorf("total_pages = %d, want 2", result.Meta.TotalPages)
	}
	if result.Meta.Page != 1 {
		t.Errorf("page = %d, want 1", result.Meta.Page)
	}
	if result.Meta.PerPage != 3 {
		t.Errorf("per_page = %d, want 3", result.Meta.PerPage)
	}
}

func TestListArticlesSearch(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token := registerUser(t, srv, "searcher@example.com", "password123", "Searcher")
	createArticle(t, srv, token, "Go Tutorial", "Learn Go programming")
	createArticle(t, srv, token, "Rust Tutorial", "Learn Rust programming")
	createArticle(t, srv, token, "Python Guide", "Learn Python programming")

	resp, err := http.Get(srv.URL + "/api/articles?search=Tutorial")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []map[string]any `json:"data"`
		Meta struct{ Total int }
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Meta.Total != 2 {
		t.Errorf("search total = %d, want 2", result.Meta.Total)
	}
}

func TestListArticlesByAuthor(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	uid1, token1 := registerUser(t, srv, "author1@example.com", "password123", "Author1")
	_, token2 := registerUser(t, srv, "author2@example.com", "password123", "Author2")
	createArticle(t, srv, token1, "Article by Author1", "Body")
	createArticle(t, srv, token1, "Another by Author1", "Body")
	createArticle(t, srv, token2, "Article by Author2", "Body")

	resp, err := http.Get(srv.URL + "/api/articles?author_id=" + uid1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []map[string]any `json:"data"`
		Meta struct{ Total int }
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Meta.Total != 2 {
		t.Errorf("author filter total = %d, want 2", result.Meta.Total)
	}
}

// --- Health Check ---

func TestHealthCheck(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("status = %q, want %q", body["status"], "ok")
	}
}
