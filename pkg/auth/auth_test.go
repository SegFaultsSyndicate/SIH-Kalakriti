// pkg/auth/auth_test.go
package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

const testSecret = "0123456789abcdef0123456789abcdef" // exactly 32 bytes

func testIssuer(t *testing.T) *Issuer {
	t.Helper()
	iss, err := NewIssuer(Config{
		Secret:     testSecret,
		Issuer:     "kalakriti-test",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	return iss
}

func TestNewIssuerRejectsBadConfig(t *testing.T) {
	base := Config{Secret: testSecret, Issuer: "k", AccessTTL: time.Minute, RefreshTTL: time.Hour}

	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"short secret", func(c *Config) { c.Secret = "tooshort" }, "at least 32 bytes"},
		{"empty secret", func(c *Config) { c.Secret = "" }, "at least 32 bytes"},
		{"no issuer", func(c *Config) { c.Issuer = "" }, "issuer must be set"},
		{"zero access ttl", func(c *Config) { c.AccessTTL = 0 }, "access TTL must be positive"},
		{"refresh not longer", func(c *Config) { c.RefreshTTL = time.Minute }, "must exceed access TTL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			tt.mutate(&cfg)
			_, err := NewIssuer(cfg)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not mention %q", err, tt.want)
			}
		})
	}
}

