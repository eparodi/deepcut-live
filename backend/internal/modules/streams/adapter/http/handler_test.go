package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/deepcut/live/internal/modules/streams/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

// ---------------------------------------------------------------------------
// mockStreamService implements streamService for handler tests
// ---------------------------------------------------------------------------

type mockStreamService struct {
	verifySRSSecretFn func(secret string) error
	onStreamStartFn   func(ctx context.Context, rawKey string, srsClientID int, title string) (*domain.Stream, error)
	onStreamEndFn     func(ctx context.Context, srsClientID int, hlsPath, recordingPath string, durationSeconds int) error
	listLiveFn        func(ctx context.Context) ([]domain.LiveStream, error)
	getChannelInfoFn  func(ctx context.Context, userID string) (*domain.ChannelInfo, error)
	heartbeatViewerFn func(ctx context.Context, streamID, userID, clientID string) error
}

func (m *mockStreamService) VerifySRSSecret(secret string) error {
	if m.verifySRSSecretFn != nil {
		return m.verifySRSSecretFn(secret)
	}
	return nil
}

func (m *mockStreamService) OnStreamStart(ctx context.Context, rawKey string, srsClientID int, title string) (*domain.Stream, error) {
	if m.onStreamStartFn != nil {
		return m.onStreamStartFn(ctx, rawKey, srsClientID, title)
	}
	return &domain.Stream{ID: "stream-1", UserID: "user-1"}, nil
}

func (m *mockStreamService) OnStreamEnd(ctx context.Context, srsClientID int, hlsPath, recordingPath string, durationSeconds int) error {
	if m.onStreamEndFn != nil {
		return m.onStreamEndFn(ctx, srsClientID, hlsPath, recordingPath, durationSeconds)
	}
	return nil
}

func (m *mockStreamService) ListLive(ctx context.Context) ([]domain.LiveStream, error) {
	if m.listLiveFn != nil {
		return m.listLiveFn(ctx)
	}
	return []domain.LiveStream{}, nil
}

func (m *mockStreamService) GetChannelInfo(ctx context.Context, userID string) (*domain.ChannelInfo, error) {
	if m.getChannelInfoFn != nil {
		return m.getChannelInfoFn(ctx, userID)
	}
	return &domain.ChannelInfo{
		UserID:   userID,
		UserName: "TestStreamer",
		IsLive:   false,
	}, nil
}

