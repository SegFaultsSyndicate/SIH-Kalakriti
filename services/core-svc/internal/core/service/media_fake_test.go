// services/core-svc/internal/core/service/media_fake_test.go
package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// fakeMediaStore is an in-memory MediaStore + MediaTx with the one database
// behaviour the service leans on: the guarded transition, which updates nothing
// when the row has already moved.
type fakeMediaStore struct {
	mu sync.Mutex

	media     map[uuid.UUID]domain.Media
	byHash    map[string]uuid.UUID
	primary   map[uuid.UUID]domain.Listing
	outbox    []outboxRow
	deleted   []uuid.UUID
	createErr error
}

func newFakeMediaStore() *fakeMediaStore {
	return &fakeMediaStore{
		media:  map[uuid.UUID]domain.Media{},
		byHash: map[string]uuid.UUID{},
	}
}

type fakeMediaTx struct {
	store  *fakeMediaStore
	writes []func()
}

func (s *fakeMediaStore) InTx(ctx context.Context, fn func(ctx context.Context, tx MediaTx) error) error {
	tx := &fakeMediaTx{store: s}
	if err := fn(ctx, tx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range tx.writes {
		w()
	}
	return nil
}

func (s *fakeMediaStore) GetMedia(_ context.Context, id uuid.UUID) (domain.Media, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.media[id]
	if !ok {
		return domain.Media{}, fmt.Errorf("media not found: %w", pkgdomain.ErrNotFound)
	}
	return m, nil
}

func (s *fakeMediaStore) GetMediaByHash(_ context.Context, artisanID uuid.UUID, sha string) (domain.Media, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byHash[artisanID.String()+":"+sha]
	if !ok {
		return domain.Media{}, fmt.Errorf("media not found: %w", pkgdomain.ErrNotFound)
	}
	return s.media[id], nil
}

func (s *fakeMediaStore) ListStalePendingMedia(_ context.Context, before time.Time, batchSize int32) ([]domain.Media, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]domain.Media, 0, len(s.media))
	for _, m := range s.media {
		if m.State == domain.MediaPending && m.CreatedAt.Before(before) {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	if len(out) > int(batchSize) {
		out = out[:batchSize]
	}
	return out, nil
}

func (t *fakeMediaTx) CreateMediaPending(_ context.Context, m domain.Media) (domain.Media, error) {
	if t.store.createErr != nil {
		return domain.Media{}, t.store.createErr
	}
	m.CreatedAt = fakeNow
	m.UploadedAt = fakeNow
	t.writes = append(t.writes, func() {
		t.store.media[m.ID] = m
		if m.SHA256Hex != nil {
			t.store.byHash[m.ArtisanID.String()+":"+*m.SHA256Hex] = m.ID
		}
	})
	return m, nil
}

func (t *fakeMediaTx) TransitionMediaState(
	_ context.Context,
	id uuid.UUID,
	from, to domain.MediaState,
	set domain.MediaTransition,
) (domain.Media, error) {
	t.store.mu.Lock()
	defer t.store.mu.Unlock()

	m, ok := t.store.media[id]
	if !ok {
		return domain.Media{}, fmt.Errorf("media not found: %w", pkgdomain.ErrNotFound)
	}
	if m.State != from {
		return domain.Media{}, fmt.Errorf(
			"media %s is in state %s, not %s: %w", id, m.State, from, pkgdomain.ErrConflict)
	}

	m.State = to
	if set.SizeBytes != nil {
		m.SizeBytes = *set.SizeBytes
	}
	if set.SHA256Hex != nil {
		m.SHA256Hex = set.SHA256Hex
	}
	m.FailureReason = nil
	if to == domain.MediaFailed {
		m.FailureReason = set.FailureReason
	}
	if m.ConfirmedAt == nil && to != domain.MediaPending {
		confirmed := fakeNow
		m.ConfirmedAt = &confirmed
	}
	t.writes = append(t.writes, func() { t.store.media[id] = m })
	return m, nil
}

func (t *fakeMediaTx) SetMediaEnhanced(
	_ context.Context,
	id uuid.UUID,
	from domain.MediaState,
	enhancedKey string,
	modelVersion *string,
) (domain.Media, error) {
	t.store.mu.Lock()
	defer t.store.mu.Unlock()

	m, ok := t.store.media[id]
	if !ok {
		return domain.Media{}, fmt.Errorf("media not found: %w", pkgdomain.ErrNotFound)
	}
	// The promotion to PROCESSING earlier in this transaction is only visible in
	// the buffered writes, so the guard is checked against the state the service
	// says it moved the row to.
	if m.State != from && !(from == domain.MediaProcessing && m.State == domain.MediaUploaded) {
		return domain.Media{}, fmt.Errorf(
			"media %s is in state %s, not %s: %w", id, m.State, from, pkgdomain.ErrConflict)
	}

	m.State = domain.MediaReady
	m.EnhancedObjectKey = &enhancedKey
	m.ModelVersion = modelVersion
	m.FailureReason = nil
	t.writes = append(t.writes, func() { t.store.media[id] = m })
	return m, nil
}

func (t *fakeMediaTx) DeleteMedia(_ context.Context, id uuid.UUID) (bool, error) {
	t.writes = append(t.writes, func() {
		delete(t.store.media, id)
		t.store.deleted = append(t.store.deleted, id)
	})
	return true, nil
}

func (t *fakeMediaTx) InsertOutbox(_ context.Context, id, aggregateID, topic, idempotencyKey string, payload []byte) error {
	row := outboxRow{
		ID: id, AggregateID: aggregateID, Topic: topic,
		IdempotencyKey: idempotencyKey, Payload: payload,
	}
	t.writes = append(t.writes, func() {
		// Mirrors ON CONFLICT (topic, idempotency_key) DO NOTHING.
		for _, existing := range t.store.outbox {
			if existing.Topic == row.Topic && existing.IdempotencyKey == row.IdempotencyKey {
				return
			}
		}
		t.store.outbox = append(t.store.outbox, row)
	})
	return nil
}

// fakeObjectStore stands in for MinIO: it records what was presigned and holds a
// map of object keys to sizes, which is the only thing Stat is asked about.
type fakeObjectStore struct {
	mu sync.Mutex

	objects    map[string]int64
	deleted    []string
	putURLs    map[string]string
	getCalls   int
	presignErr error
}

func newFakeObjectStore() *fakeObjectStore {
	return &fakeObjectStore{objects: map[string]int64{}, putURLs: map[string]string{}}
}

func (o *fakeObjectStore) PresignedPutURL(_ context.Context, key, _ string, _ time.Duration) (string, error) {
	if o.presignErr != nil {
		return "", o.presignErr
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	url := "https://minio.test/" + key + "?signature=put"
	o.putURLs[key] = url
	return url, nil
}

func (o *fakeObjectStore) PresignedGetURL(_ context.Context, key string, _ time.Duration) (string, error) {
	if o.presignErr != nil {
		return "", o.presignErr
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.getCalls++
	return fmt.Sprintf("https://minio.test/%s?signature=get&n=%d", key, o.getCalls), nil
}

func (o *fakeObjectStore) Stat(_ context.Context, key string) (ObjectInfo, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	size, ok := o.objects[key]
	if !ok {
		return ObjectInfo{}, fmt.Errorf("object %s does not exist", key)
	}
	return ObjectInfo{SizeBytes: size, ETag: "etag-" + key}, nil
}

func (o *fakeObjectStore) Delete(_ context.Context, key string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.objects, key)
	o.deleted = append(o.deleted, key)
	return nil
}

// put simulates the artisan's phone PUTting bytes to the presigned URL.
func (o *fakeObjectStore) put(key string, size int64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.objects[key] = size
}

// fakeURLCache is an in-memory URLCache that ignores TTLs.
type fakeURLCache struct {
	mu     sync.Mutex
	values map[string]string
	sets   int
}

func newFakeURLCache() *fakeURLCache { return &fakeURLCache{values: map[string]string{}} }

func (c *fakeURLCache) Get(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.values[key]
	if !ok {
		return "", fmt.Errorf("cache miss")
	}
	return v, nil
}

func (c *fakeURLCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
	c.sets++
	return nil
}

func (c *fakeURLCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.values, key)
	return nil
}
