// services/core-svc/internal/core/domain/inference.go
package domain

import "github.com/google/uuid"

// InferredAttributes is what the vision stack believes about one product, in the
// domain's own vocabulary. The proto types stay in the ml-svc client.
type InferredAttributes struct {
	// CraftCode is the ontology slug the model chose; the pipeline resolves it
	// to an id only if the artisan actually practises that craft.
	CraftCode       string
	CraftConfidence float32
	Material        string
	Technique       string
	Colours         []string
	Motifs          []string
	// Confidence by attribute name, for the rows written to listing_attribute.
	Confidence   map[string]float32
	ModelVersion string
}

// ToListingAttributes flattens the set into the rows the catalog stores. Empty
// values are dropped rather than written as blanks: an attribute the model had
// nothing to say about should be absent, not present and empty.
func (a InferredAttributes) ToListingAttributes() []ListingAttribute {
	out := make([]ListingAttribute, 0, 8)
	add := func(name, value string) {
		if value == "" {
			return
		}
		out = append(out, ListingAttribute{
			Name:       name,
			Value:      value,
			Confidence: a.confidence(name),
		})
	}

	add("craft", a.CraftCode)
	add("material", a.Material)
	add("technique", a.Technique)
	for _, colour := range a.Colours {
		add("colour", colour)
	}
	for _, motif := range a.Motifs {
		add("motif", motif)
	}
	return out
}

// confidence is the model's confidence in one attribute name, clamped into the
// range the listing_attribute check constraint allows.
func (a InferredAttributes) confidence(name string) float32 {
	value, ok := a.Confidence[name]
	if !ok {
		value = 0.5
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

// CopyRequest asks ml-svc for buyer-facing copy in one language.
type CopyRequest struct {
	Attributes  InferredAttributes
	CraftID     uuid.UUID
	CraftCode   string
	Language    string
	ArtisanNote string
	MaxChars    int32
}

// GeneratedCopy is what came back.
type GeneratedCopy struct {
	Title       string
	Description string
	Highlights  []string
	Keywords    []string
	// AttributeKeysUsed is what the model says it drew on; a reviewer can trace
	// every claim in the copy back to one of these.
	AttributeKeysUsed []string
	ModelVersion      string
}
