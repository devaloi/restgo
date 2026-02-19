package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devaloi/restgo/internal/config"
	"github.com/devaloi/restgo/internal/database"
	"github.com/devaloi/restgo/internal/repository"
	"github.com/devaloi/restgo/internal/router"
	"github.com/devaloi/restgo/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Attempt database connection; fall back to in-memory repos for demo/testing
	var userRepo repository.UserRepository
	var articleRepo repository.ArticleRepository

	db, err := database.Connect(cfg.DB)
	if err != nil {
		slog.Warn("database unavailable, using in-memory repositories", "error", err)
		userRepo = repository.NewMockUserRepository()
		articleRepo = repository.NewMockArticleRepository()
	} else {
		defer func() { _ = db.Close() }()

		if err := database.RunMigrations(db, migrations.FS); err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}
		slog.Info("database migrations applied")

		userRepo = repository.NewPostgresUserRepository(db)
		articleRepo = repository.NewPostgresArticleRepository(db)
	}

	handler := router.New(cfg, userRepo, articleRepo)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("restgo server starting", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-done
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	slog.Info("server stopped gracefully")
}
