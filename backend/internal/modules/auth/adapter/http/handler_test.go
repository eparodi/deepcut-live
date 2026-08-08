package http

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/deepcut/live/internal/modules/auth/application"
	"github.com/deepcut/live/internal/modules/auth/domain"
	streamsdomain "github.com/deepcut/live/internal/modules/streams/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

// ---------------------------------------------------------------------------
// mockAuthService implements authService for handler tests
// ---------------------------------------------------------------------------

type mockAuthService struct {
	generateOAuthURLFn     func(state string) string
	exchangeCodeForTokenFn func(ctx context.Context, code string) (*oauth2.Token, error)
	getGoogleUserFn        func(ctx context.Context, token *oauth2.Token) (*application.GoogleUserInfo, error)
	getByGoogleIDFn        func(ctx context.Context, googleID string) (*domain.User, error)
	createUserFn           func(ctx context.Context, googleID, email, name, avatarURL string) (*domain.User, error)
	generateJwtFn          func(userID string) (string, error)
	validateJwtFn          func(tokenStr string) (string, error)
	getByIDFn              func(ctx context.Context, id string) (*domain.User, error)
	regenerateStreamKeyFn  func(ctx context.Context, userID string) (string, error)
	updateSettingsFn       func(ctx context.Context, userID, title, category string) error
}

func (m *mockAuthService) GenerateOAuthURL(state string) string {
	if m.generateOAuthURLFn != nil {
		return m.generateOAuthURLFn(state)
	}
	return "https://accounts.google.com/o/oauth2/auth?state=" + state
}

func (m *mockAuthService) ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {
	if m.exchangeCodeForTokenFn != nil {
		return m.exchangeCodeForTokenFn(ctx, code)
	}
	return &oauth2.Token{AccessToken: "mock-access"}, nil
}

func (m *mockAuthService) GetGoogleUser(ctx context.Context, token *oauth2.Token) (*application.GoogleUserInfo, error) {
	if m.getGoogleUserFn != nil {
		return m.getGoogleUserFn(ctx, token)
	}
	return &application.GoogleUserInfo{ID: "g-123", Email: "test@example.com", Name: "Test"}, nil
}

func (m *mockAuthService) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	if m.getByGoogleIDFn != nil {
		return m.getByGoogleIDFn(ctx, googleID)
	}
	return nil, errs.NotFound("user not found")
}

func (m *mockAuthService) CreateUser(ctx context.Context, googleID, email, name, avatarURL string) (*domain.User, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, googleID, email, name, avatarURL)
	}
	return &domain.User{ID: "user-1", Email: email, Name: name}, nil
}

func (m *mockAuthService) GenerateJWT(userID string) (string, error) {
	if m.generateJwtFn != nil {
		return m.generateJwtFn(userID)
	}
	return "mock-jwt-token", nil
}

func (m *mockAuthService) ValidateJWT(tokenStr string) (string, error) {
	if m.validateJwtFn != nil {
		return m.validateJwtFn(tokenStr)
	}
	if tokenStr == "" {
		return "", errs.Unauthorized("invalid token")
	}
	return "user-1", nil
}

func (m *mockAuthService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &domain.User{ID: id, Email: "test@example.com", Name: "Test User"}, nil
}

func (m *mockAuthService) RegenerateStreamKey(ctx context.Context, userID string) (string, error) {
	if m.regenerateStreamKeyFn != nil {
		return m.regenerateStreamKeyFn(ctx, userID)
	}
	return "new-stream-key-abc123", nil
}

func (m *mockAuthService) UpdateSettings(ctx context.Context, userID, title, category string) error {
	if m.updateSettingsFn != nil {
		return m.updateSettingsFn(ctx, userID, title, category)
	}
	return nil
}

// ---------------------------------------------------------------------------
// mockStreamOps implements streamOps for handler tests
// ---------------------------------------------------------------------------

type mockStreamOps struct {
	getAnalyticsFn   func(ctx context.Context, userID, period string) (*streamsdomain.Analytics, error)
	forceEndStreamFn func(ctx context.Context, userID string) (string, error)
}

