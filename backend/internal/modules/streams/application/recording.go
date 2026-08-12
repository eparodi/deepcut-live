package application

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// streamRecordings tracks active ffmpeg recording goroutines by stream ID.
var streamRecordings sync.Map // map[string]context.CancelFunc

// startRecording begins recording the HLS stream to an MP4 file.
// Called from OnStreamStart. Stops when cancelled via stopRecording.
func (s *StreamService) startRecording(streamID, streamKey string) string {
	ctx, cancel := context.WithCancel(context.Background())
	streamRecordings.Store(streamID, cancel)

	recordingPath := filepath.Join("/data/recordings", streamID+".mp4")
	if err := os.MkdirAll(filepath.Dir(recordingPath), 0o755); err != nil {
		s.warnLog("vod recording: mkdir failed", "err", err)
		return ""
	}

	srsHTTP := "http://localhost:8080"
	if s.srsAPIURL != "" {
		srsHTTP = strings.Replace(s.srsAPIURL, ":1985", ":8080", 1)
		if srsHTTP == s.srsAPIURL {
			srsHTTP = s.srsAPIURL + ":8080"
		}
	}

	hlsURL := fmt.Sprintf("%s/live/%s.m3u8", srsHTTP, streamKey)

	go func() {
		defer streamRecordings.Delete(streamID)

		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-i", hlsURL,
			"-c", "copy",
			"-f", "mp4",
			recordingPath,
			"-y",
		)
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil && ctx.Err() == nil {
			s.warnLog("vod recording failed", "err", err, "stream_id", streamID)
		}
	}()

	return recordingPath
}

// stopRecording stops the ffmpeg recording goroutine for a stream.
// Returns the recording path that was being written to.
func (s *StreamService) stopRecording(streamID string) {
	if cancel, ok := streamRecordings.Load(streamID); ok {
		cancel.(context.CancelFunc)()
	}
}
