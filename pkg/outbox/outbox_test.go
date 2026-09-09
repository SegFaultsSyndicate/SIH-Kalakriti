// pkg/outbox/outbox_test.go
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeEnqueuer records InsertOutbox calls in memory, standing in for a
// sqlc-generated repo method inside a caller's transaction.
type fakeEnqueuer struct {
	mu   sync.Mutex
	rows []Message
	err  error
}

func (f *fakeEnqueuer) InsertOutbox(_ context.Context, id, aggregateID, topic, idempotencyKey string, payload []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.rows = append(f.rows, Message{ID: id, AggregateID: aggregateID, Topic: topic, Payload: payload})
	return nil
}

func TestEnqueueEncodesAndInserts(t *testing.T) {
	enq := &fakeEnqueuer{}
	payload := map[string]string{"listing_id": "abc-123"}

	if err := Enqueue(context.Background(), enq, "row-1", "agg-1", "catalog.listing_published", "idem-1", payload); err != nil {
		t.Fatalf("Enqueue returned error: %v", err)
	}

	if len(enq.rows) != 1 {
		t.Fatalf("expected 1 inserted row, got %d", len(enq.rows))
	}
	got := enq.rows[0]
	if got.Topic != "catalog.listing_published" || got.AggregateID != "agg-1" || got.ID != "row-1" {
		t.Fatalf("unexpected row: %+v", got)
	}
	var decoded map[string]string
	if err := json.Unmarshal(got.Payload, &decoded); err != nil {
		t.Fatalf("payload did not round-trip as JSON: %v", err)
	}
	if decoded["listing_id"] != "abc-123" {
		t.Fatalf("payload content mismatch: %+v", decoded)
	}
}

func TestEnqueuePropagatesInsertError(t *testing.T) {
	wantErr := errors.New("db down")
	enq := &fakeEnqueuer{err: wantErr}

	err := Enqueue(context.Background(), enq, "row-1", "agg-1", "catalog.listing_published", "idem-1", map[string]string{})
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestEnqueueRejectsUnmarshallablePayload(t *testing.T) {
	enq := &fakeEnqueuer{}
	// channels are not JSON-marshalable.
	err := Enqueue(context.Background(), enq, "row-1", "agg-1", "topic", "idem-1", make(chan int))
	if err == nil {
		t.Fatal("expected an encoding error, got nil")
	}
	if len(enq.rows) != 0 {
		t.Fatal("insert should not have been called after encoding failure")
	}
}

// fakeStore is an in-memory Store + Publisher used to exercise Relay.tick
// without a real database or Kafka broker. Its RunClaimed holds f.mu for the
// whole call, standing in for a real held-open DB transaction: no other
// claim can observe the rows mid-fn, matching the guarantee the real
// pgx-transaction-backed implementations provide.
type fakeStore struct {
	mu        sync.Mutex
	rows      []Message
	published []string
}

func (f *fakeStore) RunClaimed(ctx context.Context, limit int, fn func(ctx context.Context, rows []Message) ([]string, error)) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var claimed []Message
	if len(f.rows) > limit {
		claimed = append([]Message(nil), f.rows[:limit]...)
	} else {
		claimed = append([]Message(nil), f.rows...)
	}

	ids, fnErr := fn(ctx, claimed)

	if len(ids) > 0 {
		f.published = append(f.published, ids...)
		remaining := f.rows[:0]
		for _, r := range f.rows {
			keep := true
			for _, id := range ids {
				if r.ID == id {
					keep = false
					break
				}
			}
			if keep {
				remaining = append(remaining, r)
			}
		}
		f.rows = remaining
	}

	return fnErr
}

type fakePublisher struct {
	mu        sync.Mutex
	published []Message
	failOn    string // AggregateID that should fail to publish, once
	failed    bool
}

func (f *fakePublisher) Publish(_ context.Context, topic, aggregateID string, payload []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failOn != "" && aggregateID == f.failOn && !f.failed {
		f.failed = true
		return errors.New("broker unavailable")
	}
	f.published = append(f.published, Message{AggregateID: aggregateID, Topic: topic, Payload: payload})
	return nil
}

func TestRelayTickPublishesAndMarksAllRows(t *testing.T) {
	store := &fakeStore{rows: []Message{
		{ID: "1", AggregateID: "a", Topic: "t", Payload: []byte("{}")},
		{ID: "2", AggregateID: "b", Topic: "t", Payload: []byte("{}")},
	}}
	pub := &fakePublisher{}
	relay := NewRelay(store, pub, RelayConfig{}, nil)

	n, err := relay.tick(context.Background())
	if err != nil {
		t.Fatalf("tick returned error: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 published, got %d", n)
	}
	if len(store.rows) != 0 {
		t.Fatalf("expected all rows marked published, %d remain", len(store.rows))
	}
	if len(store.published) != 2 {
		t.Fatalf("expected MarkPublished called with 2 ids, got %d", len(store.published))
	}
}

func TestRelayTickMarksPartialBatchPublishedBeforeReturningError(t *testing.T) {
	store := &fakeStore{rows: []Message{
		{ID: "1", AggregateID: "a", Topic: "t", Payload: []byte("{}")},
		{ID: "2", AggregateID: "fails", Topic: "t", Payload: []byte("{}")},
		{ID: "3", AggregateID: "c", Topic: "t", Payload: []byte("{}")},
	}}
	pub := &fakePublisher{failOn: "fails"}
	relay := NewRelay(store, pub, RelayConfig{}, nil)

	_, err := relay.tick(context.Background())
	if err == nil {
		t.Fatal("expected an error from the failing publish")
	}
	// Row "1" published before the failure must be marked published even
	// though the batch as a whole errored, so a retry never double-publishes it.
	if len(store.published) != 1 || store.published[0] != "1" {
		t.Fatalf("expected row 1 marked published despite mid-batch failure, got %+v", store.published)
	}
	// Rows 2 and 3 remain unpublished for the next tick to retry.
	if len(store.rows) != 2 {
		t.Fatalf("expected 2 rows still pending, got %d", len(store.rows))
	}
}

func TestRelayTickNoRowsIsNotAnError(t *testing.T) {
	store := &fakeStore{}
	pub := &fakePublisher{}
	relay := NewRelay(store, pub, RelayConfig{}, nil)

	n, err := relay.tick(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("expected (0, nil) on an empty batch, got (%d, %v)", n, err)
	}
}

func TestRelayRunStopsOnContextCancel(t *testing.T) {
	store := &fakeStore{}
	pub := &fakePublisher{}
	relay := NewRelay(store, pub, RelayConfig{PollInterval: time.Millisecond}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		relay.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Relay.Run did not return after context cancellation")
	}
}

func TestNewRelayAppliesDefaults(t *testing.T) {
	relay := NewRelay(&fakeStore{}, &fakePublisher{}, RelayConfig{}, nil)
	if relay.cfg.BatchSize != defaultBatchSize {
		t.Errorf("BatchSize default = %d, want %d", relay.cfg.BatchSize, defaultBatchSize)
	}
	if relay.cfg.PollInterval != defaultPollInterval {
		t.Errorf("PollInterval default = %v, want %v", relay.cfg.PollInterval, defaultPollInterval)
	}
	if relay.cfg.MaxBackoff != defaultMaxBackoff {
		t.Errorf("MaxBackoff default = %v, want %v", relay.cfg.MaxBackoff, defaultMaxBackoff)
	}
	if relay.onError == nil {
		t.Error("onError should default to a non-nil no-op")
	}
}
