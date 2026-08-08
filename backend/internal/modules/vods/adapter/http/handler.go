package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/deepcut/live/internal/modules/vods/domain"
	"github.com/deepcut/live/internal/shared/errs"
	"github.com/deepcut/live/internal/shared/render"
)

// vodService is the subset of *application.VODService methods that VODHandler needs.
type vodService interface {
	GetVOD(ctx context.Context, vodID string) (*domain.VOD, error)
	ListVODs(ctx context.Context, userID string, limit, offset int) ([]domain.VOD, error)
	SearchVODs(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error)
	RecordViewerHeartbeat(ctx context.Context, vodID string) error
}

type VODHandler struct {
	svc    vodService
	logger *slog.Logger
}

func NewVODHandler(svc vodService, logger *slog.Logger) *VODHandler {
	return &VODHandler{svc: svc, logger: logger}
}

// RegisterRoutes registers all VOD routes on the given router.
func (h *VODHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/vods", h.SearchVODs)
	r.Get("/api/vods/{vodID}", h.GetVOD)
	r.Get("/api/channels/{userID}/vods", h.ListUserVODs)
	r.Post("/api/vods/{vodID}/heartbeat", h.ViewerHeartbeat)
}

// GetVOD returns a single VOD by ID.
func (h *VODHandler) GetVOD(w http.ResponseWriter, r *http.Request) {
	vodID := chi.URLParam(r, "vodID")
	if vodID == "" {
		render.Error(w, r, errs.BadRequest("missing vod ID"))
		return
	}

	vod, err := h.svc.GetVOD(r.Context(), vodID)
	if err != nil {
		render.Error(w, r, fmt.Errorf("get vod: %w", err))
		return
	}
	render.JSON(w, http.StatusOK, vod)
}

// SearchVODs searches VODs with filters and pagination.
func (h *VODHandler) SearchVODs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := parseIntParam(q.Get("limit"), 20, 1, 100)
	offset := parseIntParam(q.Get("offset"), 0, 0, 10000)

	params := domain.SearchParams{
		Query:    q.Get("q"),
		Category: q.Get("category"),
		Status:   q.Get("status"),
		Sort:     q.Get("sort"),
		Limit:    limit,
		Offset:   offset,
	}

	result, err := h.svc.SearchVODs(r.Context(), params)
	if err != nil {
		render.Error(w, r, fmt.Errorf("search vods: %w", err))
		return
	}
	render.JSON(w, http.StatusOK, result)
}

// ListUserVODs returns VODs for a specific user.
func (h *VODHandler) ListUserVODs(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		render.Error(w, r, errs.BadRequest("missing user ID"))
		return
	}

	q := r.URL.Query()
	limit := parseIntParam(q.Get("limit"), 20, 1, 100)
	offset := parseIntParam(q.Get("offset"), 0, 0, 10000)

	vods, err := h.svc.ListVODs(r.Context(), userID, limit, offset)
	if err != nil {
		render.Error(w, r, fmt.Errorf("list vods: %w", err))
		return
	}
	render.JSON(w, http.StatusOK, vods)
}

type vodHeartbeatRequest struct {
	ClientID string `json:"clientId"`
}

// ViewerHeartbeat records a viewer heartbeat for a VOD.
func (h *VODHandler) ViewerHeartbeat(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vodID := chi.URLParam(r, "vodID")
	if vodID == "" {
		render.Error(w, r, errs.BadRequest("missing vod ID"))
		return
	}

	var req vodHeartbeatRequest
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

	if err := h.svc.RecordViewerHeartbeat(r.Context(), vodID); err != nil {
		h.logger.Error("viewer heartbeat failed", "error", err, "vod_id", vodID)
		render.Error(w, r, fmt.Errorf("heartbeat: %w", err))
		return
	}

	render.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func parseIntParam(s string, defaultVal, min, max int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < min || v > max {
		return defaultVal
	}
	return v
}
