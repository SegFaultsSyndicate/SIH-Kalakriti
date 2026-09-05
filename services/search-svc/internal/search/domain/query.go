// services/search-svc/internal/search/domain/query.go

// Package domain holds search-svc's business types and the pure functions over
// them: normalisation, fusion, diversity. Nothing here talks to Postgres, Redis
// or gRPC, which is why the ranking rules are testable without any of them.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
)

// Filters are the structured narrowings applied to both retrieval legs.
type Filters struct {
	CraftIDs        []uuid.UUID
	Colours         []string
	Materials       []string
	MinPricePaise   *int64
	MaxPricePaise   *int64
	StateCode       *string
	ListingType     *string
	GIOnly          bool
	SealedOnly      bool
	MaxLeadTimeDays *int32
}

// IsZero reports whether anything is actually being narrowed.
func (f Filters) IsZero() bool {
	return len(f.CraftIDs) == 0 && len(f.Colours) == 0 && len(f.Materials) == 0 &&
		f.MinPricePaise == nil && f.MaxPricePaise == nil && f.StateCode == nil &&
		f.ListingType == nil && !f.GIOnly && !f.SealedOnly && f.MaxLeadTimeDays == nil
}

// CacheKey is a stable fingerprint of one query. Two requests that would produce
// the same page share a key; anything that changes the result set changes it.
func (f Filters) CacheKey(normalisedQuery, language string, version int64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "v%d|%s|%s|", version, language, normalisedQuery)

	crafts := make([]string, 0, len(f.CraftIDs))
	for _, id := range f.CraftIDs {
		crafts = append(crafts, id.String())
	}
	sort.Strings(crafts)
	colours, materials := append([]string(nil), f.Colours...), append([]string(nil), f.Materials...)
	sort.Strings(colours)
	sort.Strings(materials)

	fmt.Fprintf(&b, "c=%s|col=%s|mat=%s|min=%v|max=%v|st=%v|t=%v|gi=%t|seal=%t|lead=%v",
		strings.Join(crafts, ","), strings.Join(colours, ","), strings.Join(materials, ","),
		deref(f.MinPricePaise), deref(f.MaxPricePaise), derefString(f.StateCode),
		derefString(f.ListingType), f.GIOnly, f.SealedOnly, deref32(f.MaxLeadTimeDays))

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])[:32]
}

// Span is one craft mention located in the query text.
type Span struct {
	CraftID     uuid.UUID
	CraftCode   string
	DisplayName string
	Start       int
	End         int
	// Text is the substring as the buyer typed it, kept verbatim because it is
	// marked do-not-translate: a craft name is a proper noun of the trade.
	Text string
}

// Query is one buyer query after the whole understanding phase has run.
type Query struct {
	// Raw is what arrived, or what the transcriber heard.
	Raw string
	// Normalised is Raw folded for cache keys and lexical matching.
	Normalised string
	// Canonical is Normalised with transliterated tokens replaced by their
	// canonical spelling, which is what actually goes to retrieval.
	Canonical string
	// Residual is what is left after the understander pulled filters out; it is
	// what gets embedded.
	Residual string
	Language string
	Spans    []Span
	Filters  Filters
}

// DoNotTranslate is the vocabulary that must survive the query pipeline
// untouched: every craft mention, exactly as the ontology spells it.
func (q Query) DoNotTranslate() []string {
	out := make([]string, 0, len(q.Spans))
	for _, span := range q.Spans {
		out = append(out, span.DisplayName)
	}
	return out
}

// Normalise folds query text the same way the index folds documents: NFKC,
// lowercase, Latin diacritics stripped, punctuation collapsed to single spaces.
//
// Indic combining marks are left alone. They are the vowels, so stripping them
// the way unaccent strips an acute accent would merge unrelated words — the
// database side does the same thing, because unaccent() only has rules for
// Latin.
func Normalise(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ToLower(norm.NFKC.String(s))
	s = foldLatinDiacritics(s)

	var b strings.Builder
	b.Grow(len(s))
	pendingSpace := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) {
			if pendingSpace && b.Len() > 0 {
				b.WriteRune(' ')
			}
			pendingSpace = false
			b.WriteRune(r)
			continue
		}
		pendingSpace = true
	}
	return b.String()
}

func foldLatinDiacritics(s string) string {
	if isASCII(s) {
		return s
	}
	decomposed := norm.NFD.String(s)
	var b strings.Builder
	b.Grow(len(decomposed))
	latinBase := false
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			if latinBase {
				continue
			}
			b.WriteRune(r)
			continue
		}
		latinBase = unicode.Is(unicode.Latin, r)
		b.WriteRune(r)
	}
	return norm.NFC.String(b.String())
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= unicode.MaxASCII {
			return false
		}
	}
	return true
}

// Tokens splits normalised text into words.
func Tokens(normalised string) []string {
	if normalised == "" {
		return nil
	}
	return strings.Fields(normalised)
}

// --- retrieval and fusion ------------------------------------------------------

