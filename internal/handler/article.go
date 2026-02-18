package handler

import (
	"net/http"
	"strconv"

	"github.com/devaloi/restgo/internal/domain"
	"github.com/devaloi/restgo/internal/middleware"
	"github.com/devaloi/restgo/internal/repository"
	"github.com/devaloi/restgo/internal/service"
)

// ArticleHandler handles article-related HTTP requests.
type ArticleHandler struct {
	svc *service.ArticleService
}

// NewArticleHandler creates an ArticleHandler.
func NewArticleHandler(svc *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{svc: svc}
}

// Create handles POST /api/articles.
func (h *ArticleHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.UserFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req domain.CreateArticleRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	article, err := h.svc.Create(r.Context(), claims.UserID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	JSON(w, http.StatusCreated, article)
}

// List handles GET /api/articles.
func (h *ArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	opts := repository.ListOptions{
		Page:      page,
		PerPage:   perPage,
		Search:    q.Get("search"),
		AuthorID:  q.Get("author_id"),
		SortField: q.Get("sort"),
		SortDir:   q.Get("dir"),
	}

	articles, total, err := h.svc.List(r.Context(), opts)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PerPage < 1 {
		opts.PerPage = domain.DefaultPageSize
	}

	totalPages := total / opts.PerPage
	if total%opts.PerPage != 0 {
		totalPages++
	}

	Paginated(w, articles, domain.PaginationMeta{
		Page:       opts.Page,
		PerPage:    opts.PerPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetByID handles GET /api/articles/{id}.
func (h *ArticleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		Error(w, http.StatusBadRequest, "missing article id")
		return
	}

	article, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, article)
}

// Update handles PUT /api/articles/{id}.
func (h *ArticleHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.UserFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		Error(w, http.StatusBadRequest, "missing article id")
		return
	}

	var req domain.UpdateArticleRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	article, err := h.svc.Update(r.Context(), claims.UserID, id, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, article)
}

// Delete handles DELETE /api/articles/{id}.
func (h *ArticleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.UserFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		Error(w, http.StatusBadRequest, "missing article id")
		return
	}

	if err := h.svc.Delete(r.Context(), claims.UserID, id); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
