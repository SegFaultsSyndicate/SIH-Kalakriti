// services/core-svc/internal/core/ontology/registry.go
package ontology

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	pkgredis "github.com/segfaultsyndicate/kalakriti/pkg/redis"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/domain"
)

// Store is what the registry reads the ontology from. It is the whole graph in
// two queries: the ontology is a few hundred rows and is read on nearly every
// catalogue and search request, so it is loaded whole rather than queried per
// lookup.
type Store interface {
	ListCrafts(ctx context.Context) ([]domain.Craft, error)
	ListAllCraftAliases(ctx context.Context) ([]domain.CraftAlias, error)
}

// Snapshot is the cached form of the ontology: exactly what BuildIndex needs,
// so a replica that finds a fresh snapshot in Redis never touches Postgres.
type Snapshot struct {
	Version string              `json:"version"`
	BuiltAt time.Time           `json:"built_at"`
	Crafts  []domain.Craft      `json:"crafts"`
	Aliases []domain.CraftAlias `json:"aliases"`
}

// Broadcaster is the slice of go-redis the registry uses beyond the cache: the
// pub/sub pair that tells the other replicas a new version exists.
type Broadcaster interface {
	Publish(ctx context.Context, channel string, message any) *goredis.IntCmd
	Subscribe(ctx context.Context, channels ...string) *goredis.PubSub
}

// Defaults for the cache key, the invalidation channel and the snapshot TTL.
const (
	defaultCacheKey     = "index"
	defaultCachePrefix  = "ontology"
	defaultChannel      = "ontology:invalidate"
	defaultSnapshotTTL  = 24 * time.Hour
	subscribeRetryDelay = 5 * time.Second
)

// RegistryConfig tunes the cache. Zero values are replaced by the defaults above.
type RegistryConfig struct {
	// CacheKey is the Redis key the snapshot is stored under, within the
	// pkg/redis cache's own prefix.
	CacheKey string
	// Channel is the pub/sub channel new versions are announced on.
	Channel string
	// SnapshotTTL is a backstop only: correctness comes from the explicit
	// invalidation below, not from expiry.
	SnapshotTTL time.Duration
}

// Registry holds the live craft index, keeps it in Redis so a restarting replica
// does not have to rebuild it, and keeps every replica in step.
//
// The invalidation path is explicit, because a cached ontology with no way to
// invalidate it is a stale ontology: Refresh rebuilds from Postgres, writes the
// new snapshot to Redis and publishes its version on the invalidation channel;
// every replica running Watch reloads that snapshot. The TTL is a backstop for
// the case where a replica missed the message while it was restarting.
type Registry struct {
	store Store
	cache *pkgredis.Cache[Snapshot]
	bus   Broadcaster
	tr    Transliterator
	log   *slog.Logger
	cfg   RegistryConfig

	// index is swapped wholesale on refresh; readers never take a lock.
	index atomic.Pointer[Index]
}

// NewRegistry builds the registry. cache and bus may both be nil, in which case
// the index is held in memory only and Refresh has nothing to announce — which
// is what the service tests and the seeding CLI use.
func NewRegistry(store Store, client goredis.Cmdable, bus Broadcaster, tr Transliterator, cfg RegistryConfig, log *slog.Logger) *Registry {
	if cfg.CacheKey == "" {
		cfg.CacheKey = defaultCacheKey
	}
	if cfg.Channel == "" {
		cfg.Channel = defaultChannel
	}
	if cfg.SnapshotTTL <= 0 {
		cfg.SnapshotTTL = defaultSnapshotTTL
	}
	if tr == nil {
		tr = NoTransliteration()
	}

	r := &Registry{store: store, bus: bus, tr: tr, cfg: cfg, log: log}
	if client != nil {
		r.cache = pkgredis.NewCache[Snapshot](client, defaultCachePrefix)
	}
	r.index.Store(BuildIndex(nil, nil))
	return r
}

// Start loads the index: from the Redis snapshot when one is there, and from
// Postgres otherwise. A cache that is unreachable is logged and stepped over —
// the database is the source of truth, Redis only saves it a query.
func (r *Registry) Start(ctx context.Context) error {
	if r.cache != nil {
		snapshot, err := r.cache.Get(ctx, r.cfg.CacheKey)
		switch {
		case err == nil:
			r.index.Store(BuildIndex(snapshot.Crafts, snapshot.Aliases))
			crafts, aliases := r.index.Load().Stats()
			r.log.InfoContext(ctx, "ontology index loaded from cache",
				"version", snapshot.Version, "crafts", crafts, "aliases", aliases)
			return nil
		case errors.Is(err, pkgredis.ErrCacheMiss):
			// Expected on a cold Redis; fall through to a rebuild.
		default:
			r.log.WarnContext(ctx, "reading the ontology cache, rebuilding from postgres", "error", err)
		}
	}

	if _, err := r.Refresh(ctx); err != nil {
		return fmt.Errorf("loading the ontology: %w", err)
	}
	return nil
}

