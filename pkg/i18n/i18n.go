// pkg/i18n/i18n.go

// Package i18n provides simple message localization.
package i18n

import (
	"context"
	"strings"
)

// Locale represents a language preference.
type Locale string

const (
	English Locale = "en"
	Hindi   Locale = "hi"
)

// contextKey for locale in context.
type contextKey struct{}

// WithLocale returns a context with the given locale.
func WithLocale(ctx context.Context, locale Locale) context.Context {
	return context.WithValue(ctx, contextKey{}, locale)
}

// GetLocale returns the locale from context, defaulting to English.
func GetLocale(ctx context.Context) Locale {
	if loc, ok := ctx.Value(contextKey{}).(Locale); ok {
		return loc
	}
	return English
}

// ParseAcceptLanguage extracts preferred locale from Accept-Language header.
// Returns English if header is missing or unparseable.
func ParseAcceptLanguage(header string) Locale {
	if header == "" {
		return English
	}

	// Simple parse: "hi,en;q=0.9" → "hi"
	parts := strings.Split(header, ",")
	for _, part := range parts {
		lang := strings.TrimSpace(strings.Split(part, ";")[0])
		if strings.HasPrefix(lang, "hi") {
			return Hindi
		}
	}

	return English
}

// T translates a message key to the given locale.
// Falls back to English if translation missing.
func T(locale Locale, key string) string {
	if locale == Hindi {
		if msg, ok := hindiMessages[key]; ok {
			return msg
		}
	}
	return englishMessages[key]
}

// englishMessages maps keys to English strings.
var englishMessages = map[string]string{
	// Domain errors
	"err.not_found":       "not found",
	"err.conflict":        "conflict",
	"err.invalid_input":   "invalid input",
	"err.unauthenticated": "unauthenticated",
	"err.forbidden":       "forbidden",
	"err.unavailable":     "service unavailable",

	// Validation errors
	"validation.required":         "is required",
	"validation.invalid_format":   "invalid format",
	"validation.out_of_range":     "out of range",
	"validation.too_long":         "too long",
	"validation.too_short":        "too short",
	"validation.must_be_positive": "must be positive",

	// Auth errors
	"auth.invalid_token":       "invalid token",
	"auth.token_expired":       "token expired",
	"auth.invalid_otp":         "invalid OTP",
	"auth.otp_expired":         "OTP expired",
	"auth.missing_bearer":      "Authorization must be Bearer <token>",
	"auth.phone_not_found":     "phone number not found",
	"auth.already_verified":    "already verified",
	"auth.verification_failed": "verification failed",

	// Order errors
	"order.quantity_positive":      "quantity must be positive",
	"order.buyer_required":         "buyer_id is required",
	"order.idempotency_required":   "idempotency_key is required",
	"order.reason_required":        "reason is required",
	"order.ship_date_required":     "promised_ship_date is required to accept a lot",
	"order.decline_reason_required": "decline_reason is required to decline a lot",
	"order.progress_range":         "progress_pct must be between 0 and 100",
	"order.defects_when_passed":    "defects must be empty when passed is true",
	"order.severity_required":      "severity must be specified for every defect",

	// Payment errors
	"payment.commission_range":     "commission_pct must be between 0 and 100",
	"payment.settlement_ref_required": "settlement_ref is required",

	// Amendment errors
	"amendment.reason_required":           "reason is required",
	"amendment.quantity_required":         "proposed_quantity is required for REDUCE_QUANTITY",
	"amendment.quantity_less_than_current": "proposed_quantity must be less than current quantity",
	"amendment.deadline_required":         "proposed_required_by is required for EXTEND_DEADLINE",
	"amendment.deadline_must_extend":      "proposed_required_by must extend the current deadline",
	"amendment.unknown_type":              "unknown amendment type",
}

// hindiMessages maps keys to Hindi strings.
// ponytail: Minimal Hindi coverage, expand when users request specific messages
var hindiMessages = map[string]string{
	// Domain errors
	"err.not_found":       "नहीं मिला",
	"err.conflict":        "टकराव",
	"err.invalid_input":   "अमान्य इनपुट",
	"err.unauthenticated": "अप्रमाणित",
	"err.forbidden":       "निषिद्ध",
	"err.unavailable":     "सेवा अनुपलब्ध",

	// Validation errors
	"validation.required":         "आवश्यक है",
	"validation.invalid_format":   "अमान्य प्रारूप",
	"validation.out_of_range":     "सीमा से बाहर",
	"validation.must_be_positive": "सकारात्मक होना चाहिए",

	// Auth errors
	"auth.invalid_token":    "अमान्य टोकन",
	"auth.token_expired":    "टोकन समाप्त",
	"auth.invalid_otp":      "अमान्य OTP",
	"auth.otp_expired":      "OTP समाप्त",
	"auth.missing_bearer":   "Authorization में Bearer <token> होना चाहिए",
	"auth.phone_not_found":  "फोन नंबर नहीं मिला",
	"auth.already_verified": "पहले से सत्यापित",

	// Order errors
	"order.quantity_positive":  "मात्रा सकारात्मक होनी चाहिए",
	"order.buyer_required":     "buyer_id आवश्यक है",
	"order.reason_required":    "कारण आवश्यक है",

	// Payment errors
	"payment.commission_range": "commission_pct 0 से 100 के बीच होना चाहिए",
}
