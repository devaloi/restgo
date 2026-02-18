package domain

import "time"

type Article struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateArticleRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type UpdateArticleRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}
