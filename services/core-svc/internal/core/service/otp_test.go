// services/core-svc/internal/core/service/otp_test.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
)

// recordingSender captures what the provider was asked to deliver.
type recordingSender struct {
	sent []struct{ Phone, Code, Language string }
	err  error
}

func (s *recordingSender) Send(_ context.Context, phone, code, language string) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, struct{ Phone, Code, Language string }{phone, code, language})
	return nil
}

func newOTPHarness(t *testing.T, devMode bool) (*OTPService, *recordingSender, *redis.Client) {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{})
	sender := &recordingSender{}
	svc := NewOTPService(rdb, sender, OTPConfig{DevMode: devMode, MaxAttempts: 3}, discardLogger())
	return svc, sender, rdb
}

func TestOTPRequestStoresChallengeAndSendsCode(t *testing.T) {
	svc, sender, rdb := newOTPHarness(t, false)

	ch, err := svc.Request(context.Background(), testPhone, "GUJARATI")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if ch.ID == "" {
		t.Fatal("challenge id should be set")
	}
	if ch.DevMode {
		t.Error("DevMode should be false when not configured")
	}
	if !ch.ExpiresAt.After(time.Now()) {
		t.Error("challenge should expire in the future")
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 send, got %d", len(sender.sent))
	}
	if sender.sent[0].Phone != testPhone || sender.sent[0].Language != "GUJARATI" {
		t.Errorf("sent = %+v", sender.sent[0])
	}
	if len(sender.sent[0].Code) != otpCodeLength {
		t.Errorf("code %q should be %d digits", sender.sent[0].Code, otpCodeLength)
	}

	// The challenge must actually be stored, bound to the phone.
	raw, err := rdb.Get(context.Background(), challengeKey(ch.ID)).Bytes()
	if err != nil {
		t.Fatalf("challenge was not stored: %v", err)
	}
	var stored otpChallenge
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("stored challenge is not decodable: %v", err)
	}
	if stored.PhoneE164 != testPhone {
		t.Errorf("stored phone = %q", stored.PhoneE164)
	}
}

func TestOTPRequestGeneratesDistinctCodes(t *testing.T) {
	svc, sender, _ := newOTPHarness(t, false)

	seen := map[string]int{}
	for i := 0; i < 40; i++ {
		if _, err := svc.Request(context.Background(), testPhone, "HINDI"); err != nil {
			t.Fatalf("Request: %v", err)
		}
	}
	for _, s := range sender.sent {
		seen[s.Code]++
	}
	// A generator stuck on one value would be a catastrophic auth bug; 40 draws
	// from 10^6 should essentially never collide into a handful of values.
	if len(seen) < 30 {
		t.Fatalf("only %d distinct codes across 40 requests — the generator looks broken", len(seen))
	}
}

func TestOTPRequestDropsTheChallengeWhenSendingFails(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{})
	sender := &recordingSender{err: errors.New("provider down")}
	svc := NewOTPService(rdb, sender, OTPConfig{}, discardLogger())

	if _, err := svc.Request(context.Background(), testPhone, "HINDI"); err == nil {
		t.Fatal("expected the send failure to surface")
	}
	// No orphan challenge may be left behind: nobody could ever redeem it.
	if n := rdb.Del(context.Background(), challengeKey("any")).Val(); n != 0 {
		t.Error("unexpected leftover key")
	}
}

func TestOTPVerifyAcceptsTheRealCodeOnceOnly(t *testing.T) {
	svc, sender, _ := newOTPHarness(t, false)

	ch, err := svc.Request(context.Background(), testPhone, "HINDI")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	code := sender.sent[0].Code

	if err := svc.Verify(context.Background(), ch.ID, testPhone, code); err != nil {
		t.Fatalf("the real code should verify: %v", err)
	}
	// Redeeming burns the challenge, so a replay fails.
	if err := svc.Verify(context.Background(), ch.ID, testPhone, code); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("a redeemed challenge must not be replayable, got %v", err)
	}
}

