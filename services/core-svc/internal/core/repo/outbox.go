// services/core-svc/internal/core/repo/outbox.go
package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/outbox"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// Tx satisfies pkg/outbox.Enqueuer, so outbox.Enqueue can write an event row
// inside the same transaction as the business write it accompanies. This is the
// adapter pkg/outbox is designed around: pkg/ owns the pattern, the service owns
// the SQL.
var _ outbox.Enqueuer = (*Tx)(nil)

// InsertOutbox writes one outbox row. The query's ON CONFLICT DO NOTHING makes a
// replayed handler a no-op rather than a duplicate publish, so a zero rowcount
// is success, not an error.
func (t *Tx) InsertOutbox(ctx context.Context, id, aggregateID, topic, idempotencyKey string, payload []byte) error {
	rowID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("outbox row id %q is not a uuid: %w", id, err)
	}
	aggID, err := uuid.Parse(aggregateID)
	if err != nil {
		return fmt.Errorf("outbox aggregate id %q is not a uuid: %w", aggregateID, err)
	}

	if _, err := t.q.InsertOutbox(ctx, db.InsertOutboxParams{
		ID:             rowID,
		AggregateID:    aggID,
		Topic:          topic,
		IdempotencyKey: idempotencyKey,
		Payload:        payload,
	}); err != nil {
		return translate(err, "outbox row")
	}
	return nil
}

// OutboxStore adapts the repo to pkg/outbox.Store for the relay goroutine.
// It is a distinct type from Repo so the relay cannot reach the rest of the
// repository surface.
type OutboxStore struct {
	repo *Repo
}

// NewOutboxStore builds the relay's view of the outbox table.
func NewOutboxStore(r *Repo) *OutboxStore { return &OutboxStore{repo: r} }

// OutboxStore satisfies the relay side of pkg/outbox.
var _ outbox.Store = (*OutboxStore)(nil)

// RunClaimed claims a batch of unpublished rows with FOR UPDATE SKIP LOCKED
// and holds that claim inside one DB transaction for the lifetime of fn, so
// the claim survives past the SELECT instead of releasing the instant it
// returns — see pkg/outbox.Store.RunClaimed's doc comment for why a bare
// fetch-then-mark round trip across two calls lets two relay instances
// double-publish the same row.
func (s *OutboxStore) RunClaimed(ctx context.Context, batchSize int, fn func(ctx context.Context, rows []outbox.Message) ([]string, error)) error {
	pgtx, err := s.repo.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning outbox claim transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = pgtx.Rollback(ctx)
		}
	}()

	q := s.repo.q.WithTx(pgtx)
	claimed, err := q.FetchUnpublishedOutbox(ctx, int32(batchSize))
	if err != nil {
		return translate(err, "unpublished outbox rows")
	}
	rows := make([]outbox.Message, 0, len(claimed))
	for _, row := range claimed {
		rows = append(rows, outbox.Message{
			ID:          row.ID.String(),
			AggregateID: row.AggregateID.String(),
			Topic:       row.Topic,
			Payload:     row.Payload,
		})
	}

	published, fnErr := fn(ctx, rows)
	if len(published) > 0 {
		parsed := make([]uuid.UUID, 0, len(published))
		for _, id := range published {
			u, perr := uuid.Parse(id)
			if perr != nil {
				return fmt.Errorf("outbox row id %q is not a uuid: %w", id, perr)
			}
			parsed = append(parsed, u)
		}
		if _, err := q.MarkOutboxPublished(ctx, parsed); err != nil {
			return translate(err, "marking outbox rows published")
		}
	}

	if err := pgtx.Commit(ctx); err != nil {
		return fmt.Errorf("committing outbox claim: %w", err)
	}
	committed = true
	return fnErr
}
