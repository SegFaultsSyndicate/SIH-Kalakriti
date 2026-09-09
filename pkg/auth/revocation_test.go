// pkg/auth/revocation_test.go
package auth

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// These dial a real, empty-database local Redis, matching this repo's
// existing convention for Redis-backed tests (see core-svc's otp_test.go) —
// CI runs a redis service container; a local run without one will fail to
// dial, same as those tests do.

func TestTryBurnJTIRejectsReplay(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{})
	defer rdb.Close()
	store := NewRevocationStore(rdb)
	ctx := context.Background()
	jti := "jti-" + time.Now().Format(time.RFC3339Nano)

	fresh, err := store.TryBurnJTI(ctx, jti, time.Minute)
	if err != nil {
		t.Fatalf("first burn returned error: %v", err)
	}
	if !fresh {
		t.Fatal("first burn should report fresh=true")
	}

	fresh, err = store.TryBurnJTI(ctx, jti, time.Minute)
	if err != nil {
		t.Fatalf("second burn returned error: %v", err)
	}
	if fresh {
		t.Fatal("second burn of the same jti should report fresh=false (replay)")
	}
}

func TestRevokeSubjectInvalidatesOlderTokensOnly(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{})
	defer rdb.Close()
	store := NewRevocationStore(rdb)
	ctx := context.Background()
	subject := "subject-" + time.Now().Format(time.RFC3339Nano)

	before := time.Now().UTC()
	revoked, err := store.RevokedBefore(ctx, subject, before)
	if err != nil {
		t.Fatalf("RevokedBefore returned error before any revocation: %v", err)
	}
	if revoked {
		t.Fatal("a subject with no revocation on record should never report revoked")
	}

	time.Sleep(1100 * time.Millisecond) // Redis epoch has 1s resolution
	if err := store.RevokeSubject(ctx, subject, time.Minute); err != nil {
		t.Fatalf("RevokeSubject returned error: %v", err)
	}

	revoked, err = store.RevokedBefore(ctx, subject, before)
	if err != nil {
		t.Fatalf("RevokedBefore returned error: %v", err)
	}
	if !revoked {
		t.Fatal("a token issued before RevokeSubject should be reported revoked")
	}

	after := time.Now().UTC().Add(time.Hour)
	revoked, err = store.RevokedBefore(ctx, subject, after)
	if err != nil {
		t.Fatalf("RevokedBefore returned error: %v", err)
	}
	if revoked {
		t.Fatal("a token issued after RevokeSubject should not be reported revoked")
	}
}
