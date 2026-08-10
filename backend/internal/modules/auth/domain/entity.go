package domain

import "time"

type User struct {
	ID             string
	GoogleID       string
	Email          string
	Name           string
	AvatarURL      *string
	StreamKey      string
	StreamKeyHash  string
	StreamTitle    *string
	StreamCategory *string
	IsLive         bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
