// services/search-svc/internal/search/domain/types.go
package domain

import (
	"strings"

	"github.com/google/uuid"
)

// Sibling is a craft adjacent to one the buyer asked for, used to widen a search
// that found nothing rather than showing an empty page.
type Sibling struct {
	CraftID     uuid.UUID
	Code        string
	DisplayName string
	// Kind is the craft_relation edge that made these two crafts neighbours.
	Kind string
}

// SuggestionKind says what a type-ahead completion points at. Values match
// search.v1.SuggestionKind minus the prefix.
type SuggestionKind string

const (
	// SuggestCraft is a craft ontology node.
	SuggestCraft SuggestionKind = "CRAFT"
	// SuggestQuery is a previously successful free-text query.
	SuggestQuery SuggestionKind = "QUERY"
)

// Suggestion is one type-ahead completion.
type Suggestion struct {
	Text     string
	Kind     SuggestionKind
	EntityID *uuid.UUID
	Score    float64
}

// RerankCandidate is one hit offered to the cross-encoder.
type RerankCandidate struct {
	ListingID uuid.UUID
	Text      string
}

// IndexDocument is one listing projected into one language, ready to be written
// to listing_search. The document text is what both retrieval legs work over:
// the lexical leg tokenises it and the dense leg embeds it.
type IndexDocument struct {
	ListingID        uuid.UUID
	Language         string
	ArtisanID        uuid.UUID
	CraftID          uuid.UUID
	ClusterID        *uuid.UUID
	ListingType      string
	PricePaise       int64
	LeadTimeDays     *int32
	GICertified      bool
	ProvenanceSealed bool
	StateCode        string
	District         *string
	Colours          []string
	Materials        []string
	Document         string
	Embedding        []float32
	ModelVersion     *string
}

// IndexSource is one listing's projection in one language, as the database hands
// it over. The searchable text is assembled from it rather than stored twice.
type IndexSource struct {
	ListingID        uuid.UUID
	Language         string
	ArtisanID        uuid.UUID
	CraftID          uuid.UUID
	ClusterID        *uuid.UUID
	ListingType      string
	PricePaise       int64
	LeadTimeDays     *int32
	GICertified      bool
	ProvenanceSealed bool
	StateCode        string
	District         *string
	Colours          []string
	Materials        []string

	Title       string
	Description string
	Highlights  []string
	CraftName   string
	// CraftAliases is every spelling the ontology knows for this craft, in every
	// script. Putting them all in the document is what lets an English query
	// reach a Hindi listing without translating either: the two share a row.
	CraftAliases []string
	Techniques   []string
	Motifs       []string
	Region       []string
}

// Document assembles the searchable text and returns the row to write.
func (s IndexSource) Document() IndexDocument {
	parts := make([]string, 0, 16)
	parts = append(parts, s.Title, s.Description)
	parts = append(parts, s.Highlights...)
	parts = append(parts, s.CraftName)
	parts = append(parts, s.CraftAliases...)
	parts = append(parts, s.Materials...)
	parts = append(parts, s.Techniques...)
	parts = append(parts, s.Colours...)
	parts = append(parts, s.Motifs...)
	parts = append(parts, s.Region...)

	seen := make(map[string]struct{}, len(parts))
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, dup := seen[strings.ToLower(part)]; dup {
			continue
		}
		seen[strings.ToLower(part)] = struct{}{}
		kept = append(kept, part)
	}

	return IndexDocument{
		ListingID: s.ListingID, Language: s.Language, ArtisanID: s.ArtisanID,
		CraftID: s.CraftID, ClusterID: s.ClusterID, ListingType: s.ListingType,
		PricePaise: s.PricePaise, LeadTimeDays: s.LeadTimeDays,
		GICertified: s.GICertified, ProvenanceSealed: s.ProvenanceSealed,
		StateCode: s.StateCode, District: s.District,
		Colours: s.Colours, Materials: s.Materials,
		Document: strings.Join(kept, " "),
	}
}
