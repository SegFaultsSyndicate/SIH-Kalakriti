// services/core-svc/internal/core/service/otp.go
package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
	"github.com/segfaultsyndicate/kalakriti/pkg/ids"
)

// DevOTPCode is the well-known code the stub provider accepts. It works only
// when OTPConfig.DevMode is set, which is driven by AUTH_DEV_OTP_ENABLED and must
// never be true outside local and demo builds.
const DevOTPCode = "000000"

// otpCodeLength is the number of digits in a generated code.
const otpCodeLength = 6

// OTPSender delivers a one-time code to a phone number. This is the seam a real
// SMS provider plugs into: implement Send against Gupshup, MSG91, Twilio or the
// government's own gateway and nothing else in the service changes.
type OTPSender interface {
	// Send delivers code to phone, in the given language where the provider
	// supports it. Returning an error fails the RequestOtp call.
	Send(ctx context.Context, phone, code, language string) error
}

// LoggingOTPSender is the stub provider. It sends nothing: it logs that a code
// would have been sent, and in dev mode logs the code itself so a developer or
// demo operator can complete the flow. It is deliberately the only
// implementation shipped, so there is no chance of accidentally wiring a real
// SMS bill into a hackathon build.
type LoggingOTPSender struct {
	log     *slog.Logger
	devMode bool
}

// NewLoggingOTPSender builds the stub sender.
func NewLoggingOTPSender(log *slog.Logger, devMode bool) *LoggingOTPSender {
	return &LoggingOTPSender{log: log, devMode: devMode}
}

// Send logs the delivery instead of performing it.
func (s *LoggingOTPSender) Send(ctx context.Context, phone, code, language string) error {
	attrs := []any{"phone", maskPhone(phone), "language", language, "provider", "stub"}
	if s.devMode {
		// Only ever logged in dev mode; a real deployment must not print codes.
		attrs = append(attrs, "code", code)
	}
	s.log.InfoContext(ctx, "otp send (stubbed)", attrs...)
	return nil
}

// maskPhone keeps the last four digits so a log line is useful for support
// without printing a full phone number.
func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return "****" + phone[len(phone)-4:]
}

// OTPConfig tunes the challenge lifecycle.
type OTPConfig struct {
	// TTL is how long a challenge stays redeemable.
	TTL time.Duration
	// MaxAttempts bounds guesses against one challenge before it is burned.
	MaxAttempts int
	// DevMode makes the service accept DevOTPCode in addition to the real code.
	DevMode bool
}

// Defaults applied when a field is left zero.
const (
	defaultOTPTTL         = 5 * time.Minute
	defaultOTPMaxAttempts = 5
)

// otpChallenge is what is stored against a challenge id while it is live.
type otpChallenge struct {
	PhoneE164 string `json:"phone_e164"`
	Code      string `json:"code"`
	Attempts  int    `json:"attempts"`
}

// OTPService issues and redeems login challenges. Challenges live in Redis
// rather than Postgres: they expire in minutes, are written on every login
// attempt, and nothing needs to audit them after the fact, so a TTL key is the
// right store and keeps login traffic off the primary database.
type OTPService struct {
	redis redis.Cmdable
	send  OTPSender
	cfg   OTPConfig
	log   *slog.Logger
}

// NewOTPService builds the challenge service, filling in config defaults.
func NewOTPService(rdb redis.Cmdable, sender OTPSender, cfg OTPConfig, log *slog.Logger) *OTPService {
	if cfg.TTL <= 0 {
		cfg.TTL = defaultOTPTTL
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = defaultOTPMaxAttempts
	}
	return &OTPService{redis: rdb, send: sender, cfg: cfg, log: log}
}

// DevMode reports whether the well-known development code is accepted.
func (s *OTPService) DevMode() bool { return s.cfg.DevMode }

// Challenge is the handle returned to a caller who asked for a code.
type Challenge struct {
	ID        string
	ExpiresAt time.Time
	DevMode   bool
}

func challengeKey(id string) string { return "otp:challenge:" + id }