// Candidate is one row a retrieval leg returned.
type Candidate struct {
	ListingID uuid.UUID
	ArtisanID uuid.UUID
	CraftID   uuid.UUID
	Score     float64
}

// Hit is one result after fusion, reranking and the business rules.
type Hit struct {
	ListingID uuid.UUID
	ProductID uuid.UUID
	ArtisanID uuid.UUID
	CraftID   uuid.UUID
	Score     float64
	// MatchedTerms are the query terms the lexical leg matched, for highlighting.
	MatchedTerms []string
	// Label is the listing's title in the requester's language.
	Label       string
	Description string
	GICertified bool
	ListingType string
	// MachineGenerated is true when Label/Description are a translation, not
	// the artisan's own words in the requester's language -- a cross-lingual
	// match when the query also matched a term in it.
	MachineGenerated bool
}

// rrfK damps the contribution of the top of each list. 60 is the constant from
// Cormack, Clarke and Buettcher's original RRF paper (SIGIR 2009), where it was
// chosen empirically over TREC runs: it is large enough that rank 1 and rank 2
// score 1/61 and 1/62 rather than 1 and 1/2, so a single leg cannot dominate the
// fusion, and small enough that the tail past rank ~60 stops mattering. It is
// left as a named constant rather than tuned per query, because the whole point
// of RRF is that it needs no per-corpus calibration.
const rrfK = 60.0

// FuseRRF merges ranked candidate lists by reciprocal rank: a listing's score is
// the sum of 1/(k+rank) over every list it appears in. Only the ordering within
// each list matters, so the two legs' incomparable scores — ts_rank_cd and cosine
// similarity — never have to be put on one scale.
func FuseRRF(lists ...[]Candidate) []Candidate {
	scores := make(map[uuid.UUID]float64)
	merged := make(map[uuid.UUID]Candidate)
	// Insertion order of first appearance, so equal scores break deterministically.
	order := make([]uuid.UUID, 0, 64)

	for _, list := range lists {
		for rank, candidate := range list {
			if _, seen := merged[candidate.ListingID]; !seen {
				merged[candidate.ListingID] = candidate
				order = append(order, candidate.ListingID)
			}
			scores[candidate.ListingID] += 1.0 / (rrfK + float64(rank+1))
		}
	}

	out := make([]Candidate, 0, len(order))
	for _, id := range order {
		candidate := merged[id]
		candidate.Score = scores[id]
		out = append(out, candidate)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// giBoost is what a Geographical Indication is worth in the final ordering. It
// is a nudge, not an override: a GI piece that is a poor match for the query
// still loses to a good match without one.
const giBoost = 0.15

// maxConsecutivePerArtisan stops one prolific maker from owning the page. Three
// in a row is enough to show a coherent set from one workshop without the next
// artisan being pushed below the fold.
const maxConsecutivePerArtisan = 3

// ApplyBusinessRules reorders fused hits under the marketplace's own rules.
//
// Made-to-order is deliberately not penalised for having no stock on hand: most
// of the catalogue is made to order, and ranking it below ready stock would bury
// exactly the artisans this platform exists to reach.
func ApplyBusinessRules(hits []Hit) []Hit {
	boosted := make([]Hit, len(hits))
	copy(boosted, hits)
	for i := range boosted {
		if boosted[i].GICertified {
			boosted[i].Score *= 1 + giBoost
		}
	}
	sort.SliceStable(boosted, func(i, j int) bool { return boosted[i].Score > boosted[j].Score })
	return enforceArtisanDiversity(boosted)
}

// enforceArtisanDiversity walks the ranking and defers a hit that would be the
// fourth in a row from one artisan, putting it back as soon as someone else has
// broken the run. Ordering is otherwise preserved.
func enforceArtisanDiversity(hits []Hit) []Hit {
	out := make([]Hit, 0, len(hits))
	deferred := make([]Hit, 0, 8)

	var lastArtisan uuid.UUID
	var run int

	take := func(hit Hit) {
		if hit.ArtisanID == lastArtisan {
			run++
		} else {
			lastArtisan, run = hit.ArtisanID, 1
		}
		out = append(out, hit)
	}

	for _, hit := range hits {
		if hit.ArtisanID == lastArtisan && run >= maxConsecutivePerArtisan {
			deferred = append(deferred, hit)
			continue
		}
		take(hit)

		// A different artisan just broke the run, so whatever was held back can
		// come straight back in at its old relative position.
		for len(deferred) > 0 && deferred[0].ArtisanID != lastArtisan {
			next := deferred[0]
			deferred = deferred[1:]
			take(next)
		}
	}
	// Anything still held back goes to the end rather than being dropped: a
	// diversity rule must never lose a result.
	return append(out, deferred...)
}

func deref(v *int64) any {
	if v == nil {
		return "-"
	}
	return *v
}

func deref32(v *int32) any {
	if v == nil {
		return "-"
	}
	return *v
}

func derefString(v *string) string {
	if v == nil {
		return "-"
	}
	return *v
}
