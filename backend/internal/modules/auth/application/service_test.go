package application

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/deepcut/live/internal/modules/auth/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

// ---------------------------------------------------------------------------
// mockAuthRepo implements domain.Repository for service tests
// ---------------------------------------------------------------------------

type mockAuthRepo struct {
	createUserFn         func(ctx context.Context, googleID, email, name, avatarURL, keyHash string) (*domain.User, error)
	getByGoogleIDFn      func(ctx context.Context, googleID string) (*domain.User, error)
	getByIDFn            func(ctx context.Context, id string) (*domain.User, error)
	getByStreamKeyHashFn func(ctx context.Context, hash string) (*domain.User, error)
	updateStreamKeyFn    func(ctx context.Context, userID, keyHash string) error
	updateSettingsFn     func(ctx context.Context, userID, title, category string) error
	setLiveStatusFn      func(ctx context.Context, userID string, isLive bool) error
	getLiveUsersFn       func(ctx context.Context) ([]domain.User, error)
}

func (m *mockAuthRepo) CreateUser(ctx context.Context, googleID, email, name, avatarURL, keyHash string) (*domain.User, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, googleID, email, name, avatarURL, keyHash)
	}
	return &domain.User{
		ID:        "user-1",
		GoogleID:  googleID,
		Email:     email,
		Name:      name,
		AvatarURL: &avatarURL,
	}, nil
}

func (m *mockAuthRepo) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	if m.getByGoogleIDFn != nil {
		return m.getByGoogleIDFn(ctx, googleID)
	}
	return nil, errs.NotFound("user with google_id %s not found", googleID)
}

func (m *mockAuthRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &domain.User{ID: id, Email: "test@example.com", Name: "Test User"}, nil
}

func (m *mockAuthRepo) GetByStreamKeyHash(ctx context.Context, hash string) (*domain.User, error) {
	if m.getByStreamKeyHashFn != nil {
		return m.getByStreamKeyHashFn(ctx, hash)
	}
	return nil, errs.NotFound("user with stream key hash not found")
}

func (m *mockAuthRepo) UpdateStreamKey(ctx context.Context, userID, keyHash string) error {
	if m.updateStreamKeyFn != nil {
		return m.updateStreamKeyFn(ctx, userID, keyHash)
	}
	return nil
}

func (m *mockAuthRepo) UpdateSettings(ctx context.Context, userID, title, category string) error {
	if m.updateSettingsFn != nil {
		return m.updateSettingsFn(ctx, userID, title, category)
	}
	return nil
}

func (m *mockAuthRepo) SetLiveStatus(ctx context.Context, userID string, isLive bool) error {
	if m.setLiveStatusFn != nil {
		return m.setLiveStatusFn(ctx, userID, isLive)
	}
	return nil
}

func (m *mockAuthRepo) GetLiveUsers(ctx context.Context) ([]domain.User, error) {
	if m.getLiveUsersFn != nil {
		return m.getLiveUsersFn(ctx)
	}
	return []domain.User{}, nil
}

// generateTestKeys creates a fresh ECDSA P-256 key pair encoded as PEM for each test.
func generateTestKeys() (privPEM, pubPEM string) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		panic(err)
	}
	privBlock := &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}
	privPEM = string(pem.EncodeToMemory(privBlock))

	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		panic(err)
	}
	pubBlock := &pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}
	pubPEM = string(pem.EncodeToMemory(pubBlock))
	return
}

// newTestAuthService creates an AuthService wired to a mock repo for tests.
func newTestAuthService(repo *mockAuthRepo) *AuthService {
	privPEM, pubPEM := generateTestKeys()
	return NewAuthService(repo, "test-client-id", "test-client-secret", "http://localhost:8080", privPEM, pubPEM)
}

