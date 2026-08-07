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
	"github.com/jackc/pgx/v5/pgxpool"

	authhttp "github.com/deepcut/live/internal/modules/auth/adapter/http"
	authpg "github.com/deepcut/live/internal/modules/auth/adapter/postgres"
	authapp "github.com/deepcut/live/internal/modules/auth/application"

	streamhttp "github.com/deepcut/live/internal/modules/streams/adapter/http"
	streampg "github.com/deepcut/live/internal/modules/streams/adapter/postgres"
	streamapp "github.com/deepcut/live/internal/modules/streams/application"

	chathttp "github.com/deepcut/live/internal/modules/chat/adapter/http"
	chatpg "github.com/deepcut/live/internal/modules/chat/adapter/postgres"
	chatapp "github.com/deepcut/live/internal/modules/chat/application"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("failed to parse database URL: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Postgres adapters
	authRepo := authpg.NewAuthRepo(pool)
	streamRepo := streampg.NewStreamRepo(pool)
	chatRepo := chatpg.NewChatRepo(pool)

	// Application services
	authSvc := authapp.NewAuthService(authRepo, googleClientID, googleClientSecret, baseURL, jwtSecret)
	streamSvc := streamapp.NewStreamService(streamRepo, authRepo, srsSecret)
	chatHub := chatapp.NewChatHub(chatRepo, logger)
	chatSvc := chatapp.NewChatService(chatRepo, chatHub)

	// HTTP handlers
	authHandler := authhttp.NewAuthHandler(authSvc, logger)
	streamHandler := streamhttp.NewStreamHandler(streamSvc, logger)
	chatHandler := chathttp.NewChatHandler(chatSvc, logger)

	// Router
	r := chi.NewRouter()

	// Middleware chain (order matters)
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

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Module route registration
	authHandler.RegisterRoutes(r)
	streamHandler.RegisterRoutes(r)
	chatHandler.RegisterRoutes(r)

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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
