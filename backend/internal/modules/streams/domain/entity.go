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
	StreamID    string  `json:"streamId"`
	UserID      string  `json:"userId"`
	UserName    string  `json:"userName"`
	UserAvatar  *string `json:"userAvatar"`
	Title       *string `json:"title"`
	Category    *string `json:"category"`
	StartedAt   string  `json:"startedAt"`
	HlsPath     *string `json:"hlsPath"`
	ViewerCount int     `json:"viewerCount"`
}

// ChannelInfo provides public-facing channel/profile information.
type ChannelInfo struct {
	UserID         string  `json:"userId"`
	UserName       string  `json:"userName"`
	UserAvatar     *string `json:"userAvatar"`
	StreamTitle    *string `json:"streamTitle"`
	StreamCategory *string `json:"streamCategory"`
	IsLive         bool    `json:"isLive"`
	StartedAt      *string `json:"startedAt,omitempty"`
	HlsPath        *string `json:"hlsPath,omitempty"`
	ViewerCount    int     `json:"viewerCount"`
}
