package postgres

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/deepcut/live/internal/modules/vods/domain"
	"github.com/deepcut/live/internal/testutil"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	pool, cleanup, err := testutil.SetupDB(ctx)
	if err != nil {
		log.Fatalf("setup test db: %v", err)
	}
	testPool = pool
	defer cleanup()

	os.Exit(m.Run())
}

// seedVODUser inserts a user directly into the pool for VOD tests.
func seedVODUser(t *testing.T, ctx context.Context, repo *VODRepo, googleID, email, name string) string {
	t.Helper()
	var userID string
	err := repo.pool.QueryRow(ctx,
		`INSERT INTO users (google_id, email, name, stream_key_hash) VALUES ($1, $2, $3, $4) RETURNING id`,
		googleID, email, name, "hash-"+googleID,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return userID
}

// seedVODStream inserts an offline stream (a VOD) and returns its ID.
func seedVODStream(t *testing.T, ctx context.Context, repo *VODRepo, userID string, title string, peakViewers, totalViewers, durationSeconds int) string {
	t.Helper()
	var streamID string
	err := repo.pool.QueryRow(ctx,
		`INSERT INTO streams (user_id, title, status, peak_viewers, total_viewers, duration_seconds, recording_status, started_at, ended_at)
		 VALUES ($1, $2, 'offline', $3, $4, $5, 'ready', now() - interval '1 hour', now()) RETURNING id`,
		userID, title, peakViewers, totalViewers, durationSeconds,
	).Scan(&streamID)
	if err != nil {
		t.Fatalf("seed vod stream: %v", err)
	}
	return streamID
}

func TestVODRepo_GetVOD(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewVODRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedVODUser(t, ctx, repo, "g-vod-get", "vodget@test.com", "VOD Get User")

	t.Run("found (offline stream)", func(t *testing.T) {
		vodID := seedVODStream(t, ctx, repo, userID, "Test VOD", 100, 500, 3600)

		vod, err := repo.GetVOD(ctx, vodID)
		if err != nil {
			t.Fatalf("GetVOD: %v", err)
		}
		if vod.ID != vodID {
			t.Fatalf("id = %q, want %q", vod.ID, vodID)
		}
		if vod.Title == nil || *vod.Title != "Test VOD" {
			t.Fatalf("title = %v, want 'Test VOD'", vod.Title)
		}
		if vod.UserName != "VOD Get User" {
			t.Fatalf("user_name = %q, want 'VOD Get User'", vod.UserName)
		}
	})

	t.Run("not found (doesn't exist)", func(t *testing.T) {
		_, err := repo.GetVOD(ctx, "00000000-0000-0000-0000-000000000000")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not found error, got %v", err)
		}
	})
}

