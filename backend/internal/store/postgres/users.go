package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/deepcut/live/internal/model"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateUser(ctx context.Context, googleID, email, name, avatarURL, keyHash string) (*model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (google_id, email, name, avatar_url, stream_key_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, google_id, email, name, avatar_url, stream_key_hash,
		          stream_title, stream_category, is_live, created_at, updated_at
	`, googleID, email, name, nullString(avatarURL), keyHash).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.AvatarURL, &u.StreamKeyHash,
		&u.StreamTitle, &u.StreamCategory, &u.IsLive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (s *Store) GetUserByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, google_id, email, name, avatar_url, stream_key_hash,
		       stream_title, stream_category, is_live, created_at, updated_at
		FROM users WHERE google_id = $1
	`, googleID).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.AvatarURL, &u.StreamKeyHash,
		&u.StreamTitle, &u.StreamCategory, &u.IsLive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by google id: %w", err)
	}
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, google_id, email, name, avatar_url, stream_key_hash,
		       stream_title, stream_category, is_live, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.AvatarURL, &u.StreamKeyHash,
		&u.StreamTitle, &u.StreamCategory, &u.IsLive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (s *Store) UpdateStreamKeyHash(ctx context.Context, userID, keyHash string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET stream_key_hash = $1, updated_at = $2 WHERE id = $3
	`, keyHash, time.Now(), userID)
	return err
}

func (s *Store) GetUserByStreamKeyHash(ctx context.Context, keyHash string) (*model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, google_id, email, name, avatar_url, stream_key_hash,
		       stream_title, stream_category, is_live, created_at, updated_at
		FROM users WHERE stream_key_hash = $1
	`, keyHash).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.AvatarURL, &u.StreamKeyHash,
		&u.StreamTitle, &u.StreamCategory, &u.IsLive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by stream key hash: %w", err)
	}
	return &u, nil
}

func (s *Store) UpdateStreamSettings(ctx context.Context, userID, title, category string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET stream_title = $1, stream_category = $2, updated_at = $3 WHERE id = $4
	`, title, category, time.Now(), userID)
	return err
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
