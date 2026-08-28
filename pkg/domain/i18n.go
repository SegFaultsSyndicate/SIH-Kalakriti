// pkg/domain/i18n.go

package domain

import (
	"context"
	"fmt"
)

// LocaleProvider extracts locale from context.
// Set via i18n.WithLocale(ctx, locale).
type LocaleProvider interface {
	GetLocale(ctx context.Context) string
}

// TKey creates a translatable error with a message key.
// Use with i18n.T(locale, key) to get localized message.
func TKey(key string) error {
	return &translatableError{key: key}
}

// TKeyf creates a translatable error with formatted args.
func TKeyf(key string, args ...any) error {
	return &translatableError{key: key, args: args}
}

// translatableError wraps a message key that can be translated.
type translatableError struct {
	key  string
	args []any
}

func (e *translatableError) Error() string {
	if len(e.args) > 0 {
		return fmt.Sprintf(e.key, e.args...)
	}
	return e.key
}

// Key returns the translation key.
func (e *translatableError) Key() string {
	return e.key
}

// Args returns the format arguments.
func (e *translatableError) Args() []any {
	return e.args
}

// GetTranslationKey extracts the translation key from an error if it's translatable.
func GetTranslationKey(err error) (string, bool) {
	if te, ok := err.(*translatableError); ok {
		return te.key, true
	}
	return "", false
}
