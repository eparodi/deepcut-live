package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/deepcut/live/internal/modules/streams/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

type StreamRepo struct {
	pool *pgxpool.Pool
}

func NewStreamRepo(pool *pgxpool.Pool) *StreamRepo {
	return &StreamRepo{pool: pool}
}

func (r *StreamRepo) CreateStream(ctx context.Context, userID string, title *string, srsClientID int) (*domain.Stream, error) {
	var s domain.Stream
	err := r.pool.QueryRow(ctx, `
		INSERT INTO streams (user_id, title, status, srs_client_id)
		VALUES ($1, $2, 'live', $3)
		RETURNING id, user_id, title, started_at, ended_at, status,
		          hls_path, recording_path, recording_status,
		          peak_viewers, total_viewers, duration_seconds, srs_client_id, created_at`,
		userID, title, srsClientID,
	).Scan(
		&s.ID, &s.UserID, &s.Title, &s.StartedAt, &s.EndedAt, &s.Status,
		&s.HLSPath, &s.RecordingPath, &s.RecordingStatus,
		&s.PeakViewers, &s.TotalViewers, &s.DurationSeconds, &s.SRSClientID, &s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert stream: %w", err)
	}
	return &s, nil
}

func (r *StreamRepo) EndStream(ctx context.Context, streamID string, hlsPath, recordingPath string, durationSeconds int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE streams
		SET ended_at = now(), status = 'offline',
		    hls_path = $2, recording_path = $3,
		    duration_seconds = $4
		WHERE id = $1`,
		streamID, hlsPath, recordingPath, durationSeconds,
	)
	if err != nil {
		return fmt.Errorf("end stream: %w", err)
	}
	return nil
}

func (r *StreamRepo) UpdateStreamStatus(ctx context.Context, streamID, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE streams SET status = $2 WHERE id = $1`, streamID, status)
	if err != nil {
		return fmt.Errorf("update stream status: %w", err)
	}
	return nil
}

func (r *StreamRepo) GetStreamByUserID(ctx context.Context, userID string) (*domain.Stream, error) {
	var s domain.Stream
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, title, started_at, ended_at, status,
		       hls_path, recording_path, recording_status,
		       peak_viewers, total_viewers, duration_seconds, srs_client_id, created_at
		FROM streams WHERE user_id = $1 AND status = 'live'
		ORDER BY started_at DESC LIMIT 1`, userID,
	).Scan(
		&s.ID, &s.UserID, &s.Title, &s.StartedAt, &s.EndedAt, &s.Status,
		&s.HLSPath, &s.RecordingPath, &s.RecordingStatus,
		&s.PeakViewers, &s.TotalViewers, &s.DurationSeconds, &s.SRSClientID, &s.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, errs.NotFound("no live stream for user %s", userID)
	}
	if err != nil {
		return nil, fmt.Errorf("query stream by user: %w", err)
	}
	return &s, nil
}

func (r *StreamRepo) GetStreamBySRSClientID(ctx context.Context, srsClientID int) (*domain.Stream, error) {
	var s domain.Stream
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, title, started_at, ended_at, status,
		       hls_path, recording_path, recording_status,
		       peak_viewers, total_viewers, duration_seconds, srs_client_id, created_at
		FROM streams WHERE srs_client_id = $1
		ORDER BY started_at DESC LIMIT 1`, srsClientID,
	).Scan(
		&s.ID, &s.UserID, &s.Title, &s.StartedAt, &s.EndedAt, &s.Status,
		&s.HLSPath, &s.RecordingPath, &s.RecordingStatus,
		&s.PeakViewers, &s.TotalViewers, &s.DurationSeconds, &s.SRSClientID, &s.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, errs.NotFound("stream with srs_client_id %d not found", srsClientID)
	}
	if err != nil {
		return nil, fmt.Errorf("query stream by srs_client_id: %w", err)
	}
	return &s, nil
}

