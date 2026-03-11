package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// --- helpers ---

type apiError struct {
	Error struct {
		Message string `json:"message"`
		Details []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"details"`
	} `json:"error"`
}

func doRequest(t *testing.T, method, url, token string, body any) (*http.Response, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, data
}

// --- Full Flow Integration Test ---

func TestIntegrationFullFlow(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	// 1. Register
	resp, body := doRequest(t, http.MethodPost, srv.URL+"/api/auth/register", "", map[string]string{
		"email": "flow@example.com", "password": "password123", "name": "Flow User",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: status=%d body=%s", resp.StatusCode, body)
	}
	var regResult struct {
		Data struct {
			User  struct{ ID string } `json:"user"`
			Token string              `json:"token"`
		} `json:"data"`
	}
	json.Unmarshal(body, &regResult)
	token := regResult.Data.Token
	if token == "" {
		t.Fatal("register: empty token")
	}

	// 2. Login
	resp, body = doRequest(t, http.MethodPost, srv.URL+"/api/auth/login", "", map[string]string{
		"email": "flow@example.com", "password": "password123",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: status=%d body=%s", resp.StatusCode, body)
	}
	var loginResult struct {
		Data struct{ Token string } `json:"data"`
	}
	json.Unmarshal(body, &loginResult)
	token = loginResult.Data.Token
	if token == "" {
		t.Fatal("login: empty token")
	}

	// 3. Create article
	resp, body = doRequest(t, http.MethodPost, srv.URL+"/api/articles", token, map[string]string{
		"title": "Integration Test Article", "body": "This is the body",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create article: status=%d body=%s", resp.StatusCode, body)
	}
	var createResult struct {
		Data struct{ ID string } `json:"data"`
	}
	json.Unmarshal(body, &createResult)
	articleID := createResult.Data.ID
	if articleID == "" {
		t.Fatal("create article: empty ID")
	}

	// 4. List articles — verify article is present
	resp, body = doRequest(t, http.MethodGet, srv.URL+"/api/articles", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status=%d", resp.StatusCode)
	}
	var listResult struct {
		Data []struct{ ID string } `json:"data"`
		Meta struct{ Total int }   `json:"meta"`
	}
	json.Unmarshal(body, &listResult)
	if listResult.Meta.Total != 1 {
		t.Fatalf("list: expected total=1, got %d", listResult.Meta.Total)
	}

	// 5. Get by ID
	resp, body = doRequest(t, http.MethodGet, srv.URL+"/api/articles/"+articleID, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get by id: status=%d", resp.StatusCode)
	}
	var getResult struct {
		Data struct{ Title string } `json:"data"`
	}
	json.Unmarshal(body, &getResult)
	if getResult.Data.Title != "Integration Test Article" {
		t.Fatalf("get by id: title=%q", getResult.Data.Title)
	}

	// 6. Update
	resp, body = doRequest(t, http.MethodPut, srv.URL+"/api/articles/"+articleID, token, map[string]string{
		"title": "Updated Title", "body": "Updated body",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update: status=%d body=%s", resp.StatusCode, body)
	}
	var updateResult struct {
		Data struct{ Title string } `json:"data"`
	}
	json.Unmarshal(body, &updateResult)
	if updateResult.Data.Title != "Updated Title" {
		t.Fatalf("update: title=%q", updateResult.Data.Title)
	}

	// 7. Delete
	resp, _ = doRequest(t, http.MethodDelete, srv.URL+"/api/articles/"+articleID, token, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: status=%d", resp.StatusCode)
	}

	// 8. List — verify gone
	resp, body = doRequest(t, http.MethodGet, srv.URL+"/api/articles", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list after delete: status=%d", resp.StatusCode)
	}
	json.Unmarshal(body, &listResult)
	if listResult.Meta.Total != 0 {
		t.Fatalf("list after delete: expected total=0, got %d", listResult.Meta.Total)
	}
}

// --- Auth Protection Tests ---

func TestIntegrationAuthProtection(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	// 401 without token: create article
	resp, _ := doRequest(t, http.MethodPost, srv.URL+"/api/articles", "", map[string]string{
		"title": "No Auth", "body": "Body",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("create without token: status=%d, want 401", resp.StatusCode)
	}

	// 401 without token: update article
	resp, _ = doRequest(t, http.MethodPut, srv.URL+"/api/articles/some-id", "", map[string]string{
		"title": "No Auth",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("update without token: status=%d, want 401", resp.StatusCode)
	}

	// 401 without token: delete article
	resp, _ = doRequest(t, http.MethodDelete, srv.URL+"/api/articles/some-id", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("delete without token: status=%d, want 401", resp.StatusCode)
	}

	// 401 without token: get profile
	resp, _ = doRequest(t, http.MethodGet, srv.URL+"/api/users/me", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("profile without token: status=%d, want 401", resp.StatusCode)
	}

	// 403: wrong owner tries to update/delete
	_, token1 := registerUser(t, srv, "owner@auth.test", "password123", "Owner")
	_, token2 := registerUser(t, srv, "other@auth.test", "password123", "Other")
	articleID := createArticle(t, srv, token1, "Owned Article", "Body")

	resp, _ = doRequest(t, http.MethodPut, srv.URL+"/api/articles/"+articleID, token2, map[string]string{
		"title": "Hijack", "body": "Hijacked",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("update wrong owner: status=%d, want 403", resp.StatusCode)
	}

	resp, _ = doRequest(t, http.MethodDelete, srv.URL+"/api/articles/"+articleID, token2, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("delete wrong owner: status=%d, want 403", resp.StatusCode)
	}
}

// --- Pagination Test ---

func TestIntegrationPagination(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token := registerUser(t, srv, "paginator@test.com", "password123", "Paginator")

	// Create 25 articles
	for i := 0; i < 25; i++ {
		createArticle(t, srv, token, fmt.Sprintf("Article %02d", i), "Body content")
	}

	// Page 1, per_page=10
	resp, body := doRequest(t, http.MethodGet, srv.URL+"/api/articles?page=1&per_page=10", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page 1: status=%d", resp.StatusCode)
	}
	var page1 struct {
		Data []json.RawMessage `json:"data"`
		Meta struct {
			Page       int `json:"page"`
			PerPage    int `json:"per_page"`
			Total      int `json:"total"`
			TotalPages int `json:"total_pages"`
		} `json:"meta"`
	}
	json.Unmarshal(body, &page1)

	if len(page1.Data) != 10 {
		t.Errorf("page 1: got %d items, want 10", len(page1.Data))
	}
	if page1.Meta.Total != 25 {
		t.Errorf("page 1: total=%d, want 25", page1.Meta.Total)
	}
	if page1.Meta.TotalPages != 3 {
		t.Errorf("page 1: total_pages=%d, want 3", page1.Meta.TotalPages)
	}
	if page1.Meta.Page != 1 {
		t.Errorf("page 1: page=%d, want 1", page1.Meta.Page)
	}
	if page1.Meta.PerPage != 10 {
		t.Errorf("page 1: per_page=%d, want 10", page1.Meta.PerPage)
	}

	// Page 3 should have 5 items
	resp, body = doRequest(t, http.MethodGet, srv.URL+"/api/articles?page=3&per_page=10", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page 3: status=%d", resp.StatusCode)
	}
	var page3 struct {
		Data []json.RawMessage   `json:"data"`
		Meta struct{ Total int } `json:"meta"`
	}
	json.Unmarshal(body, &page3)
	if len(page3.Data) != 5 {
		t.Errorf("page 3: got %d items, want 5", len(page3.Data))
	}

	// Page beyond range
	resp, body = doRequest(t, http.MethodGet, srv.URL+"/api/articles?page=100&per_page=10", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page 100: status=%d", resp.StatusCode)
	}
	var pageEmpty struct {
		Data []json.RawMessage `json:"data"`
	}
	json.Unmarshal(body, &pageEmpty)
	if len(pageEmpty.Data) != 0 {
		t.Errorf("page 100: got %d items, want 0", len(pageEmpty.Data))
	}
}

// --- Search Test ---

func TestIntegrationSearch(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token := registerUser(t, srv, "searcher@test.com", "password123", "Searcher")
	createArticle(t, srv, token, "Learning Go", "A guide to Go programming")
	createArticle(t, srv, token, "Learning Rust", "A guide to Rust programming")
	createArticle(t, srv, token, "Python Basics", "Introduction to Python")
	createArticle(t, srv, token, "Go Advanced", "Advanced Go patterns")

	// Search for "Go" — should match 2 by title and 1 by body (3 total? no — "Go" in title of 2, "Go" in body of 1 more)
	resp, body := doRequest(t, http.MethodGet, srv.URL+"/api/articles?search=Go", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search: status=%d", resp.StatusCode)
	}
	var result struct {
		Data []struct{ Title string } `json:"data"`
		Meta struct{ Total int }      `json:"meta"`
	}
	json.Unmarshal(body, &result)
	// "Learning Go" (title), "Go Advanced" (title), "A guide to Go programming" (body of Learning Go - already counted)
	if result.Meta.Total < 2 {
		t.Errorf("search 'Go': total=%d, want >=2", result.Meta.Total)
	}

	// Search for "Python"
	resp, body = doRequest(t, http.MethodGet, srv.URL+"/api/articles?search=Python", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search python: status=%d", resp.StatusCode)
	}
	json.Unmarshal(body, &result)
	if result.Meta.Total != 1 {
		t.Errorf("search 'Python': total=%d, want 1", result.Meta.Total)
	}

	// Search for nonexistent
	resp, body = doRequest(t, http.MethodGet, srv.URL+"/api/articles?search=Haskell", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search haskell: status=%d", resp.StatusCode)
	}
	json.Unmarshal(body, &result)
	if result.Meta.Total != 0 {
		t.Errorf("search 'Haskell': total=%d, want 0", result.Meta.Total)
	}
}

// --- Validation Error Tests ---

func TestIntegrationValidationErrors(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token := registerUser(t, srv, "validator@test.com", "password123", "Validator")

	tests := []struct {
		name       string
		method     string
		path       string
		token      string
		body       map[string]string
		wantStatus int
		wantField  string
	}{
		{
			name:       "register: missing all fields",
			method:     http.MethodPost,
			path:       "/api/auth/register",
			body:       map[string]string{"email": "", "password": "", "name": ""},
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "email",
		},
		{
			name:       "register: invalid email format",
			method:     http.MethodPost,
			path:       "/api/auth/register",
			body:       map[string]string{"email": "notanemail", "password": "password123", "name": "Test"},
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "email",
		},
		{
			name:       "register: short password",
			method:     http.MethodPost,
			path:       "/api/auth/register",
			body:       map[string]string{"email": "short@test.com", "password": "short", "name": "Test"},
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "password",
		},
		{
			name:       "register: missing name",
			method:     http.MethodPost,
			path:       "/api/auth/register",
			body:       map[string]string{"email": "noname@test.com", "password": "password123", "name": ""},
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "name",
		},
		{
			name:       "login: missing email",
			method:     http.MethodPost,
			path:       "/api/auth/login",
			body:       map[string]string{"email": "", "password": "password123"},
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "email",
		},
		{
			name:       "login: missing password",
			method:     http.MethodPost,
			path:       "/api/auth/login",
			body:       map[string]string{"email": "test@test.com", "password": ""},
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "password",
		},
		{
			name:       "create article: missing title",
			method:     http.MethodPost,
			path:       "/api/articles",
			token:      token,
			body:       map[string]string{"title": "", "body": "Some body"},
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "title",
		},
		{
			name:       "create article: missing body",
			method:     http.MethodPost,
			path:       "/api/articles",
			token:      token,
			body:       map[string]string{"title": "Some title", "body": ""},
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "body",
		},
		{
			name:       "update article: empty body",
			method:     http.MethodPut,
			path:       "/api/articles/placeholder",
			token:      token,
			body:       map[string]string{"title": "", "body": ""},
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "title/body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For update test, create a real article first
			path := tt.path
			if tt.method == http.MethodPut && tt.path == "/api/articles/placeholder" {
				id := createArticle(t, srv, token, "To Update", "Body")
				path = "/api/articles/" + id
			}

			resp, body := doRequest(t, tt.method, srv.URL+path, tt.token, tt.body)
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status=%d, want %d, body=%s", resp.StatusCode, tt.wantStatus, body)
				return
			}

			var errResp apiError
			json.Unmarshal(body, &errResp)
			if errResp.Error.Message != "validation failed" {
				t.Errorf("error message=%q, want %q", errResp.Error.Message, "validation failed")
			}
			if len(errResp.Error.Details) == 0 {
				t.Error("expected validation details to be non-empty")
				return
			}

			found := false
			for _, d := range errResp.Error.Details {
				if d.Field == tt.wantField {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected detail for field %q, got %+v", tt.wantField, errResp.Error.Details)
			}
		})
	}
}

// --- Partial Update Test ---

func TestIntegrationPartialUpdate(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	_, token := registerUser(t, srv, "partial@test.com", "password123", "Partial")
	id := createArticle(t, srv, token, "Original Title", "Original Body")

	// Update only title
	resp, body := doRequest(t, http.MethodPut, srv.URL+"/api/articles/"+id, token, map[string]string{
		"title": "New Title",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("partial update title: status=%d body=%s", resp.StatusCode, body)
	}
	var result struct {
		Data struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		} `json:"data"`
	}
	json.Unmarshal(body, &result)
	if result.Data.Title != "New Title" {
		t.Errorf("title=%q, want %q", result.Data.Title, "New Title")
	}
	if result.Data.Body != "Original Body" {
		t.Errorf("body=%q, want %q (should be preserved)", result.Data.Body, "Original Body")
	}

	// Update only body
	resp, body = doRequest(t, http.MethodPut, srv.URL+"/api/articles/"+id, token, map[string]string{
		"body": "New Body",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("partial update body: status=%d body=%s", resp.StatusCode, body)
	}
	json.Unmarshal(body, &result)
	if result.Data.Title != "New Title" {
		t.Errorf("title=%q, want %q (should be preserved)", result.Data.Title, "New Title")
	}
	if result.Data.Body != "New Body" {
		t.Errorf("body=%q, want %q", result.Data.Body, "New Body")
	}
}

// --- Consistent JSON Format Tests ---

func TestIntegrationConsistentJSONFormat(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	// Health check has its own format
	resp, _ := doRequest(t, http.MethodGet, srv.URL+"/health", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health: status=%d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("health content-type=%q, want application/json", resp.Header.Get("Content-Type"))
	}

	// Success response wraps in {data: ...}
	_, token := registerUser(t, srv, "format@test.com", "password123", "Format")
	resp, body := doRequest(t, http.MethodGet, srv.URL+"/api/users/me", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("profile: status=%d", resp.StatusCode)
	}
	var success map[string]json.RawMessage
	json.Unmarshal(body, &success)
	if _, ok := success["data"]; !ok {
		t.Errorf("success response missing 'data' key, body=%s", body)
	}

	// Error response wraps in {error: {message: ...}}
	resp, body = doRequest(t, http.MethodGet, srv.URL+"/api/articles/nonexistent", "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("not found: status=%d", resp.StatusCode)
	}
	var errResp map[string]json.RawMessage
	json.Unmarshal(body, &errResp)
	if _, ok := errResp["error"]; !ok {
		t.Errorf("error response missing 'error' key, body=%s", body)
	}

	// Paginated response has {data: [...], meta: {...}}
	resp, body = doRequest(t, http.MethodGet, srv.URL+"/api/articles", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status=%d", resp.StatusCode)
	}
	var paginated map[string]json.RawMessage
	json.Unmarshal(body, &paginated)
	if _, ok := paginated["data"]; !ok {
		t.Errorf("paginated response missing 'data' key, body=%s", body)
	}
	if _, ok := paginated["meta"]; !ok {
		t.Errorf("paginated response missing 'meta' key, body=%s", body)
	}
}

// --- Request ID Header Test ---

func TestIntegrationRequestIDHeader(t *testing.T) {
	srv, _, _ := setupServer(t)
	defer srv.Close()

	resp, _ := doRequest(t, http.MethodGet, srv.URL+"/health", "", nil)
	if resp.Header.Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}
