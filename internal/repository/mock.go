package repository

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/devaloi/restgo/internal/domain"
)

// MockUserRepository is an in-memory implementation of UserRepository for testing.
type MockUserRepository struct {
	mu    sync.RWMutex
	users map[string]*domain.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (r *MockUserRepository) Create(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.users {
		if u.Email == user.Email {
			return domain.ErrConflict
		}
	}

	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Store a copy
	stored := *user
	r.users[user.ID] = &stored
	return nil
}

func (r *MockUserRepository) GetByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *user
	return &copy, nil
}

func (r *MockUserRepository) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Email == email {
			copy := *user
			return &copy, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *MockUserRepository) Exists(_ context.Context, email string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Email == email {
			return true, nil
		}
	}
	return false, nil
}

// MockArticleRepository is an in-memory implementation of ArticleRepository for testing.
type MockArticleRepository struct {
	mu       sync.RWMutex
	articles map[string]*domain.Article
}

func NewMockArticleRepository() *MockArticleRepository {
	return &MockArticleRepository{
		articles: make(map[string]*domain.Article),
	}
}

func (r *MockArticleRepository) Create(_ context.Context, article *domain.Article) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	article.CreatedAt = now
	article.UpdatedAt = now

	stored := *article
	r.articles[article.ID] = &stored
	return nil
}

func (r *MockArticleRepository) GetByID(_ context.Context, id string) (*domain.Article, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	article, ok := r.articles[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *article
	return &copy, nil
}

func (r *MockArticleRepository) Update(_ context.Context, article *domain.Article) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.articles[article.ID]
	if !ok {
		return domain.ErrNotFound
	}

	existing.Title = article.Title
	existing.Body = article.Body
	existing.UpdatedAt = time.Now().UTC()

	// Update the caller's article with the new timestamp
	article.UpdatedAt = existing.UpdatedAt
	return nil
}

func (r *MockArticleRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.articles[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.articles, id)
	return nil
}

func (r *MockArticleRepository) List(_ context.Context, opts ListOptions) ([]domain.Article, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Defaults
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PerPage < 1 {
		opts.PerPage = domain.DefaultPageSize
	}

	// Collect matching articles
	var filtered []domain.Article
	for _, a := range r.articles {
		if opts.AuthorID != "" && a.AuthorID != opts.AuthorID {
			continue
		}
		if opts.Search != "" {
			search := strings.ToLower(opts.Search)
			if !strings.Contains(strings.ToLower(a.Title), search) &&
				!strings.Contains(strings.ToLower(a.Body), search) {
				continue
			}
		}
		filtered = append(filtered, *a)
	}

	total := len(filtered)

	// Sort
	sortField := opts.SortField
	if sortField == "" {
		sortField = "created_at"
	}
	sortAsc := opts.SortDir == "asc"

	sort.Slice(filtered, func(i, j int) bool {
		var less bool
		switch sortField {
		case "title":
			less = filtered[i].Title < filtered[j].Title
		case "updated_at":
			less = filtered[i].UpdatedAt.Before(filtered[j].UpdatedAt)
		default: // created_at
			less = filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
		}
		if sortAsc {
			return less
		}
		return !less
	})

	// Paginate
	offset := (opts.Page - 1) * opts.PerPage
	if offset >= len(filtered) {
		return []domain.Article{}, total, nil
	}
	end := offset + opts.PerPage
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[offset:end], total, nil
}
