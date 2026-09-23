// pkg/auth/token.go
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
)

// TokenKind distinguishes an access token from a refresh token. It is carried as
// a private claim so a refresh token can never be replayed as an access token.
type TokenKind string

const (
	// KindAccess is the short-lived token presented on every call.
	KindAccess TokenKind = "access"
	// KindRefresh is the long-lived token accepted only by RefreshToken.
	KindRefresh TokenKind = "refresh"
)

// signingMethod is fixed at HS256. Pinning it here and passing
// jwt.WithValidMethods at parse time is what closes the alg-confusion hole.
const signingMethod = "HS256"

// Claims is the JWT body Kalakriti issues.
type Claims struct {
	jwt.RegisteredClaims
	// Role is the caller's authorisation role.
	Role string `json:"role"`
	// Language is the caller's preferred language.
	Language string `json:"language,omitempty"`
	// Kind separates access tokens from refresh tokens.
	Kind string `json:"kind"`
	// PhoneE164 is the verified phone a login was proven against. Present on
	// tokens minted before a profile exists, so RegisterArtisan can bind to it.
	PhoneE164 string `json:"phone,omitempty"`
	// ScopeState/ScopeDistrict narrow a staff token to one region.
	ScopeState    string `json:"scope_state,omitempty"`
	ScopeDistrict string `json:"scope_district,omitempty"`
	// Actor is the acting staff account id on an assisted-mode token (the
	// RFC 8693 "act" idea, flattened to the subject string).
	Actor string `json:"act,omitempty"`
}

// Config configures token issue and verification.
type Config struct {
	// Secret is the HS256 signing key. Must be at least 32 bytes.
	Secret string
	// Issuer is written as `iss` and required to match on parse.
	Issuer string
	// AccessTTL bounds an access token's life.
	AccessTTL time.Duration
	// RefreshTTL bounds a refresh token's life.
	RefreshTTL time.Duration
}

const minSecretLen = 32

// Issuer mints and verifies tokens. It holds no mutable state and is safe for
// concurrent use.
type Issuer struct {
	cfg    Config
	secret []byte
	// now is injected so tests can mint an already-expired token without sleeping.
	now func() time.Time
}

// NewIssuer validates the config and returns an Issuer. A short or empty secret
// is a startup error rather than a silently weak deployment.
func NewIssuer(cfg Config) (*Issuer, error) {
	if len(cfg.Secret) < minSecretLen {
		return nil, fmt.Errorf("auth: JWT secret must be at least %d bytes, got %d", minSecretLen, len(cfg.Secret))
	}
	if cfg.Issuer == "" {
		return nil, errors.New("auth: JWT issuer must be set")
	}
	if cfg.AccessTTL <= 0 {
		return nil, fmt.Errorf("auth: access TTL must be positive, got %s", cfg.AccessTTL)
	}
	if cfg.RefreshTTL <= cfg.AccessTTL {
		return nil, fmt.Errorf("auth: refresh TTL (%s) must exceed access TTL (%s)", cfg.RefreshTTL, cfg.AccessTTL)
	}
	return &Issuer{cfg: cfg, secret: []byte(cfg.Secret), now: time.Now}, nil
}

// TokenPair is a freshly minted access/refresh pair and their expiry instants.
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

// Subject describes who a token is being minted for.
type Subject struct {
	// ID is the artisan id, or the account id for non-artisan roles. May be
	// empty for a pre-registration token, which carries only a verified phone.
	ID string
	// Role is the authorisation role to grant.
	Role Role
	// Language is the caller's preferred language.
	Language string
	// PhoneE164 is the verified phone number the login was proven against.
	PhoneE164 string
	// ScopeState/ScopeDistrict narrow a staff token to one region.
	ScopeState    string
	ScopeDistrict string
	// Actor is the staff account acting for an artisan (assisted mode).
	Actor string
}

// Issue mints an access/refresh pair for sub.
func (i *Issuer) Issue(sub Subject) (TokenPair, error) {
	if !sub.Role.Valid() {
		return TokenPair{}, fmt.Errorf("auth: cannot issue a token for invalid role %q", sub.Role)
	}
	now := i.now().UTC()

	access, accessExp, err := i.sign(sub, KindAccess, now, i.cfg.AccessTTL)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, refreshExp, err := i.sign(sub, KindRefresh, now, i.cfg.RefreshTTL)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:      access,
		RefreshToken:     refresh,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
	}, nil
}

func (i *Issuer) sign(sub Subject, kind TokenKind, now time.Time, ttl time.Duration) (string, time.Time, error) {
	exp := now.Add(ttl)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.cfg.Issuer,
			Subject:   sub.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        ids.New().String(),
		},
		Role:          sub.Role.String(),
		Language:      sub.Language,
		Kind:          string(kind),
		PhoneE164:     sub.PhoneE164,
		ScopeState:    sub.ScopeState,
		ScopeDistrict: sub.ScopeDistrict,
		Actor:         sub.Actor,
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: signing %s token: %w", kind, err)
	}
	return signed, exp, nil
}

// Verify parses and validates a token, checking that it is of the expected kind.
// Every failure maps to domain.ErrUnauthenticated so the caller need not distinguish
// "expired" from "forged" — telling a caller which one it was only helps an attacker.
func (i *Issuer) Verify(token string, want TokenKind) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != signingMethod {
				return nil, fmt.Errorf("unexpected signing method %q", t.Method.Alg())
			}
			return i.secret, nil
		},
		jwt.WithValidMethods([]string{signingMethod}),
		jwt.WithIssuer(i.cfg.Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(i.now),
	)
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, fmt.Errorf("token expired: %w", domain.ErrUnauthenticated)
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, fmt.Errorf("token not valid yet: %w", domain.ErrUnauthenticated)
		default:
			return nil, fmt.Errorf("token rejected: %w", domain.ErrUnauthenticated)
		}
	}
	if TokenKind(claims.Kind) != want {
		return nil, fmt.Errorf("token is a %s token, expected %s: %w", claims.Kind, want, domain.ErrUnauthenticated)
	}
	return &claims, nil
}

// Principal converts verified claims into the principal handlers read. A token
// whose role is not one of the known roles is rejected here rather than
// defaulting to something permissive.
func (c *Claims) Principal() (Principal, error) {
	role, err := ParseRole(c.Role)
	if err != nil {
		return Principal{}, fmt.Errorf("%s: %w", err, domain.ErrUnauthenticated)
	}
	return Principal{
		Subject:       c.Subject,
		Role:          role,
		Language:      c.Language,
		TokenID:       c.ID,
		PhoneE164:     c.PhoneE164,
		ScopeState:    c.ScopeState,
		ScopeDistrict: c.ScopeDistrict,
		Actor:         c.Actor,
	}, nil
}