func TestOTPVerifyRejectsAChallengeBoundToAnotherPhone(t *testing.T) {
	svc, sender, _ := newOTPHarness(t, false)

	ch, err := svc.Request(context.Background(), testPhone, "HINDI")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	code := sender.sent[0].Code

	// A stolen challenge id must not be redeemable against a different number.
	err = svc.Verify(context.Background(), ch.ID, "+919000000009", code)
	if !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

func TestOTPVerifyBurnsTheChallengeAfterTooManyAttempts(t *testing.T) {
	svc, sender, _ := newOTPHarness(t, false) // MaxAttempts: 3

	ch, err := svc.Request(context.Background(), testPhone, "HINDI")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	code := sender.sent[0].Code
	wrong := "000001"
	if wrong == code {
		wrong = "000002"
	}

	// Attempts 1 and 2 fail but leave the challenge alive.
	for i := 0; i < 2; i++ {
		if err := svc.Verify(context.Background(), ch.ID, testPhone, wrong); !errors.Is(err, pkgdomain.ErrForbidden) {
			t.Fatalf("attempt %d: want ErrForbidden, got %v", i+1, err)
		}
	}
	// The third wrong guess spends the budget and burns the challenge.
	if err := svc.Verify(context.Background(), ch.ID, testPhone, wrong); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
	// Even the correct code is now useless.
	if err := svc.Verify(context.Background(), ch.ID, testPhone, code); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatal("a burned challenge must not accept the correct code")
	}
}

func TestOTPVerifyRejectsAnUnknownChallenge(t *testing.T) {
	svc, _, _ := newOTPHarness(t, false)
	err := svc.Verify(context.Background(), "no-such-challenge", testPhone, "123456")
	if !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

func TestOTPDevModeAcceptsTheWellKnownCode(t *testing.T) {
	svc, sender, _ := newOTPHarness(t, true)

	ch, err := svc.Request(context.Background(), testPhone, "HINDI")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if !ch.DevMode {
		t.Error("DevMode should be reported to the caller")
	}
	if !svc.DevMode() {
		t.Error("DevMode() should report true")
	}

	// The well-known code works...
	if err := svc.Verify(context.Background(), ch.ID, testPhone, DevOTPCode); err != nil {
		t.Fatalf("the development code should be accepted in dev mode: %v", err)
	}

	// ...and so does the real one, on a fresh challenge.
	ch2, err := svc.Request(context.Background(), testPhone, "HINDI")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	realCode := sender.sent[len(sender.sent)-1].Code
	if err := svc.Verify(context.Background(), ch2.ID, testPhone, realCode); err != nil {
		t.Fatalf("the real code should still work in dev mode: %v", err)
	}
}

func TestOTPDevCodeIsRejectedWhenDevModeIsOff(t *testing.T) {
	svc, sender, _ := newOTPHarness(t, false)

	ch, err := svc.Request(context.Background(), testPhone, "HINDI")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	// Guard against the astronomically unlikely case that the random code is
	// literally the dev code, which would make this test meaningless.
	if sender.sent[0].Code == DevOTPCode {
		t.Skip("random code collided with the development code")
	}

	err = svc.Verify(context.Background(), ch.ID, testPhone, DevOTPCode)
	if !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("the development code must be refused when dev mode is off, got %v", err)
	}
}

func TestNewOTPServiceAppliesDefaults(t *testing.T) {
	svc := NewOTPService(redis.NewClient(&redis.Options{}), &recordingSender{}, OTPConfig{}, discardLogger())
	if svc.cfg.TTL != defaultOTPTTL {
		t.Errorf("TTL = %v, want %v", svc.cfg.TTL, defaultOTPTTL)
	}
	if svc.cfg.MaxAttempts != defaultOTPMaxAttempts {
		t.Errorf("MaxAttempts = %d, want %d", svc.cfg.MaxAttempts, defaultOTPMaxAttempts)
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct{ in, want string }{
		{"+919876543210", "****3210"},
		{"1234", "****"},
		{"", "****"},
		{"12345", "****2345"},
	}
	for _, tt := range tests {
		if got := maskPhone(tt.in); got != tt.want {
			t.Errorf("maskPhone(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGenerateOTPCodeIsAlwaysSixDigits(t *testing.T) {
	for i := 0; i < 200; i++ {
		code, err := generateOTPCode()
		if err != nil {
			t.Fatalf("generateOTPCode: %v", err)
		}
		if len(code) != otpCodeLength {
			t.Fatalf("code %q has length %d, want %d", code, len(code), otpCodeLength)
		}
		for _, r := range code {
			if r < '0' || r > '9' {
				t.Fatalf("code %q contains a non-digit", code)
			}
		}
	}
}
