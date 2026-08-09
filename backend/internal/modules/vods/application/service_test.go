package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/deepcut/live/internal/modules/vods/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

// ---------------------------------------------------------------------------
// mockVODRepo implements domain.VODRepository for service tests
// ---------------------------------------------------------------------------

type mockVODRepo struct {
	getVODFn             func(ctx context.Context, vodID string) (*domain.VOD, error)
	listVODsFn           func(ctx context.Context, userID string, limit, offset int) ([]domain.VOD, error)
	searchVODsFn         func(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error)
	incrementViewCountFn func(ctx context.Context, vodID string) error
}

func (m *mockVODRepo) GetVOD(ctx context.Context, vodID string) (*domain.VOD, error) {
	if m.getVODFn != nil {
		return m.getVODFn(ctx, vodID)
	}
	return &domain.VOD{
		ID:           vodID,
		UserID:       "user-1",
		UserName:     "TestStreamer",
		Title:        strPtr("My Awesome VOD"),
		StartedAt:    time.Now().Add(-1 * time.Hour),
		CreatedAt:    time.Now(),
		PeakViewers:  42,
		TotalViewers: 100,
	}, nil
}

func (m *mockVODRepo) ListVODs(ctx context.Context, userID string, limit, offset int) ([]domain.VOD, error) {
	if m.listVODsFn != nil {
		return m.listVODsFn(ctx, userID, limit, offset)
	}
	return []domain.VOD{
		{ID: "vod-1", UserID: userID, UserName: "TestStreamer", Title: strPtr("Stream 1"), StartedAt: time.Now().Add(-2 * time.Hour), CreatedAt: time.Now()},
	}, nil
}

func (m *mockVODRepo) SearchVODs(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
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

func (m *mockVODRepo) IncrementViewCount(ctx context.Context, vodID string) error {
	if m.incrementViewCountFn != nil {
		return m.incrementViewCountFn(ctx, vodID)
	}
	return nil
}

func strPtr(s string) *string {
	return &s
}

// ---------------------------------------------------------------------------
// TestGetVOD
// ---------------------------------------------------------------------------

func TestGetVOD(t *testing.T) {
	tests := []struct {
		name      string
		vodID     string
		setupMock func(*mockVODRepo)
		wantErr   bool
	}{
		{
			name:  "happy path — VOD found",
			vodID: "vod-1",
		},
		{
			name:  "error — not found",
			vodID: "vod-999",
			setupMock: func(m *mockVODRepo) {
				m.getVODFn = func(ctx context.Context, vodID string) (*domain.VOD, error) {
					return nil, errs.NotFound("vod %s not found", vodID)
				}
			},
			wantErr: true,
		},
		{
			name:  "error — repo error",
			vodID: "vod-1",
			setupMock: func(m *mockVODRepo) {
				m.getVODFn = func(ctx context.Context, vodID string) (*domain.VOD, error) {
					return nil, errors.New("db down")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockVODRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			svc := NewVODService(repo)

			vod, err := svc.GetVOD(context.Background(), tt.vodID)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && vod == nil {
				t.Fatal("expected non-nil VOD")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestListVODs
// ---------------------------------------------------------------------------

func TestListVODs(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		limit     int
		offset    int
		setupMock func(*mockVODRepo)
		wantCount int
		wantErr   bool
	}{
		{
			name:      "happy path — returns VODs",
			userID:    "user-1",
			limit:     10,
			wantCount: 1,
		},
		{
			name:      "happy path — empty list",
			userID:    "user-2",
			limit:     10,
			wantCount: 0,
			setupMock: func(m *mockVODRepo) {
				m.listVODsFn = func(ctx context.Context, userID string, limit, offset int) ([]domain.VOD, error) {
					return []domain.VOD{}, nil
				}
			},
		},
		{
			name:      "happy path — pagination with offset",
			userID:    "user-1",
			limit:     5,
			offset:    10,
			wantCount: 1,
		},
		{
			name:   "error — repo error",
			userID: "user-1",
			limit:  10,
			setupMock: func(m *mockVODRepo) {
				m.listVODsFn = func(ctx context.Context, userID string, limit, offset int) ([]domain.VOD, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockVODRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			svc := NewVODService(repo)

			vods, err := svc.ListVODs(context.Background(), tt.userID, tt.limit, tt.offset)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(vods) != tt.wantCount {
				t.Fatalf("got %d vods, want %d", len(vods), tt.wantCount)
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
		params    domain.SearchParams
		setupMock func(*mockVODRepo)
		wantCount int
		wantErr   bool
	}{
		{
			name: "happy path — search with query",
			params: domain.SearchParams{
				Query: "awesome",
				Sort:  "recent",
				Limit: 10,
			},
			wantCount: 2,
			setupMock: func(m *mockVODRepo) {
				m.searchVODsFn = func(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
					return &domain.SearchResult{
						VODs: []domain.VOD{
							{ID: "vod-1", UserName: "Alice", Title: strPtr("Awesome Stream 1")},
							{ID: "vod-2", UserName: "Bob", Title: strPtr("Awesome Stream 2")},
						},
						TotalCount: 2,
						Limit:      params.Limit,
						Offset:     params.Offset,
					}, nil
				}
			},
		},
		{
			name: "happy path — empty results",
			params: domain.SearchParams{
				Query: "nonexistent",
				Limit: 10,
			},
			wantCount: 0,
		},
		{
			name: "happy path — search with all filters",
			params: domain.SearchParams{
				Query:    "stream",
				Category: "Gaming",
				Status:   "ready",
				Sort:     "popular",
				Limit:    20,
				Offset:   0,
			},
			wantCount: 1,
			setupMock: func(m *mockVODRepo) {
				m.searchVODsFn = func(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
					return &domain.SearchResult{
						VODs:       []domain.VOD{{ID: "vod-1", UserName: "Alice"}},
						TotalCount: 1,
						Limit:      params.Limit,
						Offset:     params.Offset,
					}, nil
				}
			},
		},
		{
			name: "error — repo error",
			params: domain.SearchParams{
				Limit: 10,
			},
			setupMock: func(m *mockVODRepo) {
				m.searchVODsFn = func(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockVODRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			svc := NewVODService(repo)

			result, err := svc.SearchVODs(context.Background(), tt.params)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr {
				if result == nil {
					t.Fatal("expected non-nil result")
				}
				if len(result.VODs) != tt.wantCount {
					t.Fatalf("got %d VODs, want %d", len(result.VODs), tt.wantCount)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRecordViewerHeartbeat
// ---------------------------------------------------------------------------

func TestRecordViewerHeartbeat(t *testing.T) {
	tests := []struct {
		name      string
		vodID     string
		setupMock func(*mockVODRepo)
		wantErr   bool
	}{
		{
			name:  "happy path — increments view count",
			vodID: "vod-1",
		},
		{
			name:  "error — repo error",
			vodID: "vod-1",
			setupMock: func(m *mockVODRepo) {
				m.incrementViewCountFn = func(ctx context.Context, vodID string) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockVODRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			svc := NewVODService(repo)

			err := svc.RecordViewerHeartbeat(context.Background(), tt.vodID)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
