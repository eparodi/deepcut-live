package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/deepcut/live/internal/modules/streams/application"
	"github.com/deepcut/live/internal/shared/errs"
	"github.com/deepcut/live/internal/shared/render"
)

type StreamHandler struct {
	svc    *application.StreamService
	logger *slog.Logger
}

func NewStreamHandler(svc *application.StreamService, logger *slog.Logger) *StreamHandler {
	return &StreamHandler{svc: svc, logger: logger}
}

// RegisterRoutes registers all stream routes on the given router.
func (h *StreamHandler) RegisterRoutes(r chi.Router) {
	// SRS callbacks — authenticated by shared secret
	r.Post("/api/srs/on_publish", h.SRSOnPublish)
	r.Post("/api/srs/on_unpublish", h.SRSOnUnpublish)

	// Public routes
	r.Get("/api/streams/live", h.ListLiveStreams)
	r.Get("/api/channels/{userID}", h.GetChannelInfo)
	r.Post("/api/streams/{streamID}/heartbeat", h.ViewerHeartbeat)
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
		ClientID int    `json:"client_id"`
		Param    string `json:"param"` // contains ?secret=...&key=...
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		render.Error(w, r, errs.BadRequest("invalid JSON: %v", err))
		return
	}

	// Extract stream key from param (strip leading ?)
	param := body.Param
	if len(param) > 0 && param[0] == '?' {
		param = param[1:]
	}
	vals, err := url.ParseQuery(param)
	if err != nil {
		render.Error(w, r, errs.BadRequest("invalid param: %v", err))
		return
	}
	streamKey := ""
	if v, ok := vals["key"]; ok && len(v) > 0 {
		streamKey = v[0]
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
		ClientID int    `json:"client_id"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
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

// ListLiveStreams returns all currently live streams.
func (h *StreamHandler) ListLiveStreams(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	streams, err := h.svc.ListLive(r.Context())
	if err != nil {
		render.Error(w, r, fmt.Errorf("list live: %w", err))
		return
	}
	render.JSON(w, http.StatusOK, streams)
}

// GetChannelInfo returns public channel info for a user.
func (h *StreamHandler) GetChannelInfo(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID := chi.URLParam(r, "userID")
	if userID == "" {
		render.Error(w, r, errs.BadRequest("missing user ID"))
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
