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
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	port := env("PORT", "8081")
	dbURL := env("DATABASE_URL", "postgres://live:live@localhost:5432/live?sslmode=disable")
	googleClientID := env("GOOGLE_CLIENT_ID", "")
	googleClientSecret := env("GOOGLE_CLIENT_SECRET", "")
	baseURL := env("BASE_URL", "http://localhost:3000")
	srsSecret := env("SRS_CALLBACK_SECRET", "dev-srs-secret")

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

	authSvc := authapp.NewAuthService(authRepo, googleClientID, googleClientSecret, baseURL, privateKeyPEM, publicKeyPEM)
	streamSvc := streamapp.NewStreamService(streamRepo, authRepo, srsSecret)
	vodSvc := vodapp.NewVODService(vodRepo)

	authHandler := authhttp.NewAuthHandler(authSvc, streamSvc, baseURL, logger)
	streamHandler := streamhttp.NewStreamHandler(streamSvc, logger)
	vodHandler := vodhttp.NewVODHandler(vodSvc, logger)

	r := chi.NewRouter()
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

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	authHandler.RegisterRoutes(r)
	streamHandler.RegisterRoutes(r)
	vodHandler.RegisterRoutes(r)

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

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
