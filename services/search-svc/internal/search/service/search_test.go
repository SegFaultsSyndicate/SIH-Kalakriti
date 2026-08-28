// services/search-svc/internal/search/service/search_test.go
package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/segfaultsyndicate/kalakriti/services/search-svc/internal/search/domain"
)

// --- fakes ---------------------------------------------------------------------

// fakeStore is the index: a handful of listings, matched lexically by substring
// and densely by a canned neighbour list, which is enough to exercise fusion and
// the filters without a database.
type fakeStore struct {
	lexical  []domain.Candidate
	vector   []domain.Candidate
	hits     map[uuid.UUID]domain.Hit
	siblings []domain.Sibling
	aliases  []domain.Suggestion
	queries  []domain.Suggestion

	// filtersSeen records what each leg was asked to narrow by, so a test can
	// prove the structured filters actually reached the database.
	filtersSeen []domain.Filters
	lexicalErr  error
	vectorErr   error
	recorded    int
}

func (s *fakeStore) SearchLexical(_ context.Context, _, _ string, filters domain.Filters, _ int32) ([]domain.Candidate, error) {
	s.filtersSeen = append(s.filtersSeen, filters)
	if s.lexicalErr != nil {
		return nil, s.lexicalErr
	}
	return applyFilters(s.lexical, filters, s.hits), nil
}

func (s *fakeStore) SearchVector(_ context.Context, _ []float32, _ string, filters domain.Filters, _ int32) ([]domain.Candidate, error) {
	s.filtersSeen = append(s.filtersSeen, filters)
	if s.vectorErr != nil {
		return nil, s.vectorErr
	}
	return applyFilters(s.vector, filters, s.hits), nil
}

