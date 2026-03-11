package router

import (
	"net/http"

	"github.com/devaloi/restgo/internal/auth"
	"github.com/devaloi/restgo/internal/config"
	"github.com/devaloi/restgo/internal/handler"
	"github.com/devaloi/restgo/internal/middleware"
	"github.com/devaloi/restgo/internal/repository"
	"github.com/devaloi/restgo/internal/service"
)

// New creates a fully wired HTTP handler with all routes and middleware.
// db may be nil when running with in-memory repositories.
func New(cfg *config.Config, userRepo repository.UserRepository, articleRepo repository.ArticleRepository, db handler.DBPinger) http.Handler {
	jwt := auth.New(cfg.JWT.Secret, cfg.JWT.Expiry)

	userSvc := service.NewUserService(userRepo, jwt)
	articleSvc := service.NewArticleService(articleRepo)

	userH := handler.NewUserHandler(userSvc)
	articleH := handler.NewArticleHandler(articleSvc)
	healthH := handler.NewHealthHandler(db)

	mux := http.NewServeMux()

	// Health check with dependency status
	mux.HandleFunc("GET /health", healthH.Check)

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
		middleware.Logging,
		middleware.CORS(cfg.CORS.Origins),
		middleware.RateLimit(cfg.Rate.Limit),
	)
}
