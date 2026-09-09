// pkg/auth/revocation.go
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RevocationStore tracks two independent things in Redis: which refresh
// token jtis have already been redeemed (single-token, one-time use), and
// per-subject revocation epochs ("sign out everywhere" — every token issued
// before the epoch is invalid). Both are keyed off data already on Claims
// (ID and Subject), so nothing new needs to travel in the token itself.
type RevocationStore struct {
	client redis.Cmdable
}

// NewRevocationStore builds a store over an existing Redis client.
func NewRevocationStore(client redis.Cmdable) *RevocationStore {
	return &RevocationStore{client: client}
}

func jtiKey(jti string) string              { return "auth:jti_burned:" + jti }
func revokedSinceKey(subject string) string { return "auth:revoked_since:" + subject }

// TryBurnJTI atomically marks jti as spent for the rest of ttl (its token's
// own remaining life — once the token would have expired naturally anyway,
// there is nothing left to guard against) and reports whether this call was
// the one that burned it. fresh is false when jti was already burned, which
// means the presented refresh token is being replayed — the caller should
// reject the request rather than issue a new pair. Using SETNX for the
// check-and-burn in one round trip is what closes the race two concurrent
// refresh calls for the same token would otherwise have: a separate
// "is it burned" read followed by a "burn it" write lets both callers pass
// the read before either write lands.
func (s *RevocationStore) TryBurnJTI(ctx context.Context, jti string, ttl time.Duration) (fresh bool, err error) {
	ok, err := s.client.SetNX(ctx, jtiKey(jti), 1, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("auth: burning jti: %w", err)
	}
	return ok, nil
}

// RevokeSubject invalidates every token already issued to subject as of now
// — "sign out everywhere" — for the rest of ttl. Pass at least the longest
// token TTL in play (RefreshTTL) so the epoch outlives anything it needs to
// invalidate.
func (s *RevocationStore) RevokeSubject(ctx context.Context, subject string, ttl time.Duration) error {
	if err := s.client.Set(ctx, revokedSinceKey(subject), time.Now().UTC().Unix(), ttl).Err(); err != nil {
		return fmt.Errorf("auth: revoking subject: %w", err)
	}
	return nil
}

// RevokedBefore reports whether subject has an active revocation epoch
// later than issuedAt — i.e. whether a token minted at issuedAt was
// invalidated by a later RevokeSubject call. No epoch on record at all
// means nothing has ever been revoked for this subject.
func (s *RevocationStore) RevokedBefore(ctx context.Context, subject string, issuedAt time.Time) (bool, error) {
	v, err := s.client.Get(ctx, revokedSinceKey(subject)).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("auth: checking subject revocation: %w", err)
	}
	return issuedAt.Before(time.Unix(v, 0)), nil
}
