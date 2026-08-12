package application

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// liveThumbnails tracks active thumbnail capture goroutines by stream ID.
var liveThumbnails sync.Map // map[string]context.CancelFunc

// startLiveThumbnail begins capturing frames from the live HLS stream.
// Captures immediately on start, then every 5 seconds. Requires ffmpeg.
func (s *StreamService) startLiveThumbnail(streamID, streamKey string) {
	ctx, cancel := context.WithCancel(context.Background())
	liveThumbnails.Store(streamID, cancel)

	// Derive SRS HTTP URL from the API URL (e.g. http://srs:1985 → http://srs:8080)
	srsHTTP := srsHTTPURL(s.srsAPIURL)

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		defer liveThumbnails.Delete(streamID)

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
			cmd.Stderr = nil
			if err := cmd.Run(); err != nil {
				s.warnLog("live thumbnail capture failed", "err", err, "stream_id", streamID)
			}
		}

		capture() // first frame immediately

		for {
			select {
			case <-ctx.Done():
				os.Remove(thumbnailPath)
				return
			case <-ticker.C:
				capture()
			}
		}
	}()
}

// stopLiveThumbnail stops the live thumbnail goroutine for a stream.
func (s *StreamService) stopLiveThumbnail(streamID string) {
	if cancel, ok := liveThumbnails.Load(streamID); ok {
		cancel.(context.CancelFunc)()
	}
}
