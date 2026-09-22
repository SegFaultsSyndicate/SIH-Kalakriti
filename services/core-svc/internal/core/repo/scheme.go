// services/core-svc/internal/core/repo/scheme.go

package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

func (r *Repo) ListActiveSchemes(ctx context.Context) ([]domain.Scheme, error) {
	rows, err := r.q.ListActiveSchemes(ctx)
	if err != nil {
		return nil, translate(err, "schemes")
	}
	out := make([]domain.Scheme, len(rows))
	for i, row := range rows {
		out[i] = schemeFromRow(row)
	}
	return out, nil
}

func (r *Repo) GetSchemeByID(ctx context.Context, id uuid.UUID) (domain.Scheme, error) {
	row, err := r.q.GetSchemeByID(ctx, id)
	if err != nil {
		return domain.Scheme{}, translate(err, "scheme")
	}
	return schemeFromRow(row), nil
}

// GetAllActiveSchemeCriteria fetches every active scheme's criteria in one
// query and groups them by scheme id, so MatchSchemes (service layer) does
// not issue one query per scheme.
func (r *Repo) GetAllActiveSchemeCriteria(ctx context.Context) (map[uuid.UUID][]domain.SchemeCriterion, error) {
	rows, err := r.q.GetAllActiveSchemeCriteria(ctx)
	if err != nil {
		return nil, translate(err, "scheme criteria")
	}
	out := make(map[uuid.UUID][]domain.SchemeCriterion)
	for _, row := range rows {
		out[row.SchemeID] = append(out[row.SchemeID], domain.SchemeCriterion{
			Type: domain.SchemeCriterionType(row.Type), StringValues: row.StringValues,
			IntValue: row.IntValue, Negate: row.Negate,
		})
	}
	return out, nil
}

func (r *Repo) GetAllActiveSchemeManualChecks(ctx context.Context) (map[uuid.UUID][]domain.SchemeManualCheck, error) {
	rows, err := r.q.GetAllActiveSchemeManualChecks(ctx)
	if err != nil {
		return nil, translate(err, "scheme manual checks")
	}
	out := make(map[uuid.UUID][]domain.SchemeManualCheck)
	for _, row := range rows {
		out[row.SchemeID] = append(out[row.SchemeID], domain.SchemeManualCheck{
			I18nKey: row.I18nKey, CheckText: row.CheckText,
		})
	}
	return out, nil
}

// GetArtisanMatchFacts reads everything MatchSchemes needs about one
// artisan in a single round trip.
func (r *Repo) GetArtisanMatchFacts(ctx context.Context, artisanID uuid.UUID) (domain.ArtisanMatchFacts, error) {
	row, err := r.q.GetArtisanMatchFacts(ctx, artisanID)
	if err != nil {
		return domain.ArtisanMatchFacts{}, translate(err, "artisan match facts")
	}
	var socialCategory *string
	if row.SocialCategory != nil {
		s := string(*row.SocialCategory)
		socialCategory = &s
	}
	return domain.ArtisanMatchFacts{
		StateCode: row.StateCode, YearsOfExperience: row.YearsOfExperience, SocialCategory: socialCategory,
		HasPehchanID: row.HasPehchanID, HasPMVishwakarmaID: row.HasPmVishwakarmaID,
		IsClusterMember: row.IsClusterMember, IsSHGMember: row.IsShgMember, CraftIDs: row.CraftIds,
	}, nil
}

// UpsertScheme replaces the scheme row and wholesale-replaces its criteria
// and manual checks (delete-then-insert in one transaction) since an admin
// edit form always submits the complete set, not an incremental diff --
// simpler and safer than reconciling row-by-row.
func (r *Repo) UpsertScheme(ctx context.Context, s domain.Scheme, criteria []domain.SchemeCriterion, manualChecks []domain.SchemeManualCheck) (domain.Scheme, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Scheme{}, translate(err, "scheme upsert")
	}
	defer tx.Rollback(ctx)
	qtx := r.q.WithTx(tx)

	row, err := qtx.UpsertScheme(ctx, db.UpsertSchemeParams{
		ID: s.ID, Code: s.Code, Authority: db.SchemeAuthority(s.Authority), Ministry: s.Ministry,
		OfficialUrl: s.OfficialURL, StateCode: s.StateCode, NameI18nKey: s.NameI18nKey,
		NameText: s.NameText, SummaryI18nKey: s.SummaryI18nKey, SummaryText: s.SummaryText,
		Active: true, SortOrder: s.SortOrder, CuratedBy: "admin",
	})
	if err != nil {
		return domain.Scheme{}, translate(err, "scheme upsert")
	}

	if err := qtx.DeleteSchemeCriteria(ctx, row.ID); err != nil {
		return domain.Scheme{}, translate(err, "scheme criteria replace")
	}
	for _, c := range criteria {
		if err := qtx.InsertSchemeCriterion(ctx, db.InsertSchemeCriterionParams{
			ID: ids.New(), SchemeID: row.ID, Type: db.SchemeCriterionType(c.Type),
			StringValues: c.StringValues, IntValue: c.IntValue, Negate: c.Negate,
		}); err != nil {
			return domain.Scheme{}, translate(err, "scheme criterion insert")
		}
	}

	if err := qtx.DeleteSchemeManualChecks(ctx, row.ID); err != nil {
		return domain.Scheme{}, translate(err, "scheme manual checks replace")
	}
	for _, m := range manualChecks {
		if err := qtx.InsertSchemeManualCheck(ctx, db.InsertSchemeManualCheckParams{
			ID: ids.New(), SchemeID: row.ID, I18nKey: m.I18nKey, CheckText: m.CheckText,
		}); err != nil {
			return domain.Scheme{}, translate(err, "scheme manual check insert")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Scheme{}, translate(err, "scheme upsert commit")
	}
	return schemeFromRow(row), nil
}

func (r *Repo) DeleteScheme(ctx context.Context, id uuid.UUID) (bool, error) {
	n, err := r.q.DeleteScheme(ctx, id)
	if err != nil {
		return false, translate(err, "scheme delete")
	}
	return n > 0, nil
}

func schemeFromRow(row db.GovernmentScheme) domain.Scheme {
	return domain.Scheme{
		ID: row.ID, Code: row.Code, Authority: string(row.Authority), Ministry: row.Ministry,
		OfficialURL: row.OfficialUrl, StateCode: row.StateCode, NameI18nKey: row.NameI18nKey,
		NameText: row.NameText, SummaryI18nKey: row.SummaryI18nKey, SummaryText: row.SummaryText,
		SortOrder: row.SortOrder,
	}
}
