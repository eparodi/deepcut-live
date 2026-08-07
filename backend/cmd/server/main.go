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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/deepcut/live/internal/handler"
	"github.com/deepcut/live/internal/service"
	"github.com/deepcut/live/internal/store/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Config
	dbURL := env("DATABASE_URL", "postgres://live:live@localhost:5432/live?sslmode=disable")
	port := env("PORT", "8081")
	jwtSecret := env("JWT_SECRET", "dev-secret-change-in-production")
	googleClientID := env("GOOGLE_CLIENT_ID", "")
	googleClientSecret := env("GOOGLE_CLIENT_SECRET", "")
	baseURL := env("BASE_URL", "http://localhost:3000")
	srsSecret := env("SRS_CALLBACK_SECRET", "dev-srs-secret")

	// Database
	pool, err := postgres.NewPool(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	store := postgres.New(pool)

	// Services
	authSvc := service.NewAuthService(googleClientID, googleClientSecret, baseURL, jwtSecret)
	userSvc := service.NewUserService(store)

	// Router
	r := chi.NewRouter()

	// Middleware chain
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	h := handler.New(authSvc, userSvc, jwtSecret, srsSecret)

	// Public routes
	r.Get("/api/auth/google", h.GoogleOAuth)
	r.Get("/api/auth/google/callback", h.GoogleOAuthCallback)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Get("/api/me", h.GetMe)
		r.Post("/api/me/stream-key/regenerate", h.RegenerateStreamKey)
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		logger.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
