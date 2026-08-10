package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"

	"github.com/deepcut/live/internal/modules/auth/application"
	"github.com/deepcut/live/internal/modules/auth/domain"
	streamsdomain "github.com/deepcut/live/internal/modules/streams/domain"
	"github.com/deepcut/live/internal/shared/errs"
	"github.com/deepcut/live/internal/shared/render"
)

type ctxKey int

const ctxKeyUserID ctxKey = iota

// authService is the subset of *application.AuthService methods that AuthHandler needs.
type authService interface {
	GenerateOAuthURL(state string) string
	ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error)
	GetGoogleUser(ctx context.Context, token *oauth2.Token) (*application.GoogleUserInfo, error)
	GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error)
	CreateUser(ctx context.Context, googleID, email, name, avatarURL string) (*domain.User, error)
	GenerateJWT(userID string) (string, error)
	ValidateJWT(tokenStr string) (string, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	RegenerateStreamKey(ctx context.Context, userID string) (string, error)
	UpdateSettings(ctx context.Context, userID, title, category string) error
}

// streamOps is the subset of stream service methods that AuthHandler needs.
type streamOps interface {
	GetAnalytics(ctx context.Context, userID, period string) (*streamsdomain.Analytics, error)
	ForceEndStream(ctx context.Context, userID string) (string, error)
}

type AuthHandler struct {
	svc       authService
	streamSvc streamOps
	baseURL   string
	logger    *slog.Logger
}

func NewAuthHandler(svc authService, streamSvc streamOps, baseURL string, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, streamSvc: streamSvc, baseURL: baseURL, logger: logger}
}

// RegisterRoutes registers all auth routes on the given router.
func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/auth/google", h.GoogleOAuth)
	r.Get("/api/auth/google/callback", h.GoogleOAuthCallback)

	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Get("/api/me", h.GetMe)
		r.Post("/api/me/stream-key/regenerate", h.RegenerateStreamKey)
		r.Patch("/api/me/settings", h.UpdateSettings)
		r.Get("/api/me/analytics", h.GetAnalytics)
		r.Post("/api/me/stream/end", h.ForceEndStream)
	})
}

// AuthMiddleware extracts and validates the JWT from the Authorization header
// or the "token" cookie (set during OAuth callback).
func (h *AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		// 1. Check Authorization header (preferred)
		header := r.Header.Get("Authorization")
		if header != "" && strings.HasPrefix(header, "Bearer ") {
			tokenStr = strings.TrimPrefix(header, "Bearer ")
		}

		// 2. Fall back to token cookie (set during OAuth redirect)
		if tokenStr == "" {
			cookie, err := r.Cookie("token")
			if err == nil {
				tokenStr = cookie.Value
			}
		}

		if tokenStr == "" {
			render.Error(w, r, errs.Unauthorized("missing authentication"))
			return
		}

		userID, err := h.svc.ValidateJWT(tokenStr)
		if err != nil {
			render.Error(w, r, errs.Unauthorized("invalid token"))
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromCtx extracts the authenticated user ID from context.
func UserIDFromCtx(ctx context.Context) string {
	id, _ := ctx.Value(ctxKeyUserID).(string)
	return id
}

// GoogleOAuth redirects the user to Google's OAuth consent page.
func (h *AuthHandler) GoogleOAuth(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	rand.Read(b)
	state := hex.EncodeToString(b)

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		MaxAge:   600,
		SameSite: http.SameSiteLaxMode,
	})

	url := h.svc.GenerateOAuthURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleOAuthCallback handles the OAuth callback from Google.
func (h *AuthHandler) GoogleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// Validate state
	cookie, err := r.Cookie("oauth_state")
	if err != nil {
		render.Error(w, r, errs.BadRequest("missing oauth state"))
		return
	}
	stateParam := r.URL.Query().Get("state")
	if stateParam != cookie.Value {
		render.Error(w, r, errs.BadRequest("invalid oauth state"))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		render.Error(w, r, errs.BadRequest("missing authorization code"))
		return
	}

	token, err := h.svc.ExchangeCodeForToken(r.Context(), code)
	if err != nil {
		h.logger.Error("exchange code", "error", err)
		render.Error(w, r, fmt.Errorf("exchange token: %w", err))
		return
	}

	googleUser, err := h.svc.GetGoogleUser(r.Context(), token)
	if err != nil {
		h.logger.Error("get google user", "error", err)
		render.Error(w, r, fmt.Errorf("get google user: %w", err))
		return
	}

	user, err := h.svc.GetByGoogleID(r.Context(), googleUser.ID)
	if err != nil {
		// Create new user if not found
		user, err = h.svc.CreateUser(r.Context(), googleUser.ID, googleUser.Email, googleUser.Name, googleUser.Picture)
		if err != nil {
			h.logger.Error("create user", "error", err)
			render.Error(w, r, fmt.Errorf("create user: %w", err))
			return
		}
	}

	jwt, err := h.svc.GenerateJWT(user.ID)
	if err != nil {
		h.logger.Error("generate jwt", "error", err)
		render.Error(w, r, fmt.Errorf("generate jwt: %w", err))
		return
	}

	// Set JWT as httpOnly cookie so the frontend middleware can read it
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    jwt,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		MaxAge:   259200, // 72 hours
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect back to the frontend (Next.js, same origin)
	http.Redirect(w, r, h.baseURL+"/dashboard", http.StatusTemporaryRedirect)
}

