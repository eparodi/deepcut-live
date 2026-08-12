package application

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// streamRecordings tracks active ffmpeg recording goroutines by stream ID.
var streamRecordings sync.Map // map[string]context.CancelFunc

// startRecording begins recording the HLS stream to an MP4 file.
// Called from OnStreamStart. Stops when cancelled via stopRecording.
func (s *StreamService) startRecording(streamID, streamKey string) string {
	ctx, cancel := context.WithCancel(context.Background())
	streamRecordings.Store(streamID, cancel)

	recordingPath := filepath.Join("/data/recordings", streamID+".mp4")
	os.MkdirAll(filepath.Dir(recordingPath), 0o755) // best-effort, may fail in test

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

		// Retry loop: ffmpeg may exit if the HLS playlist isn't ready yet.
		// Restart until the stream ends (context cancelled).
		for ctx.Err() == nil {
			// Fragmented MP4 (frag_keyframe+empty_moov) — every fragment is
			// independently playable, so killing ffmpeg mid-stream (SIGKILL
			// from CommandContext cancel) still leaves a valid file.
			// Reconnect flags retry the HLS playlist until SRS creates it.
			cmd := exec.CommandContext(ctx, "ffmpeg",
				"-reconnect", "1",
				"-reconnect_streamed", "1",
				"-reconnect_delay_max", "2",
				"-i", hlsURL,
				"-c", "copy",
				"-movflags", "frag_keyframe+empty_moov+default_base_moof",
				"-f", "mp4",
				recordingPath,
				"-y",
			)
			cmd.Stderr = nil
			err := cmd.Run()
			if ctx.Err() != nil {
				return // stream ended, stop retrying
			}
			if err != nil {
				s.warnLog("vod recording attempt failed, retrying", "err", err, "stream_id", streamID)
				time.Sleep(2 * time.Second)
			}
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
