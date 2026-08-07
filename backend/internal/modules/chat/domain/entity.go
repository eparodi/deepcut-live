package domain

import "time"

// ChatMessage represents a single chat message.
type ChatMessage struct {
	ID       string    `json:"id"`
	StreamID string    `json:"streamId"`
	UserID   string    `json:"userId"`
	UserName string    `json:"userName"`
	Message  string    `json:"message"`
	SentAt   time.Time `json:"sentAt"`
}

// ChatClient represents a WebSocket-connected chat client.
type ChatClient struct {
	UserID   string
	UserName string
	Send     chan []byte
}
