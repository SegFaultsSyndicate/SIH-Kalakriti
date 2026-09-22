// services/core-svc/internal/core/repo/trends.go

package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// CreateTrendLink writes a trend link to storage.
func (r *Repo) CreateTrendLink(ctx context.Context, link domain.TrendLink) (domain.TrendLink, error) {
	var metadata []byte
	if link.EmbedMetadataJSON != nil {
		metadata = []byte(*link.EmbedMetadataJSON)
	}

	row, err := r.q.CreateTrendLink(ctx, db.CreateTrendLinkParams{
		ID:            link.ID,
		Title:         link.Title,
		Description:   link.Description,
		Url:           link.URL,
		SourceType:    db.TrendSourceType(link.SourceType),
		CraftID:       link.CraftID,
		ThumbnailUrl:  link.ThumbnailURL,
		EmbedHtml:     link.EmbedHTML,
		EmbedMetadata: metadata,
		Pinned:        link.Pinned,
		CreatedBy:     link.CreatedBy,
		CuratorRole:   link.CuratorRole,
		CuratorName:   link.CuratorName,
		ExpiresAt:     link.ExpiresAt,
	})
	if err != nil {
		return domain.TrendLink{}, translate(err, "trend link")
	}
	return trendFromRow(row, nil), nil
}

// GetTrendLink reads one trend link by id.
func (r *Repo) GetTrendLink(ctx context.Context, id uuid.UUID) (domain.TrendLink, error) {
	row, err := r.q.GetTrendLink(ctx, id)
	if err != nil {
		return domain.TrendLink{}, translate(err, "trend link")
	}
	return trendFromRow(row, nil), nil
}

// ListTrendLinks lists trend links, newest/pinned first.
func (r *Repo) ListTrendLinks(ctx context.Context, filter domain.TrendFilter) ([]domain.TrendLink, error) {
	var sType *db.TrendSourceType
	if filter.SourceType != nil {
		st := db.TrendSourceType(*filter.SourceType)
		sType = &st
	}

	rows, err := r.q.ListTrendLinks(ctx, db.ListTrendLinksParams{
		CraftID:        filter.CraftID,
		SourceType:     sType,
		ExcludeExpired: filter.ExcludeExpired,
		After:          filter.Page.Cursor,
		PageSize:       filter.Page.Size,
	})
	if err != nil {
		return nil, translate(err, "trend links")
	}

	out := make([]domain.TrendLink, len(rows))
	for i, row := range rows {
		var metadataJSON *string
		if len(row.EmbedMetadata) > 0 {
			s := string(row.EmbedMetadata)
			metadataJSON = &s
		}
		out[i] = domain.TrendLink{
			ID:                row.ID,
			Title:             row.Title,
			Description:       row.Description,
			URL:               row.Url,
			SourceType:        domain.TrendSourceType(row.SourceType),
			CraftID:           row.CraftID,
			CraftName:         row.CraftName,
			ThumbnailURL:      row.ThumbnailUrl,
			EmbedHTML:         row.EmbedHtml,
			EmbedMetadataJSON: metadataJSON,
			Pinned:            row.Pinned,
			CreatedBy:         row.CreatedBy,
			CuratorRole:       row.CuratorRole,
			CuratorName:       row.CuratorName,
			ExpiresAt:         row.ExpiresAt,
			CreatedAt:         row.CreatedAt,
		}
	}
	return out, nil
}

// DeleteTrendLink removes a trend link.
func (r *Repo) DeleteTrendLink(ctx context.Context, id uuid.UUID) (bool, error) {
	affected, err := r.q.DeleteTrendLink(ctx, id)
	if err != nil {
		return false, translate(err, "trend link")
	}
	return affected > 0, nil
}

// PinTrendLink sets the pinned state of a trend link.
func (r *Repo) PinTrendLink(ctx context.Context, id uuid.UUID, pinned bool) (domain.TrendLink, error) {
	row, err := r.q.PinTrendLink(ctx, db.PinTrendLinkParams{
		Pinned: pinned,
		ID:     id,
	})
	if err != nil {
		return domain.TrendLink{}, translate(err, "trend link")
	}
	return trendFromRow(row, nil), nil
}

func trendFromRow(row db.TrendLink, craftName *string) domain.TrendLink {
	var metadataJSON *string
	if len(row.EmbedMetadata) > 0 {
		s := string(row.EmbedMetadata)
		metadataJSON = &s
	}
	return domain.TrendLink{
		ID:                row.ID,
		Title:             row.Title,
		Description:       row.Description,
		URL:               row.Url,
		SourceType:        domain.TrendSourceType(row.SourceType),
		CraftID:           row.CraftID,
		CraftName:         craftName,
		ThumbnailURL:      row.ThumbnailUrl,
		EmbedHTML:         row.EmbedHtml,
		EmbedMetadataJSON: metadataJSON,
		Pinned:            row.Pinned,
		CreatedBy:         row.CreatedBy,
		CuratorRole:       row.CuratorRole,
		CuratorName:       row.CuratorName,
		ExpiresAt:         row.ExpiresAt,
		CreatedAt:         row.CreatedAt,
	}
}
