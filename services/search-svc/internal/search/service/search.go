// services/search-svc/internal/search/service/search.go

// Package service holds search-svc's query pipeline and its indexing side. It
// takes and returns domain types; protobuf is converted in the handler.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/search-svc/internal/search/domain"
)

// candidatesPerLeg is how deep each retrieval leg goes before fusion. Fifty is
// enough for RRF to have something to fuse and short enough that the cross
// encoder can rerank the survivors inside the latency budget.
const candidatesPerLeg = 50

// minResultsToRerank is the floor below which reranking is skipped: a cross
// encoder over four candidates costs a model call to reorder a list the buyer
// can read at a glance.
const minResultsToRerank = 5

// understandingBudget is all the query understander gets. Past this the search
// runs on the raw text, because a slow filter extractor must never be the reason
// a buyer sees nothing.
const understandingBudget = 500 * time.Millisecond

// cacheTTL is how long a rendered page is reused. Five minutes is short enough
// that a price edit surfaces quickly and long enough to absorb the repeated
// queries a category page generates; the version counter handles publishes.
const cacheTTL = 5 * time.Minute

// Store is the retrieval surface search-svc reads through.
type Store interface {
	SearchLexical(ctx context.Context, query, language string, filters domain.Filters, limit int32) ([]domain.Candidate, error)
	SearchVector(ctx context.Context, embedding []float32, language string, filters domain.Filters, limit int32) ([]domain.Candidate, error)
	HydrateHits(ctx context.Context, listingIDs []uuid.UUID, language string) ([]domain.Hit, error)
	SiblingCrafts(ctx context.Context, craftIDs []uuid.UUID) ([]domain.Sibling, error)
	SuggestAliases(ctx context.Context, prefix, language string, limit int32) ([]domain.Suggestion, error)
	SuggestQueries(ctx context.Context, prefix string, limit int32) ([]domain.Suggestion, error)
	RecordQuery(ctx context.Context, normalised, language string, hits int32) error
}

// Inference is the slice of ml-svc the query pipeline uses.
type Inference interface {
	Embed(ctx context.Context, texts []string, language string) ([][]float32, error)
	Rerank(ctx context.Context, query string, candidates []domain.RerankCandidate) (map[uuid.UUID]float64, error)
	Transcribe(ctx context.Context, objectKey, language string) (string, float32, error)
}

// Ontology is core-svc's entity linker: it turns craft mentions in free text
// into craft ids, with the span each mention occupies.
type Ontology interface {
	ResolveAlias(ctx context.Context, text, language string) ([]domain.Span, error)
}

// Transliterator canonicalises a token written in one script into the spelling
// the index holds. It is hit on nearly every query, so the implementation is
// expected to be cached; the pipeline does not cache it a second time.
type Transliterator interface {
	Canonicalise(ctx context.Context, token, language string) (string, error)
}

// Understander pulls structured filters out of free text and hands back what is
// left. It is given a hard budget and its failure is never fatal.
type Understander interface {
	Understand(ctx context.Context, text, language string) (domain.Filters, string, error)
}

// Cache is the rendered-page cache. A miss is not an error.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Version is bumped whenever the index changes, which retires every key
	// derived from it without a scan or a wildcard delete.
	Version(ctx context.Context) (int64, error)
	BumpVersion(ctx context.Context) error
}

// Search is the buyer-facing query pipeline.
type Search struct {
	store        Store
	inference    Inference
	ontology     Ontology
	translit     Transliterator
	understander Understander
	cache        Cache
	log          *slog.Logger
}

// NewSearch builds the query pipeline. cache and translit may be nil, in which
// case every query is served fresh and tokens are used as typed.
func NewSearch(
	store Store,
	inference Inference,
	ontology Ontology,
	translit Transliterator,
	understander Understander,
	cache Cache,
	log *slog.Logger,
) *Search {
	return &Search{
		store: store, inference: inference, ontology: ontology,
		translit: translit, understander: understander, cache: cache, log: log,
	}
}

