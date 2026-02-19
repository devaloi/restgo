package repository

import (
	"context"
	"testing"

	"github.com/devaloi/restgo/internal/domain"
)

func TestMockUserRepository_Create(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	user := &domain.User{
		ID:           "user-1",
		Email:        "alice@example.com",
		PasswordHash: "hashed",
		Name:         "Alice",
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestMockUserRepository_CreateDuplicate(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	user := &domain.User{ID: "user-1", Email: "alice@example.com", PasswordHash: "h", Name: "Alice"}
	repo.Create(ctx, user)

	dup := &domain.User{ID: "user-2", Email: "alice@example.com", PasswordHash: "h", Name: "Alice2"}
	err := repo.Create(ctx, dup)
	if err != domain.ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestMockUserRepository_GetByID(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	user := &domain.User{ID: "user-1", Email: "alice@example.com", PasswordHash: "h", Name: "Alice"}
	repo.Create(ctx, user)

	got, err := repo.GetByID(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", got.Email)
	}
}

func TestMockUserRepository_GetByIDNotFound(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMockUserRepository_GetByEmail(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	user := &domain.User{ID: "user-1", Email: "alice@example.com", PasswordHash: "h", Name: "Alice"}
	repo.Create(ctx, user)

	got, err := repo.GetByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "user-1" {
		t.Errorf("expected id user-1, got %s", got.ID)
	}
}

func TestMockUserRepository_Exists(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	user := &domain.User{ID: "user-1", Email: "alice@example.com", PasswordHash: "h", Name: "Alice"}
	repo.Create(ctx, user)

	exists, err := repo.Exists(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected user to exist")
	}

	exists, err = repo.Exists(ctx, "nobody@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected user not to exist")
	}
}

func TestMockArticleRepository_CRUD(t *testing.T) {
	repo := NewMockArticleRepository()
	ctx := context.Background()

	// Create
	article := &domain.Article{
		ID:       "art-1",
		Title:    "Go REST APIs",
		Body:     "Building REST APIs with Go stdlib",
		AuthorID: "user-1",
	}
	if err := repo.Create(ctx, article); err != nil {
		t.Fatalf("create: %v", err)
	}
	if article.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	// GetByID
	got, err := repo.GetByID(ctx, "art-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "Go REST APIs" {
		t.Errorf("expected title 'Go REST APIs', got %s", got.Title)
	}

	// Update
	got.Title = "Updated Title"
	got.Body = "Updated body"
	err = repo.Update(ctx, got)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	updated, _ := repo.GetByID(ctx, "art-1")
	if updated.Title != "Updated Title" {
		t.Errorf("expected updated title, got %s", updated.Title)
	}

	// Delete
	err = repo.Delete(ctx, "art-1")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = repo.GetByID(ctx, "art-1")
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMockArticleRepository_DeleteNotFound(t *testing.T) {
	repo := NewMockArticleRepository()
	err := repo.Delete(context.Background(), "nonexistent")
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMockArticleRepository_UpdateNotFound(t *testing.T) {
	repo := NewMockArticleRepository()
	err := repo.Update(context.Background(), &domain.Article{ID: "nonexistent"})
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMockArticleRepository_List(t *testing.T) {
	repo := NewMockArticleRepository()
	ctx := context.Background()

	// Create multiple articles
	for i := 0; i < 5; i++ {
		a := &domain.Article{
			ID:       "art-" + string(rune('a'+i)),
			Title:    "Article " + string(rune('A'+i)),
			Body:     "Body " + string(rune('A'+i)),
			AuthorID: "user-1",
		}
		if i >= 3 {
			a.AuthorID = "user-2"
		}
		repo.Create(ctx, a)
	}

	// List all
	articles, total, err := repo.List(ctx, ListOptions{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(articles) != 5 {
		t.Errorf("expected 5 articles, got %d", len(articles))
	}

	// Filter by author
	_, total, err = repo.List(ctx, ListOptions{Page: 1, PerPage: 10, AuthorID: "user-1"})
	if err != nil {
		t.Fatalf("list by author: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3 for user-1, got %d", total)
	}
}

func TestMockArticleRepository_ListPagination(t *testing.T) {
	repo := NewMockArticleRepository()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		a := &domain.Article{
			ID:       "art-" + string(rune('a'+i)),
			Title:    "Article",
			Body:     "Body",
			AuthorID: "user-1",
		}
		repo.Create(ctx, a)
	}

	articles, total, err := repo.List(ctx, ListOptions{Page: 1, PerPage: 3})
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if total != 10 {
		t.Errorf("expected total 10, got %d", total)
	}
	if len(articles) != 3 {
		t.Errorf("expected 3 articles on page 1, got %d", len(articles))
	}

	// Page beyond range
	articles, _, err = repo.List(ctx, ListOptions{Page: 100, PerPage: 3})
	if err != nil {
		t.Fatalf("list beyond range: %v", err)
	}
	if len(articles) != 0 {
		t.Errorf("expected 0 articles beyond range, got %d", len(articles))
	}
}

func TestMockArticleRepository_ListSearch(t *testing.T) {
	repo := NewMockArticleRepository()
	ctx := context.Background()

	repo.Create(ctx, &domain.Article{ID: "1", Title: "Go REST", Body: "Building APIs", AuthorID: "u1"})
	repo.Create(ctx, &domain.Article{ID: "2", Title: "Python Flask", Body: "Web framework", AuthorID: "u1"})
	repo.Create(ctx, &domain.Article{ID: "3", Title: "Java Spring", Body: "REST API framework", AuthorID: "u1"})

	articles, total, err := repo.List(ctx, ListOptions{Page: 1, PerPage: 10, Search: "REST"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 results for 'REST', got %d", total)
	}
	if len(articles) != 2 {
		t.Errorf("expected 2 articles, got %d", len(articles))
	}
}
