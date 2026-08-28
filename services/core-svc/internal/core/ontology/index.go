// services/core-svc/internal/core/ontology/index.go
package ontology

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// maxAliasWords caps how many input words one alias may span. Nothing in the
// ontology is longer, and it bounds the sliding window the resolver walks.
const maxAliasWords = 6

// transliteratedScore is what a match found only through the transliteration
// hook scores; an exact alias hit scores 1.
const transliteratedScore float32 = 0.75

// entry is one alias pointing at one craft, with the craft's display fields
// denormalised so a match needs no second lookup.
type entry struct {
	craftID     uuid.UUID
	code        string
	displayName string
	alias       string
	script      string
	language    string
}

// Index is an immutable alias lookup built from the whole ontology. It is
// replaced wholesale on refresh rather than mutated, so readers never lock.
type Index struct {
	byAlias  map[string][]entry
	crafts   map[uuid.UUID]domain.Craft
	byCode   map[string]uuid.UUID
	maxWords int
	version  string
	builtAt  time.Time
	aliases  int
}

// BuildIndex builds the alias index from a whole ontology snapshot. Aliases are
// keyed by their normalised form, which is what Resolve looks up.
func BuildIndex(crafts []domain.Craft, aliases []domain.CraftAlias) *Index {
	ix := &Index{
		byAlias:  make(map[string][]entry, len(aliases)+len(crafts)),
		crafts:   make(map[uuid.UUID]domain.Craft, len(crafts)),
		byCode:   make(map[string]uuid.UUID, len(crafts)),
		maxWords: 1,
		builtAt:  time.Now().UTC(),
	}

	for _, c := range crafts {
		ix.crafts[c.ID] = c
		ix.byCode[c.Code] = c.ID
		// The canonical name and the slug are aliases of the craft in every
		// sense that matters here, and curators routinely forget to add them.
		ix.add(entry{
			craftID: c.ID, code: c.Code, displayName: c.DisplayName,
			alias: c.DisplayName, script: "Latn", language: "ENGLISH",
		})
		ix.add(entry{
			craftID: c.ID, code: c.Code, displayName: c.DisplayName,
			alias: c.Code, script: "Latn", language: "ENGLISH",
		})
	}

	for _, a := range aliases {
		craft, ok := ix.crafts[a.CraftID]
		if !ok {
			continue // an alias whose craft is not in the snapshot cannot resolve
		}
		ix.add(entry{
			craftID: a.CraftID, code: craft.Code, displayName: craft.DisplayName,
			alias: a.Alias, script: a.Script, language: a.Language,
		})
	}

	// Deterministic ordering makes both the version hash and the pick between
	// two entries for the same spelling stable across rebuilds.
	keys := make([]string, 0, len(ix.byAlias))
	for k, entries := range ix.byAlias {
		keys = append(keys, k)
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].script != entries[j].script {
				return entries[i].script < entries[j].script
			}
			if entries[i].code != entries[j].code {
				return entries[i].code < entries[j].code
			}
			return entries[i].alias < entries[j].alias
		})
	}
	sort.Strings(keys)

	sum := sha256.New()
	for _, k := range keys {
		for _, e := range ix.byAlias[k] {
			fmt.Fprintf(sum, "%s|%s|%s|%s\n", k, e.code, e.script, e.craftID)
		}
	}
	ix.version = hex.EncodeToString(sum.Sum(nil))[:16]
	return ix
}

// add records one alias under its normalised key, skipping blanks and exact
// duplicates of the same craft.
func (ix *Index) add(e entry) {
	key := Normalise(e.alias)
	if key == "" {
		return
	}
	for _, existing := range ix.byAlias[key] {
		if existing.craftID == e.craftID {
			return
		}
	}
	ix.byAlias[key] = append(ix.byAlias[key], e)
	ix.aliases++
	if w := strings.Count(key, " ") + 1; w > ix.maxWords && w <= maxAliasWords {
		ix.maxWords = w
	}
}

// Version is a content hash of the indexed ontology. Two replicas holding the
// same version hold the same index, which is what the Redis invalidation
// channel carries.
func (ix *Index) Version() string { return ix.version }

// BuiltAt is when this index was built, UTC.
func (ix *Index) BuiltAt() time.Time { return ix.builtAt }

// Stats reports how much of the ontology is indexed.
func (ix *Index) Stats() (crafts, aliases int) { return len(ix.crafts), ix.aliases }

// Craft returns one craft by id.
func (ix *Index) Craft(id uuid.UUID) (domain.Craft, bool) {
	c, ok := ix.crafts[id]
	return c, ok
}

// CraftByCode returns one craft by its slug.
func (ix *Index) CraftByCode(code string) (domain.Craft, bool) {
	id, ok := ix.byCode[code]
	if !ok {
		return domain.Craft{}, false
	}
	return ix.crafts[id], true
}

