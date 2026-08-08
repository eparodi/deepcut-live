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

	"github.com/deepcut/live/internal/modules/auth/application"
	"github.com/deepcut/live/internal/shared/errs"
	"github.com/deepcut/live/internal/shared/render"
)

type ctxKey int

const ctxKeyUserID ctxKey = iota

type AuthHandler struct {
	svc    *application.AuthService
	logger *slog.Logger
}

func NewAuthHandler(svc *application.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, logger: logger}
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
	})
}

// AuthMiddleware extracts and validates the JWT from the Authorization header.
func (h *AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			render.Error(w, r, errs.Unauthorized("missing authorization header"))
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		userID, err := h.svc.ValidateJWT(tokenStr)
		if err != nil {
			render.Error(w, r, errs.Unauthorized("invalid token"))
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// userIDFromCtx extracts the authenticated user ID from context.
func userIDFromCtx(ctx context.Context) string {
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

	render.JSON(w, http.StatusOK, map[string]string{"token": jwt})
}

type getMeResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	Name        string  `json:"name"`
	AvatarURL   *string `json:"avatarUrl"`
	StreamKey   *string `json:"streamKey,omitempty"`
	StreamTitle *string `json:"streamTitle"`
	IsLive      bool    `json:"isLive"`
	CreatedAt   string  `json:"createdAt"`
}

// GetMe returns the authenticated user's profile.
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID := userIDFromCtx(r.Context())
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
		ID:          user.ID,
		Email:       user.Email,
		Name:        user.Name,
		AvatarURL:   user.AvatarURL,
		StreamTitle: user.StreamTitle,
		IsLive:      user.IsLive,
		CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

type regenerateStreamKeyRequest struct {
	Confirm bool `json:"confirm"`
}

// RegenerateStreamKey generates a new stream key for the authenticated user.
func (h *AuthHandler) RegenerateStreamKey(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID := userIDFromCtx(r.Context())
	if userID == "" {
		render.Error(w, r, errs.Unauthorized("not authenticated"))
		return
	}

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

	rawKey, err := h.svc.RegenerateStreamKey(r.Context(), userID)
	if err != nil {
		render.Error(w, r, fmt.Errorf("regenerate stream key: %w", err))
		return
	}

	render.JSON(w, http.StatusOK, map[string]string{"streamKey": rawKey})
}

type updateSettingsRequest struct {
	Title    *string `json:"title"`
	Category *string `json:"category"`
}

// UpdateSettings updates the authenticated user's stream settings.
func (h *AuthHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID := userIDFromCtx(r.Context())
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

	title := ""
	if req.Title != nil {
		title = *req.Title
	}
	category := ""
	if req.Category != nil {
		category = *req.Category
	}

	if err := h.svc.UpdateSettings(r.Context(), userID, title, category); err != nil {
		render.Error(w, r, fmt.Errorf("update settings: %w", err))
		return
	}

	render.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
