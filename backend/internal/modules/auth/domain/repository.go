package domain

import "context"

type Repository interface {
	CreateUser(ctx context.Context, googleID, email, name, avatarURL, rawKey, keyHash string) (*User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByStreamKeyHash(ctx context.Context, hash string) (*User, error)
	UpdateStreamKey(ctx context.Context, userID, rawKey, keyHash string) error
	UpdateSettings(ctx context.Context, userID, title, category string) error
	SetLiveStatus(ctx context.Context, userID string, isLive bool) error
	GetLiveUsers(ctx context.Context) ([]User, error)
}
