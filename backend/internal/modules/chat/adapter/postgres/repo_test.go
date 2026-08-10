package postgres

import (
	"context"
	"log"
	"os"
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

// seedChatUser inserts a user directly into the pool for chat tests.
func seedChatUser(t *testing.T, ctx context.Context, repo *ChatRepo, googleID, email, name string) string {
	t.Helper()
	var userID string
	err := repo.pool.QueryRow(ctx,
		`INSERT INTO users (google_id, email, name, avatar_url, stream_key_hash) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		googleID, email, name, "https://example.com/"+name+".jpg", "hash-"+googleID,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return userID
}

// seedChatStream inserts a stream directly into the pool for chat tests.
func seedChatStream(t *testing.T, ctx context.Context, repo *ChatRepo, userID string) string {
	t.Helper()
	var streamID string
	err := repo.pool.QueryRow(ctx,
		`INSERT INTO streams (user_id, status, srs_client_id) VALUES ($1, 'live', 1) RETURNING id`,
		userID,
	).Scan(&streamID)
	if err != nil {
		t.Fatalf("seed stream: %v", err)
	}
	return streamID
}

func TestChatRepo_SaveMessage(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewChatRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedChatUser(t, ctx, repo, "g-chat-save", "chatsave@test.com", "Chat Save")
	streamID := seedChatStream(t, ctx, repo, userID)

	msg, err := repo.SaveMessage(ctx, streamID, userID, "Hello, world!")
	if err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}
	if msg.ID == "" {
		t.Fatal("expected non-empty message ID")
	}
	if msg.StreamID != streamID {
		t.Fatalf("stream_id = %q, want %q", msg.StreamID, streamID)
	}
	if msg.UserID != userID {
		t.Fatalf("user_id = %q, want %q", msg.UserID, userID)
	}
	if msg.Message != "Hello, world!" {
		t.Fatalf("message = %q, want 'Hello, world!'", msg.Message)
	}
	if msg.SentAt.IsZero() {
		t.Fatal("expected non-zero sent_at")
	}
}

func TestChatRepo_GetStreamStatus(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewChatRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedChatUser(t, ctx, repo, "g-status", "status@test.com", "Status User")
	streamID := seedChatStream(t, ctx, repo, userID)

	isLive, err := repo.GetStreamStatus(ctx, streamID)
	if err != nil {
		t.Fatalf("GetStreamStatus: %v", err)
	}
	if !isLive {
		t.Fatal("expected stream to be live")
	}
}

func TestChatRepo_GetMessages(t *testing.T) {
	testutil.SkipOnShort(t)
	ctx := context.Background()
	repo := NewChatRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	userID := seedChatUser(t, ctx, repo, "g-chat-get", "chatget@test.com", "Chat Get")
	streamID := seedChatStream(t, ctx, repo, userID)

	// Save a few messages (with small gaps so timestamps differ)
	_, err := repo.SaveMessage(ctx, streamID, userID, "msg 1")
	if err != nil {
		t.Fatalf("SaveMessage 1: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	_, err = repo.SaveMessage(ctx, streamID, userID, "msg 2")
	if err != nil {
		t.Fatalf("SaveMessage 2: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	_, err = repo.SaveMessage(ctx, streamID, userID, "msg 3")
	if err != nil {
		t.Fatalf("SaveMessage 3: %v", err)
	}

	t.Run("returns messages with user join", func(t *testing.T) {
		msgs, hasMore, err := repo.GetMessages(ctx, streamID, "", 10)
		if err != nil {
			t.Fatalf("GetMessages: %v", err)
		}
		if len(msgs) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(msgs))
		}
		if hasMore {
			t.Fatal("expected hasMore=false")
		}
		// Messages should be ordered by sent_at DESC
		if msgs[0].Message != "msg 3" {
			t.Fatalf("first message = %q, want 'msg 3'", msgs[0].Message)
		}
		// Check user name and avatar are populated via JOIN
		if msgs[0].UserName != "Chat Get" {
			t.Fatalf("user_name = %q, want 'Chat Get'", msgs[0].UserName)
		}
		if msgs[0].UserAvatarUrl != "https://example.com/Chat Get.jpg" {
			t.Fatalf("user_avatar_url = %q, want 'https://example.com/Chat Get.jpg'", msgs[0].UserAvatarUrl)
		}
	})

	t.Run("not found returns error", func(t *testing.T) {
		_, _, err := repo.GetMessages(ctx, "00000000-0000-0000-0000-000000000000", "", 10)
		if err == nil {
			t.Fatal("expected error for non-existent stream")
		}
	})

	t.Run("pagination with hasMore", func(t *testing.T) {
		msgs, hasMore, err := repo.GetMessages(ctx, streamID, "", 2)
		if err != nil {
			t.Fatalf("GetMessages (limit 2): %v", err)
		}
		if len(msgs) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(msgs))
		}
		if !hasMore {
			t.Fatal("expected hasMore=true")
		}

		// Use cursor (before) to get the next page.
		// The oldest message returned was "msg 2", so use its sent_at as cursor.
		cursor := msgs[1].SentAt.Format(time.RFC3339Nano)
		msgs2, hasMore2, err := repo.GetMessages(ctx, streamID, cursor, 10)
		if err != nil {
			t.Fatalf("GetMessages (cursor): %v", err)
		}
		if len(msgs2) != 1 {
			t.Fatalf("expected 1 message, got %d", len(msgs2))
		}
		if hasMore2 {
			t.Fatal("expected hasMore=false for last page")
		}
		if msgs2[0].Message != "msg 1" {
			t.Fatalf("cursor message = %q, want 'msg 1'", msgs2[0].Message)
		}
	})
}
