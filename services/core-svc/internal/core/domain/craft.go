// services/core-svc/internal/core/domain/craft.go
package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// Craft is one node of the craft ontology.
type Craft struct {
	ID               uuid.UUID
	Code             string
	DisplayName      string
	ParentCraftID    *uuid.UUID
	GIRegistrationNo *string
	Techniques       []string
	Materials        []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CraftAlias is one vernacular, transliterated or misspelt name for a craft.
type CraftAlias struct {
	ID       uuid.UUID
	CraftID  uuid.UUID
	Alias    string
	Script   string
	Language string
	Source   string
}

// Validate checks an alias row before it is written or indexed.
func (a CraftAlias) Validate() error {
	if strings.TrimSpace(a.Alias) == "" {
		return fmt.Errorf("alias must not be blank: %w", pkgdomain.ErrInvalidInput)
	}
	if len(a.Script) != 4 {
		return fmt.Errorf("alias %q has script %q, which is not an ISO 15924 code: %w",
			a.Alias, a.Script, pkgdomain.ErrInvalidInput)
	}
	if a.Language == "" {
		return fmt.Errorf("alias %q has no language: %w", a.Alias, pkgdomain.ErrInvalidInput)
	}
	return nil
}

// OntologyStats describes an ontology index: what it contains and which build
// of the ontology it came from. The version is a content hash, so two replicas
// reporting the same version hold the same graph.
type OntologyStats struct {
	Version string
	Crafts  int
	Aliases int
	BuiltAt time.Time
}

// MatchSource says how a craft was arrived at from the input text.
type MatchSource string

const (
	// MatchExact means the normalised input matched a stored alias verbatim.
	MatchExact MatchSource = "EXACT"
	// MatchTransliterated means a transliteration of the input matched a stored alias.
	MatchTransliterated MatchSource = "TRANSLITERATED"
)

// CraftMatch is one craft found inside a fragment of text, with the span of the
// input it was found at. Start and End are byte offsets into the original
// string, so a caller can highlight or strip the mention without re-scanning.
type CraftMatch struct {
	CraftID     uuid.UUID
	Code        string
	DisplayName string
	// Alias is the stored spelling that matched.
	Alias string
	// MatchedText is the exact substring of the input, before normalisation.
	MatchedText string
	Start       int
	End         int
	Script      string
	Language    string
	Score       float32
	Source      MatchSource
}
