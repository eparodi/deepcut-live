package application

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/deepcut/live/internal/modules/chat/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

// ---------------------------------------------------------------------------
// mockChatRepo implements domain.ChatRepository for service tests
// ---------------------------------------------------------------------------

type mockChatRepo struct {
	saveMessageFn     func(ctx context.Context, streamID, userID, message string) (*domain.ChatMessage, error)
	getMessagesFn     func(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error)
	getStreamStatusFn func(ctx context.Context, streamID string) (bool, error)
}

func (m *mockChatRepo) SaveMessage(ctx context.Context, streamID, userID, message string) (*domain.ChatMessage, error) {
	if m.saveMessageFn != nil {
		return m.saveMessageFn(ctx, streamID, userID, message)
	}
	return &domain.ChatMessage{
		ID:       "msg-1",
		StreamID: streamID,
		UserID:   userID,
		Message:  message,
		SentAt:   time.Now(),
	}, nil
}

func (m *mockChatRepo) GetMessages(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error) {
	if m.getMessagesFn != nil {
		return m.getMessagesFn(ctx, streamID, before, limit)
	}
	return []domain.ChatMessage{
		{ID: "msg-1", StreamID: streamID, UserID: "user-1", UserName: "Alice", UserAvatarUrl: "https://example.com/avatar.jpg", Message: "hello", SentAt: time.Now()},
	}, false, nil
}

