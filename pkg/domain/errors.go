// pkg/domain/errors.go

// Package domain holds the sentinel errors every service layer returns, and the
// mappers that turn them into transport-specific responses. Nothing here knows
// about SQL, protobuf, or HTTP frameworks.
package domain

import (
	"errors"
	"fmt"
)

// ErrNotFound means the requested aggregate does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict means the write would violate a uniqueness or state invariant.
var ErrConflict = errors.New("conflict")

// ErrInvalidInput means the caller's request failed validation.
var ErrInvalidInput = errors.New("invalid input")

// ErrUnauthenticated means the request lacks valid authentication credentials.
var ErrUnauthenticated = errors.New("unauthenticated")

// ErrForbidden means the caller is authenticated but not permitted to do this.
var ErrForbidden = errors.New("forbidden")

// ErrUnavailable means a downstream dependency could not be reached or is not
// ready to serve the request.
var ErrUnavailable = errors.New("unavailable")

// NotFound wraps ErrNotFound with a message identifying what was not found.
func NotFound(msg string) error { return fmt.Errorf("%s: %w", msg, ErrNotFound) }

// NotFoundf wraps ErrNotFound with a formatted message.
func NotFoundf(format string, args ...any) error {
	return fmt.Errorf(format+": %w", append(args, error(ErrNotFound))...)
}

// Conflict wraps ErrConflict with a message identifying the violated invariant.
func Conflict(msg string) error { return fmt.Errorf("%s: %w", msg, ErrConflict) }

// Conflictf wraps ErrConflict with a formatted message.
func Conflictf(format string, args ...any) error {
	return fmt.Errorf(format+": %w", append(args, error(ErrConflict))...)
}

// InvalidInput wraps ErrInvalidInput with a message identifying the bad field.
func InvalidInput(msg string) error { return fmt.Errorf("%s: %w", msg, ErrInvalidInput) }

// InvalidInputf wraps ErrInvalidInput with a formatted message.
func InvalidInputf(format string, args ...any) error {
	return fmt.Errorf(format+": %w", append(args, error(ErrInvalidInput))...)
}

// Unauthenticated wraps ErrUnauthenticated with a message.
func Unauthenticated(msg string) error { return fmt.Errorf("%s: %w", msg, ErrUnauthenticated) }

// Forbidden wraps ErrForbidden with a message identifying the denied action.
func Forbidden(msg string) error { return fmt.Errorf("%s: %w", msg, ErrForbidden) }

// Unavailable wraps ErrUnavailable with a message identifying the failed dependency.
func Unavailable(msg string) error { return fmt.Errorf("%s: %w", msg, ErrUnavailable) }