// Crafts returns every craft in the index, ordered by code.
func (ix *Index) Crafts() []domain.Craft {
	out := make([]domain.Craft, 0, len(ix.crafts))
	for _, c := range ix.crafts {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// token is one word of the input, with its byte span in the original string.
type token struct {
	norm  string
	start int
	end   int
}

// tokenise splits text into words, keeping each word's offsets in the original
// string so a match can report the span the caller passed in rather than the
// normalised one.
func tokenise(text string) []token {
	var (
		out   []token
		start = -1
	)
	for i, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			out = appendToken(out, text, start, i)
			start = -1
		}
	}
	if start >= 0 {
		out = appendToken(out, text, start, len(text))
	}
	return out
}

// appendToken normalises one word and drops it if normalisation left nothing.
func appendToken(out []token, text string, start, end int) []token {
	n := Normalise(text[start:end])
	if n == "" {
		return out
	}
	return append(out, token{norm: n, start: start, end: end})
}

// Resolve finds every craft mentioned in text and reports the span each mention
// occupies. Matching is longest-first and non-overlapping: in "ajrakh block
// print" the three-word alias wins over "block print" if both are in the
// ontology, and neither token is then offered to a shorter alias.
//
// language biases which alias is preferred when one spelling belongs to more
// than one craft; it never restricts what can match, because a buyer typing in
// Hindi will still type the Latin spelling half the time. tr may be nil.
func (ix *Index) Resolve(ctx context.Context, text, language string, tr Transliterator) ([]domain.CraftMatch, error) {
	tokens := tokenise(text)
	if len(tokens) == 0 {
		return nil, nil
	}

	used := make([]bool, len(tokens))
	var matches []domain.CraftMatch

	maxWords := ix.maxWords
	if maxWords > maxAliasWords {
		maxWords = maxAliasWords
	}

	for width := maxWords; width >= 1; width-- {
		for i := 0; i+width <= len(tokens); i++ {
			if anyUsed(used, i, i+width) {
				continue
			}
			key := joinTokens(tokens[i : i+width])
			e, ok := ix.pick(key, language)
			if !ok {
				continue
			}
			matches = append(matches, matchFrom(e, text, tokens[i].start, tokens[i+width-1].end, 1, domain.MatchExact))
			markUsed(used, i, i+width)
		}
	}

	if tr != nil {
		more, err := ix.resolveTransliterated(ctx, text, language, tokens, used, maxWords, tr)
		if err != nil {
			return nil, err
		}
		matches = append(matches, more...)
	}

	sort.Slice(matches, func(i, j int) bool { return matches[i].Start < matches[j].Start })
	return matches, nil
}

// resolveTransliterated makes a second pass over the spans the exact pass left
// unmatched, asking the transliterator for candidate spellings of each.
func (ix *Index) resolveTransliterated(
	ctx context.Context,
	text, language string,
	tokens []token,
	used []bool,
	maxWords int,
	tr Transliterator,
) ([]domain.CraftMatch, error) {
	var matches []domain.CraftMatch
	for width := maxWords; width >= 1; width-- {
		for i := 0; i+width <= len(tokens); i++ {
			if anyUsed(used, i, i+width) {
				continue
			}
			start, end := tokens[i].start, tokens[i+width-1].end
			candidates, err := tr.Transliterate(ctx, text[start:end], language)
			if err != nil {
				return nil, fmt.Errorf("transliterating %q: %w", text[start:end], err)
			}
			for _, candidate := range candidates {
				e, ok := ix.pick(Normalise(candidate), language)
				if !ok {
					continue
				}
				matches = append(matches, matchFrom(e, text, start, end, transliteratedScore, domain.MatchTransliterated))
				markUsed(used, i, i+width)
				break
			}
		}
	}
	return matches, nil
}

// pick chooses the best entry for a normalised key, preferring the caller's
// language and otherwise taking the first in the index's stable order.
func (ix *Index) pick(key, language string) (entry, bool) {
	entries := ix.byAlias[key]
	if len(entries) == 0 {
		return entry{}, false
	}
	if language != "" {
		for _, e := range entries {
			if e.language == language {
				return e, true
			}
		}
	}
	return entries[0], true
}

// matchFrom assembles a match from an index entry and a span of the input.
func matchFrom(e entry, text string, start, end int, score float32, source domain.MatchSource) domain.CraftMatch {
	return domain.CraftMatch{
		CraftID:     e.craftID,
		Code:        e.code,
		DisplayName: e.displayName,
		Alias:       e.alias,
		MatchedText: text[start:end],
		Start:       start,
		End:         end,
		Script:      e.script,
		Language:    e.language,
		Score:       score,
		Source:      source,
	}
}

func joinTokens(tokens []token) string {
	parts := make([]string, 0, len(tokens))
	for _, t := range tokens {
		parts = append(parts, t.norm)
	}
	return strings.Join(parts, " ")
}

func anyUsed(used []bool, from, to int) bool {
	for i := from; i < to; i++ {
		if used[i] {
			return true
		}
	}
	return false
}

func markUsed(used []bool, from, to int) {
	for i := from; i < to; i++ {
		used[i] = true
	}
}
