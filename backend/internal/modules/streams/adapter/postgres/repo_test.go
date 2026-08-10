package postgres

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

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

// seedUserRaw inserts a user via the pool directly.
func seedUserRaw(t *testing.T, ctx context.Context, repo *StreamRepo, googleID, email, name string) string {
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

func TestStreamRepo_CreateStream(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedUserRaw(t, ctx, repo, "g-stream-create", "create@test.com", "Create User")

	title := "Test Stream"
	stream, err := repo.CreateStream(ctx, userID, &title, 1001, "/hls/live/test.m3u8")
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}
	if stream.UserID != userID {
		t.Fatalf("user_id = %q, want %q", stream.UserID, userID)
	}
	if stream.Status != "live" {
		t.Fatalf("status = %q, want 'live'", stream.Status)
	}
	if stream.Title == nil || *stream.Title != "Test Stream" {
		t.Fatalf("title = %v, want 'Test Stream'", stream.Title)
	}
	if stream.SRSClientID == nil || *stream.SRSClientID != 1001 {
		t.Fatalf("srs_client_id = %v, want 1001", stream.SRSClientID)
	}
	if stream.ID == "" {
		t.Fatal("expected non-empty stream ID")
	}
}

func TestStreamRepo_EndStream(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedUserRaw(t, ctx, repo, "g-stream-end", "end@test.com", "End User")

	title := "Stream to End"
	stream, err := repo.CreateStream(ctx, userID, &title, 1002, "/hls/live/test.m3u8")
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}

	err = repo.EndStream(ctx, stream.ID, "/hls/test.m3u8", "/rec/test.mp4", 3600)
	if err != nil {
		t.Fatalf("EndStream: %v", err)
	}

	// Fetch the stream via ListLiveStreams (should not appear since status is now offline)
	liveStreams, err := repo.ListLiveStreams(ctx)
	if err != nil {
		t.Fatalf("ListLiveStreams: %v", err)
	}
	for _, ls := range liveStreams {
		if ls.StreamID == stream.ID {
			t.Fatal("stream should no longer appear in live list")
		}
	}
}

func TestStreamRepo_UpdateStreamStatus(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedUserRaw(t, ctx, repo, "g-stream-status", "status@test.com", "Status User")

	title := "Status Stream"
	stream, err := repo.CreateStream(ctx, userID, &title, 1003, "/hls/live/test.m3u8")
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}

	err = repo.UpdateStreamStatus(ctx, stream.ID, "offline")
	if err != nil {
		t.Fatalf("UpdateStreamStatus: %v", err)
	}

	// Verify by checking the stream no longer appears in live list
	liveStreams, err := repo.ListLiveStreams(ctx)
	if err != nil {
		t.Fatalf("ListLiveStreams: %v", err)
	}
	for _, ls := range liveStreams {
		if ls.StreamID == stream.ID {
			t.Fatal("stream should be offline")
		}
	}
}

func TestStreamRepo_GetStreamByUserID(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedUserRaw(t, ctx, repo, "g-stream-byuid", "byuid@test.com", "ByUID User")

	t.Run("found", func(t *testing.T) {
		title := "Found Stream"
		created, err := repo.CreateStream(ctx, userID, &title, 2001, "/hls/live/test.m3u8")
		if err != nil {
			t.Fatalf("CreateStream: %v", err)
		}

		got, err := repo.GetStreamByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("GetStreamByUserID: %v", err)
		}
		if got.ID != created.ID {
			t.Fatalf("id = %q, want %q", got.ID, created.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetStreamByUserID(ctx, "00000000-0000-0000-0000-000000000000")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "no live stream") {
			t.Fatalf("expected 'no live stream' error, got %v", err)
		}
	})
}

