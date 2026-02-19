package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/devaloi/restgo/internal/domain"
)

// ListOptions configures article listing queries.
type ListOptions struct {
	Page      int
	PerPage   int
	SortField string
	SortDir   string
	Search    string
	AuthorID  string
}

// ArticleRepository defines the interface for article persistence.
type ArticleRepository interface {
	Create(ctx context.Context, article *domain.Article) error
	GetByID(ctx context.Context, id string) (*domain.Article, error)
	Update(ctx context.Context, article *domain.Article) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, opts ListOptions) ([]domain.Article, int, error)
}

// PostgresArticleRepository implements ArticleRepository using PostgreSQL.
type PostgresArticleRepository struct {
	db *sql.DB
}

func NewPostgresArticleRepository(db *sql.DB) *PostgresArticleRepository {
	return &PostgresArticleRepository{db: db}
}

func (r *PostgresArticleRepository) Create(ctx context.Context, article *domain.Article) error {
	now := time.Now().UTC()
	article.CreatedAt = now
	article.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO articles (id, title, body, author_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		article.ID, article.Title, article.Body, article.AuthorID, article.CreatedAt, article.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting article: %w", err)
	}
	return nil
}

func (r *PostgresArticleRepository) GetByID(ctx context.Context, id string) (*domain.Article, error) {
	article := &domain.Article{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, body, author_id, created_at, updated_at FROM articles WHERE id = $1`, id,
	).Scan(&article.ID, &article.Title, &article.Body, &article.AuthorID, &article.CreatedAt, &article.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying article by id: %w", err)
	}
	return article, nil
}

func (r *PostgresArticleRepository) Update(ctx context.Context, article *domain.Article) error {
	article.UpdatedAt = time.Now().UTC()

	result, err := r.db.ExecContext(ctx,
		`UPDATE articles SET title = $1, body = $2, updated_at = $3 WHERE id = $4`,
		article.Title, article.Body, article.UpdatedAt, article.ID,
	)
	if err != nil {
		return fmt.Errorf("updating article: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresArticleRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM articles WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("deleting article: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresArticleRepository) List(ctx context.Context, opts ListOptions) ([]domain.Article, int, error) {
	// Defaults
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PerPage < 1 {
		opts.PerPage = domain.DefaultPageSize
	}

	// Validate sort field
	allowedSorts := map[string]bool{"created_at": true, "updated_at": true, "title": true}
	if !allowedSorts[opts.SortField] {
		opts.SortField = "created_at"
	}
	if opts.SortDir != "asc" {
		opts.SortDir = "desc"
	}

	var conditions []string
	var args []any
	argIdx := 1

	if opts.AuthorID != "" {
		conditions = append(conditions, fmt.Sprintf("author_id = $%d", argIdx))
		args = append(args, opts.AuthorID)
		argIdx++
	}
	if opts.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR body ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+opts.Search+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM articles %s", where)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting articles: %w", err)
	}

	// Fetch page — sort field and direction are validated above (allow-listed),
	// so interpolating them into the query is safe. All user-supplied filter
	// values are passed as parameterized arguments ($N).
	offset := (opts.Page - 1) * opts.PerPage
	query := fmt.Sprintf(
		"SELECT id, title, body, author_id, created_at, updated_at FROM articles %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, opts.SortField, opts.SortDir, argIdx, argIdx+1,
	)
	args = append(args, opts.PerPage, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing articles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var articles []domain.Article
	for rows.Next() {
		var a domain.Article
		if err := rows.Scan(&a.ID, &a.Title, &a.Body, &a.AuthorID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning article: %w", err)
		}
		articles = append(articles, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating articles: %w", err)
	}

	return articles, total, nil
}
