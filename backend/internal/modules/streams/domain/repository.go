package domain

import (
	"context"
	"time"
)

type StreamRepository interface {
	CreateStream(ctx context.Context, userID string, title *string, srsClientID string, hlsPath string) (*Stream, error)
	EndStream(ctx context.Context, streamID string, hlsPath, recordingPath string, durationSeconds int) error
	UpdateStreamStatus(ctx context.Context, streamID, status string) error
	UpdateRecordingStatus(ctx context.Context, streamID, status, errorMsg string) error
	UpdateVODPaths(ctx context.Context, streamID, hlsPath, thumbnailPath string) error
	GetStreamByUserID(ctx context.Context, userID string) (*Stream, error)
	GetStreamBySRSClientID(ctx context.Context, srsClientID string) (*Stream, error)

	ListLiveStreams(ctx context.Context) ([]LiveStream, error)
	GetChannelInfo(ctx context.Context, userID string) (*ChannelInfo, error)

	UpsertViewer(ctx context.Context, streamID, userID, clientID string) error
	HeartbeatViewer(ctx context.Context, streamID, clientID string, lastSeen time.Time) error
	RemoveViewer(ctx context.Context, streamID, clientID string) error
	GetViewerCount(ctx context.Context, streamID string) (int, error)

	GetAnalytics(ctx context.Context, userID, period string) (*Analytics, error)

	UpdateStreamAnalytics(ctx context.Context, userID string, date string, duration, peak, unique int) error
}

// AuthRepo abstracts user lookups needed by the stream service.
// The streams module does NOT import auth types — it defines its own port.
type AuthRepo interface {
	GetUserIDByStreamKeyHash(ctx context.Context, hash string) (string, error)
	SetLiveStatus(ctx context.Context, userID string, isLive bool) error
	GetStreamSettings(ctx context.Context, userID string) (title string, category string, err error)
}
