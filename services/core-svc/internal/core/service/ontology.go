// services/core-svc/internal/core/service/ontology.go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// maxResolveInput bounds the text the resolver will scan. A voice transcript or
// a search box query is short; anything longer is a caller mistake, and the
// sliding window is linear in the token count.
const maxResolveInput = 4096

// OntologyRegistry is the slice of the cached craft index the service uses. It
// is an interface so the service tests need neither Postgres nor Redis.
type OntologyRegistry interface {
	Resolve(ctx context.Context, text, language string) ([]domain.CraftMatch, error)
	Refresh(ctx context.Context) (domain.OntologyStats, error)
	Stats() domain.OntologyStats
	Craft(id uuid.UUID) (domain.Craft, bool)
	CraftByCode(code string) (domain.Craft, bool)
	Crafts() []domain.Craft
}

// Ontology is core-svc's craft-graph service: alias resolution for search and
// the cataloguing pipeline, plus the refresh that rebuilds the index in place.
type Ontology struct {
	registry OntologyRegistry
	log      *slog.Logger
}

// NewOntology builds the ontology service.
func NewOntology(registry OntologyRegistry, log *slog.Logger) *Ontology {
	return &Ontology{registry: registry, log: log}
}

// ResolveAlias links craft mentions in free text, returning each match with the
// span it occupies in the input. This is the entity-linking primitive search-svc
// calls before embedding a query, so it is readable by any authenticated caller.
func (s *Ontology) ResolveAlias(ctx context.Context, text, language string) ([]domain.CraftMatch, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text is required: %w", pkgdomain.ErrInvalidInput)
	}
	if len(text) > maxResolveInput {
		return nil, fmt.Errorf("text is %d bytes, at most %d may be resolved at once: %w",
			len(text), maxResolveInput, pkgdomain.ErrInvalidInput)
	}
	if _, err := auth.RequirePrincipal(ctx); err != nil {
		return nil, err
	}
	return s.registry.Resolve(ctx, text, language)
}

// RefreshIndex rebuilds the in-memory index from Postgres, republishes it to the
// Redis cache and tells every other replica to reload. It is how a curator's
// overnight alias import reaches a running fleet without a restart.
func (s *Ontology) RefreshIndex(ctx context.Context) (domain.OntologyStats, error) {
	principal, err := auth.RequireRole(ctx, auth.RoleClusterOfficer, auth.RoleMinistry)
	if err != nil {
		return domain.OntologyStats{}, err
	}

	stats, err := s.registry.Refresh(ctx)
	if err != nil {
		return domain.OntologyStats{}, err
	}
	s.log.InfoContext(ctx, "ontology index refreshed",
		"by", principal.Subject, "version", stats.Version, "crafts", stats.Crafts, "aliases", stats.Aliases)
	return stats, nil
}

// Stats reports what the currently loaded index holds.
func (s *Ontology) Stats(ctx context.Context) (domain.OntologyStats, error) {
	if _, err := auth.RequirePrincipal(ctx); err != nil {
		return domain.OntologyStats{}, err
	}
	return s.registry.Stats(), nil
}

// GetCraft reads one craft from the index.
func (s *Ontology) GetCraft(ctx context.Context, id uuid.UUID) (domain.Craft, error) {
	if id == uuid.Nil {
		return domain.Craft{}, fmt.Errorf("craft_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	craft, ok := s.registry.Craft(id)
	if !ok {
		return domain.Craft{}, fmt.Errorf("craft %s not found: %w", id, pkgdomain.ErrNotFound)
	}
	return craft, nil
}

// GetCraftByCode reads one craft by its slug.
func (s *Ontology) GetCraftByCode(ctx context.Context, code string) (domain.Craft, error) {
	if code == "" {
		return domain.Craft{}, fmt.Errorf("code is required: %w", pkgdomain.ErrInvalidInput)
	}
	craft, ok := s.registry.CraftByCode(code)
	if !ok {
		return domain.Craft{}, fmt.Errorf("craft %q not found: %w", code, pkgdomain.ErrNotFound)
	}
	return craft, nil
}

// ListCrafts returns the whole ontology, ordered by code. It is a few hundred
// rows and is served from memory, so it is not paged.
func (s *Ontology) ListCrafts(ctx context.Context) ([]domain.Craft, error) {
	return s.registry.Crafts(), nil
}
