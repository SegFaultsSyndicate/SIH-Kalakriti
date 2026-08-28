// services/search-svc/internal/search/service/index.go
package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/segfaultsyndicate/kalakriti/services/search-svc/internal/search/domain"
)

// IndexStore is the write side of listing_search plus the reads that build a
// document: the listing itself and the craft vocabulary behind it.
type IndexStore interface {
	// LoadIndexSource returns one projection per language the listing has copy
	// in, with the craft, region and attribute columns already denormalised and
	// the document text left empty for the service to build.
	LoadIndexSource(ctx context.Context, listingID uuid.UUID) ([]domain.IndexSource, error)
	UpsertDocument(ctx context.Context, doc domain.IndexDocument) error
	DeleteListing(ctx context.Context, listingID uuid.UUID) error
}

// Indexer projects published listings into listing_search.
type Indexer struct {
	store     IndexStore
	inference Inference
	cache     Cache
	log       *slog.Logger
}

// NewIndexer builds the indexing side.
func NewIndexer(store IndexStore, inference Inference, cache Cache, log *slog.Logger) *Indexer {
	return &Indexer{store: store, inference: inference, cache: cache, log: log}
}

// Index projects one listing into every language it has copy in. It is
// idempotent by construction: the upsert is keyed on (listing_id, language), so
// a redelivered publish rewrites the same rows.
func (i *Indexer) Index(ctx context.Context, listingID uuid.UUID) error {
	sources, err := i.store.LoadIndexSource(ctx, listingID)
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		// Published with no copy in any language: nothing searchable yet.
		return nil
	}

	texts := make([]string, 0, len(sources))
	documents := make([]domain.IndexDocument, 0, len(sources))
	for _, source := range sources {
		doc := source.Document()
		documents = append(documents, doc)
		texts = append(texts, doc.Document)
	}

	// One embed call for every language, not one per row: the batcher on the
	// ml-svc side is what makes this cheap, and asking once keeps it that way.
	vectors, err := i.inference.Embed(ctx, texts, sources[0].Language)
	if err != nil {
		// A row without a vector is still lexically searchable, and the reindex
		// sweeper will fill the vector in later. Losing the row entirely because
		// the model was down would be the worse failure.
		i.log.WarnContext(ctx, "indexing without embeddings", "listing_id", listingID, "error", err)
		vectors = nil
	}

	for n := range documents {
		if n < len(vectors) {
			documents[n].Embedding = vectors[n]
			version := "ml-svc"
			documents[n].ModelVersion = &version
		}
		if err := i.store.UpsertDocument(ctx, documents[n]); err != nil {
			return fmt.Errorf("indexing listing %s in %s: %w", listingID, documents[n].Language, err)
		}
	}

	i.bumpCache(ctx)
	i.log.InfoContext(ctx, "listing indexed", "listing_id", listingID, "languages", len(documents))
	return nil
}

// Remove drops every row for a listing, which is how a suspension leaves search.
func (i *Indexer) Remove(ctx context.Context, listingID uuid.UUID) error {
	if err := i.store.DeleteListing(ctx, listingID); err != nil {
		return err
	}
	i.bumpCache(ctx)
	i.log.InfoContext(ctx, "listing removed from the index", "listing_id", listingID)
	return nil
}

// bumpCache retires every cached page. A version counter rather than a key scan:
// Redis has no safe wildcard delete under load, and the version is one INCR.
func (i *Indexer) bumpCache(ctx context.Context) {
	if i.cache == nil {
		return
	}
	if err := i.cache.BumpVersion(ctx); err != nil {
		i.log.WarnContext(ctx, "bumping the search cache version", "error", err)
	}
}
