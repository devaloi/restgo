package service

import (
	"context"
	"fmt"

	"github.com/devaloi/restgo/internal/domain"
	"github.com/devaloi/restgo/internal/middleware"
	"github.com/devaloi/restgo/internal/repository"
)

// ArticleService handles article business logic.
type ArticleService struct {
	repo repository.ArticleRepository
}

// NewArticleService creates an ArticleService.
func NewArticleService(repo repository.ArticleRepository) *ArticleService {
	return &ArticleService{repo: repo}
}

// Create creates a new article owned by userID.
func (s *ArticleService) Create(ctx context.Context, userID string, req domain.CreateArticleRequest) (*domain.Article, error) {
	if err := validateCreateArticle(req); err != nil {
		return nil, err
	}

	article := &domain.Article{
		ID:       middleware.NewID(),
		Title:    req.Title,
		Body:     req.Body,
		AuthorID: userID,
	}

	if err := s.repo.Create(ctx, article); err != nil {
		return nil, fmt.Errorf("creating article: %w", err)
	}
	return article, nil
}

// GetByID returns an article by ID.
func (s *ArticleService) GetByID(ctx context.Context, id string) (*domain.Article, error) {
	return s.repo.GetByID(ctx, id)
}

// Update updates an article, verifying ownership.
func (s *ArticleService) Update(ctx context.Context, userID, articleID string, req domain.UpdateArticleRequest) (*domain.Article, error) {
	if err := validateUpdateArticle(req); err != nil {
		return nil, err
	}

	article, err := s.repo.GetByID(ctx, articleID)
	if err != nil {
		return nil, err
	}

	if article.AuthorID != userID {
		return nil, domain.ErrForbidden
	}

	if req.Title != "" {
		article.Title = req.Title
	}
	if req.Body != "" {
		article.Body = req.Body
	}

	if err := s.repo.Update(ctx, article); err != nil {
		return nil, fmt.Errorf("updating article: %w", err)
	}
	return article, nil
}

// Delete removes an article, verifying ownership.
func (s *ArticleService) Delete(ctx context.Context, userID, articleID string) error {
	article, err := s.repo.GetByID(ctx, articleID)
	if err != nil {
		return err
	}

	if article.AuthorID != userID {
		return domain.ErrForbidden
	}

	return s.repo.Delete(ctx, articleID)
}

// List returns a paginated list of articles.
func (s *ArticleService) List(ctx context.Context, opts repository.ListOptions) ([]domain.Article, int, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PerPage < 1 {
		opts.PerPage = 20
	}
	return s.repo.List(ctx, opts)
}

func validateCreateArticle(req domain.CreateArticleRequest) error {
	var errs []domain.ValidationError
	if req.Title == "" {
		errs = append(errs, domain.ValidationError{Field: "title", Message: "title is required"})
	} else if len(req.Title) > 255 {
		errs = append(errs, domain.ValidationError{Field: "title", Message: "title must be at most 255 characters"})
	}
	if req.Body == "" {
		errs = append(errs, domain.ValidationError{Field: "body", Message: "body is required"})
	}
	if len(errs) > 0 {
		return &domain.ValidationErrors{Errors: errs}
	}
	return nil
}

func validateUpdateArticle(req domain.UpdateArticleRequest) error {
	if req.Title == "" && req.Body == "" {
		return &domain.ValidationErrors{
			Errors: []domain.ValidationError{
				{Field: "title/body", Message: "at least one of title or body is required"},
			},
		}
	}
	if req.Title != "" && len(req.Title) > 255 {
		return &domain.ValidationErrors{
			Errors: []domain.ValidationError{
				{Field: "title", Message: "title must be at most 255 characters"},
			},
		}
	}
	return nil
}

