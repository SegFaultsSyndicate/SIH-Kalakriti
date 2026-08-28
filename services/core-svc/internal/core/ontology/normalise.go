// services/core-svc/internal/core/ontology/normalise.go

// Package ontology owns the craft graph: loading it from CSV, holding it in
// memory as an alias index, caching that index in Redis, and resolving free text
// to craft ids. It is the entity-linking primitive search-svc builds on, so it
// depends on nothing above the domain package.
package ontology

import (
	"context"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Normalise folds a fragment of text into the form the alias index is keyed by:
// NFKC-composed, lowercased, Latin diacritics stripped, every run of
// non-alphanumeric characters collapsed to one space, trimmed.
//
// Diacritic stripping is deliberately confined to Latin. In Devanagari and the
// other Indic scripts the combining marks are the vowels, so folding them the
// way unaccent folds an acute accent would merge unrelated words. The database
// side of this is `unaccent()` in SearchCraftAliases, which behaves the same way
// for the Latin spellings it is used on.
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

// foldLatinDiacritics drops combining marks that sit on a Latin base letter,
// leaving marks on every other script untouched.
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

// isASCII is the fast path: an ASCII string carries no combining marks.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= unicode.MaxASCII {
			return false
		}
	}
	return true
}

// Transliterator turns a fragment of text into candidate spellings in other
// scripts, so a buyer typing "अजरख" can reach an alias stored only in Latin.
// Batch 10 wires this to Bhashini; until then NoTransliteration is installed and
// resolution falls back to exact matching over the stored aliases alone.
type Transliterator interface {
	// Transliterate returns zero or more candidate spellings of text. language
	// is the caller's language code, e.g. "HINDI", and may be empty.
	Transliterate(ctx context.Context, text, language string) ([]string, error)
}

// noTransliteration is the default: it proposes nothing and never fails.
type noTransliteration struct{}

// NoTransliteration returns the no-op transliterator used until Bhashini is wired.
func NoTransliteration() Transliterator { return noTransliteration{} }

// Transliterate returns no candidates.
func (noTransliteration) Transliterate(context.Context, string, string) ([]string, error) {
	return nil, nil
}
