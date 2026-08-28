// services/core-svc/internal/core/repo/artisan.go
package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/domain"
	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/repo/db"
)

// CreateArtisan inserts an artisan row and its craft edges. It is a Tx method
// because registration must commit together with its outbox event.
func (t *Tx) CreateArtisan(ctx context.Context, id uuid.UUID, in domain.RegisterArtisanInput) (domain.Artisan, error) {
	row, err := t.q.CreateArtisan(ctx, db.CreateArtisanParams{
		ID:                id,
		UserID:            nil,
		DisplayName:       in.DisplayName,
		PhoneE164:         in.PhoneE164,
		PehchanID:         in.PehchanID,
		PmVishwakarmaID:   in.PMVishwakarmaID,
		PrimaryClusterID:  in.ClusterID,
		StateCode:         in.Region.StateCode,
		District:          in.Region.District,
		Block:             in.Region.Block,
		Village:           in.Region.Village,
		Pincode:           in.Region.Pincode,
		Languages:         toLanguageCodes(in.Languages),
		YearsOfExperience: in.YearsOfExperience,
		Bio:               in.Bio,
		CreatedBy:         in.CreatedBy,
	})
	if err != nil {
		return domain.Artisan{}, translate(err, "artisan")
	}

	for i, craftID := range in.CraftIDs {
		if err := t.q.AddArtisanCraft(ctx, db.AddArtisanCraftParams{
			ArtisanID: id,
			CraftID:   craftID,
			// The first craft listed is the artisan's primary craft.
			IsPrimary: i == 0,
		}); err != nil {
			return domain.Artisan{}, translate(err, fmt.Sprintf("artisan craft %s", craftID))
		}
	}

	out := artisanFromRow(row)
	out.CraftIDs = in.CraftIDs
	return out, nil
}

// GetArtisan reads one artisan by id, with their craft ids.
func (r *Repo) GetArtisan(ctx context.Context, id uuid.UUID) (domain.Artisan, error) {
	row, err := r.q.GetArtisan(ctx, id)
	if err != nil {
		return domain.Artisan{}, translate(err, "artisan")
	}
	return r.withCrafts(ctx, artisanFromRow(row))
}

// GetArtisanByPhone reads one artisan by E.164 phone number.
func (r *Repo) GetArtisanByPhone(ctx context.Context, phone string) (domain.Artisan, error) {
	row, err := r.q.GetArtisanByPhone(ctx, phone)
	if err != nil {
		return domain.Artisan{}, translate(err, "artisan")
	}
	return r.withCrafts(ctx, artisanFromRow(row))
}

// ArtisanExistsByPhone reports whether a profile exists for a phone number,
// without the cost of loading crafts. Used by the login flow to decide whether
// the caller still needs to register.
func (r *Repo) ArtisanExistsByPhone(ctx context.Context, phone string) (uuid.UUID, bool, error) {
	row, err := r.q.GetArtisanByPhone(ctx, phone)
	if err != nil {
		if isNotFound(translate(err, "artisan")) {
			return uuid.Nil, false, nil
		}
		return uuid.Nil, false, translate(err, "artisan")
	}
	return row.ID, true, nil
}

// UpdateArtisan patches an artisan profile. Only non-nil fields are written;
// COALESCE in the query leaves the rest alone.
func (r *Repo) UpdateArtisan(ctx context.Context, in domain.UpdateArtisanInput) (domain.Artisan, error) {
	params := db.UpdateArtisanParams{
		ID:                in.ArtisanID,
		DisplayName:       in.DisplayName,
		PrimaryClusterID:  in.ClusterID,
		YearsOfExperience: in.YearsOfExperience,
		Bio:               in.Bio,
		PhotoMediaID:      in.PhotoMediaID,
	}
	if in.Region != nil {
		params.StateCode = &in.Region.StateCode
		params.District = in.Region.District
		params.Block = in.Region.Block
		params.Village = in.Region.Village
		params.Pincode = in.Region.Pincode
	}
	if len(in.Languages) > 0 {
		params.Languages = toLanguageCodes(in.Languages)
	}

	row, err := r.q.UpdateArtisan(ctx, params)
	if err != nil {
		return domain.Artisan{}, translate(err, "artisan")
	}
	return r.withCrafts(ctx, artisanFromRow(row))
}

// ListArtisansByCluster pages a cluster's artisans by keyset on the UUIDv7 id.
// Craft ids are deliberately not loaded here: a roster page does not need them,
// and loading them would make this an N+1.
func (r *Repo) ListArtisansByCluster(ctx context.Context, clusterID uuid.UUID, page domain.Page) ([]domain.Artisan, error) {
	page = page.Normalise()
	rows, err := r.q.ListArtisansByCluster(ctx, db.ListArtisansByClusterParams{
		ClusterID: &clusterID,
		After:     page.Cursor,
		PageSize:  page.Size,
	})
	if err != nil {
		return nil, translate(err, "artisans by cluster")
	}
	out := make([]domain.Artisan, 0, len(rows))
	for _, row := range rows {
		out = append(out, artisanFromRow(row))
	}
	return out, nil
}

// withCrafts loads an artisan's craft ids onto an already-read artisan.
func (r *Repo) withCrafts(ctx context.Context, a domain.Artisan) (domain.Artisan, error) {
	crafts, err := r.q.ListArtisanCrafts(ctx, a.ID)
	if err != nil {
		return domain.Artisan{}, translate(err, "artisan crafts")
	}
	a.CraftIDs = make([]uuid.UUID, 0, len(crafts))
	for _, c := range crafts {
		a.CraftIDs = append(a.CraftIDs, c.ID)
	}
	return a, nil
}

// artisanFromRow converts a generated row into the domain type.
func artisanFromRow(row db.Artisan) domain.Artisan {
	return domain.Artisan{
		ID:              row.ID,
		UserID:          row.UserID,
		DisplayName:     row.DisplayName,
		PhoneE164:       row.PhoneE164,
		PehchanID:       row.PehchanID,
		PMVishwakarmaID: row.PmVishwakarmaID,

		PrimaryClusterID: row.PrimaryClusterID,
		Region: domain.Region{
			StateCode: row.StateCode,
			District:  row.District,
			Block:     row.Block,
			Village:   row.Village,
			Pincode:   row.Pincode,
		},
		Languages:         fromLanguageCodes(row.Languages),
		YearsOfExperience: row.YearsOfExperience,
		Bio:               row.Bio,
		PhotoMediaID:      row.PhotoMediaID,
		Verified:          row.Verified,
		CreatedBy:         row.CreatedBy,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

// toLanguageCodes converts domain language names into the generated enum type.
func toLanguageCodes(langs []string) []db.LanguageCode {
	out := make([]db.LanguageCode, 0, len(langs))
	for _, l := range langs {
		out = append(out, db.LanguageCode(l))
	}
	return out
}

// fromLanguageCodes converts the generated enum type back into plain strings.
func fromLanguageCodes(codes []db.LanguageCode) []string {
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		out = append(out, string(c))
	}
	return out
}
