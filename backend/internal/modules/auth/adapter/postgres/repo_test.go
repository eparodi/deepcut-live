package postgres

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/deepcut/live/internal/shared/errs"
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

func TestAuthRepo_CreateUser(t *testing.T) {
	ctx := context.Background()
	repo := NewAuthRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	tests := []struct {
		name        string
		googleID    string
		email       string
		nameVal     string
		avatarURL   string
		keyHash     string
		wantErr     bool
		errContains string
	}{
		{
			name:      "happy path",
			googleID:  "g-123",
			email:     "test@example.com",
			nameVal:   "Test User",
			avatarURL: "https://example.com/avatar.png",
			keyHash:   "hash-abc",
		},
		{
			name:        "duplicate google_id",
			googleID:    "g-123",
			email:       "test2@example.com",
			nameVal:     "Test User 2",
			avatarURL:   "",
			keyHash:     "hash-def",
			wantErr:     true,
			errContains: "duplicate key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.CreateUser(ctx, tt.googleID, tt.email, tt.nameVal, tt.avatarURL, tt.keyHash)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Fatalf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if user == nil {
				t.Fatal("expected user, got nil")
			}
			if user.GoogleID != tt.googleID {
				t.Fatalf("google_id = %q, want %q", user.GoogleID, tt.googleID)
			}
			if user.Email != tt.email {
				t.Fatalf("email = %q, want %q", user.Email, tt.email)
			}
			if user.Name != tt.nameVal {
				t.Fatalf("name = %q, want %q", user.Name, tt.nameVal)
			}
		})
	}
}

func TestAuthRepo_GetByGoogleID(t *testing.T) {
	ctx := context.Background()
	repo := NewAuthRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	// Seed a user
	_, err := repo.CreateUser(ctx, "g-123", "test@example.com", "Test User", "", "hash-abc")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		user, err := repo.GetByGoogleID(ctx, "g-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.GoogleID != "g-123" {
			t.Fatalf("google_id = %q, want %q", user.GoogleID, "g-123")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByGoogleID(ctx, "g-nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var appErr *errs.AppError
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not found error, got %v", err)
		}
		_ = appErr
	})
}

func TestAuthRepo_GetByID(t *testing.T) {
	ctx := context.Background()
	repo := NewAuthRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	user, err := repo.CreateUser(ctx, "g-123", "test@example.com", "Test User", "", "hash-abc")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != user.ID {
			t.Fatalf("id = %q, want %q", got.ID, user.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not found error, got %v", err)
		}
	})
}

func TestAuthRepo_GetByStreamKeyHash(t *testing.T) {
	ctx := context.Background()
	repo := NewAuthRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	_, err := repo.CreateUser(ctx, "g-123", "test@example.com", "Test User", "", "hash-abc")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		user, err := repo.GetByStreamKeyHash(ctx, "hash-abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.GoogleID != "g-123" {
			t.Fatalf("google_id = %q, want %q", user.GoogleID, "g-123")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByStreamKeyHash(ctx, "nonexistent-hash")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not found error, got %v", err)
		}
	})
}

func TestAuthRepo_GetUserIDByStreamKeyHash(t *testing.T) {
	ctx := context.Background()
	repo := NewAuthRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	user, err := repo.CreateUser(ctx, "g-123", "test@example.com", "Test User", "", "hash-abc")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		id, err := repo.GetUserIDByStreamKeyHash(ctx, "hash-abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != user.ID {
			t.Fatalf("id = %q, want %q", id, user.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetUserIDByStreamKeyHash(ctx, "nonexistent-hash")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not found error, got %v", err)
		}
	})
}

