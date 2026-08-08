package domain

import (
	"context"
)

type ChatRepository interface {
	SaveMessage(ctx context.Context, streamID, userID, message string) (*ChatMessage, error)
	GetMessages(ctx context.Context, streamID string, limit, offset int) ([]ChatMessage, error)
}
