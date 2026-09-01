// services/search-svc/internal/search/repo/repo.go

// Package repo adapts search-svc's sqlc-generated queries to the domain types
// the service works in. The two retrieval legs live here; fusion does not.
package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/search-svc/internal/search/domain"
	"github.com/ZoroNewbie00/kalakriti/services/search-svc/internal/search/repo/db"
)

// Repo owns the connection pool.
type Repo struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// New builds a Repo over an existing pool.
func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool, q: db.New(pool)} }

// Pool exposes the pool for health checks.
func (r *Repo) Pool() *pgxpool.Pool { return r.pool }

// translate converts a pgx error into a domain sentinel.
func translate(err error, what string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s not found: %w", what, pkgdomain.ErrNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%s already exists (%s): %w", what, pgErr.ConstraintName, pkgdomain.ErrConflict)
	}
	return fmt.Errorf("%s: %w", what, err)
}

// --- retrieval ---------------------------------------------------------------

// SearchLexical is the full-text leg.
func (r *Repo) SearchLexical(
	ctx context.Context,
	query, language string,
	filters domain.Filters,
	limit int32,
) ([]domain.Candidate, error) {
	rows, err := r.q.SearchLexicalCandidates(ctx, db.SearchLexicalCandidatesParams{
		Query:           query,
		Language:        db.LanguageCode(language),
		CraftIds:        filters.CraftIDs,
		Colours:         orEmpty(filters.Colours),
		Materials:       orEmpty(filters.Materials),
		MinPricePaise:   filters.MinPricePaise,
		MaxPricePaise:   filters.MaxPricePaise,
		StateCode:       filters.StateCode,
		ListingType:     listingType(filters.ListingType),
		GiOnly:          filters.GIOnly,
		SealedOnly:      filters.SealedOnly,
		MaxLeadTimeDays: filters.MaxLeadTimeDays,
		PageSize:        limit,
	})
	if err != nil {
		return nil, translate(err, "lexical candidates")
	}

	out := make([]domain.Candidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Candidate{
			ListingID: row.ListingID, ArtisanID: row.ArtisanID,
			CraftID: row.CraftID, Score: float64(row.Score),
		})
	}
	return out, nil
}

// SearchVector is the pgvector leg. The embedding is cosine-compared against the
// HNSW index, which is why the embedder must emit unit-normalised vectors.
func (r *Repo) SearchVector(
	ctx context.Context,
	embedding []float32,
	language string,
	filters domain.Filters,
	limit int32,
) ([]domain.Candidate, error) {
	vector := pgvector.NewVector(embedding)
	rows, err := r.q.SearchVectorCandidates(ctx, db.SearchVectorCandidatesParams{
		Embedding:       &vector,
		Language:        db.LanguageCode(language),
		CraftIds:        filters.CraftIDs,
		Colours:         orEmpty(filters.Colours),
		Materials:       orEmpty(filters.Materials),
		MinPricePaise:   filters.MinPricePaise,
		MaxPricePaise:   filters.MaxPricePaise,
		StateCode:       filters.StateCode,
		ListingType:     listingType(filters.ListingType),
		GiOnly:          filters.GIOnly,
		SealedOnly:      filters.SealedOnly,
		MaxLeadTimeDays: filters.MaxLeadTimeDays,
		PageSize:        limit,
	})
	if err != nil {
		return nil, translate(err, "vector candidates")
	}

	out := make([]domain.Candidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Candidate{
			ListingID: row.ListingID, ArtisanID: row.ArtisanID,
			CraftID: row.CraftID, Score: float64(row.Score),
		})
	}
	return out, nil
}

// HydrateHits reads the copy behind a page of results, preferring the
// requester's language and falling back to whatever the listing has.
func (r *Repo) HydrateHits(ctx context.Context, listingIDs []uuid.UUID, language string) ([]domain.Hit, error) {
	if len(listingIDs) == 0 {
		return nil, nil
	}
	rows, err := r.q.HydrateSearchHits(ctx, db.HydrateSearchHitsParams{
		ListingIds: listingIDs,
		Language:   db.LanguageCode(language),
	})
	if err != nil {
		return nil, translate(err, "search hits")
	}

	out := make([]domain.Hit, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Hit{
			ListingID:   row.ListingID,
			ProductID:   row.ProductID,
			ArtisanID:   row.ArtisanID,
			CraftID:     row.CraftID,
			Label:       derefString(row.Title),
			Description: derefString(row.Description),
			GICertified: row.GiCertified,
			ListingType: string(row.ListingType),
		})
	}
	return out, nil
}

// SiblingCrafts is the zero-result fallback's widening set.
func (r *Repo) SiblingCrafts(ctx context.Context, craftIDs []uuid.UUID) ([]domain.Sibling, error) {
	if len(craftIDs) == 0 {
		return nil, nil
	}
	rows, err := r.q.ListSiblingCrafts(ctx, db.ListSiblingCraftsParams{
		CraftIds: craftIDs,
		PageSize: 8,
	})
	if err != nil {
		return nil, translate(err, "sibling crafts")
	}

	out := make([]domain.Sibling, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Sibling{
			CraftID: row.CraftID, Code: row.Code,
			DisplayName: row.DisplayName, Kind: string(row.Kind),
		})
	}
	return out, nil
}

// --- suggest ------------------------------------------------------------------

