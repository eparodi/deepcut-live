package application

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/deepcut/live/internal/modules/streams/domain"
)

// StreamHub manages WebSocket connections for real-time stream-status notifications.
// One room per userID (the streamer). When a stream starts/stops, the SRS callback
// handler broadcasts to the streamer's room.
type StreamHub struct {
	mu     sync.RWMutex
	rooms  map[string]map[*domain.StreamStatusClient]bool // userID → clients
	logger *slog.Logger
}

func NewStreamHub(logger *slog.Logger) *StreamHub {
	return &StreamHub{
		rooms:  make(map[string]map[*domain.StreamStatusClient]bool),
		logger: logger,
	}
}

// Join adds a client to the user's room.
func (h *StreamHub) Join(userID string, client *domain.StreamStatusClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[userID]; !ok {
		h.rooms[userID] = make(map[*domain.StreamStatusClient]bool)
	}
	h.rooms[userID][client] = true
	h.logger.Info("stream-status client joined", "user_id", userID)
}

// Leave removes a client from the user's room.
func (h *StreamHub) Leave(userID string, client *domain.StreamStatusClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.rooms[userID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.rooms, userID)
		}
	}
	h.logger.Info("stream-status client left", "user_id", userID)
}

// NotifyStreamStarted broadcasts a stream-started event to all clients in the user's room.
func (h *StreamHub) NotifyStreamStarted(userID, streamID string) {
	h.broadcast(userID, domain.StreamStatusEvent{
		Type:     "streamStarted",
		StreamID: streamID,
	})
}

// NotifyStreamEnded broadcasts a stream-ended event to all clients in the user's room.
func (h *StreamHub) NotifyStreamEnded(userID string) {
	h.broadcast(userID, domain.StreamStatusEvent{
		Type: "streamEnded",
	})
}

func (h *StreamHub) broadcast(userID string, event domain.StreamStatusEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("marshal stream status event", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.rooms[userID] {
		select {
		case client.Send <- data:
		default:
			// Client buffer full, skip (client will poll as fallback)
		}
	}
}
