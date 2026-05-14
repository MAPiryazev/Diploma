package errors

import (
	"errors"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{Field: "amount", Message: "must be numeric"}
	if got, want := err.Error(), "amount: must be numeric"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestSentinelErrors(t *testing.T) {
	sentinels := []error{
		ErrNotFound,
		ErrValidation,
		ErrConflict,
		ErrAlreadyExists,
		ErrInvalidState,
		ErrNotAllowed,
		ErrInternalError,
		ErrUnauthorized,
		ErrForbidden,
		ErrTimeout,
		ErrUnavailable,
		ErrBadRequest,
		ErrDependencyFailed,
		ErrDecodeFailed,
		ErrEncodeFailed,
	}
	for _, err := range sentinels {
		if err == nil {
			t.Fatal("nil sentinel")
		}
		if err.Error() == "" {
			t.Fatalf("empty message for %T %v", err, err)
		}
	}
}

func TestErrorsIsSentinel(t *testing.T) {
	wrapped := errors.Join(ErrNotFound, errors.New("outer"))
	if !errors.Is(wrapped, ErrNotFound) {
		t.Fatal("expected errors.Is(wrapped, ErrNotFound)")
	}
}
