package domain

import "time"

// VOD represents a past stream available for on-demand viewing.
type VOD struct {
	ID               string     `json:"id"`
	UserID           string     `json:"userId"`
	UserName         string     `json:"userName"`
	UserAvatar       *string    `json:"userAvatar"`
	Title            *string    `json:"title"`
	StartedAt        time.Time  `json:"startedAt"`
	EndedAt          *time.Time `json:"endedAt"`
	DurationSeconds  *int       `json:"durationSeconds"`
	PeakViewers      int        `json:"peakViewers"`
	TotalViewers     int        `json:"totalViewers"`
	RecordingPath    *string    `json:"recordingPath"`
	RecordingStatus  string     `json:"recordingStatus"`
	VodHlsPath       *string    `json:"hlsUrl"`
	VodThumbnailPath *string    `json:"thumbnailUrl"`
	RecordingError   *string    `json:"recordingError"`
	CreatedAt        time.Time  `json:"createdAt"`
}

// SearchParams defines filters and pagination for VOD search.
type SearchParams struct {
	Query    string // search by title or streamer name
	UserID   string // filter by streamer user ID
	Category string
	Status   string // recording_status filter: 'ready', 'processing', 'failed'
	Sort     string // 'recent', 'popular', 'longest'
	Limit    int
	Offset   int
}

// SearchResult wraps a paginated VOD search response.
type SearchResult struct {
	VODs       []VOD `json:"vods"`
	TotalCount int   `json:"totalCount"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
}
