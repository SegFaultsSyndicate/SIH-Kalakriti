// services/core-svc/internal/core/repo/ontology.go
package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/ids"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/ontology"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// The repo is both halves of the ontology's persistence: the registry's read
// side and the seeder's write side.
var (
	_ ontology.Store  = (*Repo)(nil)
	_ ontology.Writer = (*Repo)(nil)
)

// ListCrafts reads the whole craft table, ordered by code.
func (r *Repo) ListCrafts(ctx context.Context) ([]domain.Craft, error) {
	rows, err := r.q.ListCrafts(ctx)
	if err != nil {
		return nil, translate(err, "crafts")
	}
	out := make([]domain.Craft, 0, len(rows))
	for _, row := range rows {
		out = append(out, craftFromRow(row))
	}
	return out, nil
}

// ListAllCraftAliases reads every alias in the ontology, for an index rebuild.
func (r *Repo) ListAllCraftAliases(ctx context.Context) ([]domain.CraftAlias, error) {
	rows, err := r.q.ListAllCraftAliases(ctx)
	if err != nil {
		return nil, translate(err, "craft aliases")
	}
	out := make([]domain.CraftAlias, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CraftAlias{
			ID:       row.ID,
			CraftID:  row.CraftID,
			Alias:    row.Alias,
			Script:   row.Script,
			Language: string(row.Language),
			Source:   row.Source,
		})
	}
	return out, nil
}

// GetCraft reads one craft by id.
func (r *Repo) GetCraft(ctx context.Context, id uuid.UUID) (domain.Craft, error) {
	row, err := r.q.GetCraft(ctx, id)
	if err != nil {
		return domain.Craft{}, translate(err, "craft")
	}
	return craftFromRow(row), nil
}

// UpsertCraft inserts or updates one craft by code and returns its id. A craft
// that already exists keeps the id it was first given, which is what lets the
// seed run against a live database without breaking every foreign key.
func (r *Repo) UpsertCraft(ctx context.Context, c domain.Craft) (uuid.UUID, error) {
	id := c.ID
	if id == uuid.Nil {
		id = ids.New()
	}
	row, err := r.q.UpsertCraft(ctx, db.UpsertCraftParams{
		ID:               id,
		Code:             c.Code,
		DisplayName:      c.DisplayName,
		ParentCraftID:    c.ParentCraftID,
		GiRegistrationNo: c.GIRegistrationNo,
		Techniques:       orEmpty(c.Techniques),
		Materials:        orEmpty(c.Materials),
	})
	if err != nil {
		return uuid.Nil, translate(err, "craft")
	}
	return row.ID, nil
}

// UpsertCraftAlias inserts or updates one alias by (lower(alias), script).
func (r *Repo) UpsertCraftAlias(ctx context.Context, a domain.CraftAlias) error {
	id := a.ID
	if id == uuid.Nil {
		id = ids.New()
	}
	_, err := r.q.UpsertCraftAlias(ctx, db.UpsertCraftAliasParams{
		ID:       id,
		CraftID:  a.CraftID,
		Alias:    a.Alias,
		Script:   a.Script,
		Language: db.LanguageCode(a.Language),
		Source:   a.Source,
	})
	return translate(err, "craft alias")
}

func craftFromRow(row db.Craft) domain.Craft {
	return domain.Craft{
		ID:               row.ID,
		Code:             row.Code,
		DisplayName:      row.DisplayName,
		ParentCraftID:    row.ParentCraftID,
		GIRegistrationNo: row.GiRegistrationNo,
		Techniques:       row.Techniques,
		Materials:        row.Materials,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}