func (m *mockChatRepo) GetStreamStatus(ctx context.Context, streamID string) (bool, error) {
	if m.getStreamStatusFn != nil {
		return m.getStreamStatusFn(ctx, streamID)
	}
	return true, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ---------------------------------------------------------------------------
// TestChatHubJoin
// ---------------------------------------------------------------------------

func TestChatHubJoin(t *testing.T) {
	hub := NewChatHub(&mockChatRepo{}, testLogger())
	client := &domain.ChatClient{UserID: "user-1", UserName: "Alice", Send: make(chan []byte, 256)}

	hub.Join("stream-1", client)

	hub.mu.RLock()
	clients, ok := hub.rooms["stream-1"]
	hub.mu.RUnlock()

	if !ok {
		t.Fatal("expected room to exist")
	}
	if !clients[client] {
		t.Fatal("expected client to be in room")
	}
}

// ---------------------------------------------------------------------------
// TestChatHubLeave
// ---------------------------------------------------------------------------

func TestChatHubLeave(t *testing.T) {
	hub := NewChatHub(&mockChatRepo{}, testLogger())
	client1 := &domain.ChatClient{UserID: "user-1", UserName: "Alice", Send: make(chan []byte, 256)}
	client2 := &domain.ChatClient{UserID: "user-2", UserName: "Bob", Send: make(chan []byte, 256)}

	hub.Join("stream-1", client1)
	hub.Join("stream-1", client2)

	// Leave client1 — room should still exist with client2.
	hub.Leave("stream-1", client1)

	hub.mu.RLock()
	clients, ok := hub.rooms["stream-1"]
	hub.mu.RUnlock()

	if !ok {
		t.Fatal("expected room to still exist")
	}
	if clients[client1] {
		t.Fatal("expected client1 to be removed")
	}
	if !clients[client2] {
		t.Fatal("expected client2 to still be in room")
	}

	// Leave client2 — room should be removed.
	hub.Leave("stream-1", client2)

	hub.mu.RLock()
	_, ok = hub.rooms["stream-1"]
	hub.mu.RUnlock()

	if ok {
		t.Fatal("expected room to be cleaned up")
	}
}

// ---------------------------------------------------------------------------
// TestChatHubBroadcast
// ---------------------------------------------------------------------------

func TestChatHubBroadcast(t *testing.T) {
	hub := NewChatHub(&mockChatRepo{}, testLogger())

	client := &domain.ChatClient{
		UserID:   "user-1",
		UserName: "Alice",
		Send:     make(chan []byte, 256),
	}
	hub.Join("stream-1", client)

	msg := &domain.ChatMessage{
		ID:            "msg-1",
		StreamID:      "stream-1",
		UserID:        "user-2",
		UserName:      "Bob",
		UserAvatarUrl: "https://example.com/bob.jpg",
		Message:       "hello everyone!",
		SentAt:        time.Now(),
	}

	hub.Broadcast("stream-1", msg)

	select {
	case data := <-client.Send:
		if len(data) == 0 {
			t.Fatal("expected non-empty broadcast data")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected to receive broadcast message")
	}
}

// ---------------------------------------------------------------------------
// TestChatHubBroadcastEmptyRoom
// ---------------------------------------------------------------------------

func TestChatHubBroadcastEmptyRoom(t *testing.T) {
	hub := NewChatHub(&mockChatRepo{}, testLogger())

	msg := &domain.ChatMessage{
		ID:       "msg-1",
		StreamID: "stream-1",
		UserID:   "user-2",
		Message:  "hello",
		SentAt:   time.Now(),
	}

	// Broadcasting to a non-existent room should not panic.
	hub.Broadcast("stream-1", msg)

	// Join, then leave, then broadcast — should not panic.
	client := &domain.ChatClient{UserID: "user-1", UserName: "Alice", Send: make(chan []byte, 256)}
	hub.Join("stream-1", client)
	hub.Leave("stream-1", client)
	hub.Broadcast("stream-1", msg)
}

// ---------------------------------------------------------------------------
// TestRateLimiting
// ---------------------------------------------------------------------------

func TestRateLimiting(t *testing.T) {
	hub := NewChatHub(&mockChatRepo{}, testLogger())

	// Should allow burst of 3.
	if !hub.AllowMessage("user-1") {
		t.Fatal("expected first message to be allowed")
	}
	if !hub.AllowMessage("user-1") {
		t.Fatal("expected second message to be allowed")
	}
	if !hub.AllowMessage("user-1") {
		t.Fatal("expected third message (burst) to be allowed")
	}

	// Fourth should be rate-limited.
	if hub.AllowMessage("user-1") {
		t.Fatal("expected fourth message to be rate-limited")
	}

	// Anonymous users are always rate-limited.
	if hub.AllowMessage("") {
		t.Fatal("expected empty userID to be rate-limited")
	}
}

// ---------------------------------------------------------------------------
// TestSendMessage
// ---------------------------------------------------------------------------

func TestSendMessage(t *testing.T) {
	tests := []struct {
		name          string
		streamID      string
		userID        string
		userName      string
		userAvatarUrl string
		message       string
		setupMock     func(*mockChatRepo)
		wantErr       bool
	}{
		{
			name:          "happy path — saves and broadcasts",
			streamID:      "stream-1",
			userID:        "user-1",
			userName:      "Alice",
			userAvatarUrl: "https://example.com/alice.jpg",
			message:       "Hello world!",
		},
		{
			name:          "happy path — empty message (passes through)",
			streamID:      "stream-1",
			userID:        "user-1",
			userName:      "Alice",
			userAvatarUrl: "",
			message:       "",
		},
		{
			name:     "error — save fails",
			streamID: "stream-1",
			userID:   "user-1",
			userName: "Alice",
			message:  "test",
			setupMock: func(m *mockChatRepo) {
				m.saveMessageFn = func(ctx context.Context, streamID, userID, message string) (*domain.ChatMessage, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockChatRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			hub := NewChatHub(repo, testLogger())
			svc := NewChatService(repo, hub)

			msg, err := svc.SendMessage(context.Background(), tt.streamID, tt.userID, tt.userName, tt.userAvatarUrl, tt.message)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr {
				if msg == nil {
					t.Fatal("expected non-nil message")
				}
				if msg.UserName != tt.userName {
					t.Fatalf("got userName %q, want %q", msg.UserName, tt.userName)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetMessages
// ---------------------------------------------------------------------------

func TestGetMessages(t *testing.T) {
	tests := []struct {
		name      string
		streamID  string
		before    string
		limit     int
		setupMock func(*mockChatRepo)
		wantCount int
		wantMore  bool
		wantErr   bool
	}{
		{
			name:      "happy path — returns messages",
			streamID:  "stream-1",
			limit:     50,
			wantCount: 1,
		},
		{
			name:      "happy path — empty list",
			streamID:  "stream-2",
			limit:     50,
			wantCount: 0,
			setupMock: func(m *mockChatRepo) {
				m.getMessagesFn = func(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error) {
					return []domain.ChatMessage{}, false, nil
				}
			},
		},
		{
			name:      "happy path — cursor-based pagination",
			streamID:  "stream-1",
			before:    "2026-01-01T00:00:00Z",
			limit:     10,
			wantCount: 1,
		},
		{
			name:      "happy path — hasMore true",
			streamID:  "stream-1",
			limit:     1,
			wantCount: 1,
			wantMore:  true,
			setupMock: func(m *mockChatRepo) {
				m.getMessagesFn = func(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error) {
					return []domain.ChatMessage{
						{ID: "msg-1"},
					}, true, nil
				}
			},
		},
		{
			name:     "error — not found",
			streamID: "stream-999",
			limit:    50,
			setupMock: func(m *mockChatRepo) {
				m.getMessagesFn = func(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error) {
					return nil, false, errs.NotFound("not found")
				}
			},
			wantErr: true,
		},
		{
			name:     "error — repo error",
			streamID: "stream-1",
			limit:    50,
			setupMock: func(m *mockChatRepo) {
				m.getMessagesFn = func(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error) {
					return nil, false, errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockChatRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			hub := NewChatHub(repo, testLogger())
			svc := NewChatService(repo, hub)

			msgs, hasMore, err := svc.GetMessages(context.Background(), tt.streamID, tt.before, tt.limit)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr {
				if len(msgs) != tt.wantCount {
					t.Fatalf("got %d messages, want %d", len(msgs), tt.wantCount)
				}
				if hasMore != tt.wantMore {
					t.Fatalf("got hasMore=%v, want %v", hasMore, tt.wantMore)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetHub
// ---------------------------------------------------------------------------

func TestGetHub(t *testing.T) {
	repo := &mockChatRepo{}
	hub := NewChatHub(repo, testLogger())
	svc := NewChatService(repo, hub)

	got := svc.GetHub()
	if got != hub {
		t.Fatal("expected same hub instance")
	}
}
