package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/deepcut/live/internal/model"
)

// UserStore is the interface for user persistence operations.
// The postgres.Store satisfies this interface.
type UserStore interface {
	CreateUser(ctx context.Context, googleID, email, name, avatarURL, keyHash string) (*model.User, error)
	GetUserByGoogleID(ctx context.Context, googleID string) (*model.User, error)
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	GetUserByStreamKeyHash(ctx context.Context, keyHash string) (*model.User, error)
	UpdateStreamKeyHash(ctx context.Context, userID, keyHash string) error
	UpdateStreamSettings(ctx context.Context, userID, title, category string) error
}

type UserService struct {
	store UserStore
}

func NewUserService(store UserStore) *UserService {
	return &UserService{store: store}
}

func (s *UserService) GetOrCreateUser(ctx context.Context, googleID, email, name, avatarURL string) (*model.User, *string, error) {
	user, err := s.store.GetUserByGoogleID(ctx, googleID)
	if err != nil {
		return nil, nil, fmt.Errorf("lookup user: %w", err)
	}
	if user != nil {
		return user, nil, nil
	}

	key, err := generateStreamKey()
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}
	keyHash := HashStreamKey(key)

	user, err = s.store.CreateUser(ctx, googleID, email, name, avatarURL, keyHash)
	if err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}
	return user, &key, nil
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*model.User, error) {
	return s.store.GetUserByID(ctx, userID)
}

func (s *UserService) GetUserByStreamKey(ctx context.Context, streamKey string) (*model.User, error) {
	keyHash := HashStreamKey(streamKey)
	return s.store.GetUserByStreamKeyHash(ctx, keyHash)
}

func (s *UserService) RegenerateStreamKey(ctx context.Context, userID string) (string, error) {
	key, err := generateStreamKey()
	if err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	keyHash := HashStreamKey(key)
	if err := s.store.UpdateStreamKeyHash(ctx, userID, keyHash); err != nil {
		return "", fmt.Errorf("update key hash: %w", err)
	}
	return key, nil
}

func (s *UserService) UpdateSettings(ctx context.Context, userID, title, category string) error {
	return s.store.UpdateStreamSettings(ctx, userID, title, category)
}

func generateStreamKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("sk-%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// HashStreamKey produces a SHA256 hash for storage and lookup.
// We use SHA256 (not bcrypt) because SRS callbacks need O(1) lookup
// by hash, and the stream key is already a 128-bit random value.
func HashStreamKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
