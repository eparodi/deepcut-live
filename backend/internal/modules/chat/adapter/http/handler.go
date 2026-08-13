package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/deepcut/live/internal/modules/chat/application"
	"github.com/deepcut/live/internal/modules/chat/domain"
	"github.com/deepcut/live/internal/shared/errs"
	"github.com/deepcut/live/internal/shared/render"
)

// chatService is the subset of *application.ChatService methods that ChatHandler needs.
type chatService interface {
	GetHub() *application.ChatHub
	SendMessage(ctx context.Context, streamID, userID, userName, userAvatarUrl, message string) (*domain.ChatMessage, error)
	GetMessages(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error)
	IsStreamLive(ctx context.Context, streamID string) (bool, error)
}

// chatAuth validates tokens and returns user info for chat display.
// Implemented by an adapter over the auth service in main.go.
type chatAuth interface {
	ValidateToken(ctx context.Context, tokenStr string) (userID, userName, userAvatarUrl string, err error)
}

const (
	wsIdleTimeout       = 2 * time.Minute
	initialBatchSize    = 50
	defaultMessageLimit = 100
	maxMessageLimit     = 200
)

// wsStatusStreamOffline is the application-defined close code sent when a
// client connects to a stream that is not live (4000-4999 range is reserved
// for applications by RFC 6455).
const wsStatusStreamOffline = websocket.StatusCode(4001)

type ChatHandler struct {
	svc    chatService
	hub    *application.ChatHub
	auth   chatAuth
	logger *slog.Logger
}

func NewChatHandler(svc chatService, auth chatAuth, logger *slog.Logger) *ChatHandler {
	return &ChatHandler{
		svc:    svc,
		hub:    svc.GetHub(),
		auth:   auth,
		logger: logger,
	}
}

// RegisterRoutes registers all chat routes on the given router.
// Note: ChatWebSocket is NOT registered here — it's registered directly
// in main.go outside the auth group at /ws/chat/{streamID}.
func (h *ChatHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/chat/{streamID}/messages", h.GetMessages)
}

// ---------------------------------------------------------------------------
// WebSocket
// ---------------------------------------------------------------------------

// wsIncoming is the client→server JSON envelope.
type wsIncoming struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// wsMessagePayload is the payload for "message" type.
type wsMessagePayload struct {
	Message string `json:"message"`
}

// ChatWebSocket upgrades an HTTP connection to WebSocket for real-time chat.
// It is NOT behind the auth middleware — auth is optional (required to send).
func (h *ChatHandler) ChatWebSocket(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "streamID")
	if streamID == "" {
		render.Error(w, r, errs.BadRequest("missing stream ID"))
		return
	}

	// Accept the WebSocket connection FIRST so the browser can receive close codes.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:3000", "localhost:8081", "127.0.0.1:3000", "127.0.0.1:8081"},
	})
	if err != nil {
		h.logger.Error("websocket accept", "error", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Validate stream exists and is live (AFTER upgrade so client gets close code).
	subCtx, subCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer subCancel()
	isLive, err := h.svc.IsStreamLive(subCtx, streamID)
	if err != nil || !isLive {
		conn.Close(wsStatusStreamOffline, "stream offline")
		return
	}

	// Try to extract auth from cookie or Authorization header (optional).
	var userID, userName, userAvatarUrl string
	tokenStr := extractToken(r)
	if tokenStr != "" {
		if uid, name, avatar, err := h.auth.ValidateToken(subCtx, tokenStr); err == nil {
			userID = uid
			userName = name
			userAvatarUrl = avatar
		}
		// Invalid/expired token is not an error — proceed as anonymous.
	}

	// connCtx governs both pumps. It is cancelled when the handler returns,
	// when readPump exits (client disconnected), or remotely via client.Close
	// (idle expiry) — any of these unblocks writePump so all cleanup runs.
	connCtx, connCancel := context.WithCancel(context.Background())
	defer connCancel()

	client := &domain.ChatClient{
		UserID:        userID,
		UserName:      userName,
		UserAvatarUrl: userAvatarUrl,
		Send:          make(chan []byte, 64),
		LastActive:    time.Now(),
		Close:         connCancel,
	}

	h.hub.Join(streamID, client)
	defer h.hub.Leave(streamID, client)

	// Send initial batch of recent messages.
	h.sendInitialBatch(subCtx, conn, streamID)

	go h.readPump(connCtx, connCancel, conn, streamID, client)
	go h.idleMonitor(connCtx, streamID)

	h.writePump(connCtx, conn, client)
}

// extractToken pulls a JWT from the Authorization header or the "token" cookie.
func extractToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header != "" && len(header) > 7 && header[:7] == "Bearer " {
		return header[7:]
	}
	cookie, err := r.Cookie("token")
	if err == nil {
		return cookie.Value
	}
	return ""
}

// sendInitialBatch sends the last 50 messages to the newly connected client.
func (h *ChatHandler) sendInitialBatch(ctx context.Context, conn *websocket.Conn, streamID string) {
	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	msgs, _, err := h.svc.GetMessages(subCtx, streamID, "", initialBatchSize)
	if err != nil {
		h.logger.Warn("failed to load initial chat batch", "stream_id", streamID, "error", err)
		return
	}

	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		envelope := map[string]any{
			"type": "message",
			"payload": map[string]any{
				"id":            m.ID,
				"userId":        m.UserID,
				"userName":      m.UserName,
				"userAvatarUrl": m.UserAvatarUrl,
				"message":       m.Message,
				"sentAt":        m.SentAt.Format(time.RFC3339Nano),
			},
		}
		if err := wsjson.Write(subCtx, conn, envelope); err != nil {
			return
		}
	}
}