func (m *mockStreamOps) GetAnalytics(ctx context.Context, userID, period string) (*streamsdomain.Analytics, error) {
	if m.getAnalyticsFn != nil {
		return m.getAnalyticsFn(ctx, userID, period)
	}
	return &streamsdomain.Analytics{Period: period}, nil
}

func (m *mockStreamOps) ForceEndStream(ctx context.Context, userID string) (string, error) {
	if m.forceEndStreamFn != nil {
		return m.forceEndStreamFn(ctx, userID)
	}
	return "Stream ended", nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

// ---------------------------------------------------------------------------
// TestGoogleOAuth
// ---------------------------------------------------------------------------

func TestGoogleOAuth(t *testing.T) {
	authMock := &mockAuthService{}
	streamMock := &mockStreamOps{}
	h := NewAuthHandler(authMock, streamMock, "http://localhost:3000", testLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/auth/google", nil)
	rec := httptest.NewRecorder()
	h.GoogleOAuth(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("got %d, want %d", rec.Code, http.StatusTemporaryRedirect)
	}
	loc := rec.Header().Get("Location")
	if loc == "" {
		t.Error("expected Location header to be set")
	}
	// Verify oauth_state cookie is set
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "oauth_state" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected oauth_state cookie")
	}
}

// ---------------------------------------------------------------------------
// TestGoogleOAuthCallback
// ---------------------------------------------------------------------------

func TestGoogleOAuthCallback(t *testing.T) {
	tests := []struct {
		name       string
		cookieVal  string
		queryState string
		queryCode  string
		setupMock  func(*mockAuthService)
		wantCode   int
	}{
		{
			name:       "happy path — new user",
			cookieVal:  "abc123",
			queryState: "abc123",
			queryCode:  "auth-code",
			setupMock: func(m *mockAuthService) {
				m.getByGoogleIDFn = func(ctx context.Context, googleID string) (*domain.User, error) {
					return nil, errs.NotFound("not found") // triggers CreateUser
				}
			},
			wantCode: http.StatusTemporaryRedirect,
		},
		{
			name:       "happy path — existing user",
			cookieVal:  "abc123",
			queryState: "abc123",
			queryCode:  "auth-code",
			setupMock: func(m *mockAuthService) {
				m.getByGoogleIDFn = func(ctx context.Context, googleID string) (*domain.User, error) {
					return &domain.User{ID: "existing-user"}, nil
				}
			},
			wantCode: http.StatusTemporaryRedirect,
		},
		{
			name:       "missing oauth_state cookie",
			cookieVal:  "",
			queryState: "abc123",
			queryCode:  "auth-code",
			wantCode:   http.StatusBadRequest,
		},
		{
			name:       "state mismatch",
			cookieVal:  "abc123",
			queryState: "xyz789",
			queryCode:  "auth-code",
			wantCode:   http.StatusBadRequest,
		},
		{
			name:       "missing authorization code",
			cookieVal:  "abc123",
			queryState: "abc123",
			queryCode:  "",
			wantCode:   http.StatusBadRequest,
		},
		{
			name:       "exchange code error",
			cookieVal:  "abc123",
			queryState: "abc123",
			queryCode:  "bad-code",
			setupMock: func(m *mockAuthService) {
				m.exchangeCodeForTokenFn = func(ctx context.Context, code string) (*oauth2.Token, error) {
					return nil, errs.BadRequest("invalid code")
				}
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:       "exchange code error — plain error",
			cookieVal:  "abc123",
			queryState: "abc123",
			queryCode:  "plain-err",
			setupMock: func(m *mockAuthService) {
				m.exchangeCodeForTokenFn = func(ctx context.Context, code string) (*oauth2.Token, error) {
					return nil, fmt.Errorf("plain network error")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := &mockAuthService{}
			streamMock := &mockStreamOps{}
			if tt.setupMock != nil {
				tt.setupMock(authMock)
			}
			h := NewAuthHandler(authMock, streamMock, "http://localhost:3000", testLogger())

			u := "/api/auth/google/callback"
			if tt.queryState != "" || tt.queryCode != "" {
				u += "?state=" + tt.queryState + "&code=" + tt.queryCode
			}
			req := httptest.NewRequest(http.MethodGet, u, nil)
			if tt.cookieVal != "" {
				req.AddCookie(&http.Cookie{Name: "oauth_state", Value: tt.cookieVal})
			}
			rec := httptest.NewRecorder()
			h.GoogleOAuthCallback(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetMe
// ---------------------------------------------------------------------------

func TestGetMe(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupMock func(*mockAuthService)
		wantCode  int
	}{
		{
			name:     "happy path",
			userID:   "user-1",
			wantCode: http.StatusOK,
		},
		{
			name:     "not authenticated — empty user ID",
			userID:   "",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:   "service error",
			userID: "user-1",
			setupMock: func(m *mockAuthService) {
				m.getByIDFn = func(ctx context.Context, id string) (*domain.User, error) {
					return nil, errs.Internal("db down")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := &mockAuthService{}
			streamMock := &mockStreamOps{}
			if tt.setupMock != nil {
				tt.setupMock(authMock)
			}
			h := NewAuthHandler(authMock, streamMock, "", testLogger())

			req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
			ctx := context.Background()
			if tt.userID != "" {
				ctx = withUserID(ctx, tt.userID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()
			h.GetMe(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
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
		body      string
		setupMock func(*mockAuthService)
		wantCode  int
	}{
		{
			name:     "happy path — empty body (confirmation handled client-side)",
			userID:   "user-1",
			body:     "",
			wantCode: http.StatusOK,
		},
		{
			name:     "happy path — explicit confirm",
			userID:   "user-1",
			body:     `{"confirm":true}`,
			wantCode: http.StatusOK,
		},
		{
			name:     "not authenticated — empty user ID",
			userID:   "",
			body:     `{"confirm":true}`,
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid JSON",
			userID:   "user-1",
			body:     `{bad`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "confirm not true",
			userID:   "user-1",
			body:     `{"confirm":false}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: "user-1",
			body:   `{"confirm":true}`,
			setupMock: func(m *mockAuthService) {
				m.regenerateStreamKeyFn = func(ctx context.Context, userID string) (string, error) {
					return "", errs.Internal("regenerate failed")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := &mockAuthService{}
			streamMock := &mockStreamOps{}
			if tt.setupMock != nil {
				tt.setupMock(authMock)
			}
			h := NewAuthHandler(authMock, streamMock, "", testLogger())

			req := httptest.NewRequest(http.MethodPost, "/api/me/stream-key/regenerate",
				strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.Background()
			if tt.userID != "" {
				ctx = withUserID(ctx, tt.userID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()
			h.RegenerateStreamKey(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestUpdateSettings
// ---------------------------------------------------------------------------

func TestUpdateSettings(t *testing.T) {
	title := "New Title"
	category := "Gaming"

	tests := []struct {
		name      string
		userID    string
		body      string
		setupMock func(*mockAuthService)
		wantCode  int
	}{
		{
			name:     "happy path",
			userID:   "user-1",
			body:     `{"streamTitle":"New Title","streamCategory":"Gaming"}`,
			wantCode: http.StatusOK,
		},
		{
			name:     "not authenticated",
			userID:   "",
			body:     `{"streamTitle":"New Title"}`,
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid JSON",
			userID:   "user-1",
			body:     `{bad`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "unknown field triggers DisallowUnknownFields",
			userID:   "user-1",
			body:     `{"streamTitle":"OK","extraField":"nope"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: "user-1",
			body:   `{"streamTitle":"x","streamCategory":"y"}`,
			setupMock: func(m *mockAuthService) {
				m.updateSettingsFn = func(ctx context.Context, userID, title, category string) error {
					return errs.Internal("db error")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	_ = title
	_ = category

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := &mockAuthService{}
			streamMock := &mockStreamOps{}
			if tt.setupMock != nil {
				tt.setupMock(authMock)
			}
			h := NewAuthHandler(authMock, streamMock, "", testLogger())

			req := httptest.NewRequest(http.MethodPatch, "/api/me/settings",
				strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.Background()
			if tt.userID != "" {
				ctx = withUserID(ctx, tt.userID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()
			h.UpdateSettings(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetAnalytics
// ---------------------------------------------------------------------------

func TestGetAnalytics(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		period    string
		setupMock func(*mockStreamOps)
		wantCode  int
	}{
		{
			name:     "happy path — default period",
			userID:   "user-1",
			period:   "",
			wantCode: http.StatusOK,
		},
		{
			name:     "happy path — month period",
			userID:   "user-1",
			period:   "month",
			wantCode: http.StatusOK,
		},
		{
			name:     "not authenticated",
			userID:   "",
			period:   "week",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:   "service error",
			userID: "user-1",
			period: "week",
			setupMock: func(m *mockStreamOps) {
				m.getAnalyticsFn = func(ctx context.Context, userID, period string) (*streamsdomain.Analytics, error) {
					return nil, errs.Internal("analytics failed")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := &mockAuthService{}
			streamMock := &mockStreamOps{}
			if tt.setupMock != nil {
				tt.setupMock(streamMock)
			}
			h := NewAuthHandler(authMock, streamMock, "", testLogger())

			u := "/api/me/analytics"
			if tt.period != "" {
				u += "?period=" + tt.period
			}
			req := httptest.NewRequest(http.MethodGet, u, nil)
			ctx := context.Background()
			if tt.userID != "" {
				ctx = withUserID(ctx, tt.userID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()
			h.GetAnalytics(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestForceEndStream
// ---------------------------------------------------------------------------

func TestForceEndStream(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupMock func(*mockStreamOps)
		wantCode  int
	}{
		{
			name:     "happy path",
			userID:   "user-1",
			wantCode: http.StatusOK,
		},
		{
			name:     "not authenticated",
			userID:   "",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:   "service error",
			userID: "user-1",
			setupMock: func(m *mockStreamOps) {
				m.forceEndStreamFn = func(ctx context.Context, userID string) (string, error) {
					return "", errs.Internal("end failed")
				}
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := &mockAuthService{}
			streamMock := &mockStreamOps{}
			if tt.setupMock != nil {
				tt.setupMock(streamMock)
			}
			h := NewAuthHandler(authMock, streamMock, "", testLogger())

			req := httptest.NewRequest(http.MethodPost, "/api/me/stream/end", nil)
			ctx := context.Background()
			if tt.userID != "" {
				ctx = withUserID(ctx, tt.userID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()
			h.ForceEndStream(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthMiddleware
// ---------------------------------------------------------------------------

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		cookie    string
		setupMock func(*mockAuthService)
		wantCode  int
	}{
		{
			name:     "happy path — Bearer token in header",
			header:   "Bearer valid-token",
			wantCode: http.StatusOK,
		},
		{
			name:     "happy path — token in cookie as fallback",
			cookie:   "valid-token",
			wantCode: http.StatusOK,
		},
		{
			name:     "missing authentication",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:   "invalid token",
			header: "Bearer bad-token",
			setupMock: func(m *mockAuthService) {
				m.validateJwtFn = func(tokenStr string) (string, error) {
					return "", errs.Unauthorized("invalid token")
				}
			},
			wantCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := &mockAuthService{}
			streamMock := &mockStreamOps{}
			if tt.setupMock != nil {
				tt.setupMock(authMock)
			}
			h := NewAuthHandler(authMock, streamMock, "", testLogger())

			// Wrap a simple handler that checks context and returns 200
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userID := userIDFromCtx(r.Context())
				if userID == "" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "token", Value: tt.cookie})
			}
			rec := httptest.NewRecorder()
			h.AuthMiddleware(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// Ensure unused imports are fine — time is used in domain.User.CreatedAt tests
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ---------------------------------------------------------------------------
// TestGetMe_EdgeCases — Task 12: null fields, user not found

// ---------------------------------------------------------------------------
// TestGetMe_EdgeCases — additional coverage beyond TestGetMe
// ---------------------------------------------------------------------------

func TestGetMe_EdgeCases(t *testing.T) {
	name := "Test Streamer"
	avatar := "https://example.com/avatar.png"

	tests := []struct {
		name      string
		setupMock func(*mockAuthService)
		wantCode  int
	}{
		{
			name: "not found — user deleted",
			setupMock: func(m *mockAuthService) {
				m.getByIDFn = func(ctx context.Context, id string) (*domain.User, error) {
					return nil, errs.NotFound("user %s not found", id)
				}
			},
			wantCode: http.StatusNotFound,
		},
		{
			name: "null fields — optional fields are nil",
			setupMock: func(m *mockAuthService) {
				m.getByIDFn = func(ctx context.Context, id string) (*domain.User, error) {
					return &domain.User{
						ID:    "user-1",
						Email: "test@example.com",
						Name:  name,
						// AvatarURL, StreamTitle, StreamCategory left nil
					}, nil
				}
			},
			wantCode: http.StatusOK,
		},
	}

	_ = name
	_ = avatar

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := &mockAuthService{}
			streamMock := &mockStreamOps{}
			if tt.setupMock != nil {
				tt.setupMock(authMock)
			}
			h := NewAuthHandler(authMock, streamMock, "", testLogger())

			req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
			ctx := withUserID(context.Background(), "user-1")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()
			h.GetMe(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// TestGetAnalytics_EdgeCases validates period enum and defaults.
func TestGetAnalytics_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		period   string
		wantCode int
	}{
		{
			name:     "invalid period returns 400",
			period:   "day",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "all period returns 200",
			period:   "all",
			wantCode: http.StatusOK,
		},
		{
			name:     "no period param defaults to week",
			period:   "",
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := &mockAuthService{}
			streamMock := &mockStreamOps{}
			h := NewAuthHandler(authMock, streamMock, "", testLogger())

			u := "/api/me/analytics"
			if tt.period != "" {
				u += "?period=" + tt.period
			}
			req := httptest.NewRequest(http.MethodGet, u, nil)
			ctx := withUserID(context.Background(), "user-1")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()
			h.GetAnalytics(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// TestUpdateSettings_Validation validates input constraints (Task 16).
func TestUpdateSettings_Validation(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "missing title returns 400",
			body:     `{"streamCategory":"Gaming"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty title returns 400",
			body:     `{"streamTitle":""}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "whitespace-only title returns 400",
			body:     `{"streamTitle":"   "}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "title over 100 chars returns 400",
			body:     `{"streamTitle":"` + strings.Repeat("x", 101) + `"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "category over 100 chars returns 400",
			body:     `{"streamTitle":"OK","streamCategory":"` + strings.Repeat("x", 101) + `"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "only title (no category) returns 200",
			body:     `{"streamTitle":"Just Chatting"}`,
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := &mockAuthService{}
			streamMock := &mockStreamOps{}
			h := NewAuthHandler(authMock, streamMock, "", testLogger())

			req := httptest.NewRequest(http.MethodPatch, "/api/me/settings",
				strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := withUserID(context.Background(), "user-1")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()
			h.UpdateSettings(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// TestForceEndStream_Conflict validates 409 on no active stream (Task 17).
func TestForceEndStream_Conflict(t *testing.T) {
	streamMock := &mockStreamOps{
		forceEndStreamFn: func(ctx context.Context, userID string) (string, error) {
			return "", errs.Conflict("no active stream to end")
		},
	}
	h := NewAuthHandler(&mockAuthService{}, streamMock, "", testLogger())

	req := httptest.NewRequest(http.MethodPost, "/api/me/stream/end", nil)
	ctx := withUserID(context.Background(), "user-1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.ForceEndStream(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("got %d, want %d (409 Conflict)", rec.Code, http.StatusConflict)
	}
}