func TestVODRepo_ListVODs(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewVODRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedVODUser(t, ctx, repo, "g-vod-list", "vodlist@test.com", "VOD List User")

	seedVODStream(t, ctx, repo, userID, "VOD 1", 50, 200, 1800)
	seedVODStream(t, ctx, repo, userID, "VOD 2", 100, 300, 3600)
	seedVODStream(t, ctx, repo, userID, "VOD 3", 75, 250, 2700)

	t.Run("returns VODs for user", func(t *testing.T) {
		vods, err := repo.ListVODs(ctx, userID, 10, 0)
		if err != nil {
			t.Fatalf("ListVODs: %v", err)
		}
		if len(vods) != 3 {
			t.Fatalf("expected 3 VODs, got %d", len(vods))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		vods, err := repo.ListVODs(ctx, "00000000-0000-0000-0000-000000000000", 10, 0)
		if err != nil {
			t.Fatalf("ListVODs (empty): %v", err)
		}
		if len(vods) != 0 {
			t.Fatalf("expected 0 VODs, got %d", len(vods))
		}
	})

	t.Run("pagination", func(t *testing.T) {
		vods, err := repo.ListVODs(ctx, userID, 2, 0)
		if err != nil {
			t.Fatalf("ListVODs (limit 2): %v", err)
		}
		if len(vods) != 2 {
			t.Fatalf("expected 2 VODs, got %d", len(vods))
		}

		vods2, err := repo.ListVODs(ctx, userID, 2, 2)
		if err != nil {
			t.Fatalf("ListVODs (offset 2): %v", err)
		}
		if len(vods2) != 1 {
			t.Fatalf("expected 1 VOD, got %d", len(vods2))
		}
	})
}

func TestVODRepo_SearchVODs(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewVODRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedVODUser(t, ctx, repo, "g-vod-search", "vodsearch@test.com", "SearchUser")

	seedVODStream(t, ctx, repo, userID, "Amazing Stream", 200, 1000, 7200)
	seedVODStream(t, ctx, repo, userID, "Boring Stream", 10, 50, 600)
	seedVODStream(t, ctx, repo, userID, "Long Marathon", 150, 800, 14400)

	t.Run("with query", func(t *testing.T) {
		result, err := repo.SearchVODs(ctx, domain.SearchParams{
			Query:  "amazing",
			Limit:  10,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("SearchVODs: %v", err)
		}
		if result.TotalCount != 1 {
			t.Fatalf("total_count = %d, want 1", result.TotalCount)
		}
		if len(result.VODs) != 1 {
			t.Fatalf("expected 1 VOD, got %d", len(result.VODs))
		}
	})

	t.Run("with status filter", func(t *testing.T) {
		result, err := repo.SearchVODs(ctx, domain.SearchParams{
			Status: "ready",
			Limit:  10,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("SearchVODs: %v", err)
		}
		if result.TotalCount != 3 {
			t.Fatalf("total_count = %d, want 3", result.TotalCount)
		}
	})

	t.Run("sort by popular", func(t *testing.T) {
		result, err := repo.SearchVODs(ctx, domain.SearchParams{
			Sort:   "popular",
			Limit:  10,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("SearchVODs: %v", err)
		}
		if len(result.VODs) != 3 {
			t.Fatalf("expected 3 VODs, got %d", len(result.VODs))
		}
		// Most popular should be first (peak_viewers=200)
		if result.VODs[0].PeakViewers != 200 {
			t.Fatalf("first VOD peak_viewers = %d, want 200", result.VODs[0].PeakViewers)
		}
	})

	t.Run("sort by longest", func(t *testing.T) {
		result, err := repo.SearchVODs(ctx, domain.SearchParams{
			Sort:   "longest",
			Limit:  10,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("SearchVODs: %v", err)
		}
		if len(result.VODs) != 3 {
			t.Fatalf("expected 3 VODs, got %d", len(result.VODs))
		}
		// Longest should be first (duration_seconds=14400)
		if result.VODs[0].DurationSeconds == nil || *result.VODs[0].DurationSeconds != 14400 {
			t.Fatalf("first VOD duration_seconds = %v, want 14400", result.VODs[0].DurationSeconds)
		}
	})

	t.Run("pagination with total count", func(t *testing.T) {
		result, err := repo.SearchVODs(ctx, domain.SearchParams{
			Sort:   "recent",
			Limit:  2,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("SearchVODs: %v", err)
		}
		if result.TotalCount != 3 {
			t.Fatalf("total_count = %d, want 3", result.TotalCount)
		}
		if len(result.VODs) != 2 {
			t.Fatalf("expected 2 VODs, got %d", len(result.VODs))
		}
		if result.Limit != 2 {
			t.Fatalf("limit = %d, want 2", result.Limit)
		}
	})
}

func TestVODRepo_IncrementViewCount(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewVODRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedVODUser(t, ctx, repo, "g-vod-incr", "vodincr@test.com", "VOD Incr User")

	vodID := seedVODStream(t, ctx, repo, userID, "View Count Stream", 5, 10, 1800)

	err := repo.IncrementViewCount(ctx, vodID)
	if err != nil {
		t.Fatalf("IncrementViewCount: %v", err)
	}

	// Fetch the VOD to verify counts
	vod, err := repo.GetVOD(ctx, vodID)
	if err != nil {
		t.Fatalf("GetVOD: %v", err)
	}
	if vod.TotalViewers != 11 {
		t.Fatalf("total_viewers = %d, want 11", vod.TotalViewers)
	}
	// peak_viewers should be GREATEST(5, 11) = 11
	if vod.PeakViewers != 11 {
		t.Fatalf("peak_viewers = %d, want 11", vod.PeakViewers)
	}
}