func TestStreamRepo_GetStreamBySRSClientID(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedUserRaw(t, ctx, repo, "g-srs", "srs@test.com", "SRS User")

	t.Run("found", func(t *testing.T) {
		title := "SRS Stream"
		created, err := repo.CreateStream(ctx, userID, &title, 9999, "/hls/live/test.m3u8")
		if err != nil {
			t.Fatalf("CreateStream: %v", err)
		}

		got, err := repo.GetStreamBySRSClientID(ctx, 9999)
		if err != nil {
			t.Fatalf("GetStreamBySRSClientID: %v", err)
		}
		if got.ID != created.ID {
			t.Fatalf("id = %q, want %q", got.ID, created.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetStreamBySRSClientID(ctx, -1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not found error, got %v", err)
		}
	})
}

func TestStreamRepo_ListLiveStreams(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	t.Run("returns live streams", func(t *testing.T) {
		userID := seedUserRaw(t, ctx, repo, "g-list-live", "listlive@test.com", "List Live User")

		title := "Live Stream"
		_, err := repo.CreateStream(ctx, userID, &title, 3001, "/hls/live/test.m3u8")
		if err != nil {
			t.Fatalf("CreateStream: %v", err)
		}

		liveStreams, err := repo.ListLiveStreams(ctx)
		if err != nil {
			t.Fatalf("ListLiveStreams: %v", err)
		}
		if len(liveStreams) != 1 {
			t.Fatalf("expected 1 live stream, got %d", len(liveStreams))
		}
	})

	t.Run("empty", func(t *testing.T) {
		if err := testutil.TruncateAll(ctx, testPool); err != nil {
			t.Fatalf("truncate: %v", err)
		}
		liveStreams, err := repo.ListLiveStreams(ctx)
		if err != nil {
			t.Fatalf("ListLiveStreams: %v", err)
		}
		if len(liveStreams) != 0 {
			t.Fatalf("expected 0 live streams, got %d", len(liveStreams))
		}
	})
}

func TestStreamRepo_GetChannelInfo(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		userID := seedUserRaw(t, ctx, repo, "g-channel", "channel@test.com", "Channel User")

		info, err := repo.GetChannelInfo(ctx, userID)
		if err != nil {
			t.Fatalf("GetChannelInfo: %v", err)
		}
		if info.UserID != userID {
			t.Fatalf("userID = %q, want %q", info.UserID, userID)
		}
		if info.UserName != "Channel User" {
			t.Fatalf("userName = %q, want 'Channel User'", info.UserName)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetChannelInfo(ctx, "00000000-0000-0000-0000-000000000000")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not found error, got %v", err)
		}
	})
}

func TestStreamRepo_UpsertViewer(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedUserRaw(t, ctx, repo, "g-viewer", "viewer@test.com", "Viewer User")

	title := "Viewer Stream"
	stream, err := repo.CreateStream(ctx, userID, &title, 4001, "/hls/live/test.m3u8")
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}

	t.Run("inserts new", func(t *testing.T) {
		err := repo.UpsertViewer(ctx, stream.ID, "", "client-1")
		if err != nil {
			t.Fatalf("UpsertViewer: %v", err)
		}

		count, err := repo.GetViewerCount(ctx, stream.ID)
		if err != nil {
			t.Fatalf("GetViewerCount: %v", err)
		}
		if count != 1 {
			t.Fatalf("viewer count = %d, want 1", count)
		}
	})

	t.Run("updates existing", func(t *testing.T) {
		// Same client_id, should not duplicate
		err := repo.UpsertViewer(ctx, stream.ID, "", "client-1")
		if err != nil {
			t.Fatalf("UpsertViewer (update): %v", err)
		}

		count, err := repo.GetViewerCount(ctx, stream.ID)
		if err != nil {
			t.Fatalf("GetViewerCount: %v", err)
		}
		if count != 1 {
			t.Fatalf("viewer count after upsert = %d, want 1", count)
		}
	})
}

func TestStreamRepo_HeartbeatViewer(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedUserRaw(t, ctx, repo, "g-heartbeat", "heartbeat@test.com", "Heartbeat User")

	title := "Heartbeat Stream"
	stream, err := repo.CreateStream(ctx, userID, &title, 5001, "/hls/live/test.m3u8")
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}

	t.Run("updates existing", func(t *testing.T) {
		// Re-create stream after truncate
		userID2 := seedUserRaw(t, ctx, repo, "g-heartbeat2", "hb2@test.com", "HB User 2")
		stream2, err := repo.CreateStream(ctx, userID2, &title, 5002, "/hls/live/test.m3u8")
		if err != nil {
			t.Fatalf("CreateStream: %v", err)
		}

		if err := repo.UpsertViewer(ctx, stream2.ID, "", "client-hb"); err != nil {
			t.Fatalf("UpsertViewer: %v", err)
		}

		err = repo.HeartbeatViewer(ctx, stream2.ID, "client-hb", time.Now())
		if err != nil {
			t.Fatalf("HeartbeatViewer: %v", err)
		}
	})

	t.Run("not found returns error", func(t *testing.T) {
		err := repo.HeartbeatViewer(ctx, stream.ID, "nonexistent-client", time.Now())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not found error, got %v", err)
		}
	})
}

