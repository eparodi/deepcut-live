package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	vodapp "github.com/deepcut/live/internal/modules/vods/application"
	voddomain "github.com/deepcut/live/internal/modules/vods/domain"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("worker exited with error", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://live:live@localhost:5432/live?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer pool.Close()

	if err := migrateRiverSchema(pool, logger); err != nil {
		return fmt.Errorf("river migration: %w", err)
	}

	vodWorker := vodapp.NewVODWorker(pool, logger)

	workers := river.NewWorkers()
	river.AddWorker[voddomain.VODProcessArgs](workers, vodWorker)

	maxWorkers := 1
	if v := os.Getenv("VOD_WORKER_MAX_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxWorkers = n
		}
	}

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: maxWorkers},
		},
		Workers: workers,
	})
	if err != nil {
		return fmt.Errorf("create river client: %w", err)
	}

	// Start the client in the background (non-blocking).
	if err := client.Start(context.Background()); err != nil {
		return fmt.Errorf("start river client: %w", err)
	}

	logger.Info("VOD worker started, waiting for jobs")

	// Block until shutdown signal.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	<-ctx.Done()

	logger.Info("shutting down worker")
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()
	if err := client.Stop(stopCtx); err != nil {
		logger.Error("client stop failed", "err", err)
	}
	logger.Info("worker shut down gracefully")
	return nil
}

func migrateRiverSchema(pool *pgxpool.Pool, logger *slog.Logger) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	existing, err := migrator.ExistingVersions(context.Background())
	if err != nil {
		if !strings.Contains(err.Error(), "does not exist") {
			return fmt.Errorf("check existing versions: %w", err)
		}
		existing = nil
	}

	logger.Info("river migration check", "already_applied", len(existing))

	_, err = migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil)
	if err != nil {
		var exists bool
		if scanErr := pool.QueryRow(context.Background(),
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'river_queue')",
		).Scan(&exists); scanErr == nil && exists {
			logger.Warn("river schema exists despite migration error, continuing", "migration_err", err)
			return nil
		}
		return fmt.Errorf("migrate: %w", err)
	}

	return nil
}
