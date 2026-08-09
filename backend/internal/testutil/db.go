package testutil

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupTestDB starts a Postgres container, runs migrations, and returns a pool.
// Each call creates a new container. Prefer SetupSharedDB + TestMain for speed.
func SetupTestDB(t *testing.T) (*pgxpool.Pool, testcontainers.Container) {
	t.Helper()
	pool, container, err := setupDB(context.Background())
	if err != nil {
		t.Fatalf("setup test db: %v", err)
	}
	return pool, container
}

// SetupDB connects to a Postgres database. If DATABASE_URL is set, it connects
// to that directly (for CI with a service container). Otherwise it starts a
// testcontainers Postgres container (for local dev with Docker).
// Returns a pool and a cleanup function (caller must defer cleanup()).
func SetupDB(ctx context.Context) (*pgxpool.Pool, func(), error) {
	if isShortFlag() {
		return nil, func() {}, nil
	}
	if url := os.Getenv("DATABASE_URL"); url != "" {
		// CI mode: connect directly; migrations were already run by golang-migrate.
		pool, err := pgxpool.New(ctx, url)
		if err != nil {
			return nil, nil, fmt.Errorf("connect to DATABASE_URL: %w", err)
		}
		return pool, func() { pool.Close() }, nil
	}

	pool, container, err := setupDB(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pool, func() {
		pool.Close()
		if err := container.Terminate(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "terminate container: %v\n", err)
		}
	}, nil
}

func setupDB(ctx context.Context) (*pgxpool.Pool, testcontainers.Container, error) {
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("start postgres container: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("get connection string: %w", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to test db: %w", err)
	}

	if err := runMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("run migrations: %w", err)
	}

	return pool, pgContainer, nil
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			google_id TEXT UNIQUE NOT NULL,
			email TEXT NOT NULL,
			name TEXT NOT NULL,
			avatar_url TEXT,
			stream_key_hash TEXT NOT NULL,
			stream_title TEXT,
			stream_category TEXT,
			is_live BOOLEAN DEFAULT false,
			live_since TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT now(),
			updated_at TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS streams (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id),
			title TEXT,
			started_at TIMESTAMPTZ DEFAULT now(),
			ended_at TIMESTAMPTZ,
			status TEXT DEFAULT 'live',
			hls_path TEXT,
			recording_path TEXT,
			recording_status TEXT DEFAULT 'pending',
			peak_viewers INT DEFAULT 0,
			total_viewers INT DEFAULT 0,
			duration_seconds INT,
			srs_client_id INT,
			created_at TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS stream_viewers (
			stream_id UUID NOT NULL REFERENCES streams(id),
			user_id UUID,
			client_id TEXT NOT NULL,
			last_seen TIMESTAMPTZ DEFAULT now(),
			PRIMARY KEY (stream_id, client_id)
		)`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			stream_id UUID NOT NULL REFERENCES streams(id),
			user_id UUID NOT NULL REFERENCES users(id),
			message TEXT NOT NULL,
			sent_at TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS stream_analytics (
			user_id UUID NOT NULL REFERENCES users(id),
			date DATE NOT NULL,
			total_seconds INT DEFAULT 0,
			peak_viewers INT DEFAULT 0,
			unique_viewers INT DEFAULT 0,
			PRIMARY KEY (user_id, date)
		)`,
	}
	for _, m := range migrations {
		if _, err := pool.Exec(ctx, m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}
	return nil
}

// TruncateAll deletes all rows from all tables for test isolation.
func TruncateAll(ctx context.Context, pool *pgxpool.Pool) error {
	tables := []string{"stream_analytics", "stream_viewers", "chat_messages", "streams", "users"}
	for _, table := range tables {
		if _, err := pool.Exec(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}
	return nil
}

// TruncateAllT is a testing.T-aware wrapper for use in non-TestMain tests.
func TruncateAllT(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if err := TruncateAll(ctx, pool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// isShortFlag checks os.Args for -test.short (without calling testing.Short,
// which panics if flag.Parse hasn't been called yet — as happens in TestMain).
func isShortFlag() bool {
	for _, a := range os.Args {
		if a == "-test.short" || strings.HasPrefix(a, "-test.short=") {
			return true
		}
	}
	return false
}

// FatalIf panics with a message to os.Stderr — for use in TestMain where testing.T is unavailable.
func FatalIf(err error, msg string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
		os.Exit(1)
	}
}

// SkipOnShort skips the test if the -short flag is set. Call this as the first
// line in any test function that requires a database connection.
func SkipOnShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping DB-dependent test in short mode")
	}
}
