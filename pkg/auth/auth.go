// pkg/auth/auth.go

// Package auth issues and verifies the JWTs every Kalakriti service accepts, and
// carries the authenticated principal through context. Token parsing lives here
// and in the gRPC interceptor — never in a service layer.
package auth

import (
	"fmt"
	"strings"
)

// Role is the authorisation role carried in an access token. The string values
// match identity.v1.Role's enum names minus the ROLE_ prefix, so a token stays
// readable in a log line.
type Role string

const (
	// RoleArtisan is a maker: owns products, accepts order lots.
	RoleArtisan Role = "ARTISAN"
	// RoleBuyer is a purchaser, individual or institutional.
	RoleBuyer Role = "BUYER"
	// RoleClusterOfficer is field staff who verify artisans and administer clusters.
	RoleClusterOfficer Role = "CLUSTER_OFFICER"
	// RoleMinistry is read-only oversight across every cluster.
	RoleMinistry Role = "MINISTRY"
)

// allRoles is the closed set a token may carry; anything else is rejected at parse.
var allRoles = map[Role]struct{}{
	RoleArtisan:        {},
	RoleBuyer:          {},
	RoleClusterOfficer: {},
	RoleMinistry:       {},
}

// ParseRole converts a wire string to a Role, rejecting anything unrecognised so
// a forged or stale role name can never widen access.
func ParseRole(s string) (Role, error) {
	r := Role(strings.ToUpper(strings.TrimSpace(s)))
	if _, ok := allRoles[r]; !ok {
		return "", fmt.Errorf("unknown role %q", s)
	}
	return r, nil
}

// String returns the role's wire representation.
func (r Role) String() string { return string(r) }

// Valid reports whether r is one of the known roles.
func (r Role) Valid() bool {
	_, ok := allRoles[r]
	return ok
}

// Principal is the authenticated caller, as derived from a verified access token.
// It is what handlers read off the context; they never see the token itself.
type Principal struct {
	// Subject is the artisan id for RoleArtisan, or the account id otherwise.
	Subject string
	// Role is the caller's authorisation role.
	Role Role
	// Language is the caller's preferred language, for server-rendered text.
	Language string
	// TokenID is the token's jti, for audit trails and revocation lists.
	TokenID string
	// PhoneE164 is the phone number the caller proved control of at login. It is
	// the only identity a pre-registration token carries, and is what binds a
	// registration to a number the caller actually holds.
	PhoneE164 string
}

// HasRole reports whether the principal holds any of the given roles.
func (p Principal) HasRole(roles ...Role) bool {
	for _, want := range roles {
		if p.Role == want {
			return true
		}
	}
	return false
}