// applyFilters is the fake's stand-in for the WHERE clause: only the narrowings
// the tests exercise are modelled.
func applyFilters(candidates []domain.Candidate, filters domain.Filters, hits map[uuid.UUID]domain.Hit) []domain.Candidate {
	out := make([]domain.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if len(filters.CraftIDs) > 0 && !containsUUID(filters.CraftIDs, candidate.CraftID) {
			continue
		}
		if filters.GIOnly && !hits[candidate.ListingID].GICertified {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

func (s *fakeStore) HydrateHits(_ context.Context, ids []uuid.UUID, _ string) ([]domain.Hit, error) {
	out := make([]domain.Hit, 0, len(ids))
	for _, id := range ids {
		if hit, ok := s.hits[id]; ok {
			out = append(out, hit)
		}
	}
	return out, nil
}

func (s *fakeStore) SiblingCrafts(context.Context, []uuid.UUID) ([]domain.Sibling, error) {
	return s.siblings, nil
}

func (s *fakeStore) SuggestAliases(context.Context, string, string, int32) ([]domain.Suggestion, error) {
	return s.aliases, nil
}

func (s *fakeStore) SuggestQueries(context.Context, string, int32) ([]domain.Suggestion, error) {
	return s.queries, nil
}

func (s *fakeStore) RecordQuery(context.Context, string, string, int32) error {
	s.recorded++
	return nil
}

// fakeInference is ml-svc: deterministic vectors, an optional rerank ordering,
// and a canned transcript.
type fakeInference struct {
	embedCalls  int
	rerankCalls int
	rerankBy    map[uuid.UUID]float64
	transcript  string
	embedErr    error
	rerankErr   error
	// embeddedText is what the dense leg actually asked to embed, which is how a
	// test sees that the residual text is what gets vectorised.
	embeddedText string
}

func (f *fakeInference) Embed(_ context.Context, texts []string, _ string) ([][]float32, error) {
	f.embedCalls++
	if len(texts) > 0 {
		f.embeddedText = texts[0]
	}
	if f.embedErr != nil {
		return nil, f.embedErr
	}
	out := make([][]float32, 0, len(texts))
	for range texts {
		out = append(out, []float32{1, 0, 0})
	}
	return out, nil
}

func (f *fakeInference) Rerank(_ context.Context, _ string, candidates []domain.RerankCandidate) (map[uuid.UUID]float64, error) {
	f.rerankCalls++
	if f.rerankErr != nil {
		return nil, f.rerankErr
	}
	if f.rerankBy != nil {
		return f.rerankBy, nil
	}
	out := make(map[uuid.UUID]float64, len(candidates))
	for i, candidate := range candidates {
		out[candidate.ListingID] = float64(len(candidates) - i)
	}
	return out, nil
}

func (f *fakeInference) Transcribe(context.Context, string, string) (string, float32, error) {
	return f.transcript, 0.94, nil
}

// fakeOntology links craft mentions by exact alias match.
type fakeOntology struct {
	aliases map[string]domain.Span
	err     error
}

func (f *fakeOntology) ResolveAlias(_ context.Context, text, _ string) ([]domain.Span, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []domain.Span
	for alias, span := range f.aliases {
		if idx := strings.Index(text, alias); idx >= 0 {
			span.Start, span.End, span.Text = idx, idx+len(alias), alias
			out = append(out, span)
		}
	}
	return out, nil
}

// fakeTransliterator maps a romanisation onto the spelling the index holds.
type fakeTransliterator struct {
	table map[string]string
	calls int
}

func (f *fakeTransliterator) Canonicalise(_ context.Context, token, _ string) (string, error) {
	f.calls++
	if canonical, ok := f.table[token]; ok {
		return canonical, nil
	}
	return token, nil
}

// slowUnderstander never answers inside the budget.
type slowUnderstander struct{ calls int }

func (s *slowUnderstander) Understand(ctx context.Context, _, _ string) (domain.Filters, string, error) {
	s.calls++
	<-ctx.Done()
	return domain.Filters{}, "", ctx.Err()
}

type fakeCache struct {
	values  map[string][]byte
	version int64
	hits    int
}

func newFakeCache() *fakeCache { return &fakeCache{values: map[string][]byte{}} }

func (c *fakeCache) Get(_ context.Context, key string) ([]byte, error) {
	value, ok := c.values[key]
	if !ok {
		return nil, errors.New("miss")
	}
	c.hits++
	return value, nil
}

func (c *fakeCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	c.values[key] = value
	return nil
}

func (c *fakeCache) Version(context.Context) (int64, error) { return c.version, nil }

func (c *fakeCache) BumpVersion(context.Context) error {
	c.version++
	return nil
}

// --- fixture -------------------------------------------------------------------

var (
	ajrakhCraft = uuid.MustParse("018f0000-0000-7000-8000-00000000c001")
	bagruCraft  = uuid.MustParse("018f0000-0000-7000-8000-00000000c002")
	artisanA    = uuid.MustParse("018f0000-0000-7000-8000-00000000a001")
	artisanB    = uuid.MustParse("018f0000-0000-7000-8000-00000000a002")
)

func listingID(n byte) uuid.UUID {
	id := uuid.MustParse("018f0000-0000-7000-8000-00000000f000")
	id[15] = n
	return id
}

// hindiIndex is the cross-lingual case: the listings were authored in Hindi and
// carry the craft's Latin aliases in the same document, which is what an English
// query matches on.
func hindiIndex() map[uuid.UUID]domain.Hit {
	return map[uuid.UUID]domain.Hit{
		listingID(1): {
			ListingID: listingID(1), ArtisanID: artisanA, CraftID: ajrakhCraft,
			Label:       "अजरख दुपट्टा",
			Description: "हाथ से छपा हुआ ajrakh block printing सूती दुपट्टा",
			GICertified: true, ListingType: "MADE_TO_ORDER",
		},
		listingID(2): {
			ListingID: listingID(2), ArtisanID: artisanA, CraftID: ajrakhCraft,
			Label: "अजरख स्टोल", Description: "ajrakh indigo cotton stole",
			ListingType: "MADE_TO_ORDER",
		},
		listingID(3): {
			ListingID: listingID(3), ArtisanID: artisanB, CraftID: bagruCraft,
			Label: "बगरू चादर", Description: "bagru block printing bedcover",
			ListingType: "READY_STOCK",
		},
	}
}

func newSearch(t *testing.T, store *fakeStore, inference *fakeInference, opts ...func(*Search)) *Search {
	t.Helper()
	s := NewSearch(store, inference,
		&fakeOntology{aliases: map[string]domain.Span{
			"ajrakh": {CraftID: ajrakhCraft, CraftCode: "ajrakh-block-printing", DisplayName: "Ajrakh Block Printing"},
		}},
		&fakeTransliterator{table: map[string]string{"अजरख": "ajrakh", "ajarakh": "ajrakh"}},
		NewRuleUnderstander(),
		nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// --- tests ---------------------------------------------------------------------

// TestEnglishQueryMatchesAHindiListing is the case the whole design exists for:
// no translation of the query, no translation of the listing, one shared row.
func TestEnglishQueryMatchesAHindiListing(t *testing.T) {
	t.Parallel()
	hits := hindiIndex()
	store := &fakeStore{
		hits:    hits,
		lexical: []domain.Candidate{{ListingID: listingID(1), ArtisanID: artisanA, CraftID: ajrakhCraft}},
		vector:  []domain.Candidate{{ListingID: listingID(2), ArtisanID: artisanA, CraftID: ajrakhCraft}},
	}
	search := newSearch(t, store, &fakeInference{})

	result, err := search.Do(context.Background(), Request{Query: "ajrakh cotton stole", Language: "ENGLISH"})
	require.NoError(t, err)
	require.NotEmpty(t, result.Hits)
	require.Equal(t, "अजरख दुपट्टा", result.Hits[0].Label, "the label is rendered in the row's own language")

	// The English query was never translated; it linked to a craft and searched.
	require.Len(t, result.Understood.Spans, 1)
	require.Equal(t, ajrakhCraft, result.Understood.Spans[0].CraftID)
	require.Equal(t, []string{"Ajrakh Block Printing"}, result.Understood.DoNotTranslate())
}

// TestSpellingsConvergeOnOneResultSet is the transliteration case: three ways of
// writing one craft, one answer.
func TestSpellingsConvergeOnOneResultSet(t *testing.T) {
	t.Parallel()

	ids := func(hits []domain.Hit) []uuid.UUID {
		out := make([]uuid.UUID, 0, len(hits))
		for _, hit := range hits {
			out = append(out, hit.ListingID)
		}
		return out
	}

	var first []uuid.UUID
	var firstKey string
	for _, spelling := range []string{"ajrakh", "Ajarakh", "अजरख"} {
		hits := hindiIndex()
		store := &fakeStore{
			hits: hits,
			lexical: []domain.Candidate{
				{ListingID: listingID(1), ArtisanID: artisanA, CraftID: ajrakhCraft},
				{ListingID: listingID(2), ArtisanID: artisanA, CraftID: ajrakhCraft},
			},
			vector: []domain.Candidate{{ListingID: listingID(2), ArtisanID: artisanA, CraftID: ajrakhCraft}},
		}
		search := newSearch(t, store, &fakeInference{})

		result, err := search.Do(context.Background(), Request{Query: spelling, Language: "ENGLISH"})
		require.NoError(t, err, spelling)
		require.NotEmpty(t, result.Hits, spelling)

		if first == nil {
			first, firstKey = ids(result.Hits), result.QueryID
			continue
		}
		require.Equal(t, first, ids(result.Hits), "%q must return the same set", spelling)
		// And they share a cache entry, because they are the same query.
		require.Equal(t, firstKey, result.QueryID, "%q must share a cache key", spelling)
	}
}

// TestRRFAgainstAHandComputedFixture pins the fusion arithmetic.
func TestRRFAgainstAHandComputedFixture(t *testing.T) {
	t.Parallel()

	a, b, c := listingID(1), listingID(2), listingID(3)
	lexical := []domain.Candidate{{ListingID: a}, {ListingID: b}, {ListingID: c}}
	dense := []domain.Candidate{{ListingID: c}, {ListingID: a}}

	fused := domain.FuseRRF(lexical, dense)

	// By hand, with k = 60:
	//   a: 1/61 + 1/62 = 0.0163934 + 0.0161290 = 0.0325224
	//   c: 1/63 + 1/61 = 0.0158730 + 0.0163934 = 0.0322664
	//   b: 1/62                                 = 0.0161290
	require.Equal(t, []uuid.UUID{a, c, b}, []uuid.UUID{fused[0].ListingID, fused[1].ListingID, fused[2].ListingID})
	require.InDelta(t, 1.0/61+1.0/62, fused[0].Score, 1e-9)
	require.InDelta(t, 1.0/63+1.0/61, fused[1].Score, 1e-9)
	require.InDelta(t, 1.0/62, fused[2].Score, 1e-9)

	// A document ranked first by one leg alone must not beat one ranked second by
	// both: that is the whole point of the damping constant.
	require.Less(t, fused[2].Score, fused[1].Score)
}

func TestRRFIsStableAcrossLegOrder(t *testing.T) {
	t.Parallel()
	a, b := listingID(1), listingID(2)
	one := domain.FuseRRF([]domain.Candidate{{ListingID: a}}, []domain.Candidate{{ListingID: b}})
	two := domain.FuseRRF([]domain.Candidate{{ListingID: b}}, []domain.Candidate{{ListingID: a}})
	require.InDelta(t, one[0].Score, two[0].Score, 1e-12)
}

// TestZeroResultsWidenToSiblings is the never-an-empty-page rule.
func TestZeroResultsWidenToSiblings(t *testing.T) {
	t.Parallel()
	hits := hindiIndex()
	store := &fakeStore{
		hits: hits,
		// Nothing under ajrakh; the bagru listing is only reachable once the
		// craft filter widens.
		lexical:  []domain.Candidate{{ListingID: listingID(3), ArtisanID: artisanB, CraftID: bagruCraft}},
		vector:   nil,
		siblings: []domain.Sibling{{CraftID: bagruCraft, Code: "bagru-block-printing", DisplayName: "Bagru Block Printing", Kind: "SHARES_TECHNIQUE"}},
	}
	search := newSearch(t, store, &fakeInference{})

	result, err := search.Do(context.Background(), Request{Query: "ajrakh", Language: "ENGLISH"})
	require.NoError(t, err)
	require.NotEmpty(t, result.Hits, "the page must never come back empty")
	require.Equal(t, listingID(3), result.Hits[0].ListingID)
	require.Equal(t, []string{"Bagru Block Printing"}, result.DidYouMean)
}

// TestSlowUnderstandingDegradesGracefully is the 500ms budget: the search runs
// on the whole query rather than failing.
func TestSlowUnderstandingDegradesGracefully(t *testing.T) {
	t.Parallel()
	hits := hindiIndex()
	store := &fakeStore{
		hits:    hits,
		lexical: []domain.Candidate{{ListingID: listingID(1), ArtisanID: artisanA, CraftID: ajrakhCraft}},
	}
	slow := &slowUnderstander{}
	search := newSearch(t, store, &fakeInference{}, func(s *Search) { s.understander = slow })

	started := time.Now()
	result, err := search.Do(context.Background(), Request{Query: "ajrakh cotton under 2000", Language: "ENGLISH"})
	elapsed := time.Since(started)

	require.NoError(t, err)
	require.NotEmpty(t, result.Hits)
	require.Equal(t, 1, slow.calls)
	// It waited for the budget and no longer, and searched the whole query.
	require.GreaterOrEqual(t, elapsed, understandingBudget)
	require.Less(t, elapsed, understandingBudget*3)
	require.Equal(t, result.Understood.Canonical, result.Understood.Residual)
}

// TestFiltersAndResidualBothAffectResults is the second acceptance criterion.
func TestFiltersAndResidualBothAffectResults(t *testing.T) {
	t.Parallel()

	t.Run("a structured filter narrows the candidate set", func(t *testing.T) {
		t.Parallel()
		hits := hindiIndex()
		store := &fakeStore{
			hits: hits,
			lexical: []domain.Candidate{
				{ListingID: listingID(1), ArtisanID: artisanA, CraftID: ajrakhCraft},
				{ListingID: listingID(3), ArtisanID: artisanB, CraftID: bagruCraft},
			},
		}
		search := newSearch(t, store, &fakeInference{})

		// "gi tagged" is read off the text and reaches the database as a filter.
		result, err := search.Do(context.Background(), Request{Query: "block printing gi tagged", Language: "ENGLISH"})
		require.NoError(t, err)
		require.True(t, store.filtersSeen[0].GIOnly)
		require.Len(t, result.Hits, 1)
		require.Equal(t, listingID(1), result.Hits[0].ListingID, "only the GI listing survives")
	})

	t.Run("the residual text is what gets embedded", func(t *testing.T) {
		t.Parallel()
		hits := hindiIndex()
		store := &fakeStore{hits: hits, vector: []domain.Candidate{{ListingID: listingID(2), ArtisanID: artisanA, CraftID: ajrakhCraft}}}
		inference := &fakeInference{}
		search := newSearch(t, store, inference)

		_, err := search.Do(context.Background(), Request{Query: "indigo stole under 2000", Language: "ENGLISH"})
		require.NoError(t, err)

		// The budget clause is pulled out of the text; the descriptive words stay.
		require.NotContains(t, inference.embeddedText, "under 2000")
		require.Contains(t, inference.embeddedText, "indigo stole")
		require.NotNil(t, store.filtersSeen[0].MaxPricePaise)
		require.Equal(t, int64(200000), *store.filtersSeen[0].MaxPricePaise, "rupees are stored as paise")
	})
}

func TestBusinessRules(t *testing.T) {
	t.Parallel()

	t.Run("made to order is not down-ranked for having no stock", func(t *testing.T) {
		t.Parallel()
		hits := []domain.Hit{
			{ListingID: listingID(1), ArtisanID: artisanA, Score: 1.0, ListingType: "MADE_TO_ORDER"},
			{ListingID: listingID(2), ArtisanID: artisanB, Score: 0.9, ListingType: "READY_STOCK"},
		}
		ordered := domain.ApplyBusinessRules(hits)
		require.Equal(t, listingID(1), ordered[0].ListingID)
	})

	t.Run("a gi tag is a nudge, not an override", func(t *testing.T) {
		t.Parallel()
		ordered := domain.ApplyBusinessRules([]domain.Hit{
			{ListingID: listingID(1), ArtisanID: artisanA, Score: 1.0},
			{ListingID: listingID(2), ArtisanID: artisanB, Score: 0.95, GICertified: true},
		})
		// 0.95 * 1.15 = 1.0925 beats 1.0: a near-tie goes to the GI piece.
		require.Equal(t, listingID(2), ordered[0].ListingID)

		ordered = domain.ApplyBusinessRules([]domain.Hit{
			{ListingID: listingID(1), ArtisanID: artisanA, Score: 1.0},
			{ListingID: listingID(2), ArtisanID: artisanB, Score: 0.5, GICertified: true},
		})
		// A poor match does not win just for carrying a tag.
		require.Equal(t, listingID(1), ordered[0].ListingID)
	})

	t.Run("no artisan takes more than three consecutive places", func(t *testing.T) {
		t.Parallel()
		hits := make([]domain.Hit, 0, 6)
		for i := byte(1); i <= 5; i++ {
			hits = append(hits, domain.Hit{ListingID: listingID(i), ArtisanID: artisanA, Score: float64(10 - i)})
		}
		hits = append(hits, domain.Hit{ListingID: listingID(9), ArtisanID: artisanB, Score: 1})

		ordered := domain.ApplyBusinessRules(hits)
		require.Len(t, ordered, 6, "diversity must not drop a result")

		var run int
		var last uuid.UUID
		for _, hit := range ordered {
			if hit.ArtisanID == last {
				run++
			} else {
				last, run = hit.ArtisanID, 1
			}
			require.LessOrEqual(t, run, 3)
		}
		require.Equal(t, artisanB, ordered[3].ArtisanID, "the other artisan breaks the run")
	})
}

func TestRerankIsSkippedForTinyResultSets(t *testing.T) {
	t.Parallel()
	hits := hindiIndex()
	store := &fakeStore{hits: hits, lexical: []domain.Candidate{
		{ListingID: listingID(1), ArtisanID: artisanA, CraftID: ajrakhCraft},
		{ListingID: listingID(2), ArtisanID: artisanA, CraftID: ajrakhCraft},
	}}
	inference := &fakeInference{}
	search := newSearch(t, store, inference)

	_, err := search.Do(context.Background(), Request{Query: "ajrakh", Language: "ENGLISH"})
	require.NoError(t, err)
	require.Zero(t, inference.rerankCalls, "two hits are not worth a model call")
}

func TestOneFailingLegStillSearches(t *testing.T) {
	t.Parallel()
	hits := hindiIndex()
	store := &fakeStore{
		hits:      hits,
		vectorErr: errors.New("pgvector timeout"),
		lexical:   []domain.Candidate{{ListingID: listingID(1), ArtisanID: artisanA, CraftID: ajrakhCraft}},
	}
	search := newSearch(t, store, &fakeInference{})

	result, err := search.Do(context.Background(), Request{Query: "ajrakh", Language: "ENGLISH"})
	require.NoError(t, err)
	require.Len(t, result.Hits, 1)
}

func TestBothLegsFailingIsAnError(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		hits:       hindiIndex(),
		lexicalErr: errors.New("postgres down"),
		vectorErr:  errors.New("postgres down"),
	}
	search := newSearch(t, store, &fakeInference{})

	_, err := search.Do(context.Background(), Request{Query: "ajrakh", Language: "ENGLISH"})
	require.Error(t, err)
}

func TestCacheServesTheSecondIdenticalQuery(t *testing.T) {
	t.Parallel()
	cache := newFakeCache()
	store := &fakeStore{hits: hindiIndex(), lexical: []domain.Candidate{
		{ListingID: listingID(1), ArtisanID: artisanA, CraftID: ajrakhCraft},
	}}
	search := newSearch(t, store, &fakeInference{}, func(s *Search) { s.cache = cache })

	first, err := search.Do(context.Background(), Request{Query: "ajrakh", Language: "ENGLISH"})
	require.NoError(t, err)
	require.False(t, first.FromCache)

	second, err := search.Do(context.Background(), Request{Query: "ajrakh", Language: "ENGLISH"})
	require.NoError(t, err)
	require.True(t, second.FromCache)
	require.Equal(t, first.Hits[0].ListingID, second.Hits[0].ListingID)

	// A publish retires the page without touching a single key.
	require.NoError(t, cache.BumpVersion(context.Background()))
	third, err := search.Do(context.Background(), Request{Query: "ajrakh", Language: "ENGLISH"})
	require.NoError(t, err)
	require.False(t, third.FromCache, "the version bump must invalidate the cached page")
}

func TestVoiceQueryIsTranscribedFirst(t *testing.T) {
	t.Parallel()
	store := &fakeStore{hits: hindiIndex(), lexical: []domain.Candidate{
		{ListingID: listingID(1), ArtisanID: artisanA, CraftID: ajrakhCraft},
	}}
	inference := &fakeInference{transcript: "अजरख दुपट्टा"}
	search := newSearch(t, store, inference)

	result, err := search.Do(context.Background(), Request{AudioObjectKey: "voice/q.m4a", Language: "HINDI"})
	require.NoError(t, err)
	require.Equal(t, "अजरख दुपट्टा", result.Transcript)
	require.InDelta(t, 0.94, result.TranscriptConfidence, 0.001)
	require.NotEmpty(t, result.Hits)
}

func TestSuggestPutsCraftsAheadOfPastQueries(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		hits:    hindiIndex(),
		aliases: []domain.Suggestion{{Text: "Ajrakh Block Printing", Kind: domain.SuggestCraft, Score: 0.9}},
		queries: []domain.Suggestion{{Text: "ajrakh dupatta", Kind: domain.SuggestQuery, Score: 0.7}},
	}
	search := newSearch(t, store, &fakeInference{})

	suggestions, err := search.Suggest(context.Background(), "ajr", "ENGLISH", 10)
	require.NoError(t, err)
	require.Len(t, suggestions, 2)
	require.Equal(t, domain.SuggestCraft, suggestions[0].Kind)
}

func TestNormaliseFoldsTheWaysBuyersType(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"AJRAKH", "ajrakh"},
		{"  ajrakh   dupatta \n", "ajrakh dupatta"},
		{"ajrakh-block-printing", "ajrakh block printing"},
		{"Kaśīdākārī", "kasidakari"},
		{"अजरख", "अजरख"},
		{"बांधणी", "बांधणी"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, domain.Normalise(tt.in), tt.in)
	}
}

func TestCacheKeyChangesWithEveryNarrowing(t *testing.T) {
	t.Parallel()
	base := domain.Filters{}
	key := base.CacheKey("ajrakh", "ENGLISH", 1)

	gi := domain.Filters{GIOnly: true}
	require.NotEqual(t, key, gi.CacheKey("ajrakh", "ENGLISH", 1))
	require.NotEqual(t, key, base.CacheKey("ajrakh", "HINDI", 1))
	require.NotEqual(t, key, base.CacheKey("bagru", "ENGLISH", 1))
	require.NotEqual(t, key, base.CacheKey("ajrakh", "ENGLISH", 2))

	// Filter order must not matter, or two identical searches miss the cache.
	one := domain.Filters{Colours: []string{"indigo", "red"}}
	two := domain.Filters{Colours: []string{"red", "indigo"}}
	require.Equal(t, one.CacheKey("q", "ENGLISH", 1), two.CacheKey("q", "ENGLISH", 1))
}

func TestIndexDocumentCarriesEveryScriptsAlias(t *testing.T) {
	t.Parallel()
	source := domain.IndexSource{
		Language: "HINDI", Title: "अजरख दुपट्टा", Description: "हाथ से छपा",
		CraftName: "Ajrakh Block Printing", CraftAliases: []string{"Ajrakh", "अजरख", "અજરખ"},
		Materials: []string{"cotton"}, Colours: []string{"indigo"},
		Region: []string{"Kutch", "IN-GJ"},
	}
	doc := source.Document()

	// One row holds the Hindi copy and the Latin aliases, which is how an English
	// query reaches it without either side being translated.
	for _, want := range []string{"अजरख दुपट्टा", "Ajrakh", "अजरख", "અજરખ", "cotton", "Kutch"} {
		require.Contains(t, doc.Document, want)
	}
	require.Equal(t, 1, strings.Count(strings.ToLower(doc.Document), "ajrakh block printing"),
		"a repeated term is written once")
}

func containsUUID(ids []uuid.UUID, want uuid.UUID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

var _ = math.Abs