// Request is one search as it arrives.
type Request struct {
	Query    string
	Language string
	Filters  domain.Filters
	Limit    int32
	// AudioObjectKey, when set, is transcribed and used as the query.
	AudioObjectKey string
	Mode           string
}

// Result is one rendered page.
type Result struct {
	Hits []domain.Hit
	// DidYouMean names the crafts substituted when the original query matched
	// nothing; empty when the results are for what was actually asked.
	DidYouMean []string
	// Transcript is what the server heard, for a voice query.
	Transcript           string
	TranscriptConfidence float32
	DetectedLanguage     string
	QueryID              string
	// Understood is the query after the whole understanding phase, exposed so a
	// caller can show what was searched for.
	Understood domain.Query
	FromCache  bool
}

// Do runs the whole pipeline. The order is fixed and each step is allowed to
// fail soft: nothing between transcription and reranking is permitted to turn a
// slow dependency into an empty page.
func (s *Search) Do(ctx context.Context, req Request) (Result, error) {
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	language := req.Language
	if language == "" {
		language = "ENGLISH"
	}

	// a. Audio first: everything downstream works on text.
	transcript, confidence := "", float32(0)
	if req.AudioObjectKey != "" {
		var err error
		transcript, confidence, err = s.inference.Transcribe(ctx, req.AudioObjectKey, language)
		if err != nil {
			return Result{}, fmt.Errorf("transcribing the query: %w", err)
		}
		req.Query = transcript
	}

	// b. Normalise, the same folding the index applied to its documents.
	query := domain.Query{
		Raw:        req.Query,
		Normalised: domain.Normalise(req.Query),
		Language:   language,
		Filters:    req.Filters,
	}

	// c. Canonicalise transliterated tokens, so "ajrakh" typed on a Latin
	//    keyboard reaches a listing indexed as अजरख.
	query.Canonical = s.canonicalise(ctx, query.Normalised, language)

	// d. Entity linking. The spans are marked do-not-translate and their craft
	//    ids become a filter, which is what makes a craft name a narrowing
	//    rather than just another bag of words.
	query.Spans = s.link(ctx, query.Canonical, language)
	query.Filters = withSpanCrafts(query.Filters, query.Spans)

	// e. Query understanding, on a hard budget.
	query.Filters, query.Residual = s.understand(ctx, query)

	// The cache key is taken after understanding, so two phrasings that mean the
	// same thing share a page.
	version := s.version(ctx)
	cacheKey := query.Filters.CacheKey(query.Canonical, language, version)
	if cached, ok := s.fromCache(ctx, cacheKey); ok {
		cached.Transcript, cached.TranscriptConfidence = transcript, confidence
		cached.FromCache = true
		return cached, nil
	}

	hits, err := s.retrieve(ctx, query, req.Limit)
	if err != nil {
		return Result{}, err
	}

	// 3. Never an empty page: widen to sibling crafts and say so.
	var didYouMean []string
	if len(hits) == 0 {
		hits, didYouMean, err = s.widen(ctx, query, req.Limit)
		if err != nil {
			return Result{}, err
		}
	}

	result := Result{
		Hits:                 hits,
		DidYouMean:           didYouMean,
		Transcript:           transcript,
		TranscriptConfidence: confidence,
		DetectedLanguage:     language,
		QueryID:              cacheKey,
		Understood:           query,
	}

	s.toCache(ctx, cacheKey, result)
	if err := s.store.RecordQuery(ctx, query.Canonical, language, int32(len(hits))); err != nil {
		s.log.WarnContext(ctx, "recording the query for suggest", "error", err)
	}
	return result, nil
}

