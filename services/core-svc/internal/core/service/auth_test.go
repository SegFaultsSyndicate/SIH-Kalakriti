// services/core-svc/internal/core/service/auth_test.go
package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

func TestRequestOtpValidatesPhone(t *testing.T) {
	otp := &fakeOTP{acceptCode: "123456"}
	svc := newTestIdentity(newFakeStore(), newFakeTokens(), otp)

	for _, bad := range []string{"", "9876543210", "+0123456789", "+91987654321012345", "not-a-phone"} {
		if _, err := svc.RequestOtp(context.Background(), bad, "HINDI"); !errors.Is(err, pkgdomain.ErrInvalidInput) {
			t.Errorf("RequestOtp(%q) should be rejected, got %v", bad, err)
		}
	}
	if len(otp.requested) != 0 {
		t.Fatal("a malformed phone must never reach the OTP provider")
	}

	if _, err := svc.RequestOtp(context.Background(), testPhone, "HINDI"); err != nil {
		t.Fatalf("a valid phone should be accepted: %v", err)
	}
	if len(otp.requested) != 1 {
		t.Fatalf("expected 1 challenge request, got %d", len(otp.requested))
	}
}

func TestRequestOtpDoesNotRevealWhetherThePhoneIsRegistered(t *testing.T) {
	store := newFakeStore()
	otp := &fakeOTP{acceptCode: "123456"}
	svc := newTestIdentity(store, newFakeTokens(), otp)

	// Unregistered number.
	unknown, err := svc.RequestOtp(context.Background(), "+919000000009", "HINDI")
	if err != nil {
		t.Fatalf("RequestOtp for an unknown number: %v", err)
	}

	// Now register a number and ask again.
	if _, _, err := svc.RegisterArtisan(artisanPhoneCtx("", testPhone), validRegisterInput(), "idem-1"); err != nil {
		t.Fatalf("RegisterArtisan: %v", err)
	}
	known, err := svc.RequestOtp(context.Background(), testPhone, "HINDI")
	if err != nil {
		t.Fatalf("RequestOtp for a known number: %v", err)
	}

	// Both responses must look the same to a caller probing for registrations.
	if (unknown.ID == "") != (known.ID == "") || unknown.DevMode != known.DevMode {
		t.Fatal("the response shape differs between registered and unregistered numbers")
	}
}

func TestVerifyOtpUnregisteredPhoneYieldsPreRegistrationToken(t *testing.T) {
	tokens := newFakeTokens()
	svc := newTestIdentity(newFakeStore(), tokens, &fakeOTP{acceptCode: "123456"})

	result, err := svc.VerifyOtp(context.Background(), "challenge-1", testPhone, "123456", "")
	if err != nil {
		t.Fatalf("VerifyOtp: %v", err)
	}
	if result.Registered {
		t.Error("an unregistered phone should report registered=false")
	}
	if result.ArtisanID != "" {
		t.Errorf("artisan_id should be empty, got %q", result.ArtisanID)
	}

	if len(tokens.issued) != 1 {
		t.Fatalf("expected 1 issued token, got %d", len(tokens.issued))
	}
	sub := tokens.issued[0]
	// The token must carry the verified phone and no subject: that is what lets
	// RegisterArtisan bind the new profile to a proven number.
	if sub.PhoneE164 != testPhone {
		t.Errorf("issued token phone = %q, want %q", sub.PhoneE164, testPhone)
	}
	if sub.ID != "" {
		t.Errorf("issued token subject should be empty pre-registration, got %q", sub.ID)
	}
	if sub.Role != auth.RoleArtisan {
		t.Errorf("issued token role = %q, want ARTISAN", sub.Role)
	}
}

