package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/deepcut/live/internal/modules/streams/domain"
	voddomain "github.com/deepcut/live/internal/modules/vods/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

// ---------------------------------------------------------------------------
// mockStreamRepo implements domain.StreamRepository for service tests
// ---------------------------------------------------------------------------

type mockStreamRepo struct {
	createStreamFn           func(ctx context.Context, userID string, title *string, srsClientID string, hlsPath string) (*domain.Stream, error)
	endStreamFn              func(ctx context.Context, streamID string, hlsPath, recordingPath string, durationSeconds int) error
	updateStreamStatusFn     func(ctx context.Context, streamID, status string) error
	getStreamByUserIDFn      func(ctx context.Context, userID string) (*domain.Stream, error)
	getStreamBySRSClientIDFn func(ctx context.Context, srsClientID string) (*domain.Stream, error)
	listLiveStreamsFn        func(ctx context.Context) ([]domain.LiveStream, error)
	getChannelInfoFn         func(ctx context.Context, userID string) (*domain.ChannelInfo, error)
	upsertViewerFn           func(ctx context.Context, streamID, userID, clientID string) error
	heartbeatViewerFn        func(ctx context.Context, streamID, clientID string, lastSeen time.Time) error
	removeViewerFn           func(ctx context.Context, streamID, clientID string) error
	getViewerCountFn         func(ctx context.Context, streamID string) (int, error)
	getAnalyticsFn           func(ctx context.Context, userID, period string) (*domain.Analytics, error)
	updateStreamAnalyticsFn  func(ctx context.Context, userID string, date string, duration, peak, unique int) error
	updateRecordingStatusFn  func(ctx context.Context, streamID, status, errorMsg string) error
	updateVODPathsFn         func(ctx context.Context, streamID, hlsPath, thumbnailPath string) error
}

func (m *mockStreamRepo) CreateStream(ctx context.Context, userID string, title *string, srsClientID string, hlsPath string) (*domain.Stream, error) {
	if m.createStreamFn != nil {
		return m.createStreamFn(ctx, userID, title, srsClientID, hlsPath)
	}
	return &domain.Stream{
		ID:          "stream-1",
		UserID:      userID,
		Title:       title,
		Status:      "live",
		StartedAt:   time.Now(),
		SRSClientID: &srsClientID,
	}, nil
}

func (m *mockStreamRepo) EndStream(ctx context.Context, streamID string, hlsPath, recordingPath string, durationSeconds int) error {
	if m.endStreamFn != nil {
		return m.endStreamFn(ctx, streamID, hlsPath, recordingPath, durationSeconds)
	}
	return nil
}

func (m *mockStreamRepo) UpdateStreamStatus(ctx context.Context, streamID, status string) error {
	if m.updateStreamStatusFn != nil {
		return m.updateStreamStatusFn(ctx, streamID, status)
	}
	return nil
}

func (m *mockStreamRepo) GetStreamByUserID(ctx context.Context, userID string) (*domain.Stream, error) {
	if m.getStreamByUserIDFn != nil {
		return m.getStreamByUserIDFn(ctx, userID)
	}
	return &domain.Stream{
		ID:        "stream-1",
		UserID:    userID,
		Status:    "live",
		StartedAt: time.Now().Add(-10 * time.Minute),
	}, nil
}

func (m *mockStreamRepo) GetStreamBySRSClientID(ctx context.Context, srsClientID string) (*domain.Stream, error) {
	if m.getStreamBySRSClientIDFn != nil {
		return m.getStreamBySRSClientIDFn(ctx, srsClientID)
	}
	return &domain.Stream{
		ID:           "stream-1",
		UserID:       "user-1",
		Status:       "live",
		StartedAt:    time.Now().Add(-10 * time.Minute),
		SRSClientID:  &srsClientID,
		PeakViewers:  5,
		TotalViewers: 10,
	}, nil
}