// readPump reads frames from the WebSocket connection. On exit it cancels
// the connection context so writePump (and the handler) unblock and clean up.
func (h *ChatHandler) readPump(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, streamID string, client *domain.ChatClient) {
	defer cancel()
	defer conn.Close(websocket.StatusNormalClosure, "")

	for {
		_, msgBytes, err := conn.Read(ctx)
		if err != nil {
			break
		}

		h.hub.Touch(streamID, client)

		var incoming wsIncoming
		if err := json.Unmarshal(msgBytes, &incoming); err != nil {
			h.sendToClient(client, "error", map[string]string{
				"code":    "invalid_message",
				"message": "Invalid message format",
			})
			continue
		}

		switch incoming.Type {
		case "message":
			h.handleChatMessage(ctx, streamID, client, incoming.Payload)
		case "ping":
			// Respond with pong (routed through writePump to avoid concurrent writes).
			h.sendToClient(client, "pong", nil)
		default:
			// Unknown message type — ignore silently.
		}
	}
}

// handleChatMessage processes an incoming chat message from the WebSocket.
func (h *ChatHandler) handleChatMessage(ctx context.Context, streamID string, client *domain.ChatClient, payload json.RawMessage) {
	if client.UserID == "" {
		h.sendToClient(client, "error", map[string]string{
			"code":    "unauthorized",
			"message": "Sign in to send messages",
		})
		return
	}

	var p wsMessagePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		h.sendToClient(client, "error", map[string]string{
			"code":    "invalid_message",
			"message": "Invalid message format",
		})
		return
	}

	// Validate message is not empty or whitespace-only.
	if strings.TrimSpace(p.Message) == "" {
		h.sendToClient(client, "error", map[string]string{
			"code":    "invalid_message",
			"message": "Message cannot be empty",
		})
		return
	}

	_, err := h.svc.SendMessage(ctx, streamID, client.UserID, client.UserName, client.UserAvatarUrl, p.Message)
	if err != nil {
		if errors.Is(err, application.ErrRateLimited) {
			h.sendToClient(client, "error", map[string]string{
				"code":    "rate_limited",
				"message": "Please wait before sending another message",
			})
		} else {
			h.logger.Error("send chat message", "error", err)
			h.sendToClient(client, "error", map[string]string{
				"code":    "internal_error",
				"message": "Failed to send message",
			})
		}
	}
}

// sendToClient marshals a message and sends it to the client via the writePump channel.
// This ensures all writes go through a single goroutine (writePump).
func (h *ChatHandler) sendToClient(client *domain.ChatClient, msgType string, payload any) {
	envelope := map[string]any{
		"type": msgType,
	}
	if payload != nil {
		envelope["payload"] = payload
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		h.logger.Error("marshal client message", "error", err)
		return
	}
	select {
	case client.Send <- data:
	default:
		// Client buffer full; drop the message rather than blocking readPump.
	}
}

// writePump sends messages from the client.Send channel to the WebSocket.
func (h *ChatHandler) writePump(ctx context.Context, conn *websocket.Conn, client *domain.ChatClient) {
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
}

// idleMonitor periodically expires idle clients from the room. ExpireIdle
// removes every expired client from the hub, so this monitor must close all
// of them (via their Close callback), not just its own connection — removed
// clients receive no further broadcasts and would otherwise leak.
func (h *ChatHandler) idleMonitor(ctx context.Context, streamID string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, c := range h.hub.ExpireIdle(streamID, wsIdleTimeout) {
				h.logger.Info("closing idle chat client", "stream_id", streamID, "user_id", c.UserID)
				if c.Close != nil {
					c.Close()
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// ---------------------------------------------------------------------------
// REST: GetMessages
// ---------------------------------------------------------------------------

// messagesResponse is the JSON shape for GET /api/chat/{streamID}/messages.
type messagesResponse struct {
	Messages []domain.ChatMessage `json:"messages"`
	HasMore  bool                 `json:"hasMore"`
}

// GetMessages returns chat message history for a stream (cursor-based pagination).
func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	streamID := chi.URLParam(r, "streamID")
	if streamID == "" {
		render.Error(w, r, errs.BadRequest("missing stream ID"))
		return
	}

	before := r.URL.Query().Get("before")

	limitStr := r.URL.Query().Get("limit")
	limit := defaultMessageLimit
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= maxMessageLimit {
			limit = v
		}
	}

	msgs, hasMore, err := h.svc.GetMessages(r.Context(), streamID, before, limit)
	if err != nil {
		h.logger.Error("get messages", "stream_id", streamID, "error", err)
		render.Error(w, r, fmt.Errorf("get messages: %w", err))
		return
	}

	// Ensure we never return null — always an empty array.
	if msgs == nil {
		msgs = []domain.ChatMessage{}
	}

	render.JSON(w, http.StatusOK, messagesResponse{
		Messages: msgs,
		HasMore:  hasMore,
	})
}