func TestAuthRepo_UpdateStreamKey(t *testing.T) {
	ctx := context.Background()
	repo := NewAuthRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	user, err := repo.CreateUser(ctx, "g-123", "test@example.com", "Test User", "", "hash-old")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	err = repo.UpdateStreamKey(ctx, user.ID, "hash-new")
	if err != nil {
		t.Fatalf("UpdateStreamKey: %v", err)
	}

	// Verify the update took effect
	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if got.StreamKeyHash != "hash-new" {
		t.Fatalf("stream_key_hash = %q, want %q", got.StreamKeyHash, "hash-new")
	}
}

func TestAuthRepo_UpdateSettings(t *testing.T) {
	ctx := context.Background()
	repo := NewAuthRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	user, err := repo.CreateUser(ctx, "g-123", "test@example.com", "Test User", "", "hash-abc")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	err = repo.UpdateSettings(ctx, user.ID, "New Title", "Gaming")
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if got.StreamTitle == nil || *got.StreamTitle != "New Title" {
		t.Fatalf("stream_title = %v, want 'New Title'", got.StreamTitle)
	}
	if got.StreamCategory == nil || *got.StreamCategory != "Gaming" {
		t.Fatalf("stream_category = %v, want 'Gaming'", got.StreamCategory)
	}
}

func TestAuthRepo_SetLiveStatus(t *testing.T) {
	ctx := context.Background()
	repo := NewAuthRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	user, err := repo.CreateUser(ctx, "g-123", "test@example.com", "Test User", "", "hash-abc")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("set true", func(t *testing.T) {
		if err := repo.SetLiveStatus(ctx, user.ID, true); err != nil {
			t.Fatalf("SetLiveStatus(true): %v", err)
		}
		got, err := repo.GetByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if !got.IsLive {
			t.Fatal("expected is_live = true")
		}
	})

	t.Run("set false", func(t *testing.T) {
		if err := repo.SetLiveStatus(ctx, user.ID, false); err != nil {
			t.Fatalf("SetLiveStatus(false): %v", err)
		}
		got, err := repo.GetByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.IsLive {
			t.Fatal("expected is_live = false")
		}
	})
}

func TestAuthRepo_GetLiveUsers(t *testing.T) {
	ctx := context.Background()
	repo := NewAuthRepo(testPool)
	if err := testutil.TruncateAll(ctx, testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	t.Run("returns empty when none are live", func(t *testing.T) {
		if err := testutil.TruncateAll(ctx, testPool); err != nil {
			t.Fatalf("truncate: %v", err)
		}

		_, err := repo.CreateUser(ctx, "g-1", "u1@example.com", "User 1", "", "hash-1")
		if err != nil {
			t.Fatalf("seed user: %v", err)
		}

		users, err := repo.GetLiveUsers(ctx)
		if err != nil {
			t.Fatalf("GetLiveUsers: %v", err)
		}
		if len(users) != 0 {
			t.Fatalf("expected 0 live users, got %d", len(users))
		}
	})

	t.Run("returns live users", func(t *testing.T) {
		if err := testutil.TruncateAll(ctx, testPool); err != nil {
			t.Fatalf("truncate: %v", err)
		}

		u1, err := repo.CreateUser(ctx, "g-1", "u1@example.com", "User 1", "", "hash-1")
		if err != nil {
			t.Fatalf("seed user 1: %v", err)
		}
		u2, err := repo.CreateUser(ctx, "g-2", "u2@example.com", "User 2", "", "hash-2")
		if err != nil {
			t.Fatalf("seed user 2: %v", err)
		}

		if err := repo.SetLiveStatus(ctx, u1.ID, true); err != nil {
			t.Fatalf("set live u1: %v", err)
		}

		users, err := repo.GetLiveUsers(ctx)
		if err != nil {
			t.Fatalf("GetLiveUsers: %v", err)
		}
		if len(users) != 1 {
			t.Fatalf("expected 1 live user, got %d", len(users))
		}
		if users[0].ID != u1.ID {
			t.Fatalf("expected user %s, got %s", u1.ID, users[0].ID)
		}
		_ = u2 // u2 is offline, should not appear
	})
}
