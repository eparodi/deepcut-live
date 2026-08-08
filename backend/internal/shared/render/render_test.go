package render

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deepcut/live/internal/shared/errs"
)

// ---------------------------------------------------------------------------
// TestJSON — table-driven tests for the JSON render helper.
// ---------------------------------------------------------------------------

func TestJSON(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        any
		wantStatus  int
		wantContent string
	}{
		{
			name:        "status 200 with body",
			status:      http.StatusOK,
			body:        map[string]string{"message": "ok"},
			wantStatus:  http.StatusOK,
			wantContent: `{"message":"ok"}`,
		},
		{
			name:        "status 201 with nil body",
			status:      http.StatusCreated,
			body:        nil,
			wantStatus:  http.StatusCreated,
			wantContent: "",
		},
		{
			name:        "status 404 with error body",
			status:      http.StatusNotFound,
			body:        errs.NotFound("user not found"),
			wantStatus:  http.StatusNotFound,
			wantContent: `{"kind":"not_found","message":"user not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			JSON(rec, tt.status, tt.body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			contentType := rec.Header().Get("Content-Type")
			wantCT := "application/json; charset=utf-8"
			if contentType != wantCT {
				t.Fatalf("Content-Type = %q, want %q", contentType, wantCT)
			}

			body := rec.Body.String()
			if tt.wantContent == "" {
				if body != "" {
					t.Fatalf("body = %q, want empty", body)
				}
				return
			}

			// Use JSON round-trip to verify valid JSON and compare structurally.
			var got, want any
			if err := json.Unmarshal([]byte(body), &got); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantContent), &want); err != nil {
				t.Fatalf("failed to unmarshal expected body: %v", err)
			}
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			if string(gotJSON) != string(wantJSON) {
				t.Fatalf("body = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestError — table-driven tests for the Error render helper.
// ---------------------------------------------------------------------------

func TestError(t *testing.T) {
	tests := []struct {
		name       string
		inputErr   error
		wantStatus int
		wantKind   errs.Kind
		wantMsg    string
	}{
		{
			name:       "AppError NotFound",
			inputErr:   errs.NotFound("user 1 not found"),
			wantStatus: http.StatusNotFound,
			wantKind:   errs.KindNotFound,
			wantMsg:    "user 1 not found",
		},
		{
			name:       "AppError BadRequest",
			inputErr:   errs.BadRequest("missing field"),
			wantStatus: http.StatusBadRequest,
			wantKind:   errs.KindBadRequest,
			wantMsg:    "missing field",
		},
		{
			name:       "AppError Unauthorized",
			inputErr:   errs.Unauthorized("invalid token"),
			wantStatus: http.StatusUnauthorized,
			wantKind:   errs.KindUnauthorized,
			wantMsg:    "invalid token",
		},
		{
			name:       "AppError Forbidden",
			inputErr:   errs.Forbidden("access denied"),
			wantStatus: http.StatusForbidden,
			wantKind:   errs.KindForbidden,
			wantMsg:    "access denied",
		},
		{
			name:       "AppError Conflict",
			inputErr:   errs.Conflict("username taken"),
			wantStatus: http.StatusConflict,
			wantKind:   errs.KindConflict,
			wantMsg:    "username taken",
		},
		{
			name:       "AppError Internal",
			inputErr:   errs.Internal("db failure"),
			wantStatus: http.StatusInternalServerError,
			wantKind:   errs.KindInternal,
			wantMsg:    "db failure",
		},
		{
			name:       "plain error (not AppError)",
			inputErr:   fmt.Errorf("some low-level error"),
			wantStatus: http.StatusInternalServerError,
			wantKind:   errs.KindInternal,
			wantMsg:    "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			Error(rec, req, tt.inputErr)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			var appErr errs.AppError
			if err := json.NewDecoder(rec.Body).Decode(&appErr); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}
			if appErr.Kind != tt.wantKind {
				t.Fatalf("Kind = %q, want %q", appErr.Kind, tt.wantKind)
			}
			if appErr.Message != tt.wantMsg {
				t.Fatalf("Message = %q, want %q", appErr.Message, tt.wantMsg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestStatusFromKind — map every Kind to the correct HTTP status code.
// ---------------------------------------------------------------------------

func TestStatusFromKind(t *testing.T) {
	tests := []struct {
		kind       errs.Kind
		wantStatus int
	}{
		{kind: errs.KindBadRequest, wantStatus: http.StatusBadRequest},
		{kind: errs.KindUnauthorized, wantStatus: http.StatusUnauthorized},
		{kind: errs.KindForbidden, wantStatus: http.StatusForbidden},
		{kind: errs.KindNotFound, wantStatus: http.StatusNotFound},
		{kind: errs.KindConflict, wantStatus: http.StatusConflict},
		{kind: errs.KindInternal, wantStatus: http.StatusInternalServerError},
		{kind: errs.Kind("unknown_kind"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("kind=%s", tt.kind), func(t *testing.T) {
			got := statusFromKind(tt.kind)
			if got != tt.wantStatus {
				t.Fatalf("statusFromKind(%q) = %d, want %d", tt.kind, got, tt.wantStatus)
			}
		})
	}
}
