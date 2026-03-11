package repository

import (
	"context"

	"github.com/devaloi/restgo/internal/circuitbreaker"
	"github.com/devaloi/restgo/internal/domain"
)

// CBArticleRepository wraps an ArticleRepository with a circuit breaker.
type CBArticleRepository struct {
	repo    ArticleRepository
	breaker *circuitbreaker.Breaker
}

// NewCBArticleRepository wraps repo with breaker.
func NewCBArticleRepository(repo ArticleRepository, breaker *circuitbreaker.Breaker) *CBArticleRepository {
	return &CBArticleRepository{repo: repo, breaker: breaker}
}

func (r *CBArticleRepository) Create(ctx context.Context, article *domain.Article) error {
	return r.breaker.Execute(func() error {
		return r.repo.Create(ctx, article)
	})
}

func (r *CBArticleRepository) GetByID(ctx context.Context, id string) (*domain.Article, error) {
	var article *domain.Article
	err := r.breaker.Execute(func() error {
		var e error
		article, e = r.repo.GetByID(ctx, id)
		return e
	})
	return article, err
}

func (r *CBArticleRepository) Update(ctx context.Context, article *domain.Article) error {
	return r.breaker.Execute(func() error {
		return r.repo.Update(ctx, article)
	})
}

func (r *CBArticleRepository) Delete(ctx context.Context, id string) error {
	return r.breaker.Execute(func() error {
		return r.repo.Delete(ctx, id)
	})
}

func (r *CBArticleRepository) List(ctx context.Context, opts ListOptions) ([]domain.Article, int, error) {
	var articles []domain.Article
	var total int
	err := r.breaker.Execute(func() error {
		var e error
		articles, total, e = r.repo.List(ctx, opts)
		return e
	})
	return articles, total, err
}

// CBUserRepository wraps a UserRepository with a circuit breaker.
type CBUserRepository struct {
	repo    UserRepository
	breaker *circuitbreaker.Breaker
}

// NewCBUserRepository wraps repo with breaker.
func NewCBUserRepository(repo UserRepository, breaker *circuitbreaker.Breaker) *CBUserRepository {
	return &CBUserRepository{repo: repo, breaker: breaker}
}

func (r *CBUserRepository) Create(ctx context.Context, user *domain.User) error {
	return r.breaker.Execute(func() error {
		return r.repo.Create(ctx, user)
	})
}

func (r *CBUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user *domain.User
	err := r.breaker.Execute(func() error {
		var e error
		user, e = r.repo.GetByID(ctx, id)
		return e
	})
	return user, err
}

func (r *CBUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user *domain.User
	err := r.breaker.Execute(func() error {
		var e error
		user, e = r.repo.GetByEmail(ctx, email)
		return e
	})
	return user, err
}

func (r *CBUserRepository) Exists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.breaker.Execute(func() error {
		var e error
		exists, e = r.repo.Exists(ctx, email)
		return e
	})
	return exists, err
}