func TestVerifyOtpRegisteredPhoneYieldsSubjectAndLanguage(t *testing.T) {
	store := newFakeStore()
	tokens := newFakeTokens()
	svc := newTestIdentity(store, tokens, &fakeOTP{acceptCode: "123456"})

	created, _, err := svc.RegisterArtisan(artisanPhoneCtx("", testPhone), validRegisterInput(), "idem-1")
	if err != nil {
		t.Fatalf("RegisterArtisan: %v", err)
	}

	result, err := svc.VerifyOtp(context.Background(), "challenge-1", testPhone, "123456", "")
	if err != nil {
		t.Fatalf("VerifyOtp: %v", err)
	}
	if !result.Registered {
		t.Error("a registered phone should report registered=true")
	}
	if result.ArtisanID != created.ID.String() {
		t.Errorf("artisan_id = %q, want %q", result.ArtisanID, created.ID)
	}

	sub := tokens.issued[len(tokens.issued)-1]
	if sub.ID != created.ID.String() {
		t.Errorf("issued token subject = %q, want the artisan id", sub.ID)
	}
	// The artisan's most fluent language rides along so server-rendered text
	// comes back in it without a further round trip.
	if sub.Language != "GUJARATI" {
		t.Errorf("issued token language = %q, want GUJARATI", sub.Language)
	}
}

func TestVerifyOtpRejectsABadCode(t *testing.T) {
	tokens := newFakeTokens()
	svc := newTestIdentity(newFakeStore(), tokens, &fakeOTP{acceptCode: "123456"})

	_, err := svc.VerifyOtp(context.Background(), "challenge-1", testPhone, "999999", "")
	if !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
	if len(tokens.issued) != 0 {
		t.Fatal("no token may be issued for a failed verification")
	}
}

func TestVerifyOtpAcceptsSIHDemoCodesOnlyInDevMode(t *testing.T) {
	for _, test := range []struct {
		phone string
		code  string
	}{
		{phone: devBypassPhone, code: devBypassCode},
		{phone: devSignupPhone, code: devSignupCode},
	} {
		t.Run(test.phone, func(t *testing.T) {
			svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{devMode: true})
			if _, err := svc.VerifyOtp(context.Background(), "challenge-1", test.phone, test.code, ""); err != nil {
				t.Fatalf("dev demo code should be accepted: %v", err)
			}
		})
	}

	for _, test := range []struct {
		phone string
		code  string
	}{
		{phone: devBypassPhone, code: devBypassCode},
		{phone: devSignupPhone, code: devSignupCode},
	} {
		svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{devMode: false})
		if _, err := svc.VerifyOtp(context.Background(), "challenge-1", test.phone, test.code, ""); !errors.Is(err, pkgdomain.ErrForbidden) {
			t.Errorf("demo code for %s should be rejected outside dev mode, got %v", test.phone, err)
		}
	}
}

