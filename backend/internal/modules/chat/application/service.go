package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/deepcut/live/internal/modules/chat/domain"
)

// ChatHub manages WebSocket connections and broadcasts messages to clients in each stream's room.
type ChatHub struct {
	mu     sync.RWMutex
	rooms  map[string]map[*domain.ChatClient]bool // streamID -> set of clients
	repo   domain.ChatRepository
	logger *slog.Logger
}

func NewChatHub(repo domain.ChatRepository, logger *slog.Logger) *ChatHub {
	return &ChatHub{
		rooms:  make(map[string]map[*domain.ChatClient]bool),
		repo:   repo,
		logger: logger,
	}
}

// Join adds a client to a stream's chat room.
func (h *ChatHub) Join(streamID string, client *domain.ChatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[streamID]; !ok {
		h.rooms[streamID] = make(map[*domain.ChatClient]bool)
	}
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

// Broadcast sends a message to all clients in a stream's room.
func (h *ChatHub) Broadcast(streamID string, msg *domain.ChatMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("marshal chat message", "error", err)
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
func (s *ChatService) SendMessage(ctx context.Context, streamID, userID, userName, message string) (*domain.ChatMessage, error) {
	msg, err := s.repo.SaveMessage(ctx, streamID, userID, message)
	if err != nil {
		return nil, fmt.Errorf("save message: %w", err)
	}
	msg.UserName = userName
	s.hub.Broadcast(streamID, msg)
	return msg, nil
}

// GetMessages retrieves chat message history for a stream.
func (s *ChatService) GetMessages(ctx context.Context, streamID string, limit, offset int) ([]domain.ChatMessage, error) {
	msgs, err := s.repo.GetMessages(ctx, streamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	return msgs, nil
}