// retrieve runs both legs, fuses them, reranks and applies the business rules.
func (s *Search) retrieve(ctx context.Context, query domain.Query, limit int32) ([]domain.Hit, error) {
	lexical, dense, err := s.legs(ctx, query)
	if err != nil {
		return nil, err
	}

	// g. Fusion on ranks, not scores: ts_rank_cd and cosine similarity are not
	//    on the same scale and never will be.
	fused := domain.FuseRRF(lexical, dense)
	if len(fused) == 0 {
		return nil, nil
	}
	if len(fused) > candidatesPerLeg {
		fused = fused[:candidatesPerLeg]
	}

	ids := make([]uuid.UUID, 0, len(fused))
	for _, candidate := range fused {
		ids = append(ids, candidate.ListingID)
	}
	hits, err := s.store.HydrateHits(ctx, ids, query.Language)
	if err != nil {
		return nil, err
	}

	byID := make(map[uuid.UUID]domain.Hit, len(hits))
	for _, hit := range hits {
		byID[hit.ListingID] = hit
	}
	ordered := make([]domain.Hit, 0, len(fused))
	for _, candidate := range fused {
		hit, ok := byID[candidate.ListingID]
		if !ok {
			continue // the listing went away between retrieval and hydration
		}
		hit.Score = candidate.Score
		hit.MatchedTerms = matchedTerms(query, hit)
		ordered = append(ordered, hit)
	}

	// h. Rerank, unless there is too little to reorder to be worth a model call.
	ordered = s.rerank(ctx, query, ordered)

	// i. Marketplace rules: made-to-order is not punished for lack of stock,
	//    GI is nudged up, no artisan owns the page.
	ordered = domain.ApplyBusinessRules(ordered)

	if int32(len(ordered)) > limit {
		ordered = ordered[:limit]
	}
	return ordered, nil
}

// legs runs lexical and dense retrieval concurrently. A leg that fails is not
// fatal on its own: hybrid search with one half working is still search, and
// only both failing is an error worth showing the buyer.
func (s *Search) legs(ctx context.Context, query domain.Query) ([]domain.Candidate, []domain.Candidate, error) {
	var lexical, dense []domain.Candidate
	var lexicalErr, denseErr error

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		lexical, lexicalErr = s.store.SearchLexical(groupCtx, query.Canonical, query.Language, query.Filters, candidatesPerLeg)
		if lexicalErr != nil {
			s.log.WarnContext(ctx, "lexical leg failed", "error", lexicalErr)
		}
		return nil
	})
	group.Go(func() error {
		text := query.Residual
		if strings.TrimSpace(text) == "" {
			text = query.Canonical
		}
		if strings.TrimSpace(text) == "" {
			return nil // a pure filter query has nothing to embed
		}

		vectors, err := s.inference.Embed(groupCtx, []string{text}, query.Language)
		if err != nil || len(vectors) == 0 {
			denseErr = err
			s.log.WarnContext(ctx, "dense leg failed to embed", "error", err)
			return nil
		}
		dense, denseErr = s.store.SearchVector(groupCtx, vectors[0], query.Language, query.Filters, candidatesPerLeg)
		if denseErr != nil {
			s.log.WarnContext(ctx, "dense leg failed", "error", denseErr)
		}
		return nil
	})
	_ = group.Wait()

	if lexicalErr != nil && denseErr != nil {
		return nil, nil, fmt.Errorf("both retrieval legs failed: %w", errors.Join(lexicalErr, denseErr))
	}
	return lexical, dense, nil
}

// rerank reorders the survivors with the cross-encoder, and keeps the fused
// order when the model is unavailable.
func (s *Search) rerank(ctx context.Context, query domain.Query, hits []domain.Hit) []domain.Hit {
	if len(hits) < minResultsToRerank {
		return hits
	}

	candidates := make([]domain.RerankCandidate, 0, len(hits))
	for _, hit := range hits {
		candidates = append(candidates, domain.RerankCandidate{
			ListingID: hit.ListingID,
			Text:      strings.TrimSpace(hit.Label + " " + hit.Description),
		})
	}

	scores, err := s.inference.Rerank(ctx, query.Canonical, candidates)
	if err != nil {
		s.log.WarnContext(ctx, "rerank failed, keeping the fused order", "error", err)
		return hits
	}

	out := make([]domain.Hit, 0, len(hits))
	for _, hit := range hits {
		if score, ok := scores[hit.ListingID]; ok {
			hit.Score = score
		}
		out = append(out, hit)
	}
	return out
}