func TestStreamRepo_RemoveViewer(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedUserRaw(t, ctx, repo, "g-remove", "remove@test.com", "Remove User")

	title := "Remove Stream"
	stream, err := repo.CreateStream(ctx, userID, &title, 6001, "/hls/live/test.m3u8")
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}

	t.Run("removes viewer", func(t *testing.T) {
		if err := repo.UpsertViewer(ctx, stream.ID, "", "client-rm"); err != nil {
			t.Fatalf("UpsertViewer: %v", err)
		}

		if err := repo.RemoveViewer(ctx, stream.ID, "client-rm"); err != nil {
			t.Fatalf("RemoveViewer: %v", err)
		}

		count, err := repo.GetViewerCount(ctx, stream.ID)
		if err != nil {
			t.Fatalf("GetViewerCount: %v", err)
		}
		if count != 0 {
			t.Fatalf("viewer count = %d, want 0", count)
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		err := repo.RemoveViewer(ctx, stream.ID, "nonexistent-client")
		if err != nil {
			t.Fatalf("RemoveViewer (idempotent): %v", err)
		}
	})
}

func TestStreamRepo_GetViewerCount(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedUserRaw(t, ctx, repo, "g-vcount", "vcount@test.com", "VCount User")

	title := "VCount Stream"
	stream, err := repo.CreateStream(ctx, userID, &title, 7001, "/hls/live/test.m3u8")
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}

	if err := repo.UpsertViewer(ctx, stream.ID, "", "c1"); err != nil {
		t.Fatalf("UpsertViewer c1: %v", err)
	}
	if err := repo.UpsertViewer(ctx, stream.ID, "", "c2"); err != nil {
		t.Fatalf("UpsertViewer c2: %v", err)
	}

	count, err := repo.GetViewerCount(ctx, stream.ID)
	if err != nil {
		t.Fatalf("GetViewerCount: %v", err)
	}
	if count != 2 {
		t.Fatalf("viewer count = %d, want 2", count)
	}
}

func TestStreamRepo_GetAnalytics(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedUserRaw(t, ctx, repo, "g-analytics", "analytics@test.com", "Analytics User")

	today := time.Now().Format("2006-01-02")

	if err := repo.UpdateStreamAnalytics(ctx, userID, today, 1800, 50, 20); err != nil {
		t.Fatalf("UpdateStreamAnalytics: %v", err)
	}

	t.Run("week", func(t *testing.T) {
		a, err := repo.GetAnalytics(ctx, userID, "week")
		if err != nil {
			t.Fatalf("GetAnalytics(week): %v", err)
		}
		if a.TotalSeconds != 1800 {
			t.Fatalf("total_seconds = %d, want 1800", a.TotalSeconds)
		}
		if a.PeakViewers != 50 {
			t.Fatalf("peak_viewers = %d, want 50", a.PeakViewers)
		}
	})

	t.Run("month", func(t *testing.T) {
		a, err := repo.GetAnalytics(ctx, userID, "month")
		if err != nil {
			t.Fatalf("GetAnalytics(month): %v", err)
		}
		if a.TotalSeconds != 1800 {
			t.Fatalf("total_seconds = %d, want 1800", a.TotalSeconds)
		}
	})

	t.Run("all", func(t *testing.T) {
		a, err := repo.GetAnalytics(ctx, userID, "all")
		if err != nil {
			t.Fatalf("GetAnalytics(all): %v", err)
		}
		if a.TotalStreams != 1 {
			t.Fatalf("total_streams = %d, want 1", a.TotalStreams)
		}
	})
}

func TestStreamRepo_UpdateStreamAnalytics(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewStreamRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedUserRaw(t, ctx, repo, "g-ana-update", "anaupdate@test.com", "AnaUpdate User")

	today := time.Now().Format("2006-01-02")

	t.Run("inserts new", func(t *testing.T) {
		if err := repo.UpdateStreamAnalytics(ctx, userID, today, 1000, 30, 10); err != nil {
			t.Fatalf("UpdateStreamAnalytics: %v", err)
		}

		a, err := repo.GetAnalytics(ctx, userID, "week")
		if err != nil {
			t.Fatalf("GetAnalytics: %v", err)
		}
		if a.TotalSeconds != 1000 {
			t.Fatalf("total_seconds = %d, want 1000", a.TotalSeconds)
		}
	})

	t.Run("updates existing", func(t *testing.T) {
		if err := repo.UpdateStreamAnalytics(ctx, userID, today, 500, 60, 15); err != nil {
			t.Fatalf("UpdateStreamAnalytics (update): %v", err)
		}

		a, err := repo.GetAnalytics(ctx, userID, "week")
		if err != nil {
			t.Fatalf("GetAnalytics: %v", err)
		}
		// total_seconds = old 1000 + new 500 = 1500
		if a.TotalSeconds != 1500 {
			t.Fatalf("total_seconds = %d, want 1500", a.TotalSeconds)
		}
		// peak_viewers = GREATEST(old 30, new 60) = 60
		if a.PeakViewers != 60 {
			t.Fatalf("peak_viewers = %d, want 60", a.PeakViewers)
		}
	})
}
