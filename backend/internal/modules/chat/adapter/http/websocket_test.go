package http

import (
	"context"
	"encoding/base64"
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
// back into a domain.ChatMessage. It handles both:
//   - direct JSON (if the handler sends raw JSON bytes)
//   - base64-encoded JSON (current writePump behaviour with wsjson.Write + []byte)
func decodeBroadcast(t *testing.T, data []byte) domain.ChatMessage {
	t.Helper()

	var msg domain.ChatMessage
	if err := json.Unmarshal(data, &msg); err == nil {
		return msg
	}

	// Fallback: wsjson.Write double-encodes []byte as a base64 JSON string.
	var b64 string
	if err := json.Unmarshal(data, &b64); err != nil {
		t.Fatalf("unmarshal received data: %v (raw=%s)", err, string(data))
	}
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	if err := json.Unmarshal(decoded, &msg); err != nil {
		t.Fatalf("unmarshal decoded message: %v", err)
	}
	return msg
}

// ---------------------------------------------------------------------------
// TestChatWebSocket_MissingParams
// ---------------------------------------------------------------------------

func TestChatWebSocket_MissingParams(t *testing.T) {
	tests := []struct {
		name     string
		streamID string
		userID   string
		userName string
		wantCode int
	}{
		{
			name:     "missing streamID",
			streamID: "",
			userID:   "user-1",
			userName: "Alice",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing userId",
			streamID: "stream-1",
			userID:   "",
			userName: "Alice",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing userName",
			streamID: "stream-1",
			userID:   "user-1",
			userName: "",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing both userId and userName",
			streamID: "stream-1",
			userID:   "",
			userName: "",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockChatService{
				getHubFn: newTestHub,
			}
			h := NewChatHandler(svc, testLogger())

			u := "/api/chat/ws/" + tt.streamID
			if tt.userID != "" || tt.userName != "" {
				u += "?userId=" + tt.userID + "&userName=" + tt.userName
			}
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
		streamID string
		userID   string
		userName string
		message  string
	}
	msgCh := make(chan capturedMsg, 1)

	svc := &mockChatService{
		getHubFn: func() *application.ChatHub { return hub },
		sendMessageFn: func(ctx context.Context, streamID, userID, userName, message string) (*domain.ChatMessage, error) {
			msg := &domain.ChatMessage{
				ID:       "msg-integration-1",
				StreamID: streamID,
				UserID:   userID,
				UserName: userName,
				Message:  message,
				SentAt:   time.Now(),
			}
			// Broadcast so the client receives its own message back via writePump.
			hub.Broadcast(streamID, msg)

			select {
			case msgCh <- capturedMsg{streamID, userID, userName, message}:
			default:
			}
			return msg, nil
		},
	}

	handler := NewChatHandler(svc, testLogger())

	r := chi.NewRouter()
	handler.RegisterRoutes(r)
	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/chat/ws/test-stream?userId=user-1&userName=Alice"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// --- Send a chat message ---
	if err := wsjson.Write(ctx, conn, map[string]string{"message": "hello world"}); err != nil {
		t.Fatalf("write message: %v", err)
	}

	// --- Assert the mock service received the correct call ---
	select {
	case cap := <-msgCh:
		if cap.streamID != "test-stream" {
			t.Errorf("streamID = %q, want %q", cap.streamID, "test-stream")
		}
		if cap.userID != "user-1" {
			t.Errorf("userID = %q, want %q", cap.userID, "user-1")
		}
		if cap.userName != "Alice" {
			t.Errorf("userName = %q, want %q", cap.userName, "Alice")
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
	if received.UserName != "Alice" {
		t.Errorf("broadcast userName = %q, want %q", received.UserName, "Alice")
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
		getHubFn: func() *application.ChatHub { return hub },
	}

	handler := NewChatHandler(svc, testLogger())

	r := chi.NewRouter()
	handler.RegisterRoutes(r)
	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/chat/ws/test-stream?userId=user-1&userName=Alice"

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
		ID:       "broadcast-ext-1",
		StreamID: "test-stream",
		UserID:   "other-user",
		UserName: "Bob",
		Message:  "external broadcast",
		SentAt:   time.Now(),
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
	h := NewChatHandler(svc, testLogger())

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	// RegisterRoutes must not panic and must make the router non-empty.
	// We verify by starting a server and hitting each route — none should 404.

	srv := httptest.NewServer(r)
	defer srv.Close()

	t.Run("WebSocket route is registered", func(t *testing.T) {
		// A plain HTTP GET won't upgrade, but it must hit the handler (200 or 400),
		// not return 404.
		resp, err := http.Get(srv.URL + "/api/chat/ws/test-stream?userId=u&userName=n")
		if err != nil {
			t.Fatalf("GET ws route: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			t.Error("WebSocket route not registered (got 404)")
		}
	})

	t.Run("Messages route is registered", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/chat/messages/test-stream")
		if err != nil {
			t.Fatalf("GET messages route: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			t.Error("Messages route not registered (got 404)")
		}
	})
}
