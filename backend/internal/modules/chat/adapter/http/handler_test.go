package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/deepcut/live/internal/modules/chat/application"
	"github.com/deepcut/live/internal/modules/chat/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

// ---------------------------------------------------------------------------
// mockChatService implements chatService for handler tests
// ---------------------------------------------------------------------------

type mockChatService struct {
	getHubFn       func() *application.ChatHub
	sendMessageFn  func(ctx context.Context, streamID, userID, userName, userAvatarUrl, message string) (*domain.ChatMessage, error)
	getMessagesFn  func(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error)
	isStreamLiveFn func(ctx context.Context, streamID string) (bool, error)
}

func (m *mockChatService) GetHub() *application.ChatHub {
	if m.getHubFn != nil {
		return m.getHubFn()
	}
	return &application.ChatHub{}
}

func (m *mockChatService) SendMessage(ctx context.Context, streamID, userID, userName, userAvatarUrl, message string) (*domain.ChatMessage, error) {
	if m.sendMessageFn != nil {
		return m.sendMessageFn(ctx, streamID, userID, userName, userAvatarUrl, message)
	}
	return &domain.ChatMessage{
		ID:       "msg-1",
		StreamID: streamID,
		UserID:   userID,
		UserName: userName,
		Message:  message,
		SentAt:   time.Now(),
	}, nil
}

func (m *mockChatService) GetMessages(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error) {
	if m.getMessagesFn != nil {
		return m.getMessagesFn(ctx, streamID, before, limit)
	}
	return []domain.ChatMessage{
		{ID: "msg-1", StreamID: streamID, UserID: "user-1", UserName: "Alice", UserAvatarUrl: "https://example.com/avatar.jpg", Message: "hello", SentAt: time.Now()},
	}, false, nil
}

func (m *mockChatService) IsStreamLive(ctx context.Context, streamID string) (bool, error) {
	if m.isStreamLiveFn != nil {
		return m.isStreamLiveFn(ctx, streamID)
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// mockChatAuth implements chatAuth for handler tests
// ---------------------------------------------------------------------------

type mockChatAuth struct {
	validateTokenFn func(ctx context.Context, tokenStr string) (userID, userName, userAvatarUrl string, err error)
}

func (m *mockChatAuth) ValidateToken(ctx context.Context, tokenStr string) (string, string, string, error) {
	if m.validateTokenFn != nil {
		return m.validateTokenFn(ctx, tokenStr)
	}
	return "user-1", "Alice", "https://example.com/avatar.jpg", nil
}

// ---------------------------------------------------------------------------
// TestGetMessages
// ---------------------------------------------------------------------------

func TestGetMessages(t *testing.T) {
	tests := []struct {
		name      string
		streamID  string
		query     string
		setupMock func(*mockChatService)
		wantCode  int
	}{
		{
			name:     "happy path — returns messages",
			streamID: "stream-1",
			wantCode: http.StatusOK,
		},
		{
			name:     "happy path — with before cursor",
			streamID: "stream-1",
			query:    "?before=2026-01-01T00:00:00Z&limit=10",
			wantCode: http.StatusOK,
		},
		{
			name:     "missing stream ID",
			streamID: "",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "service error",
			streamID: "stream-1",
			setupMock: func(m *mockChatService) {
				m.getMessagesFn = func(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error) {
					return nil, false, errs.Internal("db error")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "invalid limit — falls back to default",
			streamID: "stream-1",
			query:    "?limit=invalid",
			wantCode: http.StatusOK,
		},
		{
			name:     "limit too high — clamped to max",
			streamID: "stream-1",
			query:    "?limit=999",
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockChatService{}
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}
			h := NewChatHandler(svc, newTestAuth(), testLogger())

			u := "/api/chat/" + tt.streamID + "/messages" + tt.query
			req := httptest.NewRequest(http.MethodGet, u, nil)
			if tt.streamID != "" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("streamID", tt.streamID)
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}
			rec := httptest.NewRecorder()
			h.GetMessages(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d (body=%s)", rec.Code, tt.wantCode, rec.Body.String())
			}
		})
	}
}
