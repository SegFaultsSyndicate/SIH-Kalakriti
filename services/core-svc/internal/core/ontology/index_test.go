// services/core-svc/internal/core/ontology/index_test.go
package ontology

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// testOntology is a three-craft slice of the real ontology, with the alias
// spellings that actually arrive from voice search and from the GI registry.
func testOntology() ([]domain.Craft, []domain.CraftAlias) {
	ajrakh := uuid.MustParse("018f0000-0000-7000-8000-00000000a001")
	muga := uuid.MustParse("018f0000-0000-7000-8000-00000000a002")
	blue := uuid.MustParse("018f0000-0000-7000-8000-00000000a003")

	crafts := []domain.Craft{
		{ID: ajrakh, Code: "ajrakh-block-printing", DisplayName: "Ajrakh Block Printing"},
		{ID: muga, Code: "assam-muga-weaving", DisplayName: "Assam Muga Weaving"},
		{ID: blue, Code: "blue-pottery", DisplayName: "Blue Pottery"},
	}
	aliases := []domain.CraftAlias{
		{CraftID: ajrakh, Alias: "Ajrakh", Script: "Latn", Language: "ENGLISH"},
		{CraftID: ajrakh, Alias: "Ajarakh", Script: "Latn", Language: "ENGLISH"},
		{CraftID: ajrakh, Alias: "अजरख", Script: "Deva", Language: "HINDI"},
		{CraftID: ajrakh, Alias: "અજરખ", Script: "Gujr", Language: "GUJARATI"},
		{CraftID: muga, Alias: "Muga", Script: "Latn", Language: "ENGLISH"},
		{CraftID: muga, Alias: "মুগা", Script: "Beng", Language: "ASSAMESE"},
		{CraftID: blue, Alias: "Jaipur Blue Pottery", Script: "Latn", Language: "ENGLISH"},
	}
	return crafts, aliases
}

func TestResolveAliasAcrossScripts(t *testing.T) {
	t.Parallel()
	ix := BuildIndex(testOntology())
	wantCraft := uuid.MustParse("018f0000-0000-7000-8000-00000000a001")

	tests := []struct {
		name     string
		text     string
		language string
		want     string
	}{
		{name: "latin canonical", text: "ajrakh", language: "ENGLISH", want: "ajrakh"},
		{name: "latin variant spelling", text: "Ajarakh", language: "ENGLISH", want: "Ajarakh"},
		{name: "devanagari", text: "अजरख", language: "HINDI", want: "अजरख"},
		{name: "gujarati", text: "અજરખ", language: "GUJARATI", want: "અજરખ"},
		{name: "latin uppercase with punctuation", text: "AJRAKH!", language: "", want: "AJRAKH"},
		{name: "inside a sentence", text: "I want an Ajarakh stole", language: "ENGLISH", want: "Ajarakh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			matches, err := ix.Resolve(context.Background(), tt.text, tt.language, nil)
			require.NoError(t, err)
			require.Len(t, matches, 1)

			m := matches[0]
			require.Equal(t, wantCraft, m.CraftID)
			require.Equal(t, "ajrakh-block-printing", m.Code)
			require.Equal(t, domain.MatchExact, m.Source)
			require.Equal(t, tt.want, m.MatchedText)
			require.Equal(t, tt.want, tt.text[m.Start:m.End])
		})
	}
}

func TestResolveSpansAndLongestMatch(t *testing.T) {
	t.Parallel()
	ix := BuildIndex(testOntology())

	matches, err := ix.Resolve(context.Background(), "अजरख dupatta and Jaipur Blue Pottery bowls", "HINDI", nil)
	require.NoError(t, err)
	require.Len(t, matches, 2)

	require.Equal(t, "ajrakh-block-printing", matches[0].Code)
	require.Equal(t, "अजरख", matches[0].MatchedText)
	require.Equal(t, 0, matches[0].Start)

	// The three-word alias wins over the two-word craft name inside it, and
	// consumes all three tokens so "Blue Pottery" is not reported a second time.
	require.Equal(t, "blue-pottery", matches[1].Code)
	require.Equal(t, "Jaipur Blue Pottery", matches[1].MatchedText)
	require.Equal(t, domain.MatchExact, matches[1].Source)
}

func TestResolveNoMatch(t *testing.T) {
	t.Parallel()
	ix := BuildIndex(testOntology())

	matches, err := ix.Resolve(context.Background(), "a plain cotton towel", "ENGLISH", nil)
	require.NoError(t, err)
	require.Empty(t, matches)
}

// fakeTransliterator stands in for Bhashini: it maps one romanisation onto the
// Devanagari spelling the ontology actually stores.
type fakeTransliterator struct{ from, to string }

func (f fakeTransliterator) Transliterate(_ context.Context, text, _ string) ([]string, error) {
	if Normalise(text) == Normalise(f.from) {
		return []string{f.to}, nil
	}
	return nil, nil
}

func TestResolveFallsBackToTransliteration(t *testing.T) {
	t.Parallel()
	ix := BuildIndex(testOntology())

	// "ajrak" is in no alias list; the hook offers the Devanagari spelling.
	matches, err := ix.Resolve(context.Background(), "ajrak saree", "HINDI",
		fakeTransliterator{from: "ajrak", to: "अजरख"})
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Equal(t, "ajrakh-block-printing", matches[0].Code)
	require.Equal(t, domain.MatchTransliterated, matches[0].Source)
	require.Equal(t, "ajrak", matches[0].MatchedText)
	require.InDelta(t, transliteratedScore, matches[0].Score, 0.0001)
}

func TestResolveMatchesCodeAndDisplayName(t *testing.T) {
	t.Parallel()
	ix := BuildIndex(testOntology())

	for _, text := range []string{"assam-muga-weaving", "Assam Muga Weaving", "মুগা"} {
		matches, err := ix.Resolve(context.Background(), text, "", nil)
		require.NoError(t, err, text)
		require.Len(t, matches, 1, text)
		require.Equal(t, "assam-muga-weaving", matches[0].Code, text)
	}
}

func TestIndexVersionIsStableAndContentAddressed(t *testing.T) {
	t.Parallel()
	crafts, aliases := testOntology()

	first := BuildIndex(crafts, aliases)
	second := BuildIndex(crafts, aliases)
	require.Equal(t, first.Version(), second.Version())

	changed := BuildIndex(crafts, append(aliases, domain.CraftAlias{
		CraftID: crafts[2].ID, Alias: "Neela Bartan", Script: "Latn", Language: "HINDI",
	}))
	require.NotEqual(t, first.Version(), changed.Version())

	craftCount, aliasCount := first.Stats()
	require.Equal(t, 3, craftCount)
	require.Positive(t, aliasCount)
}

func TestNormalise(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, in, want string
	}{
		{name: "lowercases", in: "AJRAKH", want: "ajrakh"},
		{name: "collapses whitespace", in: "  blue   pottery\t\n", want: "blue pottery"},
		{name: "punctuation becomes a separator", in: "ajrakh-block-printing", want: "ajrakh block printing"},
		{name: "strips latin diacritics", in: "Kaśīdākārī", want: "kasidakari"},
		{name: "keeps devanagari matras", in: "बांधणी", want: "बांधणी"},
		{name: "nfkc folds compatibility forms", in: "ｂｌｕｅ  ｐｏｔｔｅｒｙ", want: "blue pottery"},
		{name: "empty", in: "   ---  ", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, Normalise(tt.in))
		})
	}
}