// widen is the zero-result fallback: retry against the crafts adjacent to the
// ones the query pointed at, and name the substitution so the buyer knows the
// page is not quite what they asked for.
func (s *Search) widen(ctx context.Context, query domain.Query, limit int32) ([]domain.Hit, []string, error) {
	if len(query.Filters.CraftIDs) == 0 {
		return nil, nil, nil // nothing to widen from
	}

	siblings, err := s.store.SiblingCrafts(ctx, query.Filters.CraftIDs)
	if err != nil || len(siblings) == 0 {
		return nil, nil, err
	}

	widened := query
	widened.Filters.CraftIDs = make([]uuid.UUID, 0, len(siblings))
	names := make([]string, 0, len(siblings))
	for _, sibling := range siblings {
		widened.Filters.CraftIDs = append(widened.Filters.CraftIDs, sibling.CraftID)
		names = append(names, sibling.DisplayName)
	}

	hits, err := s.retrieve(ctx, widened, limit)
	if err != nil || len(hits) == 0 {
		return nil, nil, err
	}
	s.log.InfoContext(ctx, "widened to sibling crafts", "query", query.Canonical, "siblings", names)
	return hits, names, nil
}

// Suggest is type-ahead over craft aliases and previously successful queries.
func (s *Search) Suggest(ctx context.Context, prefix, language string, limit int32) ([]domain.Suggestion, error) {
	prefix = domain.Normalise(prefix)
	if prefix == "" {
		return nil, fmt.Errorf("prefix is required: %w", pkgdomain.ErrInvalidInput)
	}
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	if language == "" {
		language = "ENGLISH"
	}

	aliases, err := s.store.SuggestAliases(ctx, prefix, language, limit)
	if err != nil {
		return nil, err
	}
	queries, err := s.store.SuggestQueries(ctx, prefix, limit)
	if err != nil {
		return nil, err
	}

	// Crafts first: a buyer typing "ajr" wants the craft, not somebody else's
	// half-finished sentence.
	out := append(aliases, queries...)
	if int32(len(out)) > limit {
		out = out[:limit]
	}
	return out, nil
}

// --- understanding steps ---------------------------------------------------------

// canonicalise replaces each token with its canonical spelling. A token the
// transliterator cannot place is left exactly as typed.
func (s *Search) canonicalise(ctx context.Context, normalised, language string) string {
	if s.translit == nil || normalised == "" {
		return normalised
	}

	tokens := domain.Tokens(normalised)
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		canonical, err := s.translit.Canonicalise(ctx, token, language)
		if err != nil || canonical == "" {
			out = append(out, token)
			continue
		}
		out = append(out, canonical)
	}
	return strings.Join(out, " ")
}

// link asks core-svc which crafts the query mentions. A failure costs the query
// its craft filter, not its results.
func (s *Search) link(ctx context.Context, text, language string) []domain.Span {
	if s.ontology == nil || text == "" {
		return nil
	}
	spans, err := s.ontology.ResolveAlias(ctx, text, language)
	if err != nil {
		s.log.WarnContext(ctx, "entity linking failed", "error", err)
		return nil
	}
	return spans
}

// understand runs the filter extractor under its own deadline and falls back to
// the whole query as residual text. This is the step most likely to be slow, and
// the one it is least acceptable to wait for.
func (s *Search) understand(ctx context.Context, query domain.Query) (domain.Filters, string) {
	if s.understander == nil || query.Canonical == "" {
		return query.Filters, query.Canonical
	}

	budgetCtx, cancel := context.WithTimeout(ctx, understandingBudget)
	defer cancel()

	extracted, residual, err := s.understander.Understand(budgetCtx, query.Canonical, query.Language)
	if err != nil {
		s.log.WarnContext(ctx, "query understanding failed, searching the whole query",
			"error", err, "budget", understandingBudget)
		return query.Filters, query.Canonical
	}
	if strings.TrimSpace(residual) == "" {
		residual = query.Canonical
	}
	return mergeFilters(query.Filters, extracted), residual
}