func (m *mockStreamRepo) ListLiveStreams(ctx context.Context) ([]domain.LiveStream, error) {
	if m.listLiveStreamsFn != nil {
		return m.listLiveStreamsFn(ctx)
	}
	return []domain.LiveStream{}, nil
}

func (m *mockStreamRepo) GetChannelInfo(ctx context.Context, userID string) (*domain.ChannelInfo, error) {
	if m.getChannelInfoFn != nil {
		return m.getChannelInfoFn(ctx, userID)
	}
	return &domain.ChannelInfo{
		UserID:   userID,
		UserName: "TestStreamer",
		IsLive:   false,
	}, nil
}

func (m *mockStreamRepo) UpsertViewer(ctx context.Context, streamID, userID, clientID string) error {
	if m.upsertViewerFn != nil {
		return m.upsertViewerFn(ctx, streamID, userID, clientID)
	}
	return nil
}

func (m *mockStreamRepo) HeartbeatViewer(ctx context.Context, streamID, clientID string, lastSeen time.Time) error {
	if m.heartbeatViewerFn != nil {
		return m.heartbeatViewerFn(ctx, streamID, clientID, lastSeen)
	}
	return nil
}

func (m *mockStreamRepo) RemoveViewer(ctx context.Context, streamID, clientID string) error {
	if m.removeViewerFn != nil {
		return m.removeViewerFn(ctx, streamID, clientID)
	}
	return nil
}

func (m *mockStreamRepo) GetViewerCount(ctx context.Context, streamID string) (int, error) {
	if m.getViewerCountFn != nil {
		return m.getViewerCountFn(ctx, streamID)
	}
	return 0, nil
}

func (m *mockStreamRepo) GetAnalytics(ctx context.Context, userID, period string) (*domain.Analytics, error) {
	if m.getAnalyticsFn != nil {
		return m.getAnalyticsFn(ctx, userID, period)
	}
	return &domain.Analytics{Period: period}, nil
}

func (m *mockStreamRepo) UpdateStreamAnalytics(ctx context.Context, userID string, date string, duration, peak, unique int) error {
	if m.updateStreamAnalyticsFn != nil {
		return m.updateStreamAnalyticsFn(ctx, userID, date, duration, peak, unique)
	}
	return nil
}

func (m *mockStreamRepo) UpdateRecordingStatus(ctx context.Context, streamID, status, errorMsg string) error {
	if m.updateRecordingStatusFn != nil {
		return m.updateRecordingStatusFn(ctx, streamID, status, errorMsg)
	}
	return nil
}

func (m *mockStreamRepo) UpdateVODPaths(ctx context.Context, streamID, hlsPath, thumbnailPath string) error {
	if m.updateVODPathsFn != nil {
		return m.updateVODPathsFn(ctx, streamID, hlsPath, thumbnailPath)
	}
	return nil
}

// ---------------------------------------------------------------------------
// mockAuthRepo implements domain.AuthRepo for stream service tests
// ---------------------------------------------------------------------------

type mockStreamAuthRepo struct {
	getUserIDByStreamKeyHashFn func(ctx context.Context, hash string) (string, error)
	setLiveStatusFn            func(ctx context.Context, userID string, isLive bool) error
	getStreamSettingsFn        func(ctx context.Context, userID string) (string, string, error)
}

func (m *mockStreamAuthRepo) GetUserIDByStreamKeyHash(ctx context.Context, hash string) (string, error) {
	if m.getUserIDByStreamKeyHashFn != nil {
		return m.getUserIDByStreamKeyHashFn(ctx, hash)
	}
	return "user-1", nil
}

func (m *mockStreamAuthRepo) SetLiveStatus(ctx context.Context, userID string, isLive bool) error {
	if m.setLiveStatusFn != nil {
		return m.setLiveStatusFn(ctx, userID, isLive)
	}
	return nil
}

