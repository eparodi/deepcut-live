package application

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// startLiveThumbnail begins capturing frames from the live HLS stream.
// Captures immediately on start, then every 5 seconds. Requires ffmpeg.
func (s *StreamService) startLiveThumbnail(streamID, streamKey string) {
	ctx, cancel := context.WithCancel(context.Background())
	s.liveThumbnails.Store(streamID, cancel)

	// Derive SRS HTTP URL from the API URL (e.g. http://srs:1985 → http://srs:8080)
	srsHTTP := srsHTTPURL(s.srsAPIURL)

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		defer s.liveThumbnails.Delete(streamID)
		defer cancel() // release the context if we exit for any other reason

		thumbnailPath := filepath.Join("/data/hls/thumbnails/live", streamID+".jpg")
		if err := os.MkdirAll(filepath.Dir(thumbnailPath), 0o755); err != nil {
			s.warnLog("live thumbnail: mkdir failed", "err", err)
			return
		}

		hlsURL := fmt.Sprintf("%s/live/%s.m3u8", srsHTTP, streamKey)

		capture := func() {
			cmd := exec.CommandContext(ctx, "ffmpeg",
				"-i", hlsURL,
				"-vframes", "1",
				"-q:v", "3",
				thumbnailPath,
				"-y",
			)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil && ctx.Err() == nil {
				s.warnLog("live thumbnail capture failed",
					"err", err, "stream_id", streamID, "ffmpeg_stderr", tail(stderr.String(), 500))
			}
		}

		capture() // first frame immediately

		for {
			select {
			case <-ctx.Done():
				if err := os.Remove(thumbnailPath); err != nil && !os.IsNotExist(err) {
					s.debugLog("live thumbnail cleanup failed", "err", err, "stream_id", streamID)
				}
				return
			case <-ticker.C:
				capture()
			}
		}
	}()
}

// stopLiveThumbnail stops the live thumbnail goroutine for a stream.
func (s *StreamService) stopLiveThumbnail(streamID string) {
	if stored, ok := s.liveThumbnails.Load(streamID); ok {
		if cancel, ok := stored.(context.CancelFunc); ok {
			cancel()
		}
	}
}
