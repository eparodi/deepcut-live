package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

	vodhttp "github.com/deepcut/live/internal/modules/vods/adapter/http"
	vodpg "github.com/deepcut/live/internal/modules/vods/adapter/postgres"
	vodapp "github.com/deepcut/live/internal/modules/vods/application"
	vodriver "github.com/deepcut/live/internal/modules/vods/adapter/river"

	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	chathttp "github.com/deepcut/live/internal/modules/chat/adapter/http"
	chatpg "github.com/deepcut/live/internal/modules/chat/adapter/postgres"
	chatapp "github.com/deepcut/live/internal/modules/chat/application"
)

func main() {
	logger := newLogger()

	port := env("PORT", "8081")
	dbURL := env("DATABASE_URL", "postgres://live:live@localhost:5432/live?sslmode=disable")
	googleClientID := env("GOOGLE_CLIENT_ID", "")
	googleClientSecret := env("GOOGLE_CLIENT_SECRET", "")
	baseURL := env("BASE_URL", "http://localhost:3000")
	corsOrigin := env("CORS_ORIGIN", "http://localhost:3000")
	srsSecret := env("SRS_CALLBACK_SECRET", "dev-srs-secret")
	srsAPIURL := env("SRS_API_URL", "http://srs:1985")

	// Load or generate ECDSA key pair for JWT
	privateKeyPEM, publicKeyPEM := loadOrGenerateKeys()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("parse db url: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	authRepo := authpg.NewAuthRepo(pool)
	streamRepo := streampg.NewStreamRepo(pool)
	vodRepo := vodpg.NewVODRepo(pool)
	chatRepo := chatpg.NewChatRepo(pool)

	authSvc := authapp.NewAuthService(authRepo, googleClientID, googleClientSecret, baseURL, privateKeyPEM, publicKeyPEM)
	streamHub := streamapp.NewStreamHub(logger)

	// Run River schema migration before creating queue
	migrator, migErr := rivermigrate.New(riverpgxv5.New(pool), nil)
	if migErr != nil {
		slog.Warn("river migrator creation failed", "err", migErr)
	} else if _, migErr := migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil); migErr != nil {
		slog.Warn("river migration failed", "err", migErr)
	}
	vodQueue, err := vodriver.NewQueue(pool)
	if err != nil {
		slog.Warn("vod queue creation failed", "err", err)
		vodQueue = nil
	}
	streamSvc := streamapp.NewStreamService(streamRepo, authRepo, streamHub, vodQueue, srsSecret, srsAPIURL, logger)
	vodSvc := vodapp.NewVODService(vodRepo)
	chatHub := chatapp.NewChatHub(chatRepo, logger)
	chatSvc := chatapp.NewChatService(chatRepo, chatHub)

	authHandler := authhttp.NewAuthHandler(authSvc, streamSvc, baseURL, logger)
	streamHandler := streamhttp.NewStreamHandler(streamSvc, streamHub, logger)
	vodHandler := vodhttp.NewVODHandler(vodSvc, logger)
	chatHandler := chathttp.NewChatHandler(chatSvc, newChatAuthAdapter(authSvc), logger)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{corsOrigin},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	authHandler.RegisterRoutes(r)
	streamHandler.RegisterRoutes(r)
	vodHandler.RegisterRoutes(r)
	chatHandler.RegisterRoutes(r)

	// WebSocket endpoints (chat WS handles auth internally — auth optional).
	r.Get("/ws/chat/{streamID}", chatHandler.ChatWebSocket)

	r.Group(func(r chi.Router) {
		r.Use(authHandler.AuthMiddleware)
		r.Get("/api/streams/ws", streamHandler.StreamWebSocket)
	})

	// Background poller: queries SRS API for active streams.
	// Falls back when SRS http_hooks callbacks don't fire.
	go streamSvc.StartSRSPoller(context.Background())

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

func loadOrGenerateKeys() (privatePEM, publicPEM string) {
	privEnv := os.Getenv("JWT_PRIVATE_KEY")
	pubEnv := os.Getenv("JWT_PUBLIC_KEY")
	if privEnv != "" && pubEnv != "" {
		return privEnv, pubEnv
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("generate ecdsa key: %v", err)
	}

	privBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		log.Fatalf("marshal private key: %v", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})

	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		log.Fatalf("marshal public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	fmt.Fprintf(os.Stderr, "Generated new ECDSA P-256 key pair. Set JWT_PRIVATE_KEY and JWT_PUBLIC_KEY env vars for persistence.\n")
	return string(privPEM), string(pubPEM)
}

func newLogger() *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// chatAuthAdapter adapts the auth service to the chat module's chatAuth interface.
type chatAuthAdapter struct {
	authSvc *authapp.AuthService
}

func newChatAuthAdapter(authSvc *authapp.AuthService) *chatAuthAdapter {
	return &chatAuthAdapter{authSvc: authSvc}
}

func (a *chatAuthAdapter) ValidateToken(ctx context.Context, tokenStr string) (userID, userName, userAvatarUrl string, err error) {
	userID, err = a.authSvc.ValidateJWT(tokenStr)
	if err != nil {
		return "", "", "", err
	}
	user, err := a.authSvc.GetByID(ctx, userID)
	if err != nil {
		return "", "", "", err
	}
	avatar := ""
	if user.AvatarURL != nil {
		avatar = *user.AvatarURL
	}
	return userID, user.Name, avatar, nil
}
