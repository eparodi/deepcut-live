package main_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	authhttp "github.com/deepcut/live/internal/modules/auth/adapter/http"
	authpg "github.com/deepcut/live/internal/modules/auth/adapter/postgres"
	"github.com/deepcut/live/internal/modules/auth/application"
	chathttp "github.com/deepcut/live/internal/modules/chat/adapter/http"
	chatpg "github.com/deepcut/live/internal/modules/chat/adapter/postgres"
	chatapp "github.com/deepcut/live/internal/modules/chat/application"
	streamhttp "github.com/deepcut/live/internal/modules/streams/adapter/http"
	streampg "github.com/deepcut/live/internal/modules/streams/adapter/postgres"
	streamapp "github.com/deepcut/live/internal/modules/streams/application"
	vodhttp "github.com/deepcut/live/internal/modules/vods/adapter/http"
	vodpg "github.com/deepcut/live/internal/modules/vods/adapter/postgres"
	vodapp "github.com/deepcut/live/internal/modules/vods/application"
	"github.com/deepcut/live/internal/shared/errs"
	"github.com/deepcut/live/internal/testutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	pool, cleanup, err := testutil.SetupDB(ctx)
	if err != nil {
		log.Fatalf("setup test db: %v", err)
	}
	testPool = pool
	defer cleanup()

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// setupTestServer wires up a full chi router with real postgres repos,
// services, handlers, and middleware — matching main.go but with a
// discard logger and no Timeout middleware for deterministic tests.
// Returns an httptest.Server and a cleanup function.
// ---------------------------------------------------------------------------

func setupTestServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	pool := testPool

	// --- repos ---
	authRepo := authpg.NewAuthRepo(pool)
	streamRepo := streampg.NewStreamRepo(pool)
	vodRepo := vodpg.NewVODRepo(pool)
	chatRepo := chatpg.NewChatRepo(pool)

	// --- services ---
	privPEM, pubPEM := generateTestKeys()
	baseURL := "http://localhost:3000"
	srsSecret := "test-srs-secret"

	authSvc := application.NewAuthService(
		authRepo, "test-client-id", "test-client-secret",
		baseURL, privPEM, pubPEM,
	)
	streamSvc := streamapp.NewStreamService(streamRepo, authRepo, nil, nil, srsSecret, "http://127.0.0.1:1985", nil)
	vodSvc := vodapp.NewVODService(vodRepo)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	chatHub := chatapp.NewChatHub(chatRepo, logger)
	chatSvc := chatapp.NewChatService(chatRepo, chatHub)

	// --- handlers ---
	authHandler := authhttp.NewAuthHandler(authSvc, streamSvc, baseURL, logger)
	streamHandler := streamhttp.NewStreamHandler(streamSvc, nil, logger)
	vodHandler := vodhttp.NewVODHandler(vodSvc, logger)
	chatHandler := chathttp.NewChatHandler(chatSvc, newChatAuthAdapter(authSvc), logger)

	// --- router (same middleware as main.go except Logger → skipped, Timeout → skipped) ---
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	authHandler.RegisterRoutes(r)
	streamHandler.RegisterRoutes(r)
	vodHandler.RegisterRoutes(r)
	chatHandler.RegisterRoutes(r)

	// Chat WebSocket (outside auth group — auth handled internally).
	r.Get("/ws/chat/{streamID}", chatHandler.ChatWebSocket)

	srv := httptest.NewServer(r)

	cleanup := func() {
		srv.Close()
		if err := testutil.TruncateAll(context.Background(), testPool); err != nil {
			t.Logf("truncate after test: %v", err)
		}
	}

	return srv, cleanup
}

// ---------------------------------------------------------------------------
// TestHealthEndpoint
// ---------------------------------------------------------------------------

