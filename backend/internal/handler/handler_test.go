package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/deepcut/live/internal/service"
	"github.com/golang-jwt/jwt/v5"
)

func TestGetMe(t *testing.T) {
	store, _, h := setupTestHandler(t)
	userID, token := createTestUser(t, h, store)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})

	rec := httptest.NewRecorder()
	h.GetMe(rec, req.WithContext(withAuth(req.Context(), userID)))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d. body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&body)
	for _, f := range []string{"id", "name", "email", "isLive"} {
		if _, ok := body[f]; !ok {
			t.Errorf("missing field %q in response", f)
		}
	}
	if body["id"] != userID {
		t.Errorf("got id %q, want %q", body["id"], userID)
	}
}

func TestRegenerateStreamKey(t *testing.T) {
	store, _, h := setupTestHandler(t)
	userID, token := createTestUser(t, h, store)

	req := httptest.NewRequest("POST", "/api/me/stream-key/regenerate", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})

	rec := httptest.NewRecorder()
	h.RegenerateStreamKey(rec, req.WithContext(withAuth(req.Context(), userID)))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	key := body["streamKey"]
	if !strings.HasPrefix(key, "sk-") {
		t.Errorf("stream key should start with 'sk-', got %q", key)
	}
	if len(key) < 10 {
		t.Errorf("stream key too short: %q", key)
	}
}

func TestUpdateSettings(t *testing.T) {
	store, _, h := setupTestHandler(t)
	userID, token := createTestUser(t, h, store)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "happy path",
			body:     `{"streamTitle":"Coding stream","streamCategory":"Programming"}`,
			wantCode: http.StatusOK,
		},
		{
			name:     "empty title — rejected",
			body:     `{"streamTitle":"","streamCategory":"Programming"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "title too long — rejected",
			body:     `{"streamTitle":"` + strings.Repeat("x", 101) + `","streamCategory":""}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PATCH", "/api/me/settings",
				strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "token", Value: token})

			rec := httptest.NewRecorder()
			h.UpdateSettings(rec, req.WithContext(withAuth(req.Context(), userID)))

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d. body: %s", rec.Code, tt.wantCode, rec.Body.String())
			}
		})
	}
}

func TestGoogleOAuth(t *testing.T) {
	_, _, h := setupTestHandler(t)

	t.Run("redirects to Google", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/google", nil)
		rec := httptest.NewRecorder()
		h.GoogleOAuth(rec, req)

		if rec.Code != http.StatusFound {
			t.Errorf("got %d, want %d", rec.Code, http.StatusFound)
		}
		location := rec.Header().Get("Location")
		if !strings.Contains(location, "accounts.google.com") {
			t.Errorf("redirect should go to Google, got %q", location)
		}
	})

	t.Run("callback with missing state", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/google/callback?code=test", nil)
		rec := httptest.NewRecorder()
		h.GoogleOAuthCallback(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("got %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("callback with missing code", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/google/callback?state=test", nil)
		req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "test"})
		rec := httptest.NewRecorder()
		h.GoogleOAuthCallback(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("got %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func TestAuthMiddleware(t *testing.T) {
	store, _, h := setupTestHandler(t)
	_, token := createTestUser(t, h, store)

	tests := []struct {
		name      string
		header    string
		useCookie bool
		wantCode  int
	}{
		{"valid token in cookie", "", true, http.StatusOK},
		{"valid token in Authorization header", "Bearer " + token, false, http.StatusOK},
		{"missing auth — rejected", "", false, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/protected", nil)
			if tt.useCookie {
				req.AddCookie(&http.Cookie{Name: "token", Value: token})
			} else if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			rec := httptest.NewRecorder()
			h.AuthMiddleware(nextHandler).ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

// --- Helpers ---

func setupTestHandler(t *testing.T) (*testStore, *service.UserService, *Handler) {
	t.Helper()
	store := newTestStore()
	userSvc := service.NewUserService(store)
	authSvc := service.NewAuthService("test-id", "test-secret", "http://localhost", "test-jwt")
	h := New(authSvc, userSvc, "test-jwt-secret-key-that-is-long-enough-for-hmac", "test-srs")
	return store, userSvc, h
}

func createTestUser(t *testing.T, h *Handler, store *testStore) (string, string) {
	t.Helper()
	user, _, err := h.userSvc.GetOrCreateUser(context.Background(),
		"google-123", "test@example.com", "Test User", "https://example.com/avatar.jpg")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	_ = store // available for direct store access in future tests

	userID := user.ID
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": user.Email,
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	})
	signed, _ := token.SignedString(h.jwtKey)
	return userID, signed
}

func withAuth(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, userID)
}