// ---------------------------------------------------------------------------
// TestCreateUser
// ---------------------------------------------------------------------------

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name      string
		googleID  string
		email     string
		userName  string
		avatarURL string
		setupMock func(*mockAuthRepo)
		wantErr   bool
	}{
		{
			name:      "happy path — creates user with stream key",
			googleID:  "g-123",
			email:     "alice@example.com",
			userName:  "Alice",
			avatarURL: "https://pic.example.com/alice.jpg",
		},
		{
			name:     "repo error — create returns error",
			googleID: "g-123",
			email:    "alice@example.com",
			userName: "Alice",
			setupMock: func(m *mockAuthRepo) {
				m.createUserFn = func(ctx context.Context, googleID, email, name, avatarURL, keyHash string) (*domain.User, error) {
					return nil, errors.New("db down")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAuthRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			svc := newTestAuthService(repo)

			user, err := svc.CreateUser(context.Background(), tt.googleID, tt.email, tt.userName, tt.avatarURL)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && user == nil {
				t.Fatal("expected user, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetByGoogleID
// ---------------------------------------------------------------------------

func TestGetByGoogleID(t *testing.T) {
	tests := []struct {
		name      string
		googleID  string
		setupMock func(*mockAuthRepo)
		wantErr   bool
	}{
		{
			name:     "happy path — user found",
			googleID: "g-123",
			setupMock: func(m *mockAuthRepo) {
				m.getByGoogleIDFn = func(ctx context.Context, googleID string) (*domain.User, error) {
					return &domain.User{ID: "user-1", GoogleID: googleID, Email: "a@b.com"}, nil
				}
			},
		},
		{
			name:     "error — not found",
			googleID: "g-unknown",
			wantErr:  true,
		},
		{
			name:     "error — repo error",
			googleID: "g-123",
			setupMock: func(m *mockAuthRepo) {
				m.getByGoogleIDFn = func(ctx context.Context, googleID string) (*domain.User, error) {
					return nil, errors.New("db connection lost")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAuthRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			svc := newTestAuthService(repo)

			user, err := svc.GetByGoogleID(context.Background(), tt.googleID)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && user == nil {
				t.Fatal("expected user, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetByID
// ---------------------------------------------------------------------------

func TestGetByID(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupMock func(*mockAuthRepo)
		wantErr   bool
	}{
		{
			name:   "happy path — user found",
			userID: "user-1",
		},
		{
			name:    "error — not found",
			userID:  "user-999",
			wantErr: true,
			setupMock: func(m *mockAuthRepo) {
				m.getByIDFn = func(ctx context.Context, id string) (*domain.User, error) {
					return nil, errs.NotFound("user %s not found", id)
				}
			},
		},
		{
			name:   "error — repo error",
			userID: "user-1",
			setupMock: func(m *mockAuthRepo) {
				m.getByIDFn = func(ctx context.Context, id string) (*domain.User, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAuthRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			svc := newTestAuthService(repo)

			user, err := svc.GetByID(context.Background(), tt.userID)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && user == nil {
				t.Fatal("expected user, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetByStreamKeyHash
// ---------------------------------------------------------------------------

func TestGetByStreamKeyHash(t *testing.T) {
	tests := []struct {
		name      string
		hash      string
		setupMock func(*mockAuthRepo)
		wantErr   bool
	}{
		{
			name: "happy path — user found by hash",
			hash: "abc123hash",
			setupMock: func(m *mockAuthRepo) {
				m.getByStreamKeyHashFn = func(ctx context.Context, hash string) (*domain.User, error) {
					return &domain.User{ID: "user-1", StreamKeyHash: hash}, nil
				}
			},
		},
		{
			name:    "error — not found",
			hash:    "badhash",
			wantErr: true,
		},
		{
			name: "error — repo error",
			hash: "abc123hash",
			setupMock: func(m *mockAuthRepo) {
				m.getByStreamKeyHashFn = func(ctx context.Context, hash string) (*domain.User, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAuthRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			svc := newTestAuthService(repo)

			user, err := svc.GetByStreamKeyHash(context.Background(), tt.hash)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && user == nil {
				t.Fatal("expected user, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRegenerateStreamKey
// ---------------------------------------------------------------------------

func TestRegenerateStreamKey(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupMock func(*mockAuthRepo)
		wantErr   bool
	}{
		{
			name:   "happy path — generates new key and updates",
			userID: "user-1",
		},
		{
			name:   "error — update fails",
			userID: "user-1",
			setupMock: func(m *mockAuthRepo) {
				m.updateStreamKeyFn = func(ctx context.Context, userID, keyHash string) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAuthRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			svc := newTestAuthService(repo)

			raw, err := svc.RegenerateStreamKey(context.Background(), tt.userID)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && raw == "" {
				t.Fatal("expected non-empty stream key")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestUpdateSettings
// ---------------------------------------------------------------------------

func TestUpdateSettings(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		title     string
		category  string
		setupMock func(*mockAuthRepo)
		wantErr   bool
	}{
		{
			name:     "happy path — updates title and category",
			userID:   "user-1",
			title:    "My Stream",
			category: "Gaming",
		},
		{
			name:     "happy path — empty title and category",
			userID:   "user-1",
			title:    "",
			category: "",
		},
		{
			name:     "error — repo error",
			userID:   "user-1",
			title:    "My Stream",
			category: "Gaming",
			setupMock: func(m *mockAuthRepo) {
				m.updateSettingsFn = func(ctx context.Context, userID, title, category string) error {
					return errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAuthRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			svc := newTestAuthService(repo)

			err := svc.UpdateSettings(context.Background(), tt.userID, tt.title, tt.category)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSetLiveStatus
// ---------------------------------------------------------------------------

func TestSetLiveStatus(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		isLive    bool
		setupMock func(*mockAuthRepo)
		wantErr   bool
	}{
		{
			name:   "happy path — set live",
			userID: "user-1",
			isLive: true,
		},
		{
			name:   "happy path — set offline",
			userID: "user-1",
			isLive: false,
		},
		{
			name:   "error — repo error",
			userID: "user-1",
			isLive: true,
			setupMock: func(m *mockAuthRepo) {
				m.setLiveStatusFn = func(ctx context.Context, userID string, isLive bool) error {
					return errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAuthRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			svc := newTestAuthService(repo)

			err := svc.SetLiveStatus(context.Background(), tt.userID, tt.isLive)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetLiveUsers
// ---------------------------------------------------------------------------

func TestGetLiveUsers(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mockAuthRepo)
		wantCount int
		wantErr   bool
	}{
		{
			name:      "happy path — returns live users",
			wantCount: 2,
			setupMock: func(m *mockAuthRepo) {
				m.getLiveUsersFn = func(ctx context.Context) ([]domain.User, error) {
					return []domain.User{
						{ID: "user-1", Name: "Alice", IsLive: true},
						{ID: "user-2", Name: "Bob", IsLive: true},
					}, nil
				}
			},
		},
		{
			name:      "happy path — empty list (no live users)",
			wantCount: 0,
			setupMock: func(m *mockAuthRepo) {
				m.getLiveUsersFn = func(ctx context.Context) ([]domain.User, error) {
					return []domain.User{}, nil
				}
			},
		},
		{
			name: "error — repo error",
			setupMock: func(m *mockAuthRepo) {
				m.getLiveUsersFn = func(ctx context.Context) ([]domain.User, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAuthRepo{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			svc := newTestAuthService(repo)

			users, err := svc.GetLiveUsers(context.Background())
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(users) != tt.wantCount {
				t.Fatalf("got %d users, want %d", len(users), tt.wantCount)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGenerateJWT
// ---------------------------------------------------------------------------

func TestGenerateJWT(t *testing.T) {
	svc := newTestAuthService(&mockAuthRepo{})

	token, err := svc.GenerateJWT("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Verify the token can be parsed back.
	userID, err := svc.ValidateJWT(token)
	if err != nil {
		t.Fatalf("failed to validate generated JWT: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("got userID %q, want %q", userID, "user-1")
	}
}

// ---------------------------------------------------------------------------
// TestValidateJWT
// ---------------------------------------------------------------------------

func TestValidateJWT(t *testing.T) {
	svc := newTestAuthService(&mockAuthRepo{})

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "happy path — valid token",
			wantErr: false,
		},
		{
			name:    "error — empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "error — malformed token",
			token:   "not.a.valid.jwt",
			wantErr: true,
		},
		{
			name:    "error — wrong signing method",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIiLCJuYW1lIjoiIn0.abc123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.token
			if token == "" && !tt.wantErr {
				// Generate a valid token for the happy path.
				var err error
				token, err = svc.GenerateJWT("user-1")
				if err != nil {
					t.Fatalf("failed to generate token: %v", err)
				}
			}

			userID, err := svc.ValidateJWT(token)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && userID == "" {
				t.Fatal("expected non-empty user ID")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGenerateStreamKey
// ---------------------------------------------------------------------------

func TestGenerateStreamKey(t *testing.T) {
	svc := newTestAuthService(&mockAuthRepo{})

	raw, hash, err := svc.GenerateStreamKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw == "" {
		t.Fatal("expected non-empty raw key")
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	// Raw key should be 64 hex chars (32 bytes).
	if len(raw) != 64 {
		t.Fatalf("raw key has %d chars, want 64", len(raw))
	}
	// Hash should be 64 hex chars (SHA-256).
	if len(hash) != 64 {
		t.Fatalf("hash has %d chars, want 64", len(hash))
	}
}

// ---------------------------------------------------------------------------
// newTestAuthServiceWithTokenURL creates an AuthService with a custom
// token endpoint for testing ExchangeCodeForToken.
// ---------------------------------------------------------------------------

func newTestAuthServiceWithTokenURL(repo *mockAuthRepo, tokenURL string) *AuthService {
	privPEM, pubPEM := generateTestKeys()
	svc := NewAuthService(repo, "test-client-id", "test-client-secret", "http://localhost:8080", privPEM, pubPEM)
	svc.googleCfg.Endpoint = oauth2.Endpoint{
		AuthURL:  svc.googleCfg.Endpoint.AuthURL,
		TokenURL: tokenURL,
	}
	return svc
}

// ---------------------------------------------------------------------------
// TestGenerateOAuthURL
// ---------------------------------------------------------------------------

func TestGenerateOAuthURL(t *testing.T) {
	svc := newTestAuthService(&mockAuthRepo{})

	tests := []struct {
		name  string
		state string
	}{
		{
			name:  "happy path — returns URL with state",
			state: "random-state-123",
		},
		{
			name:  "empty state",
			state: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.GenerateOAuthURL(tt.state)

			if !strings.Contains(got, "https://accounts.google.com/o/oauth2/auth") {
				t.Fatalf("URL missing accounts.google.com base: %s", got)
			}

			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("failed to parse generated URL: %v", err)
			}

			q := u.Query()
			if q.Get("client_id") != "test-client-id" {
				t.Fatalf("client_id = %q, want %q", q.Get("client_id"), "test-client-id")
			}
			if q.Get("response_type") != "code" {
				t.Fatalf("response_type = %q, want %q", q.Get("response_type"), "code")
			}
			if q.Get("redirect_uri") != "http://localhost:8080/api/auth/google/callback" {
				t.Fatalf("redirect_uri = %q, want %q", q.Get("redirect_uri"), "http://localhost:8080/api/auth/google/callback")
			}
			if q.Get("state") != tt.state {
				t.Fatalf("state = %q, want %q", q.Get("state"), tt.state)
			}
			scope := q.Get("scope")
			if !strings.Contains(scope, "userinfo.email") {
				t.Fatalf("scope missing userinfo.email: %s", scope)
			}
			if !strings.Contains(scope, "userinfo.profile") {
				t.Fatalf("scope missing userinfo.profile: %s", scope)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestExchangeCodeForToken
// ---------------------------------------------------------------------------

func TestExchangeCodeForToken(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		wantErr     bool
		errContains string
	}{
		{
			name: "happy path — returns token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{
					"access_token": "test-access-token",
					"token_type":   "Bearer",
					"expires_in":   3600,
				})
			},
		},
		{
			name: "error path — server returns 400",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "invalid_grant",
				})
			},
			wantErr:     true,
			errContains: "exchange code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			svc := newTestAuthServiceWithTokenURL(&mockAuthRepo{}, srv.URL)

			token, err := svc.ExchangeCodeForToken(context.Background(), "test-code")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.AccessToken != "test-access-token" {
				t.Fatalf("AccessToken = %q, want %q", token.AccessToken, "test-access-token")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetGoogleUser
// ---------------------------------------------------------------------------

func TestGetGoogleUser(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		wantErr     bool
		errContains string
		wantUser    *GoogleUserInfo
	}{
		{
			name: "happy path — returns user info",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if ah := r.Header.Get("Authorization"); ah != "Bearer test-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(GoogleUserInfo{
					ID:      "g-123",
					Email:   "alice@example.com",
					Name:    "Alice",
					Picture: "https://pic.example.com/alice.jpg",
				})
			},
			wantUser: &GoogleUserInfo{
				ID:      "g-123",
				Email:   "alice@example.com",
				Name:    "Alice",
				Picture: "https://pic.example.com/alice.jpg",
			},
		},
		{
			name: "error path — server returns 500",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:     true,
			errContains: "google userinfo returned 500",
		},
		{
			name: "error path — invalid JSON body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte("not-json"))
			},
			wantErr:     true,
			errContains: "parse google userinfo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			svc := newTestAuthService(&mockAuthRepo{})
			svc.userInfoURL = srv.URL

			token := &oauth2.Token{AccessToken: "test-token", TokenType: "Bearer"}
			user, err := svc.GetGoogleUser(context.Background(), token)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if user.ID != tt.wantUser.ID {
				t.Fatalf("ID = %q, want %q", user.ID, tt.wantUser.ID)
			}
			if user.Email != tt.wantUser.Email {
				t.Fatalf("Email = %q, want %q", user.Email, tt.wantUser.Email)
			}
			if user.Name != tt.wantUser.Name {
				t.Fatalf("Name = %q, want %q", user.Name, tt.wantUser.Name)
			}
			if user.Picture != tt.wantUser.Picture {
				t.Fatalf("Picture = %q, want %q", user.Picture, tt.wantUser.Picture)
			}
		})
	}
}
