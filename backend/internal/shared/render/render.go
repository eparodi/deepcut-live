package render

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/deepcut/live/internal/shared/errs"
)

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}

func Error(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *errs.AppError
	if errors.As(err, &appErr) {
		code := statusFromKind(appErr.Kind)
		JSON(w, code, appErr)
		return
	}
	JSON(w, http.StatusInternalServerError, errs.Internal("internal server error"))
}

func statusFromKind(k errs.Kind) int {
	switch k {
	case errs.KindBadRequest:
		return http.StatusBadRequest
	case errs.KindUnauthorized:
		return http.StatusUnauthorized
	case errs.KindForbidden:
		return http.StatusForbidden
	case errs.KindNotFound:
		return http.StatusNotFound
	case errs.KindConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
