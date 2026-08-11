package domain

import "time"

// Stream represents a streaming session from the database.
type Stream struct {
	ID              string
	UserID          string
	Title           *string
	StartedAt       time.Time
	EndedAt         *time.Time
	Status          string
	HLSPath         *string
	RecordingPath   *string
	RecordingStatus string
	PeakViewers     int
	TotalViewers    int
	DurationSeconds *int
	SRSClientID     *int
	CreatedAt       time.Time
}

// LiveStream combines a stream with its channel owner info for the public live list.
type LiveStream struct {
	StreamID     string  `json:"streamId"`
	UserID       string  `json:"userId"`
	UserName     string  `json:"streamerName"`
	UserAvatar   *string `json:"streamerAvatarUrl"`
	Title        *string `json:"title"`
	Category     *string `json:"category"`
	StartedAt    string  `json:"startedAt"`
	HlsPath      *string `json:"-"`
	ThumbnailUrl *string `json:"thumbnailUrl"`
	ViewerCount  int     `json:"viewerCount"`
}

// ChannelInfo provides public-facing channel/profile information.
type ChannelInfo struct {
	UserID         string  `json:"userId"`
	UserName       string  `json:"streamerName"`
	UserAvatar     *string `json:"streamerAvatarUrl"`
	StreamTitle    *string `json:"streamTitle"`
	StreamCategory *string `json:"streamCategory"`
	IsLive         bool    `json:"isLive"`
	StreamID       *string `json:"streamId"`
	StartedAt      *string `json:"startedAt,omitempty"`
	HlsPath        *string `json:"hlsUrl,omitempty"`
	ThumbnailUrl   *string `json:"thumbnailUrl,omitempty"`
	ViewerCount    int     `json:"viewerCount"`
}

// StreamStatusClient represents a WebSocket-connected client listening for stream status events.
type StreamStatusClient struct {
	UserID string
	Send   chan []byte
}

// StreamStatusEvent is sent over the WebSocket to notify the frontend of stream state changes.
type StreamStatusEvent struct {
	Type     string `json:"type"`               // "streamStarted" or "streamEnded"
	StreamID string `json:"streamId,omitempty"` // set on streamStarted
}

// VODStatusEvent is broadcast when a VOD's recording status changes.
type VODStatusEvent struct {
	Type         string `json:"type"` // "vod_status"
	VodID        string `json:"vodId"`
	Status       string `json:"status"` // "ready" or "failed"
	HlsURL       string `json:"hlsUrl,omitempty"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
	Error        string `json:"error,omitempty"`
}

// Analytics holds aggregated streaming statistics.
type Analytics struct {
	Period        string `json:"period"`
	StartDate     string `json:"startDate"`
	EndDate       string `json:"endDate"`
	TotalSeconds  int    `json:"totalStreamTimeSeconds"`
	PeakViewers   int    `json:"peakViewers"`
	UniqueViewers int    `json:"totalUniqueViewers"`
	TotalStreams  int    `json:"totalStreams"`
}
