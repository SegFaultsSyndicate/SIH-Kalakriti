// pkg/outbox/outbox.go

// Package outbox implements the transactional outbox pattern: a write and the
// event it produces commit atomically in one SQL transaction, and a separate
// Relay publishes the event to Kafka afterwards. This is what makes "every
// Kafka-producing write" safe against a crash between the DB commit and the
// publish. Like pkg/idempotency, this package has no SQL of its own — a service
// adapts its sqlc-generated outbox queries (migrations/queries/outbox.sql) to
// the Enqueuer and Store interfaces below.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Message is one row read back off the outbox table for publishing.
type Message struct {
	ID          string
	AggregateID string
	Topic       string
	Payload     []byte
}

// Enqueuer is what Enqueue needs from the caller's transaction. A service
// satisfies it by wrapping its sqlc *Queries with a tx already attached.
type Enqueuer interface {
	InsertOutbox(ctx context.Context, id, aggregateID, topic, idempotencyKey string, payload []byte) error
}

// Enqueue writes topic/payload into the outbox using tx, so the row commits or
// rolls back atomically with whatever business write tx also contains. It never
// talks to Kafka directly — that is the Relay's job, run out-of-band so a
// message never publishes before the transaction that produced it has committed.
func Enqueue(ctx context.Context, tx Enqueuer, id, aggregateID, topic, idempotencyKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding outbox payload for %s: %w", topic, err)
	}
	if err := tx.InsertOutbox(ctx, id, aggregateID, topic, idempotencyKey, body); err != nil {
		return fmt.Errorf("enqueueing outbox row for %s: %w", topic, err)
	}
	return nil
}

// Publisher is what the Relay needs to actually deliver a message.
type Publisher interface {
	Publish(ctx context.Context, topic, aggregateID string, payload []byte) error
}

// Store is what the Relay needs from persistence.
type Store interface {
	// FetchUnpublished claims up to limit unpublished rows (a FOR UPDATE SKIP
	// LOCKED select) so several Relay instances can run against the same table
	// without duplicating work.
	FetchUnpublished(ctx context.Context, limit int) ([]Message, error)
	// MarkPublished stamps published_at for the given row ids.
	MarkPublished(ctx context.Context, ids []string) error
}

const (
	defaultBatchSize    = 100
	defaultPollInterval = 500 * time.Millisecond
	defaultMaxBackoff   = 30 * time.Second
)

// RelayConfig controls the poller.
type RelayConfig struct {
	// BatchSize and PollInterval default to 100 and 500ms when left zero.
	BatchSize    int
	PollInterval time.Duration
	// MaxBackoff caps how long a run of consecutive failures backs off to,
	// defaulting to 30s when left zero.
	MaxBackoff time.Duration
}

// Relay polls Store for unpublished rows and publishes them via Publisher. It is
// deliberately dumb: no batching optimisation beyond one SELECT per tick, no
// exactly-once — Kafka delivery here is at-least-once, and consumers dedupe on
// the envelope's idempotency_key the same way pkg/kafka's consumer dedupes on
// commit.
type Relay struct {
	store     Store
	publisher Publisher
	cfg       RelayConfig
	onError   func(error)
}

// NewRelay builds a Relay. onError is called with every failed tick (a failed
// fetch or a failed publish); pass a logging func, or nil to ignore.
func NewRelay(store Store, publisher Publisher, cfg RelayConfig, onError func(error)) *Relay {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultBatchSize
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = defaultMaxBackoff
	}
	if onError == nil {
		onError = func(error) {}
	}
	return &Relay{store: store, publisher: publisher, cfg: cfg, onError: onError}
}

// Run polls until ctx is cancelled. A tick that errors backs off exponentially
// from PollInterval up to MaxBackoff; a successful tick resets the backoff.
func (r *Relay) Run(ctx context.Context) {
	backoff := r.cfg.PollInterval
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		n, err := r.tick(ctx)
		if err != nil {
			r.onError(err)
			backoff *= 2
			if backoff > r.cfg.MaxBackoff {
				backoff = r.cfg.MaxBackoff
			}
			continue
		}
		backoff = r.cfg.PollInterval
		if n == 0 {
			continue // nothing to do; next tick still waits a full PollInterval
		}
	}
}

// tick publishes one batch and reports how many rows it published.
func (r *Relay) tick(ctx context.Context) (int, error) {
	rows, err := r.store.FetchUnpublished(ctx, r.cfg.BatchSize)
	if err != nil {
		return 0, fmt.Errorf("fetching unpublished outbox rows: %w", err)
	}
	if len(rows) == 0 {
		return 0, nil
	}

	published := make([]string, 0, len(rows))
	for _, row := range rows {
		if err := r.publisher.Publish(ctx, row.Topic, row.AggregateID, row.Payload); err != nil {
			// Stop at the first failure: rows are claimed FOR UPDATE SKIP LOCKED, so
			// leaving the rest unpublished just means the next tick retries them
			// (possibly from a different Relay instance) rather than reordering
			// publishes within one aggregate's partition.
			if len(published) > 0 {
				if markErr := r.store.MarkPublished(ctx, published); markErr != nil {
					return 0, fmt.Errorf("marking published after a mid-batch failure: %w", markErr)
				}
			}
			return 0, fmt.Errorf("publishing outbox row %s to %s: %w", row.ID, row.Topic, err)
		}
		published = append(published, row.ID)
	}

	if err := r.store.MarkPublished(ctx, published); err != nil {
		return 0, fmt.Errorf("marking %d rows published: %w", len(published), err)
	}
	return len(published), nil
}