// Request creates a challenge and asks the provider to deliver its code.
func (s *OTPService) Request(ctx context.Context, phone, language string) (Challenge, error) {
	code, err := generateOTPCode()
	if err != nil {
		return Challenge{}, fmt.Errorf("generating otp code: %w", err)
	}

	id := ids.New().String()
	payload, err := json.Marshal(otpChallenge{PhoneE164: phone, Code: code})
	if err != nil {
		return Challenge{}, fmt.Errorf("encoding otp challenge: %w", err)
	}

	if err := s.redis.Set(ctx, challengeKey(id), payload, s.cfg.TTL).Err(); err != nil {
		return Challenge{}, fmt.Errorf("storing otp challenge: %w", err)
	}

	if err := s.send.Send(ctx, phone, code, language); err != nil {
		// The challenge is useless if the code never left, so drop it rather
		// than leaving a key nobody can redeem.
		if delErr := s.redis.Del(ctx, challengeKey(id)).Err(); delErr != nil {
			s.log.WarnContext(ctx, "could not drop otp challenge after a failed send",
				"challenge_id", id, "error", delErr)
		}
		return Challenge{}, fmt.Errorf("sending otp: %w", err)
	}

	return Challenge{
		ID:        id,
		ExpiresAt: time.Now().UTC().Add(s.cfg.TTL),
		DevMode:   s.cfg.DevMode,
	}, nil
}

// Verify redeems a challenge. A correct code burns the challenge so it cannot be
// replayed; a wrong code counts an attempt and burns the challenge once the
// attempt budget is spent.
func (s *OTPService) Verify(ctx context.Context, challengeID, phone, code string) error {
	raw, err := s.redis.Get(ctx, challengeKey(challengeID)).Bytes()
	if err != nil {
		// A missing key is an expired, already-redeemed or never-issued
		// challenge. All three are the same answer to the caller.
		return fmt.Errorf("otp challenge is not valid: %w", pkgdomain.ErrForbidden)
	}

	var ch otpChallenge
	if err := json.Unmarshal(raw, &ch); err != nil {
		return fmt.Errorf("decoding otp challenge: %w", err)
	}

	// The challenge is bound to the phone it was issued for, so a stolen
	// challenge id cannot be redeemed against a different number.
	if subtle.ConstantTimeCompare([]byte(ch.PhoneE164), []byte(phone)) != 1 {
		return fmt.Errorf("otp challenge is not valid: %w", pkgdomain.ErrForbidden)
	}

	if s.codeMatches(ch.Code, code) {
		if err := s.redis.Del(ctx, challengeKey(challengeID)).Err(); err != nil {
			s.log.WarnContext(ctx, "could not burn a redeemed otp challenge",
				"challenge_id", challengeID, "error", err)
		}
		return nil
	}

	ch.Attempts++
	if ch.Attempts >= s.cfg.MaxAttempts {
		if err := s.redis.Del(ctx, challengeKey(challengeID)).Err(); err != nil {
			s.log.WarnContext(ctx, "could not burn an exhausted otp challenge",
				"challenge_id", challengeID, "error", err)
		}
		return fmt.Errorf("otp challenge is not valid: %w", pkgdomain.ErrForbidden)
	}

	// Re-store the incremented attempt count, preserving the original TTL so a
	// wrong guess cannot extend a challenge's life.
	updated, err := json.Marshal(ch)
	if err != nil {
		return fmt.Errorf("encoding otp challenge: %w", err)
	}
	if err := s.redis.Set(ctx, challengeKey(challengeID), updated, redis.KeepTTL).Err(); err != nil {
		s.log.WarnContext(ctx, "could not record a failed otp attempt",
			"challenge_id", challengeID, "error", err)
	}
	return fmt.Errorf("otp challenge is not valid: %w", pkgdomain.ErrForbidden)
}

// codeMatches compares in constant time, and additionally accepts the well-known
// development code when dev mode is on.
func (s *OTPService) codeMatches(want, got string) bool {
	if subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1 {
		return true
	}
	if s.cfg.DevMode && subtle.ConstantTimeCompare([]byte(DevOTPCode), []byte(got)) == 1 {
		return true
	}
	return false
}

// generateOTPCode returns a uniformly random zero-padded numeric code drawn from
// crypto/rand. math/rand would make codes predictable from one observed value.
func generateOTPCode() (string, error) {
	max := big.NewInt(1)
	for i := 0; i < otpCodeLength; i++ {
		max.Mul(max, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", otpCodeLength, n), nil
}
