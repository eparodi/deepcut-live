package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/deepcut/live/internal/errs"
	"github.com/golang-jwt/jwt/v5"
)

type ctxKey int

const ctxKeyUserID ctxKey = iota

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try cookie first
		cookie, err := r.Cookie("token")
		if err != nil {
			// Try Authorization header
			header := r.Header.Get("Authorization")
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			if tokenStr == "" || tokenStr == header {
				writeError(w, errs.Unauthorized("authentication required"))
				return
			}
			cookie = &http.Cookie{Value: tokenStr}
		}

		token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return h.jwtKey, nil
		})
		if err != nil || !token.Valid {
			writeError(w, errs.Unauthorized("invalid token"))
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeError(w, errs.Unauthorized("invalid claims"))
			return
		}
		userID, ok := claims["sub"].(string)
		if !ok {
			writeError(w, errs.Unauthorized("invalid user id"))
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
