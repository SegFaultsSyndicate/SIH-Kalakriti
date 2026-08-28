package domain

import "errors"

// IsNotFound returns true if err is or wraps ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsConflict returns true if err is or wraps ErrConflict.
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

// IsForbidden returns true if err is or wraps ErrForbidden.
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}
