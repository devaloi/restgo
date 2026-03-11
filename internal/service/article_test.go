package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/devaloi/restgo/internal/domain"
	"github.com/devaloi/restgo/internal/repository"
)

func newArticleTestService() (*ArticleService, *repository.MockArticleRepository) {
	repo := repository.NewMockArticleRepository()
	svc := NewArticleService(repo)
	return svc, repo
}

func seedArticle(t *testing.T, repo *repository.MockArticleRepository, id, authorID string) {
	t.Helper()
	err := repo.Create(context.Background(), &domain.Article{
		ID:       id,
		Title:    "Seeded Article",
		Body:     "Seeded body content",
		AuthorID: authorID,
	})
	if err != nil {
		t.Fatalf("seed article: %v", err)
	}
}

func TestArticleService_Create_Success(t *testing.T) {
	svc, _ := newArticleTestService()

	article, err := svc.Create(context.Background(), "user-1", domain.CreateArticleRequest{
		Title: "My Article",
		Body:  "Article body text",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if article.ID == "" {
		t.Error("expected article ID to be generated")
	}
	if article.Title != "My Article" {
		t.Errorf("expected title 'My Article', got %q", article.Title)
	}
	if article.Body != "Article body text" {
		t.Errorf("expected body 'Article body text', got %q", article.Body)
	}
	if article.AuthorID != "user-1" {
		t.Errorf("expected author_id 'user-1', got %q", article.AuthorID)
	}
}

func TestArticleService_Create_ValidationErrors(t *testing.T) {
	svc, _ := newArticleTestService()

	tests := []struct {
		name      string
		req       domain.CreateArticleRequest
		wantField string
	}{
		{
			name:      "empty title",
			req:       domain.CreateArticleRequest{Title: "", Body: "body"},
			wantField: "title",
		},
		{
			name:      "empty body",
			req:       domain.CreateArticleRequest{Title: "title", Body: ""},
			wantField: "body",
		},
		{
			name:      "title too long",
			req:       domain.CreateArticleRequest{Title: strings.Repeat("a", domain.MaxTitleLength+1), Body: "body"},
			wantField: "title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), "user-1", tt.req)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}

			var verr *domain.ValidationErrors
			if !errors.As(err, &verr) {
				t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
			}

			found := false
			for _, e := range verr.Errors {
				if e.Field == tt.wantField {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected validation error on field %q, got %v", tt.wantField, verr.Errors)
			}
		})
	}
}

func TestArticleService_Create_BothFieldsEmpty(t *testing.T) {
	svc, _ := newArticleTestService()

	_, err := svc.Create(context.Background(), "user-1", domain.CreateArticleRequest{})
	if err == nil {
		t.Fatal("expected error for empty request")
	}

	var verr *domain.ValidationErrors
	if !errors.As(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(verr.Errors) < 2 {
		t.Errorf("expected at least 2 validation errors, got %d", len(verr.Errors))
	}
}

func TestArticleService_GetByID_Success(t *testing.T) {
	svc, repo := newArticleTestService()
	seedArticle(t, repo, "art-1", "user-1")

	article, err := svc.GetByID(context.Background(), "art-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if article.ID != "art-1" {
		t.Errorf("expected ID 'art-1', got %q", article.ID)
	}
}

func TestArticleService_GetByID_NotFound(t *testing.T) {
	svc, _ := newArticleTestService()

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestArticleService_Update_Success(t *testing.T) {
	svc, repo := newArticleTestService()
	seedArticle(t, repo, "art-1", "user-1")

	updated, err := svc.Update(context.Background(), "user-1", "art-1", domain.UpdateArticleRequest{
		Title: "Updated Title",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %q", updated.Title)
	}
}

func TestArticleService_Update_BodyOnly(t *testing.T) {
	svc, repo := newArticleTestService()
	seedArticle(t, repo, "art-1", "user-1")

	updated, err := svc.Update(context.Background(), "user-1", "art-1", domain.UpdateArticleRequest{
		Body: "New body content",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Body != "New body content" {
		t.Errorf("expected body 'New body content', got %q", updated.Body)
	}
	if updated.Title != "Seeded Article" {
		t.Errorf("expected original title preserved, got %q", updated.Title)
	}
}

func TestArticleService_Update_NotFound(t *testing.T) {
	svc, _ := newArticleTestService()

	_, err := svc.Update(context.Background(), "user-1", "nonexistent", domain.UpdateArticleRequest{
		Title: "X",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestArticleService_Update_Forbidden(t *testing.T) {
	svc, repo := newArticleTestService()
	seedArticle(t, repo, "art-1", "user-1")

	_, err := svc.Update(context.Background(), "user-2", "art-1", domain.UpdateArticleRequest{
		Title: "Hijacked",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestArticleService_Update_ValidationError(t *testing.T) {
	svc, repo := newArticleTestService()
	seedArticle(t, repo, "art-1", "user-1")

	_, err := svc.Update(context.Background(), "user-1", "art-1", domain.UpdateArticleRequest{})
	if err == nil {
		t.Fatal("expected validation error for empty update")
	}

	var verr *domain.ValidationErrors
	if !errors.As(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
}

func TestArticleService_Update_TitleTooLong(t *testing.T) {
	svc, repo := newArticleTestService()
	seedArticle(t, repo, "art-1", "user-1")

	_, err := svc.Update(context.Background(), "user-1", "art-1", domain.UpdateArticleRequest{
		Title: strings.Repeat("x", domain.MaxTitleLength+1),
	})
	if err == nil {
		t.Fatal("expected validation error for long title")
	}

	var verr *domain.ValidationErrors
	if !errors.As(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
}

func TestArticleService_Delete_Success(t *testing.T) {
	svc, repo := newArticleTestService()
	seedArticle(t, repo, "art-1", "user-1")

	err := svc.Delete(context.Background(), "user-1", "art-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify deletion
	_, err = svc.GetByID(context.Background(), "art-1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestArticleService_Delete_NotFound(t *testing.T) {
	svc, _ := newArticleTestService()

	err := svc.Delete(context.Background(), "user-1", "nonexistent")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestArticleService_Delete_Forbidden(t *testing.T) {
	svc, repo := newArticleTestService()
	seedArticle(t, repo, "art-1", "user-1")

	err := svc.Delete(context.Background(), "user-2", "art-1")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	// Verify article still exists
	_, err = svc.GetByID(context.Background(), "art-1")
	if err != nil {
		t.Fatalf("article should still exist after forbidden delete: %v", err)
	}
}

func TestArticleService_List_Defaults(t *testing.T) {
	svc, repo := newArticleTestService()

	for i := 0; i < 5; i++ {
		seedArticle(t, repo, "art-"+string(rune('a'+i)), "user-1")
	}

	articles, total, err := svc.List(context.Background(), repository.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(articles) != 5 {
		t.Errorf("expected 5 articles, got %d", len(articles))
	}
}

func TestArticleService_List_PageDefaults(t *testing.T) {
	svc, _ := newArticleTestService()

	// Zero/negative values should be normalized
	articles, _, err := svc.List(context.Background(), repository.ListOptions{
		Page:    0,
		PerPage: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = articles // empty slice is acceptable, just verify no error
}

func TestArticleService_List_CapsPerPage(t *testing.T) {
	svc, repo := newArticleTestService()

	for i := 0; i < 3; i++ {
		seedArticle(t, repo, "art-"+string(rune('a'+i)), "user-1")
	}

	// PerPage above MaxPageSize should be capped
	articles, _, err := svc.List(context.Background(), repository.ListOptions{
		Page:    1,
		PerPage: domain.MaxPageSize + 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All 3 articles should be returned since 3 < MaxPageSize
	if len(articles) != 3 {
		t.Errorf("expected 3 articles, got %d", len(articles))
	}
}

func TestArticleService_List_WithSearch(t *testing.T) {
	svc, repo := newArticleTestService()

	repo.Create(context.Background(), &domain.Article{ID: "1", Title: "Go REST API", Body: "golang", AuthorID: "u1"})
	repo.Create(context.Background(), &domain.Article{ID: "2", Title: "Python Flask", Body: "web", AuthorID: "u1"})

	articles, total, err := svc.List(context.Background(), repository.ListOptions{
		Page:    1,
		PerPage: 10,
		Search:  "Go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 search result, got %d", total)
	}
	if len(articles) != 1 {
		t.Errorf("expected 1 article, got %d", len(articles))
	}
}

func TestArticleService_List_FilterByAuthor(t *testing.T) {
	svc, repo := newArticleTestService()

	repo.Create(context.Background(), &domain.Article{ID: "1", Title: "A", Body: "B", AuthorID: "user-1"})
	repo.Create(context.Background(), &domain.Article{ID: "2", Title: "C", Body: "D", AuthorID: "user-2"})
	repo.Create(context.Background(), &domain.Article{ID: "3", Title: "E", Body: "F", AuthorID: "user-1"})

	articles, total, err := svc.List(context.Background(), repository.ListOptions{
		Page:     1,
		PerPage:  10,
		AuthorID: "user-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 articles for user-1, got %d", total)
	}
	if len(articles) != 2 {
		t.Errorf("expected 2 articles, got %d", len(articles))
	}
}
