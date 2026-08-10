package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// srsClient represents an active RTMP publisher from the SRS API.
type srsClient struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"` // stream key
	Publish bool    `json:"publish"`
	Alive   float64 `json:"alive"`
}

type srsClientsResponse struct {
	Clients []srsClient `json:"clients"`
}

// StartSRSPoller begins a background loop that polls the SRS API for active
// publishers and syncs them with the database. This is a fallback for when
// SRS http_hooks callbacks don't fire (known compatibility issue with some
// SRS Docker configurations).
func (s *StreamService) StartSRSPoller(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Track which keys we've already processed to avoid duplicates.
	seen := make(map[string]bool)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollSRS(ctx, seen)
		}
	}
}

func (s *StreamService) pollSRS(ctx context.Context, seen map[string]bool) {
	clients, err := s.fetchSRSClients(ctx)
	if err != nil {
		slog.Warn("srs poller: fetch clients failed", "err", err)
		return // next poll will retry
	}

	activeKeys := make(map[string]bool)
	for _, c := range clients {
		if !c.Publish || c.Name == "" {
			continue
		}
		activeKeys[c.Name] = true

		if seen[c.Name] {
			continue // already processed
		}

		// Try to authenticate and start the stream.
		userID, err := s.AuthenticateStreamKey(ctx, c.Name)
		if err != nil {
			slog.Warn("srs poller: authenticate stream key failed", "err", err, "key", c.Name)
			continue
		}

		// Check if user already has an active stream.
		_, err = s.repo.GetStreamByUserID(ctx, userID)
		if err == nil {
			seen[c.Name] = true
			continue // already live
		}

		hlsPath := "/hls/live/" + c.Name + ".m3u8"
		stream, err := s.repo.CreateStream(ctx, userID, nil, 0, hlsPath)
		if err != nil {
			slog.Error("srs poller: create stream", "err", err, "user_id", userID)
			continue
		}

		if err := s.authRepo.SetLiveStatus(ctx, userID, true); err != nil {
			slog.Error("srs poller: set live status", "err", err, "user_id", userID)
			continue
		}

		s.hub.NotifyStreamStarted(userID, stream.ID)
		seen[c.Name] = true
		slog.Info("srs poller: stream started", "stream_id", stream.ID, "user_id", userID)
	}

	// Detect streams that ended (key in seen but no longer active).
	for key := range seen {
		if !activeKeys[key] {
			// Find the stream by key hash and end it.
			userID, err := s.AuthenticateStreamKey(ctx, key)
			if err == nil {
				stream, err := s.repo.GetStreamByUserID(ctx, userID)
				if err == nil {
					if endErr := s.repo.EndStream(ctx, stream.ID, "", "", 0); endErr != nil {
						slog.Error("srs poller: end stream failed", "err", endErr, "stream_id", stream.ID)
					}
					if statusErr := s.authRepo.SetLiveStatus(ctx, userID, false); statusErr != nil {
						slog.Error("srs poller: set live status failed", "err", statusErr, "user_id", userID)
					}
					s.hub.NotifyStreamEnded(userID)
					slog.Info("srs poller: stream ended", "user_id", userID)
				}
			}
			delete(seen, key)
		}
	}
}

func (s *StreamService) fetchSRSClients(ctx context.Context) ([]srsClient, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/v1/clients/", s.srsAPIURL), nil)
	if err != nil {
		return nil, fmt.Errorf("build srs request: %w", err)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("srs api call: %w", err)
	}
	defer resp.Body.Close()

	var body srsClientsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode srs response: %w", err)
	}

	return body.Clients, nil
}
