package domain

import "time"

// ChatMessage represents a single chat message.
type ChatMessage struct {
	ID            string    `json:"id"`
	StreamID      string    `json:"streamId"`
	UserID        string    `json:"userId"`
	UserName      string    `json:"userName"`
	UserAvatarUrl string    `json:"userAvatarUrl"`
	Message       string    `json:"message"`
	SentAt        time.Time `json:"sentAt"`
}

// ChatClient represents a WebSocket-connected chat client.
type ChatClient struct {
	UserID        string
	UserName      string
	UserAvatarUrl string
	Send          chan []byte
	LastActive    time.Time

	// Close terminates the client's connection. Set by the transport layer
	// (WebSocket handler) at connect time; called to disconnect the client
	// remotely, e.g. when the hub expires it for idleness. Must be safe to
	// call multiple times.
	Close func()
}
