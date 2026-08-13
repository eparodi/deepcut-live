package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"nhooyr.io/websocket"

	"github.com/deepcut/live/internal/modules/streams/domain"
	"github.com/deepcut/live/internal/shared/errs"
	"github.com/deepcut/live/internal/shared/render"

	authhttp "github.com/deepcut/live/internal/modules/auth/adapter/http"
)

// streamService is the subset of *application.StreamService methods that StreamHandler needs.
type streamService interface {
	VerifySRSSecret(secret string) error
	OnStreamStart(ctx context.Context, rawKey string, srsClientID string, title string) (*domain.Stream, error)
	OnStreamEnd(ctx context.Context, srsClientID string, hlsPath, recordingPath string, durationSeconds int) error
	ListLive(ctx context.Context) ([]domain.LiveStream, error)
	GetChannelInfo(ctx context.Context, userID string) (*domain.ChannelInfo, error)
	HeartbeatViewer(ctx context.Context, streamID, userID, clientID string) error
}

// streamHub is the subset of *application.StreamHub that the WebSocket handler needs.
type streamHub interface {
	Join(userID string, client *domain.StreamStatusClient)
	Leave(userID string, client *domain.StreamStatusClient)
	NotifyVODStatus(event domain.VODStatusEvent)
}

type StreamHandler struct {
	svc    streamService
	hub    streamHub
	logger *slog.Logger
}

func NewStreamHandler(svc streamService, hub streamHub, logger *slog.Logger) *StreamHandler {
	return &StreamHandler{svc: svc, hub: hub, logger: logger}
}

// RegisterRoutes registers all stream routes on the given router.
func (h *StreamHandler) RegisterRoutes(r chi.Router) {
	// SRS callback — authenticated by shared secret
	r.Post("/api/srs/callback", h.SRSCallback)

	// Public routes
	r.Get("/api/streams/live", h.ListLiveStreams)
	r.Get("/api/channel/{userID}", h.GetChannelInfo)
	r.Post("/api/streams/{streamID}/viewer-heartbeat", h.ViewerHeartbeat)

	// Internal route — called by cmd/worker after VOD processing
	r.Post("/api/internal/vod-status", h.VODStatusNotify)
}

// SRSCallback handles SRS on_publish/on_unpublish by dispatching based on the action field.
// Reads the body once, then restores it so dispatch handlers can re-read.
func (h *StreamHandler) SRSCallback(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// Read the body once into memory so dispatch handlers can re-read it
	bodyBytes, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	if err != nil {
		w.Write([]byte("1"))
		return
	}

	var actionCheck struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(bodyBytes, &actionCheck); err != nil {
		w.Write([]byte("1"))
		return
	}

	// Restore body so dispatch handlers can read it
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	switch actionCheck.Action {
	case "on_publish":
		h.SRSOnPublish(w, r)
	case "on_unpublish":
		h.SRSOnUnpublish(w, r)
	default:
		h.logger.Warn("unknown srs action", "action", actionCheck.Action)
		w.Write([]byte("0"))
	}
}

