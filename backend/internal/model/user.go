package model

import "time"

type User struct {
	ID             string     `json:"id"`
	GoogleID       string     `json:"-"`
	Email          string     `json:"email"`
	Name           string     `json:"name"`
	AvatarURL      *string    `json:"avatarUrl"`
	StreamKeyHash  string     `json:"-"`
	StreamTitle    *string    `json:"streamTitle"`
	StreamCategory *string    `json:"streamCategory"`
	IsLive         bool       `json:"isLive"`
	LiveSince      *time.Time `json:"liveSince,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}
