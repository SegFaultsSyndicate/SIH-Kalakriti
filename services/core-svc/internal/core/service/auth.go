// services/core-svc/internal/core/service/auth.go
package service

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// phoneE164Pattern mirrors the artisan_phone_e164_check constraint, so a login
// attempt with a malformed number is rejected before it reaches Redis.
var phoneE164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// RequestOtp issues a login challenge for a phone number. It deliberately does
// not reveal whether the number belongs to a registered artisan: an unregistered
// number gets a challenge too, and the caller learns their registration state
// only after proving control of the phone.
func (s *Identity) RequestOtp(ctx context.Context, phone, language string) (Challenge, error) {
	if !phoneE164Pattern.MatchString(phone) {
		return Challenge{}, fmt.Errorf("phone_e164 %q must be E.164, e.g. +919876543210: %w",
			phone, pkgdomain.ErrInvalidInput)
	}
	return s.otp.Request(ctx, phone, language)
}

// LoginResult is what a successful OTP verification yields.
type LoginResult struct {
	Tokens auth.TokenPair
	// ArtisanID is empty when no profile exists yet.
	ArtisanID string
	// Registered is false when the caller must still call RegisterArtisan.
	Registered bool
}

// VerifyOtp redeems a challenge and mints a token pair.
//
// A caller with no profile still gets tokens, carrying the verified phone and no
// subject. That is what lets RegisterArtisan authenticate the registration and
// bind it to a phone the caller has proven they control, without a second
// unauthenticated endpoint.
func (s *Identity) VerifyOtp(ctx context.Context, challengeID, phone, code string) (LoginResult, error) {
	if challengeID == "" || code == "" {
		return LoginResult{}, fmt.Errorf("challenge_id and code are required: %w", pkgdomain.ErrInvalidInput)
	}
	if !phoneE164Pattern.MatchString(phone) {
		return LoginResult{}, fmt.Errorf("phone_e164 %q must be E.164: %w", phone, pkgdomain.ErrInvalidInput)
	}

	if err := s.otp.Verify(ctx, challengeID, phone, code); err != nil {
		return LoginResult{}, err
	}

	artisanID, registered, err := s.store.ArtisanExistsByPhone(ctx, phone)
	if err != nil {
		return LoginResult{}, fmt.Errorf("looking up the profile for a verified phone: %w", err)
	}

	sub := auth.Subject{
		Role:      auth.RoleArtisan,
		PhoneE164: phone,
	}
	result := LoginResult{Registered: registered}
	if registered {
		sub.ID = artisanID.String()
		result.ArtisanID = sub.ID

		// Load the profile so the token carries the artisan's own language and
		// server-rendered text comes back in it without a further round trip.
		if profile, err := s.store.GetArtisan(ctx, artisanID); err == nil && len(profile.Languages) > 0 {
			sub.Language = profile.Languages[0]
		}
	}

	tokens, err := s.tokens.Issue(sub)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issuing tokens after otp verification: %w", err)
	}
	result.Tokens = tokens

	s.log.InfoContext(ctx, "otp verified", "registered", registered, "artisan_id", result.ArtisanID)
	return result, nil
}

// RefreshToken exchanges a valid refresh token for a fresh pair. Verify rejects
// an access token presented here, because the two kinds are distinguished by a
// private claim.
//
// Each refresh token is one-time use: RefreshToken burns the presented
// token's jti before issuing a replacement, so a stolen refresh token that
// gets redeemed by an attacker is detected the moment the legitimate client
// (or the attacker, whichever goes second) tries to redeem it again. That
// second, rejected redeemer's whole session family is then revoked
// (RevokeSubject) — reuse means the family is already compromised, so
// letting whoever went first keep rotating would just lock out the victim
// while leaving the attacker with a live session. A revoked session
// (RevokeSubject — "sign out everywhere") is rejected the same way. Neither
// check touches already-issued access tokens, which stay valid for their own
// short remaining TTL; this is a deliberate tradeoff for staying out of the
// per-request verify path — see pkg/auth.RevocationStore.
func (s *Identity) RefreshToken(ctx context.Context, refreshToken string) (auth.TokenPair, error) {
	if refreshToken == "" {
		return auth.TokenPair{}, fmt.Errorf("refresh_token is required: %w", pkgdomain.ErrInvalidInput)
	}

	claims, err := s.tokens.Verify(refreshToken, auth.KindRefresh)
	if err != nil {
		return auth.TokenPair{}, err
	}
	role, err := auth.ParseRole(claims.Role)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("%s: %w", err, pkgdomain.ErrForbidden)
	}

	if s.revocations != nil {
		if claims.IssuedAt == nil {
			return auth.TokenPair{}, fmt.Errorf("refresh token has no iat: %w", pkgdomain.ErrForbidden)
		}
		revoked, err := s.revocations.RevokedBefore(ctx, claims.Subject, claims.IssuedAt.Time)
		if err != nil {
			return auth.TokenPair{}, fmt.Errorf("checking session revocation: %w", err)
		}
		if revoked {
			return auth.TokenPair{}, fmt.Errorf("session has been revoked: %w", pkgdomain.ErrUnauthenticated)
		}

		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl <= 0 {
			ttl = time.Minute // Verify already rejects an expired token; this is just a safe floor for SETNX's TTL.
		}
		fresh, err := s.revocations.TryBurnJTI(ctx, claims.ID, ttl)
		if err != nil {
			return auth.TokenPair{}, fmt.Errorf("checking refresh token reuse: %w", err)
		}
		if !fresh {
			// Reuse of an already-burned refresh token means someone other than
			// the last legitimate redeemer just tried to use it — a stolen
			// token, redeemed by an attacker, with the real owner now locked
			// out, or the reverse. Either way the pair in front of whoever
			// went first is now attacker-controlled from the other party's
			// perspective, so the whole session family must die, not just
			// this one request: revoke the subject so every token issued up
			// to and including the one just minted from this jti stops
			// working, forcing a fresh login on every device.
			if revokeErr := s.revocations.RevokeSubject(ctx, claims.Subject, ttl); revokeErr != nil {
				s.log.ErrorContext(ctx, "revoking subject after refresh token reuse", "error", revokeErr, "subject", claims.Subject)
			}
			return auth.TokenPair{}, fmt.Errorf("refresh token already used: %w", pkgdomain.ErrUnauthenticated)
		}
	}

	tokens, err := s.tokens.Issue(auth.Subject{
		ID:        claims.Subject,
		Role:      role,
		Language:  claims.Language,
		PhoneE164: claims.PhoneE164,
	})
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("issuing refreshed tokens: %w", err)
	}
	return tokens, nil
}
