package application

import (
	"context"
	"fmt"

	"github.com/deepcut/live/internal/modules/vods/domain"
)

type VODService struct {
	repo domain.VODRepository
}

func NewVODService(repo domain.VODRepository) *VODService {
	return &VODService{repo: repo}
}

// GetVOD returns a single VOD by ID.
func (s *VODService) GetVOD(ctx context.Context, vodID string) (*domain.VOD, error) {
	vod, err := s.repo.GetVOD(ctx, vodID)
	if err != nil {
		return nil, fmt.Errorf("get vod: %w", err)
	}
	return vod, nil
}

// ListVODs returns VODs for a specific user.
func (s *VODService) ListVODs(ctx context.Context, userID string, limit, offset int) ([]domain.VOD, error) {
	vods, err := s.repo.ListVODs(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list vods: %w", err)
	}
	return vods, nil
}

// SearchVODs searches VODs with filters and pagination.
func (s *VODService) SearchVODs(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
	result, err := s.repo.SearchVODs(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search vods: %w", err)
	}
	return result, nil
}

// RecordViewerHeartbeat increments the view count for a VOD.
func (s *VODService) RecordViewerHeartbeat(ctx context.Context, vodID string) error {
	if err := s.repo.IncrementViewCount(ctx, vodID); err != nil {
		return fmt.Errorf("record viewer heartbeat: %w", err)
	}
	return nil
}
