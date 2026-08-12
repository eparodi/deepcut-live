package application

import (
	"bytes"
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

// startRecording begins recording the HLS stream to an MPEG-TS file.
// Called from OnStreamStart. Stops when cancelled via stopRecording.
//
// TS is used instead of MP4 on purpose: it is a streaming container with no
// moov/index, so killing ffmpeg mid-stream (SIGKILL from CommandContext
// cancel at stream end) leaves a fully readable file. With MP4 the moov atom
// is lost on SIGKILL, which corrupts the AAC track's extradata and makes
// later TS muxing fail ("AAC bitstream not in ADTS format and extradata
// missing").
func (s *StreamService) startRecording(streamID, streamKey string) string {
	ctx, cancel := context.WithCancel(context.Background())
	streamRecordings.Store(streamID, cancel)

	recordingPath := filepath.Join("/data/recordings", streamID+".ts")
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
		// Reconnect flags retry the HLS playlist until SRS creates it.
		for ctx.Err() == nil {
			cmd := exec.CommandContext(ctx, "ffmpeg",
				"-reconnect", "1",
				"-reconnect_streamed", "1",
				"-reconnect_delay_max", "2",
				"-i", hlsURL,
				"-c", "copy",
				"-f", "mpegts",
				recordingPath,
				"-y",
			)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			err := cmd.Run()
			if ctx.Err() != nil {
				return // stream ended, stop retrying
			}
			if err != nil {
				s.warnLog("vod recording attempt failed, retrying",
					"err", err, "stream_id", streamID, "ffmpeg_stderr", tail(stderr.String(), 500))
				time.Sleep(2 * time.Second)
			}
		}
	}()

	return recordingPath
}

// tail returns the last n characters of s, used to log ffmpeg stderr without
// flooding logs with the full output.
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// stopRecording stops the ffmpeg recording goroutine for a stream.
// Returns the recording path that was being written to.
func (s *StreamService) stopRecording(streamID string) {
	if cancel, ok := streamRecordings.Load(streamID); ok {
		cancel.(context.CancelFunc)()
	}
}