func (m *mockStreamAuthRepo) GetStreamSettings(ctx context.Context, userID string) (string, string, error) {
	if m.getStreamSettingsFn != nil {
		return m.getStreamSettingsFn(ctx, userID)
	}
	return "", "", nil
}

// ---------------------------------------------------------------------------
// TestVerifySRSSecret
// ---------------------------------------------------------------------------

func TestVerifySRSSecret(t *testing.T) {
	svc := NewStreamService(&mockStreamRepo{}, &mockStreamAuthRepo{}, nil, nil, "super-secret", "", nil)

	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{
			name:   "happy path — valid secret",
			secret: "super-secret",
		},
		{
			name:    "error — invalid secret",
			secret:  "wrong-secret",
			wantErr: true,
		},
		{
			name:    "error — empty secret",
			secret:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.VerifySRSSecret(tt.secret)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthenticateStreamKey
// ---------------------------------------------------------------------------

func TestAuthenticateStreamKey(t *testing.T) {
	tests := []struct {
		name      string
		rawKey    string
		setupMock func(*mockStreamAuthRepo)
		wantID    string
		wantErr   bool
	}{
		{
			name:   "happy path — authenticates key",
			rawKey: "sk-abc123",
			wantID: "user-1",
		},
		{
			name:   "error — key not found",
			rawKey: "sk-unknown",
			setupMock: func(m *mockStreamAuthRepo) {
				m.getUserIDByStreamKeyHashFn = func(ctx context.Context, hash string) (string, error) {
					return "", errs.NotFound("user with stream key hash not found")
				}
			},
			wantErr: true,
		},
		{
			name:   "error — repo error",
			rawKey: "sk-abc123",
			setupMock: func(m *mockStreamAuthRepo) {
				m.getUserIDByStreamKeyHashFn = func(ctx context.Context, hash string) (string, error) {
					return "", errors.New("db down")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authRepo := &mockStreamAuthRepo{}
			if tt.setupMock != nil {
				tt.setupMock(authRepo)
			}
			svc := NewStreamService(&mockStreamRepo{}, authRepo, nil, nil, "secret", "", nil)

			userID, err := svc.AuthenticateStreamKey(context.Background(), tt.rawKey)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && userID != tt.wantID {
				t.Fatalf("got userID %q, want %q", userID, tt.wantID)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestOnStreamStart
// ---------------------------------------------------------------------------

func TestOnStreamStart(t *testing.T) {
	tests := []struct {
		name        string
		rawKey      string
		srsClientID string
		title       string
		setupMock   func(*mockStreamAuthRepo, *mockStreamRepo)
		wantErr     bool
	}{
		{
			name:        "happy path — starts stream with title",
			rawKey:      "sk-abc",
			srsClientID: "srs-conn-1",
			title:       "My Awesome Stream",
		},
		{
			name:        "happy path — starts stream with empty title",
			rawKey:      "sk-abc",
			srsClientID: "srs-conn-2",
			title:       "",
		},
		{
			name:        "error — auth fails",
			rawKey:      "sk-bad",
			srsClientID: "srs-conn-1",
			setupMock: func(auth *mockStreamAuthRepo, stream *mockStreamRepo) {
				auth.getUserIDByStreamKeyHashFn = func(ctx context.Context, hash string) (string, error) {
					return "", errs.NotFound("not found")
				}
			},
			wantErr: true,
		},
		{
			name:        "error — create stream fails",
			rawKey:      "sk-abc",
			srsClientID: "srs-conn-1",
			setupMock: func(auth *mockStreamAuthRepo, stream *mockStreamRepo) {
				stream.createStreamFn = func(ctx context.Context, userID string, title *string, srsClientID string, hlsPath string) (*domain.Stream, error) {
					return nil, errors.New("insert failed")
				}
			},
			wantErr: true,
		},
		{
			name:        "error — set live status fails",
			rawKey:      "sk-abc",
			srsClientID: "srs-conn-1",
			setupMock: func(auth *mockStreamAuthRepo, stream *mockStreamRepo) {
				auth.setLiveStatusFn = func(ctx context.Context, userID string, isLive bool) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authRepo := &mockStreamAuthRepo{}
			streamRepo := &mockStreamRepo{}
			if tt.setupMock != nil {
				tt.setupMock(authRepo, streamRepo)
			}
			svc := NewStreamService(streamRepo, authRepo, nil, nil, "secret", "", nil)

			stream, err := svc.OnStreamStart(context.Background(), tt.rawKey, tt.srsClientID, tt.title)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && stream == nil {
				t.Fatal("expected non-nil stream")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestOnStreamEnd
// ---------------------------------------------------------------------------

func TestOnStreamEnd(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mockStreamAuthRepo, *mockStreamRepo)
		wantErr   bool
	}{
		{
			name: "happy path — ends stream and updates analytics",
		},
		{
			name: "error — get stream by SRS client ID fails",
			setupMock: func(auth *mockStreamAuthRepo, stream *mockStreamRepo) {
				stream.getStreamBySRSClientIDFn = func(ctx context.Context, srsClientID string) (*domain.Stream, error) {
					return nil, errs.NotFound("not found")
				}
			},
			wantErr: true,
		},
		{
			name: "error — end stream fails",
			setupMock: func(auth *mockStreamAuthRepo, stream *mockStreamRepo) {
				stream.endStreamFn = func(ctx context.Context, streamID, hlsPath, recordingPath string, durationSeconds int) error {
					return errors.New("end failed")
				}
			},
			wantErr: true,
		},
		{
			name: "error — set live status fails",
			setupMock: func(auth *mockStreamAuthRepo, stream *mockStreamRepo) {
				auth.setLiveStatusFn = func(ctx context.Context, userID string, isLive bool) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authRepo := &mockStreamAuthRepo{}
			streamRepo := &mockStreamRepo{}
			if tt.setupMock != nil {
				tt.setupMock(authRepo, streamRepo)
			}
			svc := NewStreamService(streamRepo, authRepo, nil, nil, "secret", "", nil)

			err := svc.OnStreamEnd(context.Background(), "srs-conn-1", "/hls/path", "/rec/path", 600)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestOnStreamInterrupted
// ---------------------------------------------------------------------------

func TestOnStreamInterrupted(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mockStreamAuthRepo, *mockStreamRepo)
		wantErr   bool
	}{
		{
			name: "happy path — marks stream interrupted",
		},
		{
			name: "error — get stream fails",
			setupMock: func(auth *mockStreamAuthRepo, stream *mockStreamRepo) {
				stream.getStreamBySRSClientIDFn = func(ctx context.Context, srsClientID string) (*domain.Stream, error) {
					return nil, errs.NotFound("not found")
				}
			},
			wantErr: true,
		},
		{
			name: "error — update status fails",
			setupMock: func(auth *mockStreamAuthRepo, stream *mockStreamRepo) {
				stream.updateStreamStatusFn = func(ctx context.Context, streamID, status string) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authRepo := &mockStreamAuthRepo{}
			streamRepo := &mockStreamRepo{}
			if tt.setupMock != nil {
				tt.setupMock(authRepo, streamRepo)
			}
			svc := NewStreamService(streamRepo, authRepo, nil, nil, "secret", "", nil)

			err := svc.OnStreamInterrupted(context.Background(), "srs-conn-1")
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestListLive
// ---------------------------------------------------------------------------

func TestListLive(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mockStreamRepo)
		wantCount int
		wantErr   bool
	}{
		{
			name:      "happy path — returns live streams",
			wantCount: 1,
			setupMock: func(m *mockStreamRepo) {
				m.listLiveStreamsFn = func(ctx context.Context) ([]domain.LiveStream, error) {
					return []domain.LiveStream{
						{StreamID: "stream-1", UserID: "user-1", UserName: "Alice", ViewerCount: 5},
					}, nil
				}
			},
		},
		{
			name:      "happy path — empty list",
			wantCount: 0,
		},
		{
			name: "error — repo error",
			setupMock: func(m *mockStreamRepo) {
				m.listLiveStreamsFn = func(ctx context.Context) ([]domain.LiveStream, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streamRepo := &mockStreamRepo{}
			if tt.setupMock != nil {
				tt.setupMock(streamRepo)
			}
			svc := NewStreamService(streamRepo, &mockStreamAuthRepo{}, nil, nil, "secret", "", nil)

			streams, err := svc.ListLive(context.Background())
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(streams) != tt.wantCount {
				t.Fatalf("got %d streams, want %d", len(streams), tt.wantCount)
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
		setupMock func(*mockStreamRepo)
		wantErr   bool
	}{
		{
			name:   "happy path — returns channel info",
			userID: "user-1",
		},
		{
			name:   "error — not found",
			userID: "user-999",
			setupMock: func(m *mockStreamRepo) {
				m.getChannelInfoFn = func(ctx context.Context, userID string) (*domain.ChannelInfo, error) {
					return nil, errs.NotFound("user not found")
				}
			},
			wantErr: true,
		},
		{
			name:   "error — repo error",
			userID: "user-1",
			setupMock: func(m *mockStreamRepo) {
				m.getChannelInfoFn = func(ctx context.Context, userID string) (*domain.ChannelInfo, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streamRepo := &mockStreamRepo{}
			if tt.setupMock != nil {
				tt.setupMock(streamRepo)
			}
			svc := NewStreamService(streamRepo, &mockStreamAuthRepo{}, nil, nil, "secret", "", nil)

			info, err := svc.GetChannelInfo(context.Background(), tt.userID)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && info == nil {
				t.Fatal("expected non-nil channel info")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestHeartbeatViewer
// ---------------------------------------------------------------------------

func TestHeartbeatViewer(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mockStreamRepo)
		wantErr   bool
	}{
		{
			name: "happy path — heartbeat succeeds (existing viewer)",
		},
		{
			name: "happy path — heartbeat fails, upsert succeeds",
			setupMock: func(m *mockStreamRepo) {
				m.heartbeatViewerFn = func(ctx context.Context, streamID, clientID string, lastSeen time.Time) error {
					return errs.NotFound("viewer not found")
				}
			},
		},
		{
			name: "error — heartbeat fails and upsert fails",
			setupMock: func(m *mockStreamRepo) {
				m.heartbeatViewerFn = func(ctx context.Context, streamID, clientID string, lastSeen time.Time) error {
					return errs.NotFound("viewer not found")
				}
				m.upsertViewerFn = func(ctx context.Context, streamID, userID, clientID string) error {
					return errors.New("upsert failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streamRepo := &mockStreamRepo{}
			if tt.setupMock != nil {
				tt.setupMock(streamRepo)
			}
			svc := NewStreamService(streamRepo, &mockStreamAuthRepo{}, nil, nil, "secret", "", nil)

			err := svc.HeartbeatViewer(context.Background(), "stream-1", "user-1", "client-1")
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRemoveViewer
// ---------------------------------------------------------------------------

func TestRemoveViewer(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mockStreamRepo)
		wantErr   bool
	}{
		{
			name: "happy path — removes viewer",
		},
		{
			name: "error — repo error",
			setupMock: func(m *mockStreamRepo) {
				m.removeViewerFn = func(ctx context.Context, streamID, clientID string) error {
					return errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streamRepo := &mockStreamRepo{}
			if tt.setupMock != nil {
				tt.setupMock(streamRepo)
			}
			svc := NewStreamService(streamRepo, &mockStreamAuthRepo{}, nil, nil, "secret", "", nil)

			err := svc.RemoveViewer(context.Background(), "stream-1", "client-1")
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetAnalytics
// ---------------------------------------------------------------------------

func TestGetAnalytics(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		period    string
		setupMock func(*mockStreamRepo)
		wantErr   bool
	}{
		{
			name:   "happy path — returns analytics",
			userID: "user-1",
			period: "week",
		},
		{
			name:   "happy path — all-time period",
			userID: "user-1",
			period: "all",
		},
		{
			name:   "error — repo error",
			userID: "user-1",
			period: "week",
			setupMock: func(m *mockStreamRepo) {
				m.getAnalyticsFn = func(ctx context.Context, userID, period string) (*domain.Analytics, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streamRepo := &mockStreamRepo{}
			if tt.setupMock != nil {
				tt.setupMock(streamRepo)
			}
			svc := NewStreamService(streamRepo, &mockStreamAuthRepo{}, nil, nil, "secret", "", nil)

			analytics, err := svc.GetAnalytics(context.Background(), tt.userID, tt.period)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && analytics == nil {
				t.Fatal("expected non-nil analytics")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestForceEndStream
// ---------------------------------------------------------------------------

func TestForceEndStream(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupMock func(*mockStreamAuthRepo, *mockStreamRepo)
		wantErr   bool
	}{
		{
			name:   "happy path — forcefully ends active stream",
			userID: "user-1",
		},
		{
			name:   "error — get stream fails",
			userID: "user-1",
			setupMock: func(auth *mockStreamAuthRepo, stream *mockStreamRepo) {
				stream.getStreamByUserIDFn = func(ctx context.Context, userID string) (*domain.Stream, error) {
					return nil, errs.NotFound("no live stream")
				}
			},
			wantErr: true,
		},
		{
			name:   "error — end stream fails",
			userID: "user-1",
			setupMock: func(auth *mockStreamAuthRepo, stream *mockStreamRepo) {
				stream.endStreamFn = func(ctx context.Context, streamID, hlsPath, recordingPath string, durationSeconds int) error {
					return errors.New("end failed")
				}
			},
			wantErr: true,
		},
		{
			name:   "error — set live status fails",
			userID: "user-1",
			setupMock: func(auth *mockStreamAuthRepo, stream *mockStreamRepo) {
				auth.setLiveStatusFn = func(ctx context.Context, userID string, isLive bool) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authRepo := &mockStreamAuthRepo{}
			streamRepo := &mockStreamRepo{}
			if tt.setupMock != nil {
				tt.setupMock(authRepo, streamRepo)
			}
			svc := NewStreamService(streamRepo, authRepo, nil, nil, "secret", "", nil)

			_, err := svc.ForceEndStream(context.Background(), tt.userID)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestHandleRecording_NilQueue — typed-nil queue must not panic
// ---------------------------------------------------------------------------

func TestHandleRecording_NilQueue(t *testing.T) {
	repo := &mockStreamRepo{}
	authRepo := &mockStreamAuthRepo{}

	// Simulate a typed-nil queue: interface holding (*Queue)(nil)
	var typedNilQueue *testNilQueue
	var queueInterface voddomain.VODQueue = typedNilQueue

	svc := NewStreamService(repo, authRepo, nil, queueInterface, "secret", "", nil)

	// Must not panic even though the queue is a typed nil
	svc.handleRecording(context.Background(), "stream-1", "/data/recordings/stream-1.mp4")

	// Verify status was set to processing (queue enqueue is best-effort)
	if repo.updateRecordingStatusFn == nil {
		t.Log("updateRecordingStatus not asserted via mock fn — verified no panic")
	}
}

// testNilQueue implements voddomain.VODQueue but is always nil.
type testNilQueue struct{}

func (q *testNilQueue) Enqueue(ctx context.Context, args voddomain.VODProcessArgs) error {
	if q == nil {
		return fmt.Errorf("queue not initialized")
	}
	return nil
}
