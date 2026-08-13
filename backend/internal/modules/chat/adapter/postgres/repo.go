package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/deepcut/live/internal/modules/chat/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

type ChatRepo struct {
	pool *pgxpool.Pool
}

func NewChatRepo(pool *pgxpool.Pool) *ChatRepo {
	return &ChatRepo{pool: pool}
}

func (r *ChatRepo) SaveMessage(ctx context.Context, streamID, userID, message string) (*domain.ChatMessage, error) {
	var msg domain.ChatMessage
	err := r.pool.QueryRow(ctx, `
			INSERT INTO chat_messages (stream_id, user_id, message)
			VALUES ($1, $2, $3)
			RETURNING id, stream_id, user_id, message, sent_at`,
		streamID, userID, message,
	).Scan(&msg.ID, &msg.StreamID, &msg.UserID, &msg.Message, &msg.SentAt)
	if err != nil {
		return nil, fmt.Errorf("insert chat message: %w", err)
	}
	return &msg, nil
}

// GetStreamStatus checks whether a stream exists and is currently live.
func (r *ChatRepo) GetStreamStatus(ctx context.Context, streamID string) (bool, error) {
	var status string
	err := r.pool.QueryRow(ctx, `SELECT status FROM streams WHERE id = $1`, streamID).Scan(&status)
	if err != nil {
		return false, fmt.Errorf("get stream status: %w", err)
	}
	return status == "live", nil
}

func (r *ChatRepo) GetMessages(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error) {
	// Validate stream exists first
	var streamExists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM streams WHERE id = $1)`, streamID).Scan(&streamExists)
	if err != nil {
		return nil, false, fmt.Errorf("check stream exists: %w", err)
	}
	if !streamExists {
		return nil, false, fmt.Errorf("stream %s not found: %w", streamID, errs.NotFound("stream not found"))
	}

	// Fetch one extra to determine hasMore
	fetchLimit := limit + 1

	var rows pgx.Rows
	if before == "" {
		rows, err = r.pool.Query(ctx, `
			SELECT cm.id, cm.stream_id, cm.user_id, u.name, COALESCE(u.avatar_url, ''), cm.message, cm.sent_at
			FROM chat_messages cm
			JOIN users u ON cm.user_id = u.id
			WHERE cm.stream_id = $1
			ORDER BY cm.sent_at DESC
			LIMIT $2`,
			streamID, fetchLimit)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT cm.id, cm.stream_id, cm.user_id, u.name, COALESCE(u.avatar_url, ''), cm.message, cm.sent_at
			FROM chat_messages cm
			JOIN users u ON cm.user_id = u.id
			WHERE cm.stream_id = $1 AND cm.sent_at < $2
			ORDER BY cm.sent_at DESC
			LIMIT $3`,
			streamID, before, fetchLimit)
	}
	if err != nil {
		return nil, false, fmt.Errorf("query chat messages: %w", err)
	}
	defer rows.Close()

	var msgs []domain.ChatMessage
	for rows.Next() {
		var m domain.ChatMessage
		if err := rows.Scan(&m.ID, &m.StreamID, &m.UserID, &m.UserName, &m.UserAvatarUrl, &m.Message, &m.SentAt); err != nil {
			return nil, false, fmt.Errorf("scan chat message: %w", err)
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate chat messages: %w", err)
	}

	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}

	return msgs, hasMore, nil
}

// Ensure interface compliance.
var _ domain.ChatRepository = (*ChatRepo)(nil)