// SRSOnPublish handles the SRS on_publish callback.
func (h *StreamHandler) SRSOnPublish(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	secret := r.URL.Query().Get("secret")
	if err := h.svc.VerifySRSSecret(secret); err != nil {
		render.Error(w, r, fmt.Errorf("verify srs secret: %w", err))
		return
	}

	var body struct {
		Action   string `json:"action"`
		ClientID string `json:"client_id"`
		Stream   string `json:"stream"` // primary: the RTMP stream name (= OBS stream key)
		Param    string `json:"param"`  // fallback: query-string params from the RTMP URL
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	dec := json.NewDecoder(r.Body)
	// Do NOT DisallowUnknownFields — SRS sends ip, vhost, app, and other
	// fields that aren't relevant here.
	if err := dec.Decode(&body); err != nil {
		render.Error(w, r, errs.BadRequest("invalid JSON: %v", err))
		return
	}

	// Primary: the stream key is the RTMP stream name (SRS "stream" field).
	streamKey := body.Stream

	// Fallback: if stream is empty, try extracting from param query string.
	// Some RTMP clients encode the key as ?key=... in the URL query.
	if streamKey == "" {
		param := body.Param
		if len(param) > 0 && param[0] == '?' {
			param = param[1:]
		}
		vals, err := url.ParseQuery(param)
		if err != nil {
			render.Error(w, r, errs.BadRequest("invalid param: %v", err))
			return
		}
		if v, ok := vals["key"]; ok && len(v) > 0 {
			streamKey = v[0]
		}
	}

	stream, err := h.svc.OnStreamStart(r.Context(), streamKey, body.ClientID, "")
	if err != nil {
		h.logger.Error("on_publish failed", "error", err, "client_id", body.ClientID)
		render.Error(w, r, fmt.Errorf("on publish: %w", err))
		return
	}

	h.logger.Info("stream started", "stream_id", stream.ID, "user_id", stream.UserID)

	// SRS expects HTTP 200 with empty body or specific response
	render.JSON(w, http.StatusOK, map[string]any{"code": 0})
}

// SRSOnUnpublish handles the SRS on_unpublish callback.
func (h *StreamHandler) SRSOnUnpublish(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	secret := r.URL.Query().Get("secret")
	if err := h.svc.VerifySRSSecret(secret); err != nil {
		render.Error(w, r, fmt.Errorf("verify srs secret: %w", err))
		return
	}

	var body struct {
		Action   string `json:"action"`
		ClientID string `json:"client_id"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	dec := json.NewDecoder(r.Body)
	// Do NOT DisallowUnknownFields — SRS sends ip, vhost, app, tcUrl,
	// stream_url and other fields that aren't relevant here.
	if err := dec.Decode(&body); err != nil {
		render.Error(w, r, errs.BadRequest("invalid JSON: %v", err))
		return
	}

	hlsPath := r.URL.Query().Get("hls_path")
	recordingPath := r.URL.Query().Get("recording_path")
	durationStr := r.URL.Query().Get("duration")
	duration, _ := strconv.Atoi(durationStr)

	if err := h.svc.OnStreamEnd(r.Context(), body.ClientID, hlsPath, recordingPath, duration); err != nil {
		h.logger.Error("on_unpublish failed", "error", err, "client_id", body.ClientID)
		render.Error(w, r, fmt.Errorf("on unpublish: %w", err))
		return
	}

	h.logger.Info("stream ended", "client_id", body.ClientID)
	render.JSON(w, http.StatusOK, map[string]any{"code": 0})
}

type liveStreamsResponse struct {
	Streams []domain.LiveStream `json:"streams"`
	Total   int                 `json:"total"`
}

// ListLiveStreams returns all currently live streams.
func (h *StreamHandler) ListLiveStreams(w http.ResponseWriter, r *http.Request) {
	streams, err := h.svc.ListLive(r.Context())
	if err != nil {
		render.Error(w, r, fmt.Errorf("list live: %w", err))
		return
	}

	// Ensure we never marshal a null "streams" field — empty list must be [].
	if streams == nil {
		streams = []domain.LiveStream{}
	}

	render.JSON(w, http.StatusOK, liveStreamsResponse{
		Streams: streams,
		Total:   len(streams),
	})
}

// GetChannelInfo returns public channel info for a user.
func (h *StreamHandler) GetChannelInfo(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		render.Error(w, r, errs.BadRequest("missing user ID"))
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		render.Error(w, r, errs.BadRequest("invalid user ID — must be a UUID"))
		return
	}

	info, err := h.svc.GetChannelInfo(r.Context(), userID)
	if err != nil {
		render.Error(w, r, fmt.Errorf("get channel info: %w", err))
		return
	}
	render.JSON(w, http.StatusOK, info)
}

type heartbeatRequest struct {
	ClientID string `json:"clientId"`
}

// ViewerHeartbeat records a viewer heartbeat for a live stream.
func (h *StreamHandler) ViewerHeartbeat(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	streamID := chi.URLParam(r, "streamID")
	if streamID == "" {
		render.Error(w, r, errs.BadRequest("missing stream ID"))
		return
	}

	var req heartbeatRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		render.Error(w, r, errs.BadRequest("invalid JSON: %v", err))
		return
	}
	if req.ClientID == "" {
		render.Error(w, r, errs.BadRequest("missing client ID"))
		return
	}

	// userID comes from auth middleware if authenticated, empty for anonymous
	userID := ""

	if err := h.svc.HeartbeatViewer(r.Context(), streamID, userID, req.ClientID); err != nil {
		render.Error(w, r, fmt.Errorf("heartbeat: %w", err))
		return
	}

	render.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// VODStatusNotify is called by the cmd/worker after VOD processing completes.
// It broadcasts a vod_status event to all connected WebSocket clients.
func (h *StreamHandler) VODStatusNotify(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var event domain.VODStatusEvent
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&event); err != nil {
		render.Error(w, r, errs.BadRequest("invalid JSON: %v", err))
		return
	}

	if event.VodID == "" || event.Status == "" {
		render.Error(w, r, errs.BadRequest("missing vodId or status"))
		return
	}

	event.Type = "vod_status"

	if h.hub != nil {
		h.hub.NotifyVODStatus(event)
	}

	render.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// StreamWebSocket upgrades to WebSocket for real-time stream-status events.
// Must be inside the auth middleware group — userID comes from request context.
func (h *StreamHandler) StreamWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := authhttp.UserIDFromCtx(r.Context())
	if userID == "" {
		render.Error(w, r, errs.Unauthorized("not authenticated"))
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:3000", "localhost:8081"},
	})
	if err != nil {
		h.logger.Error("stream-status ws accept", "error", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	client := &domain.StreamStatusClient{
		UserID: userID,
		Send:   make(chan []byte, 64),
	}

	h.hub.Join(userID, client)
	defer h.hub.Leave(userID, client)

	// Connection lifetime — NOT derived from r.Context().
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	// writePump: send events from the hub to the WebSocket client.
	go func() {
		for {
			select {
			case data, ok := <-client.Send:
				if !ok {
					return
				}
				if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// readPump: keep the connection alive until the client disconnects.
	// We don't expect any messages from the client for this endpoint.
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}
}
