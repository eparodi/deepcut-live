package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/deepcut/live/internal/modules/streams/domain"
	voddomain "github.com/deepcut/live/internal/modules/vods/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

// recordingPaths tracks the recording file path for each active stream.
var recordingPaths sync.Map // map[string]string (streamID → path)

type StreamService struct {
	repo      domain.StreamRepository
	authRepo  domain.AuthRepo
	hub       *StreamHub
	vodQueue  voddomain.VODQueue
	srsSecret string
	srsAPIURL string
	http      *http.Client
	logger    *slog.Logger
}

func NewStreamService(repo domain.StreamRepository, authRepo domain.AuthRepo, hub *StreamHub, vodQueue voddomain.VODQueue, srsSecret, srsAPIURL string, logger *slog.Logger) *StreamService {
	return &StreamService{
		repo:      repo,
		authRepo:  authRepo,
		hub:       hub,
		vodQueue:  vodQueue,
		srsSecret: srsSecret,
		srsAPIURL: srsAPIURL,
		http: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
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
	} else {
		// Fetch user's stream title from settings (same as poller)
		if userTitle, _, err := s.authRepo.GetStreamSettings(ctx, userID); err == nil && userTitle != "" {
			t = &userTitle
		}
	}

	// Construct the HLS playlist URL. SRS writes HLS files by default to
	// ./objs/nginx/html/[app]/[stream].m3u8, served on port 8080.
	// The frontend proxies /hls/* to SRS:8080, stripping the prefix.
	hlsPath := "/hls/live/" + rawKey + ".m3u8"

	stream, err := s.repo.CreateStream(ctx, userID, t, srsClientID, hlsPath)
	if err != nil {
		return nil, fmt.Errorf("on stream start: %w", err)
	}

	if err := s.authRepo.SetLiveStatus(ctx, userID, true); err != nil {
		return nil, fmt.Errorf("set live status: %w", err)
	}

	if s.hub != nil {
		s.hub.NotifyStreamStarted(userID, stream.ID)
	}

	s.startLiveThumbnail(stream.ID, rawKey)
	recordingPaths.Store(stream.ID, s.startRecording(stream.ID, rawKey))

	return stream, nil
}

// OnStreamEnd handles the SRS on_unpublish callback: ends the stream, marks user offline,
// and enqueues VOD processing if a recording path is available.
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

	if s.hub != nil {
		s.hub.NotifyStreamEnded(stream.UserID)
	}

	s.stopLiveThumbnail(stream.ID)
	s.stopRecording(stream.ID)

	// Use ffmpeg recording path over SRS callback path
	recPath := recordingPath
	if stored, ok := recordingPaths.Load(stream.ID); ok {
		recPath = stored.(string)
		recordingPaths.Delete(stream.ID)
	}

	// Update analytics
	date := time.Now().Format("2006-01-02")
	if err := s.repo.UpdateStreamAnalytics(ctx, stream.UserID, date, durationSeconds, stream.PeakViewers, stream.TotalViewers); err != nil {
		slog.Error("failed to update stream analytics", "err", err, "stream_id", stream.ID)
	}

	// Set recording status and enqueue VOD processing job
	s.handleRecording(ctx, stream.ID, recPath)

	return nil
}

// handleRecording sets the recording status and enqueues a VOD processing job.
func (s *StreamService) handleRecording(ctx context.Context, streamID, recordingPath string) {
	if recordingPath == "" {
		if err := s.repo.UpdateRecordingStatus(ctx, streamID, "failed", "No recording available"); err != nil {
			slog.Error("update recording status failed", "err", err, "stream_id", streamID)
		}
		return
	}

	if err := s.repo.UpdateRecordingStatus(ctx, streamID, "processing", ""); err != nil {
		slog.Error("update recording status to processing failed", "err", err, "stream_id", streamID)
		return
	}

	if s.vodQueue == nil {
		slog.Warn("vod queue not configured, skipping job enqueue", "stream_id", streamID)
		return
	}

	args := voddomain.VODProcessArgs{
		StreamID:      streamID,
		RecordingPath: recordingPath,
	}
	if err := s.vodQueue.Enqueue(ctx, args); err != nil {
		slog.Error("enqueue vod processing failed", "err", err, "stream_id", streamID)
		// Don't fail the whole on_unpublish — the recording exists on disk.
	}
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
	if s.hub != nil {
		s.hub.NotifyStreamEnded(stream.UserID)

		s.stopLiveThumbnail(stream.ID)
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

// GetAnalytics returns aggregated streaming analytics for the user.
func (s *StreamService) GetAnalytics(ctx context.Context, userID, period string) (*domain.Analytics, error) {
	if period == "" {
		period = "week"
	}
	return s.repo.GetAnalytics(ctx, userID, period)
}

// ForceEndStream terminates the user's current live stream.
// Returns a user-facing message on success; on error the caller should render an error response.
func (s *StreamService) ForceEndStream(ctx context.Context, userID string) (string, error) {
	stream, err := s.repo.GetStreamByUserID(ctx, userID)
	if err != nil {
		// Convert NotFound to Conflict — the user exists but has no active stream.
		var appErr *errs.AppError
		if errors.As(err, &appErr) && appErr.Kind == errs.KindNotFound {
			return "", errs.Conflict("no active stream to end")
		}
		return "", fmt.Errorf("get active stream: %w", err)
	}

	// Try to disconnect the publisher via SRS (graceful degradation).
	srsFailed := false
	if stream.SRSClientID != nil && s.srsAPIURL != "" {
		if err := s.disconnectSRSClient(ctx, *stream.SRSClientID); err != nil {
			slog.Warn("failed to disconnect SRS publisher", "err", err, "user_id", userID, "srs_client_id", *stream.SRSClientID)
			srsFailed = true
		}
	}

	duration := int(time.Since(stream.StartedAt).Seconds())
	if err := s.repo.EndStream(ctx, stream.ID, "", "", duration); err != nil {
		return "", fmt.Errorf("end stream: %w", err)
	}

	if err := s.authRepo.SetLiveStatus(ctx, userID, false); err != nil {
		return "", fmt.Errorf("set live status: %w", err)
	}

	if s.hub != nil {
		s.hub.NotifyStreamEnded(userID)
	}

	// Update analytics (same pattern as OnStreamEnd).
	date := time.Now().Format("2006-01-02")
	if err := s.repo.UpdateStreamAnalytics(ctx, userID, date, duration, stream.PeakViewers, stream.TotalViewers); err != nil {
		slog.Error("failed to update stream analytics", "err", err, "user_id", userID)
	}

	// Force-end has no recording path from SRS
	s.handleRecording(ctx, stream.ID, "")

	if srsFailed {
		return "Stream ended (publisher disconnect may have failed)", nil
	}
	return "Stream ended", nil
}

// disconnectSRSClient sends a DELETE to SRS to drop the publisher connection.
func (s *StreamService) disconnectSRSClient(ctx context.Context, clientID int) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/api/v1/clients/%d", s.srsAPIURL, clientID), nil)
	if err != nil {
		return fmt.Errorf("build srs request: %w", err)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("srs call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("srs responded with %d", resp.StatusCode)
	}
	return nil
}
