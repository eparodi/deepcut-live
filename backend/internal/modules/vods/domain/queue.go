package domain

import "context"

// VODProcessArgs is the payload for a VOD processing job.
type VODProcessArgs struct {
	StreamID      string `json:"stream_id"`
	RecordingPath string `json:"recording_path"`
}

// Kind returns the job kind for River's worker registry.
func (VODProcessArgs) Kind() string { return "vod_process" }

// VODQueue is the interface for enqueuing VOD processing jobs.
// Swapping River for Redis/RabbitMQ only requires a new implementation
// of this interface — the StreamService never changes.
type VODQueue interface {
	Enqueue(ctx context.Context, args VODProcessArgs) error
}
