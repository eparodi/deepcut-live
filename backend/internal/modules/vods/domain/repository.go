package domain

import "context"

type VODRepository interface {
	GetVOD(ctx context.Context, vodID string) (*VOD, error)
	ListVODs(ctx context.Context, userID string, limit, offset int) ([]VOD, error)
	SearchVODs(ctx context.Context, params SearchParams) (*SearchResult, error)
	IncrementViewCount(ctx context.Context, vodID string) error
}
