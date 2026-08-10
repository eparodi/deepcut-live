package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/deepcut/live/internal/modules/chat/application"
	"github.com/deepcut/live/internal/modules/chat/domain"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestHub returns a properly initialised ChatHub (non-nil rooms map).
func newTestHub() *application.ChatHub {
	return application.NewChatHub(nil, testLogger())
}

// decodeBroadcast attempts to decode a []byte received from the WebSocket
// back into the spec envelope: {"type":"message","payload":{...}}.
func decodeBroadcast(t *testing.T, data []byte) domain.ChatMessage {
	t.Helper()

	var envelope struct {
		Type    string             `json:"type"`
		Payload domain.ChatMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal received data: %v (raw=%s)", err, string(data))
	}
	if envelope.Type != "message" {
		t.Fatalf("expected type 'message', got %q", envelope.Type)
	}
	return envelope.Payload
}

// newTestAuth returns a mock auth that always authenticates.
func newTestAuth() *mockChatAuth {
	return &mockChatAuth{}
}

// ---------------------------------------------------------------------------
// TestChatWebSocket_MissingParams
// ---------------------------------------------------------------------------

func TestChatWebSocket_MissingParams(t *testing.T) {
	tests := []struct {
		name     string
		streamID string
		wantCode int
	}{
		{
			name:     "missing streamID",
			streamID: "",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockChatService{
				getHubFn:       newTestHub,
				isStreamLiveFn: func(ctx context.Context, streamID string) (bool, error) { return true, nil },
			}
			h := NewChatHandler(svc, newTestAuth(), testLogger())

			u := "/ws/chat/" + tt.streamID
			req := httptest.NewRequest(http.MethodGet, u, nil)

			if tt.streamID != "" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("streamID", tt.streamID)
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}

			rec := httptest.NewRecorder()
			h.ChatWebSocket(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestChatWebSocket_Integration
// ---------------------------------------------------------------------------

func TestChatWebSocket_Integration(t *testing.T) {
	hub := newTestHub()

	type capturedMsg struct {
		streamID      string
		userID        string
		userName      string
		userAvatarUrl string
		message       string
	}
	msgCh := make(chan capturedMsg, 1)

	svc := &mockChatService{
		getHubFn:       func() *application.ChatHub { return hub },
		isStreamLiveFn: func(ctx context.Context, streamID string) (bool, error) { return true, nil },
		getMessagesFn: func(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error) {
			return []domain.ChatMessage{}, false, nil
		},
		sendMessageFn: func(ctx context.Context, streamID, userID, userName, userAvatarUrl, message string) (*domain.ChatMessage, error) {
			msg := &domain.ChatMessage{
				ID:            "msg-integration-1",
				StreamID:      streamID,
				UserID:        userID,
				UserName:      userName,
				UserAvatarUrl: userAvatarUrl,
				Message:       message,
				SentAt:        time.Now(),
			}
			// Broadcast so the client receives its own message back via writePump.
			hub.Broadcast(streamID, msg)

			select {
			case msgCh <- capturedMsg{streamID, userID, userName, userAvatarUrl, message}:
			default:
			}
			return msg, nil
		},
	}

	handler := NewChatHandler(svc, newTestAuth(), testLogger())

	r := chi.NewRouter()
	r.Get("/ws/chat/{streamID}", handler.ChatWebSocket)
	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/chat/test-stream"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Cookie": {"token=test-token"}},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// --- Send a chat message (spec envelope) ---
	sendEnvelope := map[string]interface{}{
		"type":    "message",
		"payload": map[string]string{"message": "hello world"},
	}
	if err := wsjson.Write(ctx, conn, sendEnvelope); err != nil {
		t.Fatalf("write message: %v", err)
	}

	// --- Assert the mock service received the correct call ---
	select {
	case cap := <-msgCh:
		if cap.streamID != "test-stream" {
			t.Errorf("streamID = %q, want %q", cap.streamID, "test-stream")
		}
		if cap.message != "hello world" {
			t.Errorf("message = %q, want %q", cap.message, "hello world")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SendMessage call")
	}

	// --- Assert the broadcast arrived back via writePump ---
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read broadcast: %v", err)
	}
	received := decodeBroadcast(t, data)
	if received.Message != "hello world" {
		t.Errorf("broadcast message = %q, want %q", received.Message, "hello world")
	}

	// --- Close and let goroutines tear down ---
	conn.Close(websocket.StatusNormalClosure, "")
	time.Sleep(100 * time.Millisecond) // give readPump / writePump time to exit
}

// ---------------------------------------------------------------------------
// TestChatWebSocket_BroadcastReceived — exercises writePump indirectly.
// ---------------------------------------------------------------------------

func TestChatWebSocket_BroadcastReceived(t *testing.T) {
	hub := newTestHub()

	svc := &mockChatService{
		getHubFn:       func() *application.ChatHub { return hub },
		isStreamLiveFn: func(ctx context.Context, streamID string) (bool, error) { return true, nil },
		getMessagesFn: func(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error) {
			return []domain.ChatMessage{}, false, nil
		},
	}

	handler := NewChatHandler(svc, newTestAuth(), testLogger())

	r := chi.NewRouter()
	r.Get("/ws/chat/{streamID}", handler.ChatWebSocket)
	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/chat/test-stream"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Give the client time to join the hub room.
	time.Sleep(50 * time.Millisecond)

	// Broadcast a message from outside (simulates another client).
	hub.Broadcast("test-stream", &domain.ChatMessage{
		ID:            "broadcast-ext-1",
		StreamID:      "test-stream",
		UserID:        "other-user",
		UserName:      "Bob",
		UserAvatarUrl: "https://example.com/bob.jpg",
		Message:       "external broadcast",
		SentAt:        time.Now(),
	})

	// Read the broadcast from the WebSocket.
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read broadcast: %v", err)
	}
	received := decodeBroadcast(t, data)
	if received.Message != "external broadcast" {
		t.Errorf("message = %q, want %q", received.Message, "external broadcast")
	}
	if received.UserName != "Bob" {
		t.Errorf("userName = %q, want %q", received.UserName, "Bob")
	}
}

// ---------------------------------------------------------------------------
// TestRegisterRoutes
// ---------------------------------------------------------------------------

func TestRegisterRoutes(t *testing.T) {
	svc := &mockChatService{
		getHubFn: newTestHub,
	}
	h := NewChatHandler(svc, newTestAuth(), testLogger())

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	srv := httptest.NewServer(r)
	defer srv.Close()

	t.Run("Messages route is registered", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/chat/test-stream/messages")
		if err != nil {
			t.Fatalf("GET messages route: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			t.Error("Messages route not registered (got 404)")
		}
	})
}
