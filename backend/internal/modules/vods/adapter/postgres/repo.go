package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/deepcut/live/internal/modules/vods/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

type VODRepo struct {
	pool *pgxpool.Pool
}

func NewVODRepo(pool *pgxpool.Pool) *VODRepo {
	return &VODRepo{pool: pool}
}

func (r *VODRepo) GetVOD(ctx context.Context, vodID string) (*domain.VOD, error) {
	query := `
		SELECT s.id, s.user_id, u.name, u.avatar_url,
		       s.title, s.started_at, s.ended_at,
		       s.duration_seconds, s.peak_viewers, s.total_viewers,
		       s.recording_path, s.recording_status, s.created_at
		FROM streams s
		JOIN users u ON s.user_id = u.id
		WHERE s.id = $1 AND s.status = 'offline'`

	var vod domain.VOD
	err := r.pool.QueryRow(ctx, query, vodID).Scan(
		&vod.ID, &vod.UserID, &vod.UserName, &vod.UserAvatar,
		&vod.Title, &vod.StartedAt, &vod.EndedAt,
		&vod.DurationSeconds, &vod.PeakViewers, &vod.TotalViewers,
		&vod.RecordingPath, &vod.RecordingStatus, &vod.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, errs.NotFound("vod %s not found", vodID)
	}
	if err != nil {
		return nil, fmt.Errorf("query vod: %w", err)
	}
	return &vod, nil
}

func (r *VODRepo) ListVODs(ctx context.Context, userID string, limit, offset int) ([]domain.VOD, error) {
	query := `
		SELECT s.id, s.user_id, u.name, u.avatar_url,
		       s.title, s.started_at, s.ended_at,
		       s.duration_seconds, s.peak_viewers, s.total_viewers,
		       s.recording_path, s.recording_status, s.created_at
		FROM streams s
		JOIN users u ON s.user_id = u.id
		WHERE s.user_id = $1 AND s.status = 'offline'
		ORDER BY s.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query vods: %w", err)
	}
	defer rows.Close()

	return scanVODs(rows)
}

func (r *VODRepo) SearchVODs(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
	baseQuery := `
		FROM streams s
		JOIN users u ON s.user_id = u.id
		WHERE s.status = 'offline'`

	countQuery := "SELECT COUNT(*) " + baseQuery
	dataQuery := "SELECT s.id, s.user_id, u.name, u.avatar_url, s.title, s.started_at, s.ended_at, s.duration_seconds, s.peak_viewers, s.total_viewers, s.recording_path, s.recording_status, s.created_at " + baseQuery

	args := []any{}
	argIdx := 1

	// Search filter
	if params.Query != "" {
		filter := fmt.Sprintf(" AND (s.title ILIKE $%d OR u.name ILIKE $%d)", argIdx, argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, "%"+params.Query+"%")
		argIdx++
	}

	// Recording status filter
	if params.Status != "" {
		filter := fmt.Sprintf(" AND s.recording_status = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, params.Status)
		argIdx++
	}

	// Sorting
	switch params.Sort {
	case "popular":
		dataQuery += " ORDER BY s.peak_viewers DESC"
	case "longest":
		dataQuery += " ORDER BY s.duration_seconds DESC NULLS LAST"
	default:
		dataQuery += " ORDER BY s.created_at DESC"
	}

	// Pagination
	var totalCount int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("count vods: %w", err)
	}

	dataQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, params.Limit, params.Offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query search vods: %w", err)
	}
	defer rows.Close()

	vods, err := scanVODs(rows)
	if err != nil {
		return nil, err
	}

	return &domain.SearchResult{
		VODs:       vods,
		TotalCount: totalCount,
		Limit:      params.Limit,
		Offset:     params.Offset,
	}, nil
}

func (r *VODRepo) IncrementViewCount(ctx context.Context, vodID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE streams SET total_viewers = total_viewers + 1, peak_viewers = GREATEST(peak_viewers, total_viewers + 1) WHERE id = $1`, vodID)
	if err != nil {
		return fmt.Errorf("increment view count: %w", err)
	}
	return nil
}

func scanVODs(rows pgx.Rows) ([]domain.VOD, error) {
	var vods []domain.VOD
	for rows.Next() {
		var v domain.VOD
		if err := rows.Scan(
			&v.ID, &v.UserID, &v.UserName, &v.UserAvatar,
			&v.Title, &v.StartedAt, &v.EndedAt,
			&v.DurationSeconds, &v.PeakViewers, &v.TotalViewers,
			&v.RecordingPath, &v.RecordingStatus, &v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan vod: %w", err)
		}
		vods = append(vods, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vods: %w", err)
	}
	return vods, nil
}
