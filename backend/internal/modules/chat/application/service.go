package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/deepcut/live/internal/modules/chat/domain"
)

// Rate limit: 1 message per 2 seconds, burst of 3.
const (
	rateLimitInterval = 2 * time.Second
	rateLimitBurst    = 3
)

// userRateLimiter tracks per-user rate limiting state.
type userRateLimiter struct {
	tokens     float64
	lastRefill time.Time
}

// ChatHub manages WebSocket connections and broadcasts messages to clients in each stream's room.
type ChatHub struct {
	mu       sync.RWMutex
	rooms    map[string]map[*domain.ChatClient]bool // streamID -> set of clients
	limiters map[string]*userRateLimiter            // userID -> rate limiter
	repo     domain.ChatRepository
	logger   *slog.Logger
}

func NewChatHub(repo domain.ChatRepository, logger *slog.Logger) *ChatHub {
	return &ChatHub{
		rooms:    make(map[string]map[*domain.ChatClient]bool),
		limiters: make(map[string]*userRateLimiter),
		repo:     repo,
		logger:   logger,
	}
}

// Join adds a client to a stream's chat room.
func (h *ChatHub) Join(streamID string, client *domain.ChatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[streamID]; !ok {
		h.rooms[streamID] = make(map[*domain.ChatClient]bool)
	}
	client.LastActive = time.Now()
	h.rooms[streamID][client] = true
	h.logger.Info("chat client joined", "stream_id", streamID, "user_id", client.UserID)
}

// Leave removes a client from a stream's chat room.
func (h *ChatHub) Leave(streamID string, client *domain.ChatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.rooms[streamID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.rooms, streamID)
		}
	}
	h.logger.Info("chat client left", "stream_id", streamID, "user_id", client.UserID)
}

// Touch updates the client's last active timestamp.
func (h *ChatHub) Touch(streamID string, client *domain.ChatClient) {
	h.mu.Lock()
	client.LastActive = time.Now()
	h.mu.Unlock()
}

// ExpireIdle removes clients that have been idle for longer than idleTimeout.
func (h *ChatHub) ExpireIdle(streamID string, idleTimeout time.Duration) []*domain.ChatClient {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.rooms[streamID]
	if !ok {
		return nil
	}

	now := time.Now()
	var expired []*domain.ChatClient
	for client := range clients {
		if now.Sub(client.LastActive) > idleTimeout {
			expired = append(expired, client)
			delete(clients, client)
		}
	}

	if len(clients) == 0 {
		delete(h.rooms, streamID)
	}

	return expired
}

// AllowMessage checks rate limiting for a user. Returns true if allowed, false if rate limited.
func (h *ChatHub) AllowMessage(userID string) bool {
	if userID == "" {
		return false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	limiter, ok := h.limiters[userID]
	if !ok {
		limiter = &userRateLimiter{
			tokens:     rateLimitBurst,
			lastRefill: time.Now(),
		}
		h.limiters[userID] = limiter
	}

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(limiter.lastRefill)
	limiter.tokens += elapsed.Seconds() / rateLimitInterval.Seconds()
	if limiter.tokens > rateLimitBurst {
		limiter.tokens = rateLimitBurst
	}
	limiter.lastRefill = now

	if limiter.tokens >= 1 {
		limiter.tokens--
		return true
	}
	return false
}

// Broadcast sends a message to all clients in a stream's room using the spec envelope format.
func (h *ChatHub) Broadcast(streamID string, msg *domain.ChatMessage) {
	envelope := map[string]interface{}{
		"type": "message",
		"payload": map[string]interface{}{
			"id":            msg.ID,
			"userId":        msg.UserID,
			"userName":      msg.UserName,
			"userAvatarUrl": msg.UserAvatarUrl,
			"message":       msg.Message,
			"sentAt":        msg.SentAt.Format(time.RFC3339),
		},
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		h.logger.Error("marshal broadcast envelope", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[streamID]
	if !ok {
		return
	}
	for client := range clients {
		select {
		case client.Send <- data:
		default:
			// Client buffer full, skip
		}
	}
}

// SendToClient sends a JSON message to a specific client.
func (h *ChatHub) SendToClient(client *domain.ChatClient, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("marshal client message", "error", err)
		return
	}
	select {
	case client.Send <- data:
	default:
	}
}

// ClientCount returns the number of connected clients for a stream.
func (h *ChatHub) ClientCount(streamID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[streamID])
}

// ChatService coordinates chat operations.
type ChatService struct {
	hub  *ChatHub
	repo domain.ChatRepository
}

func NewChatService(repo domain.ChatRepository, hub *ChatHub) *ChatService {
	return &ChatService{
		hub:  hub,
		repo: repo,
	}
}

// GetHub returns the ChatHub for WebSocket handling.
func (s *ChatService) GetHub() *ChatHub {
	return s.hub
}

// SendMessage persists a chat message and broadcasts it to the stream's room.
func (s *ChatService) SendMessage(ctx context.Context, streamID, userID, userName, userAvatarUrl, message string) (*domain.ChatMessage, error) {
	// Check rate limit
	if !s.hub.AllowMessage(userID) {
		return nil, fmt.Errorf("rate limited")
	}

	msg, err := s.repo.SaveMessage(ctx, streamID, userID, message)
	if err != nil {
		return nil, fmt.Errorf("save message: %w", err)
	}
	msg.UserName = userName
	msg.UserAvatarUrl = userAvatarUrl
	s.hub.Broadcast(streamID, msg)
	return msg, nil
}

// GetMessages retrieves chat message history for a stream using cursor-based pagination.
func (s *ChatService) GetMessages(ctx context.Context, streamID string, before string, limit int) ([]domain.ChatMessage, bool, error) {
	msgs, hasMore, err := s.repo.GetMessages(ctx, streamID, before, limit)
	if err != nil {
		return nil, false, fmt.Errorf("get messages: %w", err)
	}
	return msgs, hasMore, nil
}

// IsStreamLive checks whether a stream exists and is currently live.
func (s *ChatService) IsStreamLive(ctx context.Context, streamID string) (bool, error) {
	return s.repo.GetStreamStatus(ctx, streamID)
}
