package errs

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// TestAppError_Error — verify each constructor returns correct Kind, Message,
// and that Error() returns the message.
// ---------------------------------------------------------------------------

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		wantKind Kind
		wantMsg  string
	}{
		{
			name:     "BadRequest",
			err:      BadRequest("bad request: %s", "missing field"),
			wantKind: KindBadRequest,
			wantMsg:  "bad request: missing field",
		},
		{
			name:     "Unauthorized",
			err:      Unauthorized("invalid token"),
			wantKind: KindUnauthorized,
			wantMsg:  "invalid token",
		},
		{
			name:     "NotFound",
			err:      NotFound("user %d not found", 42),
			wantKind: KindNotFound,
			wantMsg:  "user 42 not found",
		},
		{
			name:     "Forbidden",
			err:      Forbidden("access denied"),
			wantKind: KindForbidden,
			wantMsg:  "access denied",
		},
		{
			name:     "Conflict",
			err:      Conflict("duplicate key: %s", "email"),
			wantKind: KindConflict,
			wantMsg:  "duplicate key: email",
		},
		{
			name:     "Internal",
			err:      Internal("something went wrong"),
			wantKind: KindInternal,
			wantMsg:  "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Kind != tt.wantKind {
				t.Fatalf("Kind = %q, want %q", tt.err.Kind, tt.wantKind)
			}
			if tt.err.Message != tt.wantMsg {
				t.Fatalf("Message = %q, want %q", tt.err.Message, tt.wantMsg)
			}
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Fatalf("Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestAppError_Unwrap — verify Unwrap() returns the wrapped error.
// ---------------------------------------------------------------------------

func TestAppError_Unwrap(t *testing.T) {
	tests := []struct {
		name    string
		appErr  *AppError
		wantErr error
		wantNil bool
	}{
		{
			name:    "wraps a real error",
			appErr:  &AppError{Kind: KindInternal, Message: "boom", Err: fmt.Errorf("root cause")},
			wantErr: fmt.Errorf("root cause"),
		},
		{
			name:    "no wrapped error (nil)",
			appErr:  Internal("just a message"),
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unwrapped := tt.appErr.Unwrap()
			if tt.wantNil {
				if unwrapped != nil {
					t.Fatalf("Unwrap() = %v, want nil", unwrapped)
				}
				return
			}
			if unwrapped == nil {
				t.Fatal("Unwrap() = nil, want non-nil")
			}
			if unwrapped.Error() != tt.wantErr.Error() {
				t.Fatalf("Unwrap().Error() = %q, want %q", unwrapped.Error(), tt.wantErr.Error())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBadRequest — verify Kind and formatted Message.
// ---------------------------------------------------------------------------

func TestBadRequest(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		args    []any
		wantMsg string
	}{
		{
			name:    "with formatting args",
			msg:     "invalid field %q",
			args:    []any{"email"},
			wantMsg: `invalid field "email"`,
		},
		{
			name:    "plain message, no args",
			msg:     "missing required fields",
			args:    nil,
			wantMsg: "missing required fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := BadRequest(tt.msg, tt.args...)
			if err.Kind != KindBadRequest {
				t.Fatalf("Kind = %q, want %q", err.Kind, KindBadRequest)
			}
			if err.Message != tt.wantMsg {
				t.Fatalf("Message = %q, want %q", err.Message, tt.wantMsg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestUnauthorized — verify Kind and Message.
// ---------------------------------------------------------------------------

func TestUnauthorized(t *testing.T) {
	tests := []struct {
		name string
		msg  string
	}{
		{name: "expired token", msg: "token expired"},
		{name: "missing credentials", msg: "missing credentials"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unauthorized(tt.msg)
			if err.Kind != KindUnauthorized {
				t.Fatalf("Kind = %q, want %q", err.Kind, KindUnauthorized)
			}
			if err.Message != tt.msg {
				t.Fatalf("Message = %q, want %q", err.Message, tt.msg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestNotFound — verify Kind and formatted Message.
// ---------------------------------------------------------------------------

func TestNotFound(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		args    []any
		wantMsg string
	}{
		{
			name:    "with formatting args",
			msg:     "stream %s not found",
			args:    []any{"abc123"},
			wantMsg: "stream abc123 not found",
		},
		{
			name:    "plain message, no args",
			msg:     "resource not found",
			args:    nil,
			wantMsg: "resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NotFound(tt.msg, tt.args...)
			if err.Kind != KindNotFound {
				t.Fatalf("Kind = %q, want %q", err.Kind, KindNotFound)
			}
			if err.Message != tt.wantMsg {
				t.Fatalf("Message = %q, want %q", err.Message, tt.wantMsg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestForbidden — verify Kind and Message.
// ---------------------------------------------------------------------------

func TestForbidden(t *testing.T) {
	tests := []struct {
		name string
		msg  string
	}{
		{name: "no permission", msg: "you do not own this resource"},
		{name: "role insufficient", msg: "admin role required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Forbidden(tt.msg)
			if err.Kind != KindForbidden {
				t.Fatalf("Kind = %q, want %q", err.Kind, KindForbidden)
			}
			if err.Message != tt.msg {
				t.Fatalf("Message = %q, want %q", err.Message, tt.msg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestConflict — verify Kind and formatted Message.
// ---------------------------------------------------------------------------

func TestConflict(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		args    []any
		wantMsg string
	}{
		{
			name:    "with formatting args",
			msg:     "username %q already taken",
			args:    []any{"alice"},
			wantMsg: `username "alice" already taken`,
		},
		{
			name:    "plain message, no args",
			msg:     "resource already exists",
			args:    nil,
			wantMsg: "resource already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Conflict(tt.msg, tt.args...)
			if err.Kind != KindConflict {
				t.Fatalf("Kind = %q, want %q", err.Kind, KindConflict)
			}
			if err.Message != tt.wantMsg {
				t.Fatalf("Message = %q, want %q", err.Message, tt.wantMsg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestInternal — verify Kind and Message.
// ---------------------------------------------------------------------------

func TestInternal(t *testing.T) {
	tests := []struct {
		name string
		msg  string
	}{
		{name: "db connection failure", msg: "database connection failed"},
		{name: "unknown error", msg: "unexpected error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Internal(tt.msg)
			if err.Kind != KindInternal {
				t.Fatalf("Kind = %q, want %q", err.Kind, KindInternal)
			}
			if err.Message != tt.msg {
				t.Fatalf("Message = %q, want %q", err.Message, tt.msg)
			}
		})
	}
}
