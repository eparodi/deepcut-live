package river

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/deepcut/live/internal/modules/vods/domain"
)

// Queue implements domain.VODQueue using River (PostgreSQL-backed job queue).
// The main server uses this to enqueue VOD processing jobs; a separate
// cmd/worker binary runs the actual processing.
type Queue struct {
	client *river.Client[pgx.Tx]
}

// NewQueue creates a River-backed VOD queue. The pool is used for job
// insertion only — job processing happens in cmd/worker.
func NewQueue(pool *pgxpool.Pool) (*Queue, error) {
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			// River requires at least 1 worker per queue even though this
			// client only inserts jobs (processing happens in cmd/worker).
			river.QueueDefault: {MaxWorkers: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("river new client: %w", err)
	}

	return &Queue{client: client}, nil
}

// Enqueue inserts a VOD processing job into the River queue.
func (q *Queue) Enqueue(ctx context.Context, args domain.VODProcessArgs) error {
	// Guard against typed-nil receiver (interface holding (*Queue)(nil))
	if q == nil || q.client == nil {
		return fmt.Errorf("vod queue not initialized")
	}
	_, err := q.client.Insert(ctx, &args, nil)
	if err != nil {
		return fmt.Errorf("river insert: %w", err)
	}
	return nil
}

// Close stops the River client. Call during graceful shutdown.
func (q *Queue) Close() error {
	return q.client.Stop(context.Background())
}