func (r *StreamRepo) ListLiveStreams(ctx context.Context) ([]domain.LiveStream, error) {
	query := `
		SELECT s.id, u.id, u.name, u.avatar_url, s.title,
		       COALESCE(u.stream_category, ''),
		       s.started_at, s.hls_path,
		       COALESCE(vc.viewer_count, 0)
		FROM streams s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN (
			SELECT stream_id, COUNT(*) as viewer_count
			FROM stream_viewers
			GROUP BY stream_id
		) vc ON vc.stream_id = s.id
		WHERE s.status = 'live'
		ORDER BY s.started_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query live streams: %w", err)
	}
	defer rows.Close()

	var result []domain.LiveStream
	for rows.Next() {
		var ls domain.LiveStream
		var startedAt time.Time
		if err := rows.Scan(
			&ls.StreamID, &ls.UserID, &ls.UserName, &ls.UserAvatar,
			&ls.Title, &ls.Category,
			&startedAt, &ls.HlsPath, &ls.ViewerCount,
		); err != nil {
			return nil, fmt.Errorf("scan live stream: %w", err)
		}
		ls.StartedAt = startedAt.Format("2006-01-02T15:04:05Z")
		result = append(result, ls)
	}
	return result, nil
}

func (r *StreamRepo) GetChannelInfo(ctx context.Context, userID string) (*domain.ChannelInfo, error) {
	query := `
		SELECT u.id, u.name, u.avatar_url,
		       u.stream_title, u.stream_category,
		       u.is_live, u.live_since,
		       COALESCE(s.hls_path, ''),
		       COALESCE(vc.viewer_count, 0)
		FROM users u
		LEFT JOIN streams s ON s.user_id = u.id AND s.status = 'live'
		LEFT JOIN (
			SELECT stream_id, COUNT(*) as viewer_count
			FROM stream_viewers
			GROUP BY stream_id
		) vc ON vc.stream_id = s.id
		WHERE u.id = $1`

	var info domain.ChannelInfo
	var liveSince *time.Time
	var hlsPath string
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&info.UserID, &info.UserName, &info.UserAvatar,
		&info.StreamTitle, &info.StreamCategory,
		&info.IsLive, &liveSince, &hlsPath, &info.ViewerCount,
	)
	if err == pgx.ErrNoRows {
		return nil, errs.NotFound("user %s not found", userID)
	}
	if err != nil {
		return nil, fmt.Errorf("query channel info: %w", err)
	}
	if info.IsLive && liveSince != nil {
		ts := liveSince.Format("2006-01-02T15:04:05Z")
		info.StartedAt = &ts
	}
	if hlsPath != "" {
		info.HlsPath = &hlsPath
	}
	return &info, nil
}

func (r *StreamRepo) UpsertViewer(ctx context.Context, streamID, userID, clientID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO stream_viewers (stream_id, user_id, client_id)
		VALUES ($1, NULLIF($2, ''), $3)
		ON CONFLICT (stream_id, client_id) DO UPDATE SET last_seen = now()`,
		streamID, userID, clientID,
	)
	if err != nil {
		return fmt.Errorf("upsert viewer: %w", err)
	}
	return nil
}

func (r *StreamRepo) HeartbeatViewer(ctx context.Context, streamID, clientID string, lastSeen time.Time) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE stream_viewers SET last_seen = $3
		WHERE stream_id = $1 AND client_id = $2`,
		streamID, clientID, lastSeen,
	)
	if err != nil {
		return fmt.Errorf("heartbeat viewer: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errs.NotFound("viewer not found")
	}
	return nil
}

func (r *StreamRepo) RemoveViewer(ctx context.Context, streamID, clientID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM stream_viewers WHERE stream_id = $1 AND client_id = $2`,
		streamID, clientID,
	)
	if err != nil {
		return fmt.Errorf("remove viewer: %w", err)
	}
	return nil
}

func (r *StreamRepo) GetViewerCount(ctx context.Context, streamID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM stream_viewers WHERE stream_id = $1`, streamID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get viewer count: %w", err)
	}
	return count, nil
}

func (r *StreamRepo) UpdateStreamAnalytics(ctx context.Context, userID string, date string, duration, peak, unique int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO stream_analytics (user_id, date, total_seconds, peak_viewers, unique_viewers)
		VALUES ($1, $2::date, $3, $4, $5)
		ON CONFLICT (user_id, date) DO UPDATE SET
			total_seconds = stream_analytics.total_seconds + $3,
			peak_viewers = GREATEST(stream_analytics.peak_viewers, $4),
			unique_viewers = GREATEST(stream_analytics.unique_viewers, $5)`,
		userID, date, duration, peak, unique,
	)
	if err != nil {
		return fmt.Errorf("update analytics: %w", err)
	}
	return nil
}
