package errs

import "fmt"

type Kind string

const (
	KindNotFound     Kind = "not_found"
	KindBadRequest   Kind = "bad_request"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindConflict     Kind = "conflict"
	KindInternal     Kind = "internal"
)

type AppError struct {
	Kind    Kind   `json:"kind"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

func BadRequest(msg string, args ...any) *AppError {
	return &AppError{Kind: KindBadRequest, Message: fmt.Sprintf(msg, args...)}
}
func Unauthorized(msg string) *AppError {
	return &AppError{Kind: KindUnauthorized, Message: msg}
}
func NotFound(msg string, args ...any) *AppError {
	return &AppError{Kind: KindNotFound, Message: fmt.Sprintf(msg, args...)}
}
func Forbidden(msg string) *AppError {
	return &AppError{Kind: KindForbidden, Message: msg}
}
func Conflict(msg string, args ...any) *AppError {
	return &AppError{Kind: KindConflict, Message: fmt.Sprintf(msg, args...)}
}
func Internal(msg string) *AppError {
	return &AppError{Kind: KindInternal, Message: msg}
}
