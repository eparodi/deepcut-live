package domain

import (
	"context"
)

type ChatRepository interface {
	SaveMessage(ctx context.Context, streamID, userID, message string) (*ChatMessage, error)
	GetMessages(ctx context.Context, streamID string, before string, limit int) ([]ChatMessage, bool, error)
	GetStreamStatus(ctx context.Context, streamID string) (isLive bool, err error)
}