// --- cache -----------------------------------------------------------------------

func (s *Search) version(ctx context.Context) int64 {
	if s.cache == nil {
		return 0
	}
	version, err := s.cache.Version(ctx)
	if err != nil {
		s.log.WarnContext(ctx, "reading the search cache version", "error", err)
		return 0
	}
	return version
}

func (s *Search) fromCache(ctx context.Context, key string) (Result, bool) {
	if s.cache == nil {
		return Result{}, false
	}
	raw, err := s.cache.Get(ctx, key)
	if err != nil || len(raw) == 0 {
		return Result{}, false
	}
	result, err := decodeResult(raw)
	if err != nil {
		s.log.WarnContext(ctx, "decoding a cached page", "error", err)
		return Result{}, false
	}
	return result, true
}

func (s *Search) toCache(ctx context.Context, key string, result Result) {
	if s.cache == nil {
		return
	}
	raw, err := encodeResult(result)
	if err != nil {
		return
	}
	if err := s.cache.Set(ctx, key, raw, cacheTTL); err != nil {
		s.log.WarnContext(ctx, "caching a page", "error", err)
	}
}

// --- helpers ---------------------------------------------------------------------

// withSpanCrafts turns linked craft mentions into a craft filter, unless the
// caller already asked for specific crafts.
func withSpanCrafts(filters domain.Filters, spans []domain.Span) domain.Filters {
	if len(spans) == 0 || len(filters.CraftIDs) > 0 {
		return filters
	}
	seen := make(map[uuid.UUID]struct{}, len(spans))
	for _, span := range spans {
		if _, dup := seen[span.CraftID]; dup {
			continue
		}
		seen[span.CraftID] = struct{}{}
		filters.CraftIDs = append(filters.CraftIDs, span.CraftID)
	}
	return filters
}

// mergeFilters lets the caller's explicit filters win over the extracted ones:
// a buyer who ticked a box means it more than a model reading their sentence.
func mergeFilters(explicit, extracted domain.Filters) domain.Filters {
	if len(explicit.CraftIDs) == 0 {
		explicit.CraftIDs = extracted.CraftIDs
	}
	if len(explicit.Colours) == 0 {
		explicit.Colours = extracted.Colours
	}
	if len(explicit.Materials) == 0 {
		explicit.Materials = extracted.Materials
	}
	if explicit.MinPricePaise == nil {
		explicit.MinPricePaise = extracted.MinPricePaise
	}
	if explicit.MaxPricePaise == nil {
		explicit.MaxPricePaise = extracted.MaxPricePaise
	}
	if explicit.StateCode == nil {
		explicit.StateCode = extracted.StateCode
	}
	if explicit.ListingType == nil {
		explicit.ListingType = extracted.ListingType
	}
	if explicit.MaxLeadTimeDays == nil {
		explicit.MaxLeadTimeDays = extracted.MaxLeadTimeDays
	}
	explicit.GIOnly = explicit.GIOnly || extracted.GIOnly
	explicit.SealedOnly = explicit.SealedOnly || extracted.SealedOnly
	return explicit
}

// matchedTerms is which query tokens appear in the hit's own text, for the
// highlighting the buyer surface does.
func matchedTerms(query domain.Query, hit domain.Hit) []string {
	haystack := domain.Normalise(hit.Label + " " + hit.Description)
	if haystack == "" {
		return nil
	}
	out := make([]string, 0, 4)
	for _, token := range domain.Tokens(query.Canonical) {
		if strings.Contains(haystack, token) {
			out = append(out, token)
		}
	}
	return out
}
