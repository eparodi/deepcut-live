package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/deepcut/live/internal/modules/vods/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

// ---------------------------------------------------------------------------
// mockVODService implements vodService for handler tests
// ---------------------------------------------------------------------------

type mockVODService struct {
	getVODFn                func(ctx context.Context, vodID string) (*domain.VOD, error)
	listVODsFn              func(ctx context.Context, userID string, limit, offset int) ([]domain.VOD, error)
	searchVODsFn            func(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error)
	recordViewerHeartbeatFn func(ctx context.Context, vodID string) error
}

func (m *mockVODService) GetVOD(ctx context.Context, vodID string) (*domain.VOD, error) {
	if m.getVODFn != nil {
		return m.getVODFn(ctx, vodID)
	}
	return &domain.VOD{
		ID:        vodID,
		UserID:    "user-1",
		UserName:  "TestUser",
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockVODService) ListVODs(ctx context.Context, userID string, limit, offset int) ([]domain.VOD, error) {
	if m.listVODsFn != nil {
		return m.listVODsFn(ctx, userID, limit, offset)
	}
	return []domain.VOD{}, nil
}

func (m *mockVODService) SearchVODs(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
	if m.searchVODsFn != nil {
		return m.searchVODsFn(ctx, params)
	}
	return &domain.SearchResult{
		VODs:       []domain.VOD{},
		TotalCount: 0,
		Limit:      params.Limit,
		Offset:     params.Offset,
	}, nil
}

func (m *mockVODService) RecordViewerHeartbeat(ctx context.Context, vodID string) error {
	if m.recordViewerHeartbeatFn != nil {
		return m.recordViewerHeartbeatFn(ctx, vodID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// TestGetVOD
// ---------------------------------------------------------------------------

func TestGetVOD(t *testing.T) {
	tests := []struct {
		name      string
		vodID     string
		setupMock func(*mockVODService)
		wantCode  int
	}{
		{
			name:     "happy path",
			vodID:    "vod-1",
			wantCode: http.StatusOK,
		},
		{
			name:     "missing vod ID",
			vodID:    "",
			wantCode: http.StatusBadRequest,
		},
		{
			name:  "service error — not found",
			vodID: "vod-999",
			setupMock: func(m *mockVODService) {
				m.getVODFn = func(ctx context.Context, vodID string) (*domain.VOD, error) {
					return nil, errs.NotFound("vod not found")
				}
			},
			wantCode: http.StatusNotFound,
		},
		{
			name:  "service error — internal",
			vodID: "vod-1",
			setupMock: func(m *mockVODService) {
				m.getVODFn = func(ctx context.Context, vodID string) (*domain.VOD, error) {
					return nil, errs.Internal("db down")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockVODService{}
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}
			h := NewVODHandler(svc, testLogger())

			req := httptest.NewRequest(http.MethodGet, "/api/vods/"+tt.vodID, nil)
			if tt.vodID != "" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("vodID", tt.vodID)
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}
			rec := httptest.NewRecorder()
			h.GetVOD(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSearchVODs
// ---------------------------------------------------------------------------

func TestSearchVODs(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		setupMock func(*mockVODService)
		wantCode  int
	}{
		{
			name:     "happy path — empty filters",
			query:    "",
			wantCode: http.StatusOK,
		},
		{
			name:     "happy path — with search query",
			query:    "?q=setup&category=Tech&sort=recent&limit=10&offset=0",
			wantCode: http.StatusOK,
		},
		{
			name:  "service error",
			query: "?q=test",
			setupMock: func(m *mockVODService) {
				m.searchVODsFn = func(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
					return nil, errs.Internal("search failed")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockVODService{}
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}
			h := NewVODHandler(svc, testLogger())

			req := httptest.NewRequest(http.MethodGet, "/api/vods"+tt.query, nil)
			rec := httptest.NewRecorder()
			h.SearchVODs(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestListUserVODs
// ---------------------------------------------------------------------------

func TestListUserVODs(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		query     string
		setupMock func(*mockVODService)
		wantCode  int
	}{
		{
			name:     "happy path",
			userID:   "user-1",
			query:    "",
			wantCode: http.StatusOK,
		},
		{
			name:     "happy path — with pagination",
			userID:   "user-1",
			query:    "?limit=5&offset=10",
			wantCode: http.StatusOK,
		},
		{
			name:     "missing user ID",
			userID:   "",
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: "user-1",
			setupMock: func(m *mockVODService) {
				m.listVODsFn = func(ctx context.Context, userID string, limit, offset int) ([]domain.VOD, error) {
					return nil, errs.Internal("list failed")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockVODService{}
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}
			h := NewVODHandler(svc, testLogger())

			req := httptest.NewRequest(http.MethodGet, "/api/channels/"+tt.userID+"/vods"+tt.query, nil)
			if tt.userID != "" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("userID", tt.userID)
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}
			rec := httptest.NewRecorder()
			h.ListUserVODs(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestViewerHeartbeat (VODs)
// ---------------------------------------------------------------------------

func TestVODViewerHeartbeat(t *testing.T) {
	tests := []struct {
		name      string
		vodID     string
		body      string
		setupMock func(*mockVODService)
		wantCode  int
	}{
		{
			name:     "happy path",
			vodID:    "vod-1",
			body:     `{"clientId":"client-abc"}`,
			wantCode: http.StatusOK,
		},
		{
			name:     "missing vod ID",
			vodID:    "",
			body:     `{"clientId":"client-abc"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid JSON",
			vodID:    "vod-1",
			body:     `{bad`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing client ID",
			vodID:    "vod-1",
			body:     `{}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:  "service error",
			vodID: "vod-1",
			body:  `{"clientId":"client-abc"}`,
			setupMock: func(m *mockVODService) {
				m.recordViewerHeartbeatFn = func(ctx context.Context, vodID string) error {
					return errs.Internal("heartbeat failed")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockVODService{}
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}
			h := NewVODHandler(svc, testLogger())

			u := "/api/vods/" + tt.vodID + "/heartbeat"
			req := httptest.NewRequest(http.MethodPost, u,
				strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.vodID != "" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("vodID", tt.vodID)
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
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
