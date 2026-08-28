// pkg/idempotency/idempotency_test.go
package idempotency

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/segfaultsyndicate/kalakriti/pkg/domain"
)

// fakeStore is an in-memory Store for tests; GetOrInsert is guarded by a mutex to
// mimic the atomicity a real INSERT ... ON CONFLICT gives.
type fakeStore struct {
	mu   sync.Mutex
	rows map[string]Record
}

func newFakeStore() *fakeStore { return &fakeStore{rows: map[string]Record{}} }

func (s *fakeStore) GetOrInsert(_ context.Context, scope, key, requestHash string, _ time.Time) (Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := scope + "/" + key
	if rec, ok := s.rows[k]; ok {
		return rec, false, nil
	}
	rec := Record{RequestHash: requestHash}
	s.rows[k] = rec
	return rec, true, nil
}

func (s *fakeStore) SaveResponse(_ context.Context, scope, key string, response []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := scope + "/" + key
	rec, ok := s.rows[k]
	if !ok {
		return errors.New("no such key")
	}
	rec.Response = response
	s.rows[k] = rec
	return nil
}

type result struct {
	Value string `json:"value"`
}

func TestDoFirstCallRunsFn(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	calls := 0
	fn := func(context.Context) (result, error) {
		calls++
		return result{Value: "computed"}, nil
	}

	got, err := Do(context.Background(), store, "listing.create", "key-1", "hash-a", fn)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got.Value != "computed" || calls != 1 {
		t.Errorf("got %+v, calls=%d, want computed/1", got, calls)
	}
}

func TestDoReplaySameHashReturnsStoredResultWithoutRerunning(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	calls := 0
	fn := func(context.Context) (result, error) {
		calls++
		return result{Value: "computed"}, nil
	}

	first, err := Do(context.Background(), store, "listing.create", "key-1", "hash-a", fn)
	if err != nil {
		t.Fatalf("first Do: %v", err)
	}
	second, err := Do(context.Background(), store, "listing.create", "key-1", "hash-a", fn)
	if err != nil {
		t.Fatalf("second Do: %v", err)
	}
	if calls != 1 {
		t.Errorf("fn called %d times, want 1", calls)
	}
	if first != second {
		t.Errorf("first %+v != second %+v", first, second)
	}
}

func TestDoReplayDifferentHashReturnsConflict(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	fn := func(context.Context) (result, error) { return result{Value: "computed"}, nil }

	if _, err := Do(context.Background(), store, "listing.create", "key-1", "hash-a", fn); err != nil {
		t.Fatalf("first Do: %v", err)
	}
	_, err := Do(context.Background(), store, "listing.create", "key-1", "hash-b", fn)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("Do with different hash = %v, want domain.ErrConflict", err)
	}
}

func TestDoInFlightReplayReturnsUnavailable(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	// Simulate a key inserted by a first caller that has not yet saved a response.
	if _, _, err := store.GetOrInsert(context.Background(), "order.create", "key-2", "hash-a", time.Now().Add(TTL)); err != nil {
		t.Fatalf("seeding store: %v", err)
	}

	fn := func(context.Context) (result, error) {
		t.Fatal("fn should not run while the key is still in flight")
		return result{}, nil
	}
	_, err := Do(context.Background(), store, "order.create", "key-2", "hash-a", fn)
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("Do while in flight = %v, want domain.ErrUnavailable", err)
	}
}

func TestDoFnErrorIsNotStored(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	boom := errors.New("boom")
	calls := 0
	fn := func(context.Context) (result, error) {
		calls++
		if calls == 1 {
			return result{}, boom
		}
		return result{Value: "computed"}, nil
	}

	_, err := Do(context.Background(), store, "listing.create", "key-3", "hash-a", fn)
	if !errors.Is(err, boom) {
		t.Fatalf("first Do error = %v, want boom", err)
	}

	// A retry after a failed first attempt is not a replay of a completed call:
	// the key is still unresolved, so it is correctly rejected as in-flight
	// rather than silently re-running side effects.
	_, err = Do(context.Background(), store, "listing.create", "key-3", "hash-a", fn)
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("second Do error = %v, want domain.ErrUnavailable", err)
	}
	if calls != 1 {
		t.Errorf("fn called %d times, want 1", calls)
	}
}

func TestHash256Hex(t *testing.T) {
	t.Parallel()
	got := Hash256Hex([]byte("hello"))
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Errorf("Hash256Hex(%q) = %q, want %q", "hello", got, want)
	}
}
