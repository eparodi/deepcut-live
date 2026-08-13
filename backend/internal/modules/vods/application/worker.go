package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	voddomain "github.com/deepcut/live/internal/modules/vods/domain"
)

// VODWorker processes River jobs to transcode VOD recordings.
type VODWorker struct {
	river.WorkerDefaults[voddomain.VODProcessArgs]
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewVODWorker creates a River-compatible VOD processing worker.
func NewVODWorker(pool *pgxpool.Pool, logger *slog.Logger) *VODWorker {
	return &VODWorker{pool: pool, logger: logger}
}

// Work processes a VOD recording job.
func (w *VODWorker) Work(ctx context.Context, job *river.Job[voddomain.VODProcessArgs]) error {
	args := job.Args
	w.logger.Info("processing vod", "stream_id", args.StreamID, "recording_path", args.RecordingPath)

	encodingPreset := os.Getenv("VOD_ENCODING_PRESET")
	if encodingPreset == "" {
		encodingPreset = "copy"
	}

	hlsDir := filepath.Join("/data/hls/vods", args.StreamID)
	if err := os.MkdirAll(hlsDir, 0o755); err != nil {
		return w.fail(ctx, args.StreamID, fmt.Errorf("mkdir hls dir: %w", err))
	}

	hlsPath := filepath.Join(hlsDir, "index.m3u8")
	if err := w.transcode(ctx, args.RecordingPath, hlsPath, encodingPreset); err != nil {
		return w.fail(ctx, args.StreamID, fmt.Errorf("transcode: %w", err))
	}

	thumbnailDir := "/data/hls/thumbnails"
	if err := os.MkdirAll(thumbnailDir, 0o755); err != nil {
		return w.fail(ctx, args.StreamID, fmt.Errorf("mkdir thumbnail dir: %w", err))
	}

	thumbnailPath := filepath.Join(thumbnailDir, args.StreamID+".jpg")
	if err := w.generateThumbnail(ctx, args.RecordingPath, thumbnailPath); err != nil {
		w.logger.Warn("thumbnail generation failed, continuing", "err", err, "stream_id", args.StreamID)
	}

	if err := w.markReady(ctx, args.StreamID, "/hls/vods/"+args.StreamID+"/index.m3u8", "/hls/thumbnails/"+args.StreamID+".jpg"); err != nil {
		return fmt.Errorf("mark ready: %w", err)
	}

	w.notifyServer(args.StreamID, string(voddomain.RecordingStatusReady), "/hls/vods/"+args.StreamID+"/index.m3u8", "/hls/thumbnails/"+args.StreamID+".jpg", "")
	w.logger.Info("vod processing complete", "stream_id", args.StreamID)
	return nil
}

func (w *VODWorker) transcode(ctx context.Context, inputPath, outputPath, preset string) error {
	var args []string
	switch preset {
	case "720p":
		args = []string{"-i", inputPath, "-vf", "scale=-2:720", "-c:v", "libx264", "-b:v", "2M", "-c:a", "aac", "-b:a", "128k"}
	case "480p":
		args = []string{"-i", inputPath, "-vf", "scale=-2:480", "-c:v", "libx264", "-b:v", "1M", "-c:a", "aac", "-b:a", "96k"}
	default: // "copy"
		args = []string{"-i", inputPath, "-c:v", "copy", "-c:a", "copy"}
	}

	args = append(args,
		"-hls_time", "4",
		"-hls_list_size", "0",
		"-hls_segment_filename", filepath.Join(filepath.Dir(outputPath), "%03d.ts"),
		outputPath,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg: %w", err)
	}
	return nil
}

func (w *VODWorker) generateThumbnail(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-ss", "00:00:01",
		"-vframes", "1",
		"-q:v", "3",
		outputPath,
		"-y",
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg thumbnail: %w", err)
	}
	return nil
}

func (w *VODWorker) markReady(ctx context.Context, streamID, hlsPath, thumbnailPath string) error {
	_, err := w.pool.Exec(ctx, `
		UPDATE streams
		SET recording_status = 'ready',
		    vod_hls_path = $2,
		    vod_thumbnail_path = $3
		WHERE id = $1`, streamID, hlsPath, thumbnailPath)
	if err != nil {
		return fmt.Errorf("update stream: %w", err)
	}
	return nil
}

func (w *VODWorker) fail(ctx context.Context, streamID string, err error) error {
	w.logger.Error("vod processing failed", "err", err, "stream_id", streamID)
	if dbErr := w.markFailed(ctx, streamID, err.Error()); dbErr != nil {
		w.logger.Error("failed to mark vod as failed", "err", dbErr, "stream_id", streamID)
	}
	return err
}

func (w *VODWorker) markFailed(ctx context.Context, streamID, errorMsg string) error {
	_, err := w.pool.Exec(ctx, `
		UPDATE streams
		SET recording_status = 'failed',
		    recording_error = $2
		WHERE id = $1`, streamID, errorMsg)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	return nil
}

// notifyServer sends a VOD status update to the main backend server.
func (w *VODWorker) notifyServer(vodID, status, hlsURL, thumbnailURL, errMsg string) {
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8081"
	}

	payload := map[string]string{
		"vodId":        vodID,
		"status":       status,
		"hlsUrl":       hlsURL,
		"thumbnailUrl": thumbnailURL,
		"error":        errMsg,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		w.logger.Warn("vod status notify: marshal payload failed", "err", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, backendURL+"/api/internal/vod-status", bytes.NewReader(body))
	if err != nil {
		w.logger.Warn("vod status notify: build request failed", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.logger.Warn("vod status notify: request failed", "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.logger.Warn("vod status notify: unexpected status", "status", resp.StatusCode)
	}
}