type getMeResponse struct {
	ID             string  `json:"id"`
	Email          string  `json:"email"`
	Name           string  `json:"name"`
	AvatarURL      *string `json:"avatarUrl"`
	StreamKey      *string `json:"streamKey,omitempty"`
	StreamTitle    *string `json:"streamTitle"`
	StreamCategory *string `json:"streamCategory"`
	IsLive         bool    `json:"isLive"`
}

// GetMe returns the authenticated user's profile.
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID := UserIDFromCtx(r.Context())
	if userID == "" {
		render.Error(w, r, errs.Unauthorized("not authenticated"))
		return
	}

	user, err := h.svc.GetByID(r.Context(), userID)
	if err != nil {
		render.Error(w, r, fmt.Errorf("get me: %w", err))
		return
	}

	render.JSON(w, http.StatusOK, getMeResponse{
		ID:             user.ID,
		Email:          user.Email,
		Name:           user.Name,
		AvatarURL:      user.AvatarURL,
		StreamKey:      &user.StreamKey,
		StreamTitle:    user.StreamTitle,
		StreamCategory: user.StreamCategory,
		IsLive:         user.IsLive,
	})
}

type regenerateStreamKeyRequest struct {
	Confirm bool `json:"confirm"`
}

// RegenerateStreamKey generates a new stream key for the authenticated user.
// The confirmation dialog is handled client-side; the backend always regenerates
// when called (the user already clicked "Regenerate" in the UI dialog).
func (h *AuthHandler) RegenerateStreamKey(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID := UserIDFromCtx(r.Context())
	if userID == "" {
		render.Error(w, r, errs.Unauthorized("not authenticated"))
		return
	}

	// Check confirmation if body is provided, but skip for empty POST bodies
	// (the frontend dialog already serves as the confirmation step).
	if r.ContentLength > 0 {
		var req regenerateStreamKeyRequest
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			render.Error(w, r, errs.BadRequest("invalid JSON: %v", err))
			return
		}
		if !req.Confirm {
			render.Error(w, r, errs.BadRequest("must confirm stream key regeneration"))
			return
		}
	}

	rawKey, err := h.svc.RegenerateStreamKey(r.Context(), userID)
	if err != nil {
		render.Error(w, r, fmt.Errorf("regenerate stream key: %w", err))
		return
	}

	render.JSON(w, http.StatusOK, map[string]string{"streamKey": rawKey})
}

type updateSettingsRequest struct {
	Title    *string `json:"streamTitle"`
	Category *string `json:"streamCategory"`
}

type updateSettingsResponse struct {
	StreamTitle    string  `json:"streamTitle"`
	StreamCategory *string `json:"streamCategory"`
}

// UpdateSettings updates the authenticated user's stream settings.
func (h *AuthHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID := UserIDFromCtx(r.Context())
	if userID == "" {
		render.Error(w, r, errs.Unauthorized("not authenticated"))
		return
	}

	var req updateSettingsRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		render.Error(w, r, errs.BadRequest("invalid JSON: %v", err))
		return
	}

	// Validate and extract title (required, 1-100 chars after trimming).
	title := ""
	if req.Title != nil {
		title = strings.TrimSpace(*req.Title)
	}
	if title == "" || len(title) > 100 {
		render.Error(w, r, errs.BadRequest("streamTitle is required and must be 1-100 characters"))
		return
	}

	// Validate category (optional, max 100 chars when provided).
	category := ""
	if req.Category != nil {
		category = strings.TrimSpace(*req.Category)
	}
	if category != "" && len(category) > 100 {
		render.Error(w, r, errs.BadRequest("streamCategory must be 100 characters or fewer"))
		return
	}

	if err := h.svc.UpdateSettings(r.Context(), userID, title, category); err != nil {
		render.Error(w, r, fmt.Errorf("update settings: %w", err))
		return
	}

	// Build response: category is null when cleared/not-provided (nil or empty).
	var respCategory *string
	if req.Category != nil && *req.Category != "" {
		respCategory = req.Category
	}

	render.JSON(w, http.StatusOK, updateSettingsResponse{
		StreamTitle:    title,
		StreamCategory: respCategory,
	})
}

// GetAnalytics returns stream analytics for the authenticated user.
func (h *AuthHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromCtx(r.Context())
	if userID == "" {
		render.Error(w, r, errs.Unauthorized("not authenticated"))
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "week"
	}
	if period != "week" && period != "month" && period != "all" {
		render.Error(w, r, errs.BadRequest("invalid period: %s; expected week, month, or all", period))
		return
	}

	analytics, err := h.streamSvc.GetAnalytics(r.Context(), userID, period)
	if err != nil {
		render.Error(w, r, fmt.Errorf("get analytics: %w", err))
		return
	}

	render.JSON(w, http.StatusOK, analytics)
}

// ForceEndStream terminates the current live stream.
func (h *AuthHandler) ForceEndStream(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromCtx(r.Context())
	if userID == "" {
		render.Error(w, r, errs.Unauthorized("not authenticated"))
		return
	}

	msg, err := h.streamSvc.ForceEndStream(r.Context(), userID)
	if err != nil {
		render.Error(w, r, fmt.Errorf("force end stream: %w", err))
		return
	}

	render.JSON(w, http.StatusOK, map[string]string{
		"status":  "offline",
		"message": msg,
	})
}
