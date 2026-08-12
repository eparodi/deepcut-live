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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	vodapp "github.com/deepcut/live/internal/modules/vods/application"
	voddomain "github.com/deepcut/live/internal/modules/vods/domain"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://live:live@localhost:5432/live?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := migrateRiverSchema(pool, logger); err != nil {
		logger.Error("river migration failed", "err", err)
		os.Exit(1)
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
		logger.Error("failed to create river client", "err", err)
		os.Exit(1)
	}

	// Start the client in the background (non-blocking).
	if err := client.Start(context.Background()); err != nil {
		logger.Error("client start failed", "err", err)
		os.Exit(1)
	}

	logger.Info("VOD worker started, waiting for jobs")

	// Block until shutdown signal.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	<-ctx.Done()

	logger.Info("shutting down worker")
	if err := client.Stop(context.Background()); err != nil {
		logger.Error("client stop failed", "err", err)
	}
	fmt.Println("worker shut down gracefully")
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
