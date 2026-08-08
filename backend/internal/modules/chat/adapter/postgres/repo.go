package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/deepcut/live/internal/modules/chat/domain"
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

func (r *ChatRepo) GetMessages(ctx context.Context, streamID string, limit, offset int) ([]domain.ChatMessage, error) {
	query := `
		SELECT cm.id, cm.stream_id, cm.user_id, u.name, cm.message, cm.sent_at
		FROM chat_messages cm
		JOIN users u ON cm.user_id = u.id
		WHERE cm.stream_id = $1
		ORDER BY cm.sent_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, streamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query chat messages: %w", err)
	}
	defer rows.Close()

	var msgs []domain.ChatMessage
	for rows.Next() {
		var m domain.ChatMessage
		if err := rows.Scan(&m.ID, &m.StreamID, &m.UserID, &m.UserName, &m.Message, &m.SentAt); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

// Ensure interface compliance.
var _ domain.ChatRepository = (*ChatRepo)(nil)
