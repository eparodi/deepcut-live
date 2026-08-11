package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
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

	// Run River schema migration before starting
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		logger.Error("failed to create river migrator", "err", err)
		os.Exit(1)
	}
	if _, err := migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil); err != nil {
		logger.Error("river migration failed", "err", err)
		os.Exit(1)
	}

	vodWorker := vodapp.NewVODWorker(pool, logger)

	workers := river.NewWorkers()
	river.AddWorker[voddomain.VODProcessArgs](workers, vodWorker)

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 1},
		},
		Workers: workers,
	})
	if err != nil {
		logger.Error("failed to create river client", "err", err)
		os.Exit(1)
	}

	logger.Info("VOD worker started, waiting for jobs")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := client.Start(ctx); err != nil {
		logger.Error("worker stopped with error", "err", err)
		os.Exit(1)
	}

	fmt.Println("worker shut down gracefully")
}