// SuggestAliases is type-ahead over the craft ontology.
func (r *Repo) SuggestAliases(ctx context.Context, prefix, _ string, limit int32) ([]domain.Suggestion, error) {
	rows, err := r.q.SuggestCraftAliases(ctx, db.SuggestCraftAliasesParams{
		Prefix: prefix, PageSize: limit,
	})
	if err != nil {
		return nil, translate(err, "craft suggestions")
	}

	// One suggestion per craft: three spellings of Ajrakh are one thing to click.
	seen := make(map[uuid.UUID]struct{}, len(rows))
	out := make([]domain.Suggestion, 0, len(rows))
	for _, row := range rows {
		if _, dup := seen[row.CraftID]; dup {
			continue
		}
		seen[row.CraftID] = struct{}{}
		craftID := row.CraftID
		out = append(out, domain.Suggestion{
			Text: row.DisplayName, Kind: domain.SuggestCraft,
			EntityID: &craftID, Score: toFloat64(row.Score),
		})
	}
	return out, nil
}

// SuggestQueries is type-ahead over what other buyers have searched for.
func (r *Repo) SuggestQueries(ctx context.Context, prefix string, limit int32) ([]domain.Suggestion, error) {
	rows, err := r.q.SuggestPopularQueries(ctx, db.SuggestPopularQueriesParams{
		Prefix: prefix, PageSize: limit,
	})
	if err != nil {
		return nil, translate(err, "query suggestions")
	}

	out := make([]domain.Suggestion, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Suggestion{
			Text: row.Normalised, Kind: domain.SuggestQuery,
			Score: float64(row.TimesSeen),
		})
	}
	return out, nil
}

// RecordQuery remembers a query for type-ahead. A query that found nothing is
// still recorded, with its zero, so it is never suggested to anyone else.
func (r *Repo) RecordQuery(ctx context.Context, normalised, language string, hits int32) error {
	if len(normalised) < 2 || len(normalised) > 120 {
		return nil
	}
	return translate(r.q.RecordSearchQuery(ctx, db.RecordSearchQueryParams{
		Normalised: normalised,
		Language:   db.LanguageCode(language),
		HitCount:   hits,
	}), "search query log")
}

// --- indexing -----------------------------------------------------------------

// LoadIndexSource reads everything one publish needs to build its documents.
func (r *Repo) LoadIndexSource(ctx context.Context, listingID uuid.UUID) ([]domain.IndexSource, error) {
	rows, err := r.q.LoadIndexSource(ctx, listingID)
	if err != nil {
		return nil, translate(err, "index source")
	}

	out := make([]domain.IndexSource, 0, len(rows))
	for _, row := range rows {
		region := make([]string, 0, 2)
		region = append(region, row.StateCode)
		if row.District != nil {
			region = append(region, *row.District)
		}

		out = append(out, domain.IndexSource{
			ListingID: row.ListingID, Language: string(row.Language),
			ArtisanID: row.ArtisanID, CraftID: row.CraftID, ClusterID: row.ClusterID,
			ListingType: string(row.ListingType), PricePaise: row.PricePaise,
			LeadTimeDays: row.LeadTimeDays, GICertified: row.GiCertified,
			ProvenanceSealed: toBool(row.ProvenanceSealed),
			StateCode:        row.StateCode, District: row.District,
			Colours: row.Colours, Materials: row.Materials,
			Title: row.Title, Description: row.Description, Highlights: row.Highlights,
			CraftName: row.CraftName, CraftAliases: toStringSlice(row.CraftAliases),
			Techniques: row.Techniques, Motifs: row.Motifs, Region: region,
		})
	}
	return out, nil
}

// UpsertDocument writes one listing's projection in one language.
func (r *Repo) UpsertDocument(ctx context.Context, doc domain.IndexDocument) error {
	params := db.UpsertListingSearchParams{
		ListingID: doc.ListingID, Language: db.LanguageCode(doc.Language),
		ArtisanID: doc.ArtisanID, CraftID: doc.CraftID, ClusterID: doc.ClusterID,
		ListingType: db.ListingType(doc.ListingType), PricePaise: doc.PricePaise,
		LeadTimeDays: doc.LeadTimeDays, GiCertified: doc.GICertified,
		ProvenanceSealed: doc.ProvenanceSealed, StateCode: doc.StateCode,
		District: doc.District, Colours: orEmpty(doc.Colours),
		Materials: orEmpty(doc.Materials), Document: doc.Document,
		ModelVersion: doc.ModelVersion,
	}
	if len(doc.Embedding) > 0 {
		vector := pgvector.NewVector(doc.Embedding)
		params.Embedding = &vector
	}
	return translate(r.q.UpsertListingSearch(ctx, params), "search document")
}

// DeleteListing drops every row for a listing, which is how a suspension leaves
// search.
func (r *Repo) DeleteListing(ctx context.Context, listingID uuid.UUID) error {
	_, err := r.q.DeleteListingSearch(ctx, listingID)
	return translate(err, "search document")
}

func orEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func listingType(value *string) *db.ListingType {
	if value == nil {
		return nil
	}
	converted := db.ListingType(*value)
	return &converted
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// toFloat64 handles a computed SQL expression sqlc couldn't type statically
// (a CASE mixing a literal with a real-returning function). pgx decodes
// Postgres real as float32.
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	default:
		return 0
	}
}

func toBool(v interface{}) bool {
	b, _ := v.(bool)
	return b
}

func toStringSlice(v interface{}) []string {
	s, _ := v.([]string)
	return s
}
