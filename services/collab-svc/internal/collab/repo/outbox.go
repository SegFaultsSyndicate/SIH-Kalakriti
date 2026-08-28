// services/collab-svc/internal/collab/repo/outbox.go
package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/segfaultsyndicate/kalakriti/pkg/outbox"

	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/repo/db"
)

// Tx satisfies pkg/outbox.Enqueuer, so outbox.Enqueue can write an event row
// inside the same transaction as the saga write it accompanies.
var _ outbox.Enqueuer = (*Tx)(nil)

// InsertOutbox writes one outbox row. ON CONFLICT DO NOTHING makes a replayed
// call a no-op, matching core-svc's same-named method.
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
type OutboxStore struct {
	repo *Repo
}

// NewOutboxStore builds the relay's view of the outbox table.
func NewOutboxStore(r *Repo) *OutboxStore { return &OutboxStore{repo: r} }

// OutboxStore satisfies the relay side of pkg/outbox.
var _ outbox.Store = (*OutboxStore)(nil)

// FetchUnpublished claims a batch of unpublished rows with FOR UPDATE SKIP
// LOCKED, so several relay instances can share the table without blocking.
func (s *OutboxStore) FetchUnpublished(ctx context.Context, limit int) ([]outbox.Message, error) {
	rows, err := s.repo.q.FetchUnpublishedOutbox(ctx, int32(limit))
	if err != nil {
		return nil, translate(err, "unpublished outbox rows")
	}
	out := make([]outbox.Message, 0, len(rows))
	for _, row := range rows {
		out = append(out, outbox.Message{
			ID:          row.ID.String(),
			AggregateID: row.AggregateID.String(),
			Topic:       row.Topic,
			Payload:     row.Payload,
		})
	}
	return out, nil
}

// MarkPublished stamps published_at on the given rows.
func (s *OutboxStore) MarkPublished(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	parsed := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		u, err := uuid.Parse(id)
		if err != nil {
			return fmt.Errorf("outbox row id %q is not a uuid: %w", id, err)
		}
		parsed = append(parsed, u)
	}
	if _, err := s.repo.q.MarkOutboxPublished(ctx, parsed); err != nil {
		return translate(err, "marking outbox rows published")
	}
	return nil
}