// Refresh rebuilds the index from Postgres, republishes the snapshot and
// announces the new version to the other replicas.
func (r *Registry) Refresh(ctx context.Context) (domain.OntologyStats, error) {
	crafts, err := r.store.ListCrafts(ctx)
	if err != nil {
		return domain.OntologyStats{}, fmt.Errorf("reading crafts: %w", err)
	}
	aliases, err := r.store.ListAllCraftAliases(ctx)
	if err != nil {
		return domain.OntologyStats{}, fmt.Errorf("reading craft aliases: %w", err)
	}

	index := BuildIndex(crafts, aliases)
	r.index.Store(index)

	if r.cache != nil {
		snapshot := Snapshot{
			Version: index.Version(),
			BuiltAt: index.BuiltAt(),
			Crafts:  crafts,
			Aliases: aliases,
		}
		if err := r.cache.Set(ctx, r.cfg.CacheKey, snapshot, r.cfg.SnapshotTTL); err != nil {
			// The rebuild itself succeeded; a cache write failure only costs the
			// next replica a query, so it must not fail the refresh.
			r.log.WarnContext(ctx, "writing the ontology cache", "error", err)
		}
	}
	if r.bus != nil {
		if err := r.bus.Publish(ctx, r.cfg.Channel, index.Version()).Err(); err != nil {
			r.log.WarnContext(ctx, "announcing the new ontology version", "error", err)
		}
	}

	return r.Stats(), nil
}

// Watch reloads the index whenever another replica announces a new version. It
// blocks until ctx is cancelled and is meant to be run in its own goroutine.
func (r *Registry) Watch(ctx context.Context) {
	if r.bus == nil {
		return
	}
	sub := r.bus.Subscribe(ctx, r.cfg.Channel)
	defer func() {
		if err := sub.Close(); err != nil {
			r.log.Warn("closing the ontology invalidation subscription", "error", err)
		}
	}()

	messages := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				// go-redis closes the channel only when the subscription is
				// gone for good; wait a beat rather than spinning.
				select {
				case <-ctx.Done():
				case <-time.After(subscribeRetryDelay):
				}
				return
			}
			if msg.Payload == r.index.Load().Version() {
				continue // our own announcement, or a no-op rebuild
			}
			r.reload(ctx, msg.Payload)
		}
	}
}

// reload pulls the announced snapshot out of Redis, falling back to a full
// rebuild from Postgres when the cache cannot supply it.
func (r *Registry) reload(ctx context.Context, version string) {
	if r.cache != nil {
		snapshot, err := r.cache.Get(ctx, r.cfg.CacheKey)
		if err == nil {
			r.index.Store(BuildIndex(snapshot.Crafts, snapshot.Aliases))
			r.log.InfoContext(ctx, "ontology index reloaded", "version", snapshot.Version)
			return
		}
		if !errors.Is(err, pkgredis.ErrCacheMiss) {
			r.log.WarnContext(ctx, "reading the announced ontology snapshot", "error", err)
		}
	}
	if _, err := r.Refresh(ctx); err != nil {
		r.log.ErrorContext(ctx, "rebuilding the ontology after an invalidation",
			"announced_version", version, "error", err)
	}
}

// Resolve links craft mentions in text, returning the span of each mention.
func (r *Registry) Resolve(ctx context.Context, text, language string) ([]domain.CraftMatch, error) {
	return r.index.Load().Resolve(ctx, text, language, r.tr)
}

// Craft returns one craft by id.
func (r *Registry) Craft(id uuid.UUID) (domain.Craft, bool) { return r.index.Load().Craft(id) }

// CraftByCode returns one craft by slug.
func (r *Registry) CraftByCode(code string) (domain.Craft, bool) {
	return r.index.Load().CraftByCode(code)
}

// Crafts returns the whole ontology, ordered by code.
func (r *Registry) Crafts() []domain.Craft { return r.index.Load().Crafts() }

// Stats describes the index currently loaded.
func (r *Registry) Stats() domain.OntologyStats {
	index := r.index.Load()
	crafts, aliases := index.Stats()
	return domain.OntologyStats{
		Version: index.Version(),
		Crafts:  crafts,
		Aliases: aliases,
		BuiltAt: index.BuiltAt(),
	}
}
