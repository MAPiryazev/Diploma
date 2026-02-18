package errors

import "errors"

var (
	// доменные ошибки
	ErrNotFound      = errors.New("not found")
	ErrValidation    = errors.New("validation error")
	ErrConflict      = errors.New("conflict")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidState  = errors.New("invalid state")
	ErrNotAllowed    = errors.New("operation not allowed")

	// инфраструктура / доступ
	ErrInternalError = errors.New("internal error")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrTimeout       = errors.New("timeout")
	ErrUnavailable   = errors.New("service unavailable")

	// данные / интеграции
	ErrBadRequest       = errors.New("bad request")
	ErrDependencyFailed = errors.New("dependency failed")
	ErrDecodeFailed     = errors.New("decode failed")
	ErrEncodeFailed     = errors.New("encode failed")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
