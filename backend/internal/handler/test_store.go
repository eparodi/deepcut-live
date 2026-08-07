package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/deepcut/live/internal/model"
)

// testStore is an in-memory store for handler tests.
type testStore struct {
	mu    sync.RWMutex
	users map[string]*model.User
}

func newTestStore() *testStore {
	return &testStore{users: make(map[string]*model.User)}
}

func (s *testStore) CreateUser(ctx context.Context, googleID, email, name, avatarURL, keyHash string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	u := &model.User{
		ID:            fmt.Sprintf("user-%d", len(s.users)+1),
		GoogleID:      googleID,
		Email:         email,
		Name:          name,
		StreamKeyHash: keyHash,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if avatarURL != "" {
		u.AvatarURL = &avatarURL
	}
	s.users[u.ID] = u
	return u, nil
}

func (s *testStore) GetUserByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.GoogleID == googleID {
			return u, nil
		}
	}
	return nil, nil
}

func (s *testStore) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (s *testStore) GetUserByStreamKeyHash(ctx context.Context, keyHash string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.StreamKeyHash == keyHash {
			return u, nil
		}
	}
	return nil, nil
}

func (s *testStore) UpdateStreamKeyHash(ctx context.Context, userID, keyHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}
	u.StreamKeyHash = keyHash
	u.UpdatedAt = time.Now()
	return nil
}

func (s *testStore) UpdateStreamSettings(ctx context.Context, userID, title, category string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}
	u.StreamTitle = &title
	u.StreamCategory = &category
	u.UpdatedAt = time.Now()
	return nil
}
