// services/core-svc/internal/core/service/auth_test.go
package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
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
	if _, err := svc.RegisterArtisan(artisanPhoneCtx("", testPhone), validRegisterInput(), "idem-1"); err != nil {
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

	result, err := svc.VerifyOtp(context.Background(), "challenge-1", testPhone, "123456")
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

	created, err := svc.RegisterArtisan(artisanPhoneCtx("", testPhone), validRegisterInput(), "idem-1")
	if err != nil {
		t.Fatalf("RegisterArtisan: %v", err)
	}

	result, err := svc.VerifyOtp(context.Background(), "challenge-1", testPhone, "123456")
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

	_, err := svc.VerifyOtp(context.Background(), "challenge-1", testPhone, "999999")
	if !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
	if len(tokens.issued) != 0 {
		t.Fatal("no token may be issued for a failed verification")
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
			_, err := svc.VerifyOtp(context.Background(), tt.challenge, tt.phone, tt.code)
			if !errors.Is(err, pkgdomain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
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
