package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/deepcut/live/internal/modules/auth/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

type AuthRepo struct {
	pool *pgxpool.Pool
}

func NewAuthRepo(pool *pgxpool.Pool) *AuthRepo {
	return &AuthRepo{pool: pool}
}

func (r *AuthRepo) CreateUser(ctx context.Context, googleID, email, name, avatarURL, rawKey, keyHash string) (*domain.User, error) {
	query := `
		INSERT INTO users (google_id, email, name, avatar_url, stream_key, stream_key_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, google_id, email, name, avatar_url, stream_key, stream_key_hash,
		          COALESCE(stream_title, ''), COALESCE(stream_category, ''),
		          is_live, created_at, updated_at`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, googleID, email, name, avatarURL, rawKey, keyHash).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.AvatarURL,
		&u.StreamKey, &u.StreamKeyHash, &u.StreamTitle, &u.StreamCategory,
		&u.IsLive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &u, nil
}

func (r *AuthRepo) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	query := `
		SELECT id, google_id, email, name, avatar_url, stream_key, stream_key_hash,
		       COALESCE(stream_title, ''), COALESCE(stream_category, ''),
		       is_live, created_at, updated_at
		FROM users WHERE google_id = $1`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, googleID).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.AvatarURL,
		&u.StreamKey, &u.StreamKeyHash, &u.StreamTitle, &u.StreamCategory,
		&u.IsLive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.NotFound("user with google_id %s not found", googleID)
	}
	if err != nil {
		return nil, fmt.Errorf("query user by google_id: %w", err)
	}
	return &u, nil
}

func (r *AuthRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, google_id, email, name, avatar_url, stream_key, stream_key_hash,
		       COALESCE(stream_title, ''), COALESCE(stream_category, ''),
		       is_live, created_at, updated_at
		FROM users WHERE id = $1`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.AvatarURL,
		&u.StreamKey, &u.StreamKeyHash, &u.StreamTitle, &u.StreamCategory,
		&u.IsLive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.NotFound("user %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query user by id: %w", err)
	}
	return &u, nil
}

// GetUserIDByStreamKeyHash returns just the user ID for a given stream key hash.
// This satisfies the streams module's AuthRepo port.
func (r *AuthRepo) GetUserIDByStreamKeyHash(ctx context.Context, hash string) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx, `SELECT id FROM users WHERE stream_key_hash = $1`, hash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errs.NotFound("user with stream key hash not found")
	}
	if err != nil {
		return "", fmt.Errorf("query user id by stream_key_hash: %w", err)
	}
	return userID, nil
}

func (r *AuthRepo) GetByStreamKeyHash(ctx context.Context, hash string) (*domain.User, error) {
	query := `
		SELECT id, google_id, email, name, avatar_url, stream_key, stream_key_hash,
		       COALESCE(stream_title, ''), COALESCE(stream_category, ''),
		       is_live, created_at, updated_at
		FROM users WHERE stream_key_hash = $1`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, hash).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.AvatarURL,
		&u.StreamKey, &u.StreamKeyHash, &u.StreamTitle, &u.StreamCategory,
		&u.IsLive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.NotFound("user with stream key hash not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query user by stream_key_hash: %w", err)
	}
	return &u, nil
}

func (r *AuthRepo) UpdateStreamKey(ctx context.Context, userID, rawKey, keyHash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET stream_key = $2, stream_key_hash = $3, updated_at = now() WHERE id = $1`, userID, rawKey, keyHash)
	if err != nil {
		return fmt.Errorf("update stream key: %w", err)
	}
	return nil
}

func (r *AuthRepo) UpdateSettings(ctx context.Context, userID, title, category string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET stream_title = $2, stream_category = $3, updated_at = now() WHERE id = $1`,
		userID, title, category,
	)
	if err != nil {
		return fmt.Errorf("update settings: %w", err)
	}
	return nil
}

func (r *AuthRepo) SetLiveStatus(ctx context.Context, userID string, isLive bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET is_live = $2, live_since = CASE WHEN $2 THEN now() ELSE NULL END, updated_at = now() WHERE id = $1`,
		userID, isLive,
	)
	if err != nil {
		return fmt.Errorf("set live status: %w", err)
	}
	return nil
}

func (r *AuthRepo) GetStreamSettings(ctx context.Context, userID string) (title string, category string, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(stream_title, ''), COALESCE(stream_category, '') FROM users WHERE id = $1`,
		userID,
	).Scan(&title, &category)
	if err != nil {
		return "", "", fmt.Errorf("get stream settings: %w", err)
	}
	return title, category, nil
}

func (r *AuthRepo) GetLiveUsers(ctx context.Context) ([]domain.User, error) {
	query := `
		SELECT id, google_id, email, name, avatar_url, stream_key, stream_key_hash,
		       COALESCE(stream_title, ''), COALESCE(stream_category, ''),
		       is_live, created_at, updated_at
		FROM users WHERE is_live = true ORDER BY updated_at DESC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query live users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.AvatarURL,
			&u.StreamKey, &u.StreamKeyHash, &u.StreamTitle, &u.StreamCategory,
			&u.IsLive, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan live user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return users, nil
}
