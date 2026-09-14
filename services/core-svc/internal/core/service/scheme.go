// services/core-svc/internal/core/service/scheme.go

package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// SchemeStore is the persistence surface the scheme service depends on.
type SchemeStore interface {
	ListActiveSchemes(ctx context.Context) ([]domain.Scheme, error)
	GetAllActiveSchemeCriteria(ctx context.Context) (map[uuid.UUID][]domain.SchemeCriterion, error)
	GetAllActiveSchemeManualChecks(ctx context.Context) (map[uuid.UUID][]domain.SchemeManualCheck, error)
	GetArtisanMatchFacts(ctx context.Context, artisanID uuid.UUID) (domain.ArtisanMatchFacts, error)
	GetSchemeByID(ctx context.Context, id uuid.UUID) (domain.Scheme, error)
	UpsertScheme(ctx context.Context, s domain.Scheme, criteria []domain.SchemeCriterion, manualChecks []domain.SchemeManualCheck) (domain.Scheme, error)
	DeleteScheme(ctx context.Context, id uuid.UUID) (bool, error)
}

// Schemes is the service for government scheme reference data and matching.
type Schemes struct {
	store SchemeStore
	log   *slog.Logger
}

// NewSchemes builds the scheme service.
func NewSchemes(store SchemeStore, log *slog.Logger) *Schemes {
	if log == nil {
		log = slog.Default()
	}
	return &Schemes{store: store, log: log}
}

// ListSchemes is public reference data -- no principal required.
func (s *Schemes) ListSchemes(ctx context.Context) ([]domain.Scheme, error) {
	return s.store.ListActiveSchemes(ctx)
}

// MatchSchemes matches the CALLER against every active scheme. There is no
// artisan-id parameter and therefore no authorization check to get wrong --
// this method can only ever be asked about the caller's own record.
func (s *Schemes) MatchSchemes(ctx context.Context) ([]domain.SchemeMatch, error) {
	principal, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	artisanID, err := uuid.Parse(principal.Subject)
	if err != nil {
		return nil, err
	}

	schemes, err := s.store.ListActiveSchemes(ctx)
	if err != nil {
		return nil, err
	}
	criteriaByScheme, err := s.store.GetAllActiveSchemeCriteria(ctx)
	if err != nil {
		return nil, err
	}
	manualChecksByScheme, err := s.store.GetAllActiveSchemeManualChecks(ctx)
	if err != nil {
		return nil, err
	}
	facts, err := s.store.GetArtisanMatchFacts(ctx, artisanID)
	if err != nil {
		return nil, err
	}

	matches := make([]domain.SchemeMatch, len(schemes))
	for i, scheme := range schemes {
		matches[i] = domain.MatchScheme(scheme, criteriaByScheme[scheme.ID], manualChecksByScheme[scheme.ID], facts)
	}
	return matches, nil
}

// UpsertScheme creates or updates a scheme. MINISTRY only.
func (s *Schemes) UpsertScheme(ctx context.Context, scheme domain.Scheme, criteria []domain.SchemeCriterion, manualChecks []domain.SchemeManualCheck) (domain.Scheme, error) {
	principal, err := auth.RequireRole(ctx, auth.RoleMinistry)
	if err != nil {
		return domain.Scheme{}, err
	}
	if scheme.ID == uuid.Nil {
		scheme.ID = uuid.New()
	}
	_ = principal
	return s.store.UpsertScheme(ctx, scheme, criteria, manualChecks)
}

// DeleteScheme removes a scheme. MINISTRY only.
func (s *Schemes) DeleteScheme(ctx context.Context, id uuid.UUID) (bool, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return false, err
	}
	return s.store.DeleteScheme(ctx, id)
}