func TestVerifyOtpValidatesItsArguments(t *testing.T) {
	svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{acceptCode: "123456"})

	tests := []struct {
		name                   string
		challenge, phone, code string
	}{
		{"no challenge", "", testPhone, "123456"},
		{"no code", "challenge-1", testPhone, ""},
		{"bad phone", "challenge-1", "9876543210", "123456"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.VerifyOtp(context.Background(), tt.challenge, tt.phone, tt.code, "")
			if !errors.Is(err, pkgdomain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

// Regression coverage for WIRING_AUDIT_PLAN.md F-5: before this, no path in
// the product could ever mint a BUYER, CLUSTER_OFFICER or MINISTRY token --
// every one of the 20 bff handlers gating on those roles was unreachable by
// any real login.
func TestVerifyOtpDevRoleMintsTheRequestedRoleWhenDevModeIsOn(t *testing.T) {
	tokens := newFakeTokens()
	svc := newTestIdentity(newFakeStore(), tokens, &fakeOTP{acceptCode: "123456", devMode: true})

	result, err := svc.VerifyOtp(context.Background(), "challenge-1", testPhone, "123456", "ministry")
	if err != nil {
		t.Fatalf("VerifyOtp: %v", err)
	}
	if !result.Registered {
		t.Error("a dev-role login should report registered=true so no client treats it as needing artisan registration")
	}
	if len(tokens.issued) != 1 || tokens.issued[0].Role != auth.RoleMinistry {
		t.Fatalf("expected one MINISTRY subject issued, got %+v", tokens.issued)
	}
	if tokens.issued[0].PhoneE164 != testPhone {
		t.Errorf("issued subject phone = %q, want %q", tokens.issued[0].PhoneE164, testPhone)
	}
}

func TestVerifyOtpDevRoleIsRejectedWhenDevModeIsOff(t *testing.T) {
	store := newFakeStore()
	tokens := newFakeTokens()
	svc := newTestIdentity(store, tokens, &fakeOTP{acceptCode: "123456", devMode: false})

	_, err := svc.VerifyOtp(context.Background(), "challenge-1", testPhone, "123456", "ministry")
	if !errors.Is(err, pkgdomain.ErrInvalidInput) {
		t.Fatalf("dev_role without dev OTP enabled should be rejected as invalid input, got %v", err)
	}
	if len(tokens.issued) != 0 {
		t.Fatal("no token should be issued when dev_role is rejected")
	}
}

func TestVerifyOtpDevRoleRejectsAnUnknownRole(t *testing.T) {
	svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{acceptCode: "123456", devMode: true})

	_, err := svc.VerifyOtp(context.Background(), "challenge-1", testPhone, "123456", "superadmin")
	if !errors.Is(err, pkgdomain.ErrInvalidInput) {
		t.Fatalf("unknown dev_role should be rejected as invalid input, got %v", err)
	}
}

func TestVerifyOtpDevRoleArtisanFallsThroughToTheOrdinaryLoginPath(t *testing.T) {
	// "artisan" is a legal dev_role value, but it's also the default --
	// requesting it explicitly must behave exactly like the ordinary path
	// (looking up the real profile), not like the short-circuit the other
	// three roles take.
	store := newFakeStore()
	tokens := newFakeTokens()
	svc := newTestIdentity(store, tokens, &fakeOTP{acceptCode: "123456", devMode: true})

	result, err := svc.VerifyOtp(context.Background(), "challenge-1", testPhone, "123456", "artisan")
	if err != nil {
		t.Fatalf("VerifyOtp: %v", err)
	}
	if result.Registered {
		t.Error("an unregistered phone should still report registered=false via the ordinary path")
	}
	if len(tokens.issued) != 1 || tokens.issued[0].Role != auth.RoleArtisan {
		t.Fatalf("expected one ARTISAN subject issued, got %+v", tokens.issued)
	}
}

func TestRefreshTokenIssuesAFreshPair(t *testing.T) {
	tokens := newFakeTokens()
	tokens.verify["good-refresh"] = &auth.Claims{
		Role:      string(auth.RoleArtisan),
		Language:  "GUJARATI",
		Kind:      string(auth.KindRefresh),
		PhoneE164: testPhone,
	}
	tokens.verify["good-refresh"].Subject = "artisan-1"

	svc := newTestIdentity(newFakeStore(), tokens, &fakeOTP{})

	pair, err := svc.RefreshToken(context.Background(), "good-refresh")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("a refreshed pair should carry an access token")
	}

	sub := tokens.issued[len(tokens.issued)-1]
	if sub.ID != "artisan-1" || sub.Role != auth.RoleArtisan || sub.PhoneE164 != testPhone {
		t.Errorf("refreshed subject = %+v", sub)
	}
}

func TestRefreshTokenRejectsAnUnknownOrExpiredToken(t *testing.T) {
	svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{})

	if _, err := svc.RefreshToken(context.Background(), ""); !errors.Is(err, pkgdomain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput for an empty token, got %v", err)
	}
	// The fake verifier rejects anything it was not given, standing in for an
	// expired or forged token.
	if _, err := svc.RefreshToken(context.Background(), "expired"); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

func TestRefreshTokenRejectsAnUnknownRole(t *testing.T) {
	tokens := newFakeTokens()
	tokens.verify["weird"] = &auth.Claims{Role: "SUPERUSER", Kind: string(auth.KindRefresh)}
	svc := newTestIdentity(newFakeStore(), tokens, &fakeOTP{})

	_, err := svc.RefreshToken(context.Background(), "weird")
	if !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
	if !strings.Contains(err.Error(), "SUPERUSER") {
		t.Errorf("error should name the bad role, got %q", err)
	}
}

func TestVerifyOtpActiveStaffPhoneMintsTheStaffRoleWithoutDevRole(t *testing.T) {
	store := newFakeStore()
	staffID := uuid.New()
	state, district := "IN-UP", "Varanasi"
	store.staff[staffID] = domain.StaffAccount{
		ID: staffID, PhoneE164: testPhone, DisplayName: "Ramesh", Role: string(auth.RoleFieldAgent),
		StateCode: &state, District: &district, Active: true,
	}
	tokens := newFakeTokens()
	// Dev mode off: this must work in a real deployment, with no dev_role.
	svc := newTestIdentity(store, tokens, &fakeOTP{acceptCode: "123456"})

	result, err := svc.VerifyOtp(context.Background(), "challenge-1", testPhone, "123456", "")
	if err != nil {
		t.Fatalf("VerifyOtp: %v", err)
	}
	if !result.Registered || result.ArtisanID != "" {
		t.Errorf("staff login result = %+v, want registered with no artisan id", result)
	}
	sub := tokens.issued[len(tokens.issued)-1]
	if sub.Role != auth.RoleFieldAgent || sub.ID != staffID.String() {
		t.Errorf("issued subject = %+v, want FIELD_AGENT %s", sub, staffID)
	}
	if sub.ScopeState != state || sub.ScopeDistrict != district {
		t.Errorf("scope = %q/%q, want %q/%q", sub.ScopeState, sub.ScopeDistrict, state, district)
	}
	if sub.PhoneE164 != "" {
		t.Error("a staff token must not carry a phone: RegisterArtisan binds to it")
	}
}

func TestVerifyOtpInactiveStaffFallsThroughToTheArtisanFlow(t *testing.T) {
	store := newFakeStore()
	staffID := uuid.New()
	store.staff[staffID] = domain.StaffAccount{ID: staffID, PhoneE164: testPhone, Role: string(auth.RoleMinistry), Active: false}
	tokens := newFakeTokens()
	svc := newTestIdentity(store, tokens, &fakeOTP{acceptCode: "123456"})

	if _, err := svc.VerifyOtp(context.Background(), "challenge-1", testPhone, "123456", ""); err != nil {
		t.Fatalf("VerifyOtp: %v", err)
	}
	if got := tokens.issued[len(tokens.issued)-1].Role; got != auth.RoleArtisan {
		t.Errorf("a deactivated staff phone logged in as %s, want ARTISAN", got)
	}
}

func TestRefreshTokenRejectsADeactivatedStaffAccount(t *testing.T) {
	store := newFakeStore()
	staffID := uuid.New()
	store.staff[staffID] = domain.StaffAccount{ID: staffID, Role: string(auth.RoleClusterOfficer), Active: false}
	tokens := newFakeTokens()
	tokens.verify["staff-refresh"] = &auth.Claims{Role: string(auth.RoleClusterOfficer), Kind: string(auth.KindRefresh)}
	tokens.verify["staff-refresh"].Subject = staffID.String()
	svc := newTestIdentity(store, tokens, &fakeOTP{})

	if _, err := svc.RefreshToken(context.Background(), "staff-refresh"); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden for a deactivated staffer, got %v", err)
	}

	store.staff[staffID] = domain.StaffAccount{ID: staffID, Role: string(auth.RoleClusterOfficer), Active: true}
	if _, err := svc.RefreshToken(context.Background(), "staff-refresh"); err != nil {
		t.Fatalf("an active staffer should refresh: %v", err)
	}
}
