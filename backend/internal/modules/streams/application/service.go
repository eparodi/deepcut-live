package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/deepcut/live/internal/modules/streams/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

type StreamService struct {
	repo      domain.StreamRepository
	authRepo  domain.AuthRepo
	srsSecret string
}

func NewStreamService(repo domain.StreamRepository, authRepo domain.AuthRepo, srsSecret string) *StreamService {
	return &StreamService{
		repo:      repo,
		authRepo:  authRepo,
		srsSecret: srsSecret,
	}
}

// VerifySRSSecret validates the secret token from SRS callbacks.
func (s *StreamService) VerifySRSSecret(secret string) error {
	if secret != s.srsSecret {
		return errs.Forbidden("invalid srs secret")
	}
	return nil
}

// AuthenticateStreamKey hashes a raw stream key and looks up the user ID.
func (s *StreamService) AuthenticateStreamKey(ctx context.Context, rawKey string) (string, error) {
	sum := sha256.Sum256([]byte(rawKey))
	hash := hex.EncodeToString(sum[:])
	userID, err := s.authRepo.GetUserIDByStreamKeyHash(ctx, hash)
	if err != nil {
		return "", fmt.Errorf("authenticate stream key: %w", err)
	}
	return userID, nil
}

// OnStreamStart handles the SRS on_publish callback: validates key, creates stream, marks user live.
func (s *StreamService) OnStreamStart(ctx context.Context, rawKey string, srsClientID int, title string) (*domain.Stream, error) {
	userID, err := s.AuthenticateStreamKey(ctx, rawKey)
	if err != nil {
		return nil, fmt.Errorf("on stream start: %w", err)
	}

	var t *string
	if title != "" {
		t = &title
	}

	stream, err := s.repo.CreateStream(ctx, userID, t, srsClientID)
	if err != nil {
		return nil, fmt.Errorf("on stream start: %w", err)
	}

	if err := s.authRepo.SetLiveStatus(ctx, userID, true); err != nil {
		return nil, fmt.Errorf("set live status: %w", err)
	}

	return stream, nil
}

// OnStreamEnd handles the SRS on_unpublish callback: ends the stream, marks user offline.
func (s *StreamService) OnStreamEnd(ctx context.Context, srsClientID int, hlsPath, recordingPath string, durationSeconds int) error {
	stream, err := s.repo.GetStreamBySRSClientID(ctx, srsClientID)
	if err != nil {
		return fmt.Errorf("on stream end: %w", err)
	}

	if err := s.repo.EndStream(ctx, stream.ID, hlsPath, recordingPath, durationSeconds); err != nil {
		return fmt.Errorf("end stream: %w", err)
	}

	if err := s.authRepo.SetLiveStatus(ctx, stream.UserID, false); err != nil {
		return fmt.Errorf("set live status: %w", err)
	}

	// Update analytics
	date := time.Now().Format("2006-01-02")
	peak := stream.PeakViewers
	unique := stream.TotalViewers
	if peak == 0 {
		peak = 1
	}
	if unique == 0 {
		unique = 1
	}
	if err := s.repo.UpdateStreamAnalytics(ctx, stream.UserID, date, durationSeconds, peak, unique); err != nil {
		slog.Error("update stream analytics failed", "error", err, "user_id", stream.UserID)
	}

	return nil
}

// OnStreamInterrupted marks a stream as interrupted.
func (s *StreamService) OnStreamInterrupted(ctx context.Context, srsClientID int) error {
	stream, err := s.repo.GetStreamBySRSClientID(ctx, srsClientID)
	if err != nil {
		return fmt.Errorf("on stream interrupted: %w", err)
	}
	if err := s.repo.UpdateStreamStatus(ctx, stream.ID, "interrupted"); err != nil {
		return fmt.Errorf("update stream status: %w", err)
	}
	if err := s.authRepo.SetLiveStatus(ctx, stream.UserID, false); err != nil {
		return fmt.Errorf("set live status: %w", err)
	}
	return nil
}

// ListLive returns all currently live streams with viewer counts.
func (s *StreamService) ListLive(ctx context.Context) ([]domain.LiveStream, error) {
	streams, err := s.repo.ListLiveStreams(ctx)
	if err != nil {
		return nil, fmt.Errorf("list live: %w", err)
	}
	return streams, nil
}

// GetChannelInfo returns public channel info for a user.
func (s *StreamService) GetChannelInfo(ctx context.Context, userID string) (*domain.ChannelInfo, error) {
	info, err := s.repo.GetChannelInfo(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get channel info: %w", err)
	}
	return info, nil
}

// HeartbeatViewer records a viewer heartbeat.
func (s *StreamService) HeartbeatViewer(ctx context.Context, streamID, userID, clientID string) error {
	if err := s.repo.HeartbeatViewer(ctx, streamID, clientID, time.Now()); err != nil {
		if err := s.repo.UpsertViewer(ctx, streamID, userID, clientID); err != nil {
			return fmt.Errorf("upsert viewer: %w", err)
		}
	}
	return nil
}

// RemoveViewer removes a viewer from the count.
func (s *StreamService) RemoveViewer(ctx context.Context, streamID, clientID string) error {
	if err := s.repo.RemoveViewer(ctx, streamID, clientID); err != nil {
		return fmt.Errorf("remove viewer: %w", err)
	}
	return nil
}
