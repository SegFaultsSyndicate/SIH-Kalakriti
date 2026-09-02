// services/bff/internal/bff/client/auth.go
package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"
	pkgredis "github.com/ZoroNewbie00/kalakriti/pkg/redis"
)

// challengeTTL bounds how long a requested OTP challenge stays redeemable in
// the cache, used only when RequestOtp's own expiry can't be read.
const challengeTTL = 10 * time.Minute

// Auth is bff's view of core-svc's OTP login flow.
//
// RequestOTP and VerifyOTP are two separate HTTP calls, but VerifyOtp's RPC
// needs the challenge_id RequestOtp minted, and handler.AuthService's
// signature has nowhere to carry it between the two. This caches it in Redis
// keyed by phone, entirely inside the adapter — the interface and HTTP
// contract are unchanged.
type Auth struct {
	identity  identityv1.IdentityServiceClient
	challenge *pkgredis.Cache[string]
}

// NewAuth builds the auth client, sharing conn with core-svc's other services.
func NewAuth(conn grpc.ClientConnInterface, rdb redis.Cmdable) *Auth {
	return &Auth{
		identity:  identityv1.NewIdentityServiceClient(conn),
		challenge: pkgredis.NewCache[string](rdb, "otp_challenge"),
	}
}

// RequestOTP sends a login code and remembers the challenge it opened.
// RequestOtp is one of core-svc's three public RPCs, so ctx carries no
// bearer token here — there's no session to forward yet.
func (a *Auth) RequestOTP(ctx context.Context, phone string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := a.identity.RequestOtp(ctx, &identityv1.RequestOtpRequest{
		PhoneE164: phone,
		Language:  commonv1.Language_LANGUAGE_ENGLISH,
	})
	if err != nil {
		return grpcErr(err)
	}

	ttl := challengeTTL
	if exp := resp.GetExpiresAt(); exp != nil {
		if d := time.Until(exp.AsTime()); d > 0 {
			ttl = d
		}
	}
	if err := a.challenge.Set(ctx, phone, resp.GetChallengeId(), ttl); err != nil {
		return fmt.Errorf("caching otp challenge: %w", err)
	}
	return nil
}

// VerifyOTP redeems the challenge RequestOTP opened for this phone number.
func (a *Auth) VerifyOTP(ctx context.Context, phone, otp string) (accessToken, refreshToken string, err error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	challengeID, err := a.challenge.Get(ctx, phone)
	if err != nil {
		if errors.Is(err, pkgredis.ErrCacheMiss) {
			return "", "", domain.InvalidInput("no OTP challenge pending for this phone number, request a new code")
		}
		return "", "", domain.Unavailable("otp challenge cache unavailable")
	}

	resp, err := a.identity.VerifyOtp(ctx, &identityv1.VerifyOtpRequest{
		ChallengeId: challengeID,
		PhoneE164:   phone,
		Code:        otp,
	})
	if err != nil {
		return "", "", grpcErr(err)
	}
	_ = a.challenge.Delete(ctx, phone) // best-effort; a stale entry just expires on its own TTL

	tokens := resp.GetTokens()
	return tokens.GetAccessToken(), tokens.GetRefreshToken(), nil
}

// RefreshToken exchanges a valid refresh token for a fresh pair.
func (a *Auth) RefreshToken(ctx context.Context, refreshToken string) (accessToken string, err error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := a.identity.RefreshToken(ctx, &identityv1.RefreshTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		return "", grpcErr(err)
	}
	return resp.GetTokens().GetAccessToken(), nil
}