func (m *mockStreamService) HeartbeatViewer(ctx context.Context, streamID, userID, clientID string) error {
	if m.heartbeatViewerFn != nil {
		return m.heartbeatViewerFn(ctx, streamID, userID, clientID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func chiURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// ---------------------------------------------------------------------------
// TestSRSCallback
// ---------------------------------------------------------------------------

func TestSRSCallback(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
		wantBody string // raw body check for SRS protocol responses
	}{
		{
			name:     "on_publish action",
			body:     `{"action":"on_publish","client_id":1,"param":"?secret=valid&key=sk-abc"}`,
			wantCode: http.StatusOK,
		},
		{
			name:     "on_unpublish action",
			body:     `{"action":"on_unpublish","client_id":1}`,
			wantCode: http.StatusOK,
		},
		{
			name:     "unknown action",
			body:     `{"action":"unknown","client_id":1}`,
			wantCode: http.StatusOK,
			wantBody: "0",
		},
		{
			name:     "empty body",
			body:     ``,
			wantCode: http.StatusOK,
			wantBody: "1",
		},
		{
			name:     "invalid JSON",
			body:     `{bad`,
			wantCode: http.StatusOK,
			wantBody: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockStreamService{}
			h := NewStreamHandler(svc, testLogger())

			req := httptest.NewRequest(http.MethodPost, "/api/srs/callback",
				strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.SRSCallback(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
			if tt.wantBody != "" {
				got := rec.Body.String()
				if got != tt.wantBody {
					t.Errorf("body: got %q, want %q", got, tt.wantBody)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSRSOnPublish
// ---------------------------------------------------------------------------

func TestSRSOnPublish(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		secret    string
		setupMock func(*mockStreamService)
		wantCode  int
	}{
		{
			name:     "happy path",
			body:     `{"action":"on_publish","client_id":1,"param":"?key=sk-abc"}`,
			secret:   "valid-secret",
			wantCode: http.StatusOK,
		},
		{
			name:   "invalid srs secret",
			body:   `{"action":"on_publish","client_id":1,"param":"?key=sk-abc"}`,
			secret: "",
			setupMock: func(m *mockStreamService) {
				m.verifySRSSecretFn = func(secret string) error {
					return errs.Forbidden("invalid srs secret")
				}
			},
			wantCode: http.StatusForbidden,
		},
		{
			name:     "invalid JSON body",
			body:     `{bad`,
			secret:   "valid-secret",
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "on stream start error",
			body:   `{"action":"on_publish","client_id":1,"param":"?key=bad-key"}`,
			secret: "valid-secret",
			setupMock: func(m *mockStreamService) {
				m.onStreamStartFn = func(ctx context.Context, rawKey string, srsClientID int, title string) (*domain.Stream, error) {
					return nil, errs.NotFound("user not found")
				}
			},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockStreamService{}
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}
			h := NewStreamHandler(svc, testLogger())

			u := "/api/srs/callback?secret=" + tt.secret
			req := httptest.NewRequest(http.MethodPost, u,
				strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.SRSOnPublish(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSRSOnUnpublish
// ---------------------------------------------------------------------------

func TestSRSOnUnpublish(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		secret    string
		setupMock func(*mockStreamService)
		wantCode  int
	}{
		{
			name:     "happy path",
			body:     `{"action":"on_unpublish","client_id":1}`,
			secret:   "valid-secret",
			wantCode: http.StatusOK,
		},
		{
			name:   "invalid srs secret",
			body:   `{"action":"on_unpublish","client_id":1}`,
			secret: "",
			setupMock: func(m *mockStreamService) {
				m.verifySRSSecretFn = func(secret string) error {
					return errs.Forbidden("invalid srs secret")
				}
			},
			wantCode: http.StatusForbidden,
		},
		{
			name:     "invalid JSON body",
			body:     `{bad`,
			secret:   "valid-secret",
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "on stream end error",
			body:   `{"action":"on_unpublish","client_id":1}`,
			secret: "valid-secret",
			setupMock: func(m *mockStreamService) {
				m.onStreamEndFn = func(ctx context.Context, srsClientID int, hlsPath, recordingPath string, durationSeconds int) error {
					return errs.Internal("end failed")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockStreamService{}
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}
			h := NewStreamHandler(svc, testLogger())

			u := "/api/srs/callback?secret=" + tt.secret
			req := httptest.NewRequest(http.MethodPost, u,
				strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.SRSOnUnpublish(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestListLiveStreams
// ---------------------------------------------------------------------------

func TestListLiveStreams(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mockStreamService)
		wantCode  int
	}{
		{
			name:     "happy path — empty list",
			wantCode: http.StatusOK,
		},
		{
			name:     "happy path — with streams",
			wantCode: http.StatusOK,
		},
		{
			name: "service error",
			setupMock: func(m *mockStreamService) {
				m.listLiveFn = func(ctx context.Context) ([]domain.LiveStream, error) {
					return nil, errs.Internal("db error")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockStreamService{}
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}
			h := NewStreamHandler(svc, testLogger())

			req := httptest.NewRequest(http.MethodGet, "/api/streams/live", nil)
			rec := httptest.NewRecorder()
			h.ListLiveStreams(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetChannelInfo
// ---------------------------------------------------------------------------

func TestGetChannelInfo(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupMock func(*mockStreamService)
		wantCode  int
	}{
		{
			name:     "happy path",
			userID:   "550e8400-e29b-41d4-a716-446655440000",
			wantCode: http.StatusOK,
		},
		{
			name:     "missing user ID",
			userID:   "",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid UUID",
			userID:   "not-a-uuid",
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: "550e8400-e29b-41d4-a716-446655440000",
			setupMock: func(m *mockStreamService) {
				m.getChannelInfoFn = func(ctx context.Context, userID string) (*domain.ChannelInfo, error) {
					return nil, errs.NotFound("channel not found")
				}
			},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockStreamService{}
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}
			h := NewStreamHandler(svc, testLogger())

			req := httptest.NewRequest(http.MethodGet, "/api/channel/"+tt.userID, nil)
			if tt.userID != "" {
				req = chiURLParam(req, "userID", tt.userID)
			}
			rec := httptest.NewRecorder()
			h.GetChannelInfo(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestViewerHeartbeat
// ---------------------------------------------------------------------------

func TestViewerHeartbeat(t *testing.T) {
	tests := []struct {
		name      string
		streamID  string
		body      string
		setupMock func(*mockStreamService)
		wantCode  int
	}{
		{
			name:     "happy path",
			streamID: "stream-1",
			body:     `{"clientId":"client-abc"}`,
			wantCode: http.StatusOK,
		},
		{
			name:     "missing stream ID",
			streamID: "",
			body:     `{"clientId":"client-abc"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid JSON",
			streamID: "stream-1",
			body:     `{bad`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing client ID",
			streamID: "stream-1",
			body:     `{}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "service error",
			streamID: "stream-1",
			body:     `{"clientId":"client-abc"}`,
			setupMock: func(m *mockStreamService) {
				m.heartbeatViewerFn = func(ctx context.Context, streamID, userID, clientID string) error {
					return errs.Internal("heartbeat failed")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockStreamService{}
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}
			h := NewStreamHandler(svc, testLogger())

			u := "/api/streams/" + tt.streamID + "/viewer-heartbeat"
			req := httptest.NewRequest(http.MethodPost, u,
				strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.streamID != "" {
				req = chiURLParam(req, "streamID", tt.streamID)
			}
			rec := httptest.NewRecorder()
			h.ViewerHeartbeat(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
