// services/core-svc/internal/core/service/scheme_test.go

package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

type fakeSchemeStore struct {
	schemes      []domain.Scheme
	criteria     map[uuid.UUID][]domain.SchemeCriterion
	manualChecks map[uuid.UUID][]domain.SchemeManualCheck
	facts        domain.ArtisanMatchFacts
}

func (f *fakeSchemeStore) ListActiveSchemes(ctx context.Context) ([]domain.Scheme, error) {
	return f.schemes, nil
}

func (f *fakeSchemeStore) GetAllActiveSchemeCriteria(ctx context.Context) (map[uuid.UUID][]domain.SchemeCriterion, error) {
	if f.criteria == nil {
		return make(map[uuid.UUID][]domain.SchemeCriterion), nil
	}
	return f.criteria, nil
}

func (f *fakeSchemeStore) GetAllActiveSchemeManualChecks(ctx context.Context) (map[uuid.UUID][]domain.SchemeManualCheck, error) {
	if f.manualChecks == nil {
		return make(map[uuid.UUID][]domain.SchemeManualCheck), nil
	}
	return f.manualChecks, nil
}

func (f *fakeSchemeStore) GetArtisanMatchFacts(ctx context.Context, artisanID uuid.UUID) (domain.ArtisanMatchFacts, error) {
	return f.facts, nil
}

func (f *fakeSchemeStore) GetSchemeByID(ctx context.Context, id uuid.UUID) (domain.Scheme, error) {
	return domain.Scheme{}, nil
}

func (f *fakeSchemeStore) UpsertScheme(ctx context.Context, s domain.Scheme, c []domain.SchemeCriterion, m []domain.SchemeManualCheck) (domain.Scheme, error) {
	return s, nil
}

func (f *fakeSchemeStore) DeleteScheme(ctx context.Context, id uuid.UUID) (bool, error) {
	return true, nil
}

func TestSchemes_MatchSchemes_ReturnsOneMatchPerActiveScheme(t *testing.T) {
	schemeID := uuid.New()
	store := &fakeSchemeStore{
		schemes: []domain.Scheme{{ID: schemeID, Code: "test_scheme"}},
	}
	svc := NewSchemes(store, nil)
	artisanID := uuid.New()
	ctx := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: artisanID.String(), Role: auth.RoleArtisan})

	matches, err := svc.MatchSchemes(ctx)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Equal(t, domain.MayQualify, matches[0].Status) // no criteria, no manual checks -> vacuously MayQualify
}
