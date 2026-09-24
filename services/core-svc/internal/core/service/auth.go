// services/core-svc/internal/core/service/auth.go
package service

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// phoneE164Pattern mirrors the artisan_phone_e164_check constraint, so a login
// attempt with a malformed number is rejected before it reaches Redis.
var phoneE164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// devBypassPhone/devBypassCode let one specific number always log in during
// local development without requesting a real OTP challenge first. Only ever
// honored when s.otp.DevMode() is true (see VerifyOtp).
const (
	devBypassPhone = "+918779279060"
	devBypassCode  = "123456"
)

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
func (s *Identity) VerifyOtp(ctx context.Context, challengeID, phone, code, devRole string) (LoginResult, error) {
	if challengeID == "" || code == "" {
		return LoginResult{}, fmt.Errorf("challenge_id and code are required: %w", pkgdomain.ErrInvalidInput)
	}
	if !phoneE164Pattern.MatchString(phone) {
		return LoginResult{}, fmt.Errorf("phone_e164 %q must be E.164: %w", phone, pkgdomain.ErrInvalidInput)
	}

	// devPhoneBypass skips real OTP verification entirely for one hardcoded
	// dev phone number, so it can always log in with a fixed code during local
	// development regardless of what a real (or expired) challenge holds. Only
	// honored when dev OTP is enabled -- same guard as DevOTPCode and dev_role
	// above -- so this can never activate against a real deployment.
	if s.otp.DevMode() && phone == devBypassPhone && code == devBypassCode {
		s.log.WarnContext(ctx, "dev phone bypass otp login", "phone", phone)
	} else if err := s.otp.Verify(ctx, challengeID, phone, code); err != nil {
		return LoginResult{}, err
	}

	// Staff first: a ministry-issued staff account always logs in as its own
	// role, so staff never depend on the dev_role hack below.
	staff, isStaff, err := s.store.GetActiveStaffByPhone(ctx, phone)
	if err != nil {
		return LoginResult{}, fmt.Errorf("looking up staff for a verified phone: %w", err)
	}
	if isStaff {
		tokens, err := s.tokens.Issue(staffSubject(staff))
		if err != nil {
			return LoginResult{}, fmt.Errorf("issuing staff tokens after otp verification: %w", err)
		}
		s.log.InfoContext(ctx, "staff otp verified", "role", staff.Role, "staff_id", staff.ID)
		// Registered: true -- staff never go through RegisterArtisan.
		return LoginResult{Tokens: tokens, Registered: true}, nil
	}

	// devRole requests a token for a role other than ARTISAN. BUYER,
	// CLUSTER_OFFICER and MINISTRY currently have no other issuance path
	// anywhere in the product -- see cmd/seed-demo/main.go, which mints
	// them directly with pkg/auth.Issuer as a documented workaround for
	// exactly this gap. Only honored when the server is actually running
	// with dev OTP enabled: the same condition that makes `code`
	// DevOTPCode ("000000") always accepted, so this can never mint a
	// non-artisan token against a real deployment. See
	// WIRING_AUDIT_PLAN.md F-5.
	if devRole != "" {
		if !s.otp.DevMode() {
			return LoginResult{}, fmt.Errorf("dev_role is only honored when dev OTP is enabled: %w", pkgdomain.ErrInvalidInput)
		}
		role, err := auth.ParseRole(devRole)
		if err != nil {
			return LoginResult{}, fmt.Errorf("dev_role %q: %w", devRole, pkgdomain.ErrInvalidInput)
		}
		if role != auth.RoleArtisan {
			tokens, err := s.tokens.Issue(auth.Subject{ID: phone, Role: role, PhoneE164: phone})
			if err != nil {
				return LoginResult{}, fmt.Errorf("issuing dev-role tokens after otp verification: %w", err)
			}
			s.log.WarnContext(ctx, "dev-role otp login", "role", role, "phone", phone)
			// Registered: true -- this LoginResult field means "must still
			// call RegisterArtisan", an artisan-only flow this role never
			// goes through.
			return LoginResult{Tokens: tokens, Registered: true}, nil
		}
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

	// A staff refresh re-reads the account, so deactivating a staffer (or
	// changing their scope) takes effect at the next refresh rather than when
	// a 7-day refresh token finally expires. A dev_role token's subject is a
	// phone, not a staff id, and is left alone -- it only exists in dev.
	if role != auth.RoleArtisan && role != auth.RoleBuyer {
		if staffID, perr := uuid.Parse(claims.Subject); perr == nil {
			staff, err := s.store.GetStaffAccount(ctx, staffID)
			if err != nil || !staff.Active {
				return auth.TokenPair{}, fmt.Errorf("staff account is not active: %w", pkgdomain.ErrForbidden)
			}
			tokens, err := s.tokens.Issue(staffSubject(staff))
			if err != nil {
				return auth.TokenPair{}, fmt.Errorf("issuing refreshed staff tokens: %w", err)
			}
			return tokens, nil
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

// staffSubject builds the token subject for a staff account: sub is the
// staff id (never an artisan id), and the account's region rides along as
// scope claims so the bff can narrow an officer's dashboards without a lookup.
func staffSubject(staff domain.StaffAccount) auth.Subject {
	// No PhoneE164: RegisterArtisan binds a new profile to the token's phone,
	// and a staff token must never be able to register itself as an artisan.
	sub := auth.Subject{ID: staff.ID.String(), Role: auth.Role(staff.Role)}
	if staff.StateCode != nil {
		sub.ScopeState = *staff.StateCode
	}
	if staff.District != nil {
		sub.ScopeDistrict = *staff.District
	}
	return sub
}
