package router

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/devaloi/restgo/internal/auth"
	"github.com/devaloi/restgo/internal/config"
	"github.com/devaloi/restgo/internal/handler"
	"github.com/devaloi/restgo/internal/middleware"
	"github.com/devaloi/restgo/internal/repository"
	"github.com/devaloi/restgo/internal/service"
)

// New creates a fully wired HTTP handler with all routes and middleware.
func New(cfg *config.Config, userRepo repository.UserRepository, articleRepo repository.ArticleRepository) http.Handler {
	jwt := auth.New(cfg.JWT.Secret, cfg.JWT.Expiry)

	userSvc := service.NewUserService(userRepo, jwt)
	articleSvc := service.NewArticleService(articleRepo)

	userH := handler.NewUserHandler(userSvc)
	articleH := handler.NewArticleHandler(articleSvc)

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			slog.Error("failed to encode health response", "error", err)
		}
	})

	// Public auth routes
	mux.HandleFunc("POST /api/auth/register", userH.Register)
	mux.HandleFunc("POST /api/auth/login", userH.Login)

	// Public article routes
	mux.HandleFunc("GET /api/articles", articleH.List)
	mux.HandleFunc("GET /api/articles/{id}", articleH.GetByID)

	// Protected routes — wrap with auth middleware
	authMW := middleware.Auth(jwt)
	mux.Handle("GET /api/users/me", authMW(http.HandlerFunc(userH.GetProfile)))
	mux.Handle("POST /api/articles", authMW(http.HandlerFunc(articleH.Create)))
	mux.Handle("PUT /api/articles/{id}", authMW(http.HandlerFunc(articleH.Update)))
	mux.Handle("DELETE /api/articles/{id}", authMW(http.HandlerFunc(articleH.Delete)))

	// Global middleware stack
	return middleware.Chain(mux,
		middleware.Recovery,
		middleware.RequestID,
		middleware.SecurityHeaders,
		middleware.Logging,
		middleware.CORS(cfg.CORS.Origins),
		middleware.RateLimit(cfg.Rate.Limit),
	)
}