func TestIssueAndVerifyRoundTrip(t *testing.T) {
	iss := testIssuer(t)
	pair, err := iss.Issue(Subject{ID: "artisan-1", Role: RoleArtisan, Language: "HINDI", PhoneE164: "+919876543210"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	claims, err := iss.Verify(pair.AccessToken, KindAccess)
	if err != nil {
		t.Fatalf("Verify access: %v", err)
	}
	if claims.Subject != "artisan-1" {
		t.Errorf("Subject = %q, want artisan-1", claims.Subject)
	}
	if claims.Role != "ARTISAN" {
		t.Errorf("Role = %q, want ARTISAN", claims.Role)
	}
	if claims.Language != "HINDI" {
		t.Errorf("Language = %q, want HINDI", claims.Language)
	}
	if claims.PhoneE164 != "+919876543210" {
		t.Errorf("PhoneE164 = %q, want +919876543210", claims.PhoneE164)
	}
	if claims.ID == "" {
		t.Error("jti should be set")
	}

	p, err := claims.Principal()
	if err != nil {
		t.Fatalf("Principal: %v", err)
	}
	if p.Role != RoleArtisan || p.Subject != "artisan-1" {
		t.Errorf("principal = %+v", p)
	}
}

func TestVerifyRejectsWrongTokenKind(t *testing.T) {
	iss := testIssuer(t)
	pair, err := iss.Issue(Subject{ID: "a", Role: RoleArtisan})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// A refresh token must never be accepted where an access token is required.
	if _, err := iss.Verify(pair.RefreshToken, KindAccess); err == nil {
		t.Fatal("refresh token was accepted as an access token")
	} else if !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated, got %v", err)
	}

	// ...and the reverse.
	if _, err := iss.Verify(pair.AccessToken, KindRefresh); err == nil {
		t.Fatal("access token was accepted as a refresh token")
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	iss := testIssuer(t)
	// Mint at a fixed instant, then verify well past the access TTL.
	minted := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	iss.now = func() time.Time { return minted }

	pair, err := iss.Issue(Subject{ID: "a", Role: RoleArtisan})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	iss.now = func() time.Time { return minted.Add(16 * time.Minute) } // AccessTTL is 15m
	_, err = iss.Verify(pair.AccessToken, KindAccess)
	if err == nil {
		t.Fatal("expired token was accepted")
	}
	if !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated, got %v", err)
	}

	// The refresh token has a 24h TTL and is still good at the same instant.
	if _, err := iss.Verify(pair.RefreshToken, KindRefresh); err != nil {
		t.Fatalf("refresh token should still be valid: %v", err)
	}
}

func TestVerifyRejectsForeignSecretAndIssuer(t *testing.T) {
	iss := testIssuer(t)
	pair, err := iss.Issue(Subject{ID: "a", Role: RoleArtisan})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	other, err := NewIssuer(Config{
		Secret:     "ffffffffffffffffffffffffffffffff",
		Issuer:     "kalakriti-test",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	if _, err := other.Verify(pair.AccessToken, KindAccess); err == nil {
		t.Fatal("token signed with a different secret was accepted")
	}

	wrongIssuer, err := NewIssuer(Config{
		Secret:     testSecret,
		Issuer:     "somebody-else",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	if _, err := wrongIssuer.Verify(pair.AccessToken, KindAccess); err == nil {
		t.Fatal("token from a different issuer was accepted")
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	iss := testIssuer(t)
	for _, tok := range []string{"", "not-a-jwt", "a.b.c", "..."} {
		if _, err := iss.Verify(tok, KindAccess); err == nil {
			t.Errorf("garbage token %q was accepted", tok)
		}
	}
}

func TestIssueRejectsInvalidRole(t *testing.T) {
	iss := testIssuer(t)
	if _, err := iss.Issue(Subject{ID: "a", Role: Role("SUPERUSER")}); err == nil {
		t.Fatal("expected an error for an unknown role")
	}
}

func TestParseRole(t *testing.T) {
	tests := []struct {
		in      string
		want    Role
		wantErr bool
	}{
		{"ARTISAN", RoleArtisan, false},
		{"artisan", RoleArtisan, false},
		{"  BUYER  ", RoleBuyer, false},
		{"CLUSTER_OFFICER", RoleClusterOfficer, false},
		{"MINISTRY", RoleMinistry, false},
		{"ADMIN", "", true},
		{"", "", true},
		{"ROLE_ARTISAN", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseRole(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseRole(%q) = %v, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRole(%q): %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("ParseRole(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestClaimsPrincipalRejectsUnknownRole(t *testing.T) {
	c := &Claims{Role: "SUPERUSER"}
	if _, err := c.Principal(); err == nil {
		t.Fatal("expected an unknown role to be rejected")
	} else if !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated, got %v", err)
	}
}

func TestRequireRole(t *testing.T) {
	ctx := ContextWithPrincipal(context.Background(), Principal{Subject: "a", Role: RoleArtisan})

	if _, err := RequireRole(ctx, RoleArtisan, RoleMinistry); err != nil {
		t.Fatalf("artisan should be allowed: %v", err)
	}
	if _, err := RequireRole(ctx, RoleMinistry); err == nil {
		t.Fatal("artisan should not pass a ministry-only check")
	} else if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
	if _, err := RequireRole(context.Background(), RoleArtisan); err == nil {
		t.Fatal("an unauthenticated context should be rejected")
	}
}

func TestRequireSelfOrRole(t *testing.T) {
	artisan := ContextWithPrincipal(context.Background(), Principal{Subject: "a1", Role: RoleArtisan})
	officer := ContextWithPrincipal(context.Background(), Principal{Subject: "o1", Role: RoleClusterOfficer})

	if _, err := RequireSelfOrRole(artisan, "a1", RoleClusterOfficer); err != nil {
		t.Fatalf("artisan acting on self should be allowed: %v", err)
	}
	if _, err := RequireSelfOrRole(artisan, "a2", RoleClusterOfficer); err == nil {
		t.Fatal("artisan acting on another subject should be denied")
	}
	if _, err := RequireSelfOrRole(officer, "a2", RoleClusterOfficer); err != nil {
		t.Fatalf("officer override should be allowed: %v", err)
	}
	// An empty subject must never match an empty principal subject.
	anon := ContextWithPrincipal(context.Background(), Principal{Subject: "", Role: RoleBuyer})
	if _, err := RequireSelfOrRole(anon, "", RoleClusterOfficer); err == nil {
		t.Fatal("empty subject must not self-match")
	}
}

// --- interceptor ---

func bearerCtx(token string) context.Context {
	return metadata.NewIncomingContext(context.Background(),
		metadata.Pairs(authorizationHeader, "Bearer "+token))
}

func okHandler(ctx context.Context, _ any) (any, error) {
	p, ok := PrincipalFrom(ctx)
	if !ok {
		return "anonymous", nil
	}
	return p.Subject, nil
}

func TestInterceptorRejectsExpiredTokenWithUnauthenticated(t *testing.T) {
	iss := testIssuer(t)
	minted := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	iss.now = func() time.Time { return minted }
	pair, err := iss.Issue(Subject{ID: "artisan-1", Role: RoleArtisan})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	iss.now = func() time.Time { return minted.Add(time.Hour) }

	interceptor := UnaryServerInterceptor(iss, NewPublicMethods())
	_, err = interceptor(bearerCtx(pair.AccessToken), nil,
		&grpc.UnaryServerInfo{FullMethod: "/identity.v1.IdentityService/GetArtisan"}, okHandler)

	if err == nil {
		t.Fatal("expired token was accepted")
	}
	if got := status.Code(err); got != codes.Unauthenticated {
		t.Fatalf("status code = %v, want Unauthenticated", got)
	}
}

func TestInterceptorAcceptsValidTokenAndInjectsPrincipal(t *testing.T) {
	iss := testIssuer(t)
	pair, err := iss.Issue(Subject{ID: "artisan-7", Role: RoleArtisan})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	interceptor := UnaryServerInterceptor(iss, NewPublicMethods())
	got, err := interceptor(bearerCtx(pair.AccessToken), nil,
		&grpc.UnaryServerInfo{FullMethod: "/identity.v1.IdentityService/GetArtisan"}, okHandler)
	if err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	if got != "artisan-7" {
		t.Fatalf("handler saw subject %v, want artisan-7", got)
	}
}

func TestInterceptorRejectsMalformedAuthorizationHeader(t *testing.T) {
	iss := testIssuer(t)
	pair, err := iss.Issue(Subject{ID: "a", Role: RoleArtisan})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	interceptor := UnaryServerInterceptor(iss, NewPublicMethods())
	info := &grpc.UnaryServerInfo{FullMethod: "/identity.v1.IdentityService/GetArtisan"}

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{"no metadata at all", context.Background()},
		{"no authorization key", metadata.NewIncomingContext(context.Background(), metadata.Pairs("x", "y"))},
		{"missing bearer prefix", metadata.NewIncomingContext(context.Background(), metadata.Pairs(authorizationHeader, pair.AccessToken))},
		{"bearer with no token", metadata.NewIncomingContext(context.Background(), metadata.Pairs(authorizationHeader, "Bearer "))},
		{"wrong scheme", metadata.NewIncomingContext(context.Background(), metadata.Pairs(authorizationHeader, "Basic "+pair.AccessToken))},
		{"refresh token presented as access", bearerCtx(pair.RefreshToken)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := interceptor(tt.ctx, nil, info, okHandler); err == nil {
				t.Fatal("expected rejection")
			} else if got := status.Code(err); got != codes.Unauthenticated {
				t.Fatalf("status code = %v, want Unauthenticated", got)
			}
		})
	}
}

func TestInterceptorAcceptsLowercaseBearerPrefix(t *testing.T) {
	iss := testIssuer(t)
	pair, err := iss.Issue(Subject{ID: "a", Role: RoleArtisan})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs(authorizationHeader, "bearer "+pair.AccessToken))

	interceptor := UnaryServerInterceptor(iss, NewPublicMethods())
	if _, err := interceptor(ctx, nil,
		&grpc.UnaryServerInfo{FullMethod: "/identity.v1.IdentityService/GetArtisan"}, okHandler); err != nil {
		t.Fatalf("lowercase bearer prefix should be accepted: %v", err)
	}
}

func TestInterceptorAllowsPublicMethodsWithoutToken(t *testing.T) {
	iss := testIssuer(t)
	public := NewPublicMethods("/identity.v1.IdentityService/RequestOtp")
	interceptor := UnaryServerInterceptor(iss, public)

	got, err := interceptor(context.Background(), nil,
		&grpc.UnaryServerInfo{FullMethod: "/identity.v1.IdentityService/RequestOtp"}, okHandler)
	if err != nil {
		t.Fatalf("public method rejected: %v", err)
	}
	if got != "anonymous" {
		t.Fatalf("handler saw %v, want anonymous", got)
	}
}

func TestInterceptorAttachesPrincipalToPublicMethodWhenTokenPresent(t *testing.T) {
	iss := testIssuer(t)
	pair, err := iss.Issue(Subject{ID: "artisan-9", Role: RoleArtisan})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	public := NewPublicMethods("/identity.v1.IdentityService/RequestOtp")
	interceptor := UnaryServerInterceptor(iss, public)

	got, err := interceptor(bearerCtx(pair.AccessToken), nil,
		&grpc.UnaryServerInfo{FullMethod: "/identity.v1.IdentityService/RequestOtp"}, okHandler)
	if err != nil {
		t.Fatalf("public method with a token should still succeed: %v", err)
	}
	if got != "artisan-9" {
		t.Fatalf("handler saw %v, want artisan-9", got)
	}
}

func TestInterceptorPublicMethodIgnoresBadToken(t *testing.T) {
	iss := testIssuer(t)
	public := NewPublicMethods("/identity.v1.IdentityService/RequestOtp")
	interceptor := UnaryServerInterceptor(iss, public)

	// A garbage token on a public method must not fail the call; it just means
	// the handler sees an anonymous caller.
	got, err := interceptor(bearerCtx("garbage"), nil,
		&grpc.UnaryServerInfo{FullMethod: "/identity.v1.IdentityService/RequestOtp"}, okHandler)
	if err != nil {
		t.Fatalf("public method should tolerate a bad token: %v", err)
	}
	if got != "anonymous" {
		t.Fatalf("handler saw %v, want anonymous", got)
	}
}

func TestBearerMetadata(t *testing.T) {
	md := BearerMetadata("tok")
	if got := md.Get(authorizationHeader); len(got) != 1 || got[0] != "Bearer tok" {
		t.Fatalf("BearerMetadata = %v", got)
	}
}

func TestTokenFromRoundTripsThroughContext(t *testing.T) {
	if _, ok := TokenFrom(context.Background()); ok {
		t.Fatal("TokenFrom should report false on a context that never carried a token")
	}

	ctx := ContextWithToken(context.Background(), "tok-123")
	got, ok := TokenFrom(ctx)
	if !ok || got != "tok-123" {
		t.Fatalf("TokenFrom = %q, %v; want \"tok-123\", true", got, ok)
	}
}
