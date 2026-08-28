// pkg/auth/context.go
package auth

import (
	"context"
	"fmt"

	"github.com/segfaultsyndicate/kalakriti/pkg/domain"
)

type ctxKey int

const principalKey ctxKey = iota

// ContextWithPrincipal attaches an authenticated principal to ctx. Only the
// interceptor (and tests) should call this — a service layer receives a ctx that
// already carries the principal.
func ContextWithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// PrincipalFrom returns the principal carried by ctx. ok is false on an
// unauthenticated call, which is normal for the public RPCs.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

// RequirePrincipal returns the principal carried by ctx, or ErrForbidden when the
// call is unauthenticated.
func RequirePrincipal(ctx context.Context) (Principal, error) {
	p, ok := PrincipalFrom(ctx)
	if !ok {
		return Principal{}, fmt.Errorf("no authenticated principal: %w", domain.ErrForbidden)
	}
	return p, nil
}

// RequireRole returns the principal carried by ctx if it holds one of roles, and
// ErrForbidden otherwise. Handlers call this at the top of every guarded RPC.
func RequireRole(ctx context.Context, roles ...Role) (Principal, error) {
	p, err := RequirePrincipal(ctx)
	if err != nil {
		return Principal{}, err
	}
	if !p.HasRole(roles...) {
		return Principal{}, fmt.Errorf("role %s may not perform this action: %w", p.Role, domain.ErrForbidden)
	}
	return p, nil
}

// RequireSelfOrRole allows a caller acting on their own subject, or any caller
// holding one of the given override roles. This is the common shape for profile
// reads and edits: an artisan may touch their own row, an officer may touch any.
func RequireSelfOrRole(ctx context.Context, subject string, roles ...Role) (Principal, error) {
	p, err := RequirePrincipal(ctx)
	if err != nil {
		return Principal{}, err
	}
	if p.Subject == subject && subject != "" {
		return p, nil
	}
	if p.HasRole(roles...) {
		return p, nil
	}
	return Principal{}, fmt.Errorf("role %s may not act on subject %s: %w", p.Role, subject, domain.ErrForbidden)
}