func TestHealthEndpoint(t *testing.T) {
	testutil.SkipOnShort(t)
	srv, cleanup := setupTestServer(t)
	t.Cleanup(cleanup)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "GET /health returns 200 ok",
			method:     http.MethodGet,
			path:       "/health",
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status: got %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}

			got := strings.TrimSpace(string(body))
			if got != tt.wantBody {
				t.Errorf("body: got %q, want %q", got, tt.wantBody)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestListLiveStreams
// ---------------------------------------------------------------------------

func TestListLiveStreams(t *testing.T) {
	testutil.SkipOnShort(t)
	srv, cleanup := setupTestServer(t)
	t.Cleanup(cleanup)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "GET /api/streams/live returns wrapped response with empty streams",
			path:       "/api/streams/live",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status: got %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			// Verify Content-Type is JSON
			ct := resp.Header.Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				t.Errorf("Content-Type: got %q, want application/json", ct)
			}

			// Verify body is valid JSON and has the expected wrapper shape.
			var body struct {
				Streams []any `json:"streams"`
				Total   int   `json:"total"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode JSON: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetChannelInfo_NotFound
// ---------------------------------------------------------------------------

func TestGetChannelInfo_NotFound(t *testing.T) {
	testutil.SkipOnShort(t)
	srv, cleanup := setupTestServer(t)
	t.Cleanup(cleanup)

	tests := []struct {
		name       string
		userID     string
		wantStatus int
	}{
		{
			name:       "nonexistent user returns 404",
			userID:     "00000000-0000-0000-0000-000000000000",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/channel/"+tt.userID, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status: got %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			// Verify error response has not_found kind
			var appErr errs.AppError
			if err := json.NewDecoder(resp.Body).Decode(&appErr); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if appErr.Kind != errs.KindNotFound {
				t.Errorf("error kind: got %q, want %q", appErr.Kind, errs.KindNotFound)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSearchVODs
// ---------------------------------------------------------------------------

func TestSearchVODs(t *testing.T) {
	testutil.SkipOnShort(t)
	srv, cleanup := setupTestServer(t)
	t.Cleanup(cleanup)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "GET /api/vods returns search result JSON",
			path:       "/api/vods",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status: got %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			// Verify response is a search result object with vods array and pagination
			var result struct {
				VODs       []any `json:"vods"`
				TotalCount int   `json:"totalCount"`
				Limit      int   `json:"limit"`
				Offset     int   `json:"offset"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("decode search result: %v", err)
			}
			if result.VODs == nil && result.TotalCount > 0 {
				t.Error("expected vods array when totalCount > 0, got null")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetVOD_NotFound
// ---------------------------------------------------------------------------

func TestGetVOD_NotFound(t *testing.T) {
	testutil.SkipOnShort(t)
	srv, cleanup := setupTestServer(t)
	t.Cleanup(cleanup)

	tests := []struct {
		name       string
		vodID      string
		wantStatus int
	}{
		{
			name:       "nonexistent VOD returns 404",
			vodID:      "00000000-0000-0000-0000-000000000000",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/vods/"+tt.vodID, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status: got %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			var appErr errs.AppError
			if err := json.NewDecoder(resp.Body).Decode(&appErr); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if appErr.Kind != errs.KindNotFound {
				t.Errorf("error kind: got %q, want %q", appErr.Kind, errs.KindNotFound)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestCORSHeaders
// ---------------------------------------------------------------------------

func TestCORSHeaders(t *testing.T) {
	testutil.SkipOnShort(t)
	srv, cleanup := setupTestServer(t)
	t.Cleanup(cleanup)

	tests := []struct {
		name            string
		method          string
		path            string
		origin          string
		wantStatus      int
		wantAllowOrigin string
	}{
		{
			name:            "OPTIONS /api/streams/live returns CORS headers",
			method:          http.MethodOptions,
			path:            "/api/streams/live",
			origin:          "http://localhost:3000",
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "http://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Access-Control-Request-Method", "GET")
			req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")

			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status: got %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
			if allowOrigin != tt.wantAllowOrigin {
				t.Errorf("Access-Control-Allow-Origin: got %q, want %q", allowOrigin, tt.wantAllowOrigin)
			}

			// Verify other critical CORS headers exist
			if resp.Header.Get("Access-Control-Allow-Methods") == "" {
				t.Error("Access-Control-Allow-Methods header missing")
			}
			if resp.Header.Get("Access-Control-Allow-Headers") == "" {
				t.Error("Access-Control-Allow-Headers header missing")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthFlow — simulate OAuth redirect + unauthenticated /api/me
// ---------------------------------------------------------------------------

func TestAuthFlow(t *testing.T) {
	testutil.SkipOnShort(t)
	srv, cleanup := setupTestServer(t)
	t.Cleanup(cleanup)

	client := srv.Client()
	// Do not follow redirects so we can inspect the 302 response
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	t.Run("GET /api/auth/google redirects to Google OAuth", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/google", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusTemporaryRedirect {
			t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusTemporaryRedirect)
		}

		location := resp.Header.Get("Location")
		if location == "" {
			t.Fatal("expected Location header in redirect")
		}

		// Verify redirect URL format
		if !strings.HasPrefix(location, "https://accounts.google.com/o/oauth2/auth") {
			t.Errorf("redirect URL should start with Google OAuth endpoint, got: %s", location)
		}
		if !strings.Contains(location, "client_id=test-client-id") {
			t.Errorf("redirect URL should contain client_id, got: %s", location)
		}
		if !strings.Contains(location, "redirect_uri=") {
			t.Errorf("redirect URL should contain redirect_uri, got: %s", location)
		}

		// Verify oauth_state cookie is set
		cookies := resp.Cookies()
		var stateCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "oauth_state" {
				stateCookie = c
				break
			}
		}
		if stateCookie == nil {
			t.Error("expected oauth_state cookie to be set")
		} else {
			if stateCookie.Value == "" {
				t.Error("oauth_state cookie has empty value")
			}
			if !stateCookie.HttpOnly {
				t.Error("oauth_state cookie should be HttpOnly")
			}
		}
	})

	t.Run("GET /api/me without auth returns 401", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/me", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusUnauthorized)
		}

		var appErr errs.AppError
		if err := json.NewDecoder(resp.Body).Decode(&appErr); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if appErr.Kind != errs.KindUnauthorized {
			t.Errorf("error kind: got %q, want %q", appErr.Kind, errs.KindUnauthorized)
		}
		if appErr.Message != "missing authentication" {
			t.Errorf("error message: got %q, want %q", appErr.Message, "missing authentication")
		}
	})
}

// ---------------------------------------------------------------------------
// generateTestKeys creates a fresh ECDSA P-256 key pair encoded as PEM
// for each test. Matches the pattern from auth/application/service_test.go.
// ---------------------------------------------------------------------------

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

// chatAuthAdapter adapts the auth service to the chat module's chatAuth interface.
type chatAuthAdapter struct {
	authSvc *application.AuthService
}

func newChatAuthAdapter(authSvc *application.AuthService) *chatAuthAdapter {
	return &chatAuthAdapter{authSvc: authSvc}
}

func (a *chatAuthAdapter) ValidateToken(ctx context.Context, tokenStr string) (userID, userName, userAvatarUrl string, err error) {
	userID, err = a.authSvc.ValidateJWT(tokenStr)
	if err != nil {
		return "", "", "", err
	}
	user, err := a.authSvc.GetByID(ctx, userID)
	if err != nil {
		return "", "", "", err
	}
	avatar := ""
	if user.AvatarURL != nil {
		avatar = *user.AvatarURL
	}
	return userID, user.Name, avatar, nil
}

// ---------------------------------------------------------------------------
// TestStreamLifecycle simulates OBS publish/unpublish via SRS callbacks
// ---------------------------------------------------------------------------

func TestStreamLifecycle(t *testing.T) {
	testutil.SkipOnShort(t)
	srv, cleanup := setupTestServer(t)
	t.Cleanup(cleanup)

	// Create a test user with a stream key
	userID := uuid.New().String()
	streamKey := "test-stream-key-12345"
	sum := sha256.Sum256([]byte(streamKey))
	keyHash := hex.EncodeToString(sum[:])

	_, err := testPool.Exec(context.Background(),
		`INSERT INTO users (id, google_id, email, name, stream_key_hash, stream_title)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "g-"+userID[:8], "test@test.com", "Test User", keyHash, "My Stream Title")
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	client := srv.Client()

	// 1. Simulate on_publish
	publishBody := fmt.Sprintf(
		`{"action":"on_publish","client_id":1,"stream":"%s","ip":"127.0.0.1","vhost":"__defaultVhost__","app":"live"}`,
		streamKey,
	)
	req, _ := http.NewRequest(http.MethodPost,
		srv.URL+"/api/srs/callback?secret=test-srs-secret",
		strings.NewReader(publishBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("publish request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("on_publish: got %d, body: %s", resp.StatusCode, body)
	}

	// Verify stream created and is live
	var streamID string
	err = testPool.QueryRow(context.Background(),
		`SELECT id FROM streams WHERE user_id = $1 AND status = 'live'`, userID).Scan(&streamID)
	if err != nil {
		t.Fatalf("stream not created: %v", err)
	}
	t.Logf("stream created: %s", streamID)

	// 2. Create a dummy recording file (simulating ffmpeg recording output)
	// The recording goroutine was started by OnStreamStart; create the file
	// at the path it would produce so handleRecording finds it.
	// The recording goroutine was started by OnStreamStart and the path
	// was stored in recordingPaths. OnStreamEnd will retrieve it.

	// 2. Simulate on_unpublish
	unpublishBody := `{"action":"on_unpublish","client_id":1}`
	req, _ = http.NewRequest(http.MethodPost,
		srv.URL+"/api/srs/callback?secret=test-srs-secret",
		strings.NewReader(unpublishBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("unpublish request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("on_unpublish: got %d, body: %s", resp.StatusCode, body)
	}

	// Verify recording status is set (failed since test has no VOD queue)
	var recordingStatus string
	err = testPool.QueryRow(context.Background(),
		`SELECT recording_status FROM streams WHERE id = $1`, streamID).Scan(&recordingStatus)
	if err != nil {
		t.Fatalf("query recording status: %v", err)
	}

	// With a recording file, status should be 'processing' (enqueued for worker)
	// Note: the VOD queue is nil in tests, so no River job is created
	if recordingStatus != "processing" {
		t.Errorf("recording_status = %q, want %q", recordingStatus, "processing")
	}
	t.Logf("recording status: %s", recordingStatus)
}
