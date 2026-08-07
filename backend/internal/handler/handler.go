package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	"github.com/deepcut/live/internal/errs"
	"github.com/deepcut/live/internal/service"
)

type Handler struct {
	authSvc   *service.AuthService
	userSvc   *service.UserService
	jwtKey    []byte
	srsSecret string
}

func New(authSvc *service.AuthService, userSvc *service.UserService, jwtSecret, srsSecret string) *Handler {
	return &Handler{
		authSvc:   authSvc,
		userSvc:   userSvc,
		jwtKey:    []byte(jwtSecret),
		srsSecret: srsSecret,
	}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/api/auth/google", h.GoogleOAuth)
	r.Get("/api/auth/google/callback", h.GoogleOAuthCallback)

	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Get("/api/me", h.GetMe)
		r.Post("/api/me/stream-key/regenerate", h.RegenerateStreamKey)
		r.Patch("/api/me/settings", h.UpdateSettings)
	})
}

// GoogleOAuth redirects to Google's consent screen.
func (h *Handler) GoogleOAuth(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		writeError(w, errs.Internal("failed to generate state"))
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, h.authSvc.AuthCodeURL(state), http.StatusFound)
}

// GoogleOAuthCallback handles the OAuth callback.
func (h *Handler) GoogleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Validate state
	cookie, err := r.Cookie("oauth_state")
	if err != nil {
		writeError(w, errs.BadRequest("missing state cookie"))
		return
	}
	if r.URL.Query().Get("state") != cookie.Value {
		writeError(w, errs.BadRequest("invalid state"))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, errs.BadRequest("missing code"))
		return
	}

	googleUser, err := h.authSvc.Exchange(r.Context(), code)
	if err != nil {
		slog.Error("oauth exchange failed", "error", err)
		writeError(w, errs.Internal("authentication failed"))
		return
	}

	user, plaintextKey, err := h.userSvc.GetOrCreateUser(r.Context(),
		googleUser.Sub, googleUser.Email, googleUser.Name, googleUser.Picture)
	if err != nil {
		slog.Error("get or create user failed", "error", err)
		writeError(w, errs.Internal("failed to process user"))
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"exp":   time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString(h.jwtKey)
	if err != nil {
		writeError(w, errs.Internal("failed to sign token"))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenStr,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// If this is a new signup and we have the plaintext key, store it in a
	// temporary cookie so the frontend can display it on the dashboard.
	// (The frontend reads this once, then the cookie expires.)
	if plaintextKey != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "stream_key_display",
			Value:    *plaintextKey,
			Path:     "/",
			MaxAge:   300,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// GetMe returns the authenticated user's profile + stream key.
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxKeyUserID).(string)
	user, err := h.userSvc.GetUser(r.Context(), userID)
	if err != nil {
		writeError(w, errs.Internal("failed to get user"))
		return
	}
	if user == nil {
		writeError(w, errs.NotFound("user not found"))
		return
	}

	// Check if there's a stream key to display (from initial signup)
	keyCookie, _ := r.Cookie("stream_key_display")
	streamKey := ""
	if keyCookie != nil {
		streamKey = keyCookie.Value
	}

	resp := map[string]interface{}{
		"id":             user.ID,
		"name":           user.Name,
		"email":          user.Email,
		"avatarUrl":      user.AvatarURL,
		"streamKey":      streamKey,
		"streamTitle":    user.StreamTitle,
		"streamCategory": user.StreamCategory,
		"isLive":         user.IsLive,
	}
	writeJSON(w, http.StatusOK, resp)
}

// RegenerateStreamKey revokes and regenerates the stream key.
func (h *Handler) RegenerateStreamKey(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxKeyUserID).(string)
	key, err := h.userSvc.RegenerateStreamKey(r.Context(), userID)
	if err != nil {
		writeError(w, errs.Internal("failed to regenerate key"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"streamKey": key})
}

// UpdateSettings updates stream title and category.
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxKeyUserID).(string)

	var input struct {
		StreamTitle    string `json:"streamTitle"`
		StreamCategory string `json:"streamCategory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, errs.BadRequest("invalid JSON"))
		return
	}
	if input.StreamTitle == "" || len(input.StreamTitle) > 100 {
		writeError(w, errs.BadRequest("title must be 1-100 characters"))
		return
	}

	if err := h.userSvc.UpdateSettings(r.Context(), userID, input.StreamTitle, input.StreamCategory); err != nil {
		writeError(w, errs.Internal("failed to update settings"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"streamTitle":    input.StreamTitle,
		"streamCategory": input.StreamCategory,
	})
}

// Helpers
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err *errs.AppError) {
	status := http.StatusInternalServerError
	switch err.Kind {
	case errs.KindBadRequest:
		status = http.StatusBadRequest
	case errs.KindUnauthorized:
		status = http.StatusUnauthorized
	case errs.KindNotFound:
		status = http.StatusNotFound
	case errs.KindConflict:
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": err.Message})
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
