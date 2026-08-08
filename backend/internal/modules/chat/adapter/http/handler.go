package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/deepcut/live/internal/modules/chat/application"
	"github.com/deepcut/live/internal/modules/chat/domain"
	"github.com/deepcut/live/internal/shared/errs"
	"github.com/deepcut/live/internal/shared/render"
)

type ChatHandler struct {
	svc    *application.ChatService
	hub    *application.ChatHub
	logger *slog.Logger
}

func NewChatHandler(svc *application.ChatService, logger *slog.Logger) *ChatHandler {
	return &ChatHandler{
		svc:    svc,
		hub:    svc.GetHub(),
		logger: logger,
	}
}

// RegisterRoutes registers all chat routes on the given router.
func (h *ChatHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/chat/ws/{streamID}", h.ChatWebSocket)
	r.Get("/api/chat/messages/{streamID}", h.GetMessages)
}

// ChatWebSocket upgrades an HTTP connection to WebSocket for real-time chat.
func (h *ChatHandler) ChatWebSocket(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "streamID")
	if streamID == "" {
		render.Error(w, r, errs.BadRequest("missing stream ID"))
		return
	}

	// Extract user info from query params (auth middleware not required for chat)
	userID := r.URL.Query().Get("userId")
	userName := r.URL.Query().Get("userName")
	if userID == "" || userName == "" {
		render.Error(w, r, errs.BadRequest("missing userId or userName"))
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.logger.Error("websocket accept", "error", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	client := &domain.ChatClient{
		UserID:   userID,
		UserName: userName,
		Send:     make(chan []byte, 64),
	}

	h.hub.Join(streamID, client)
	defer h.hub.Leave(streamID, client)

	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	// Read messages from client
	go h.readPump(ctx, conn, streamID, client)

	// Write messages to client
	h.writePump(ctx, conn, client)
}

func (h *ChatHandler) readPump(ctx context.Context, conn *websocket.Conn, streamID string, client *domain.ChatClient) {
	defer conn.Close(websocket.StatusNormalClosure, "")

	for {
		_, msgBytes, err := conn.Read(ctx)
		if err != nil {
			break
		}

		var msg struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(msgBytes, &msg); err != nil || msg.Message == "" {
			continue
		}

		_, err = h.svc.SendMessage(ctx, streamID, client.UserID, client.UserName, msg.Message)
		if err != nil {
			h.logger.Error("send chat message", "error", err)
		}
	}
}

func (h *ChatHandler) writePump(ctx context.Context, conn *websocket.Conn, client *domain.ChatClient) {
	for {
		select {
		case data, ok := <-client.Send:
			if !ok {
				return
			}
			if err := wsjson.Write(ctx, conn, data); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// GetMessages returns chat message history for a stream.
func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	streamID := chi.URLParam(r, "streamID")
	if streamID == "" {
		render.Error(w, r, errs.BadRequest("missing stream ID"))
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	limit := 50
	offset := 0
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	msgs, err := h.svc.GetMessages(r.Context(), streamID, limit, offset)
	if err != nil {
		render.Error(w, r, fmt.Errorf("get messages: %w", err))
		return
	}
	render.JSON(w, http.StatusOK, msgs)
}
