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
	s.streamRecordings.Store(streamID, cancel)

	recordingPath := filepath.Join("/data/recordings", streamID+".ts")
	if err := os.MkdirAll(filepath.Dir(recordingPath), 0o755); err != nil {
		// Best-effort (fails outside the container, e.g. in tests): ffmpeg
		// will fail below and keep retrying until the stream ends.
		s.warnLog("vod recording: mkdir failed", "err", err, "stream_id", streamID)
	}

	hlsURL := fmt.Sprintf("%s/live/%s.m3u8", srsHTTPURL(s.srsAPIURL), streamKey)

	go func() {
		defer s.streamRecordings.Delete(streamID)
		defer cancel() // release the context if we exit for any other reason

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
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
				}
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
func (s *StreamService) stopRecording(streamID string) {
	if stored, ok := s.streamRecordings.Load(streamID); ok {
		if cancel, ok := stored.(context.CancelFunc); ok {
			cancel()
		}
	}
}
