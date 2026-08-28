// services/core-svc/internal/core/repo/media.go
package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// CreateMediaPending inserts the row that a presigned upload URL points at. The
// bytes do not exist yet, which is why sha256_hex may still be NULL here.
func (t *Tx) CreateMediaPending(ctx context.Context, m domain.Media) (domain.Media, error) {
	row, err := t.q.CreateMediaPending(ctx, db.CreateMediaPendingParams{
		ID:         m.ID,
		ArtisanID:  m.ArtisanID,
		ProductID:  m.ProductID,
		Bucket:     m.Bucket,
		ObjectKey:  m.ObjectKey,
		Kind:       db.MediaKind(m.Kind),
		MimeType:   m.MimeType,
		SizeBytes:  m.SizeBytes,
		Sha256Hex:  m.SHA256Hex,
		WidthPx:    m.WidthPx,
		HeightPx:   m.HeightPx,
		DurationMs: m.DurationMs,
		Source:     m.Source,
	})
	if err != nil {
		return domain.Media{}, translate(err, "media")
	}
	return mediaFromRow(row), nil
}

// TransitionMediaState performs the guarded state change. A zero-row update
// means another caller moved the row first, which is a conflict: it is what
// makes a duplicated ConfirmUpload a no-op rather than a second event.
func (t *Tx) TransitionMediaState(
	ctx context.Context,
	id uuid.UUID,
	from, to domain.MediaState,
	set domain.MediaTransition,
) (domain.Media, error) {
	row, err := t.q.TransitionMediaState(ctx, db.TransitionMediaStateParams{
		ID:            id,
		NextState:     db.MediaState(to),
		ExpectedState: db.MediaState(from),
		SizeBytes:     set.SizeBytes,
		Sha256Hex:     set.SHA256Hex,
		FailureReason: set.FailureReason,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Media{}, fmt.Errorf(
				"media %s is no longer in %s, so it cannot move to %s: %w",
				id, from, to, pkgdomain.ErrConflict)
		}
		return domain.Media{}, translate(err, "media state")
	}
	return mediaFromRow(row), nil
}

// SetMediaEnhanced records the enhanced rendition and moves the row to READY.
func (t *Tx) SetMediaEnhanced(
	ctx context.Context,
	id uuid.UUID,
	from domain.MediaState,
	enhancedKey string,
	modelVersion *string,
) (domain.Media, error) {
	row, err := t.q.SetMediaEnhanced(ctx, db.SetMediaEnhancedParams{
		ID:                id,
		ExpectedState:     db.MediaState(from),
		EnhancedObjectKey: &enhancedKey,
		ModelVersion:      modelVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Media{}, fmt.Errorf(
				"media %s is no longer in %s: %w", id, from, pkgdomain.ErrConflict)
		}
		return domain.Media{}, translate(err, "media enhancement")
	}
	return mediaFromRow(row), nil
}

// DeleteMedia removes one row, reporting whether it was still there.
func (t *Tx) DeleteMedia(ctx context.Context, id uuid.UUID) (bool, error) {
	n, err := t.q.DeleteMedia(ctx, id)
	if err != nil {
		return false, translate(err, "media")
	}
	return n > 0, nil
}

// GetMedia reads one media row.
func (r *Repo) GetMedia(ctx context.Context, id uuid.UUID) (domain.Media, error) {
	row, err := r.q.GetMedia(ctx, id)
	if err != nil {
		return domain.Media{}, translate(err, "media")
	}
	return mediaFromRow(row), nil
}

// GetMediaByHash finds an artisan's already-stored copy of the same bytes.
func (r *Repo) GetMediaByHash(ctx context.Context, artisanID uuid.UUID, sha256Hex string) (domain.Media, error) {
	row, err := r.q.GetMediaByHash(ctx, db.GetMediaByHashParams{
		ArtisanID: artisanID,
		Sha256Hex: &sha256Hex,
	})
	if err != nil {
		return domain.Media{}, translate(err, "media")
	}
	return mediaFromRow(row), nil
}

// ListStalePendingMedia reads the rows whose bytes never arrived.
func (r *Repo) ListStalePendingMedia(ctx context.Context, before time.Time, batchSize int32) ([]domain.Media, error) {
	rows, err := r.q.ListStalePendingMedia(ctx, db.ListStalePendingMediaParams{
		Before:    before,
		BatchSize: batchSize,
	})
	if err != nil {
		return nil, translate(err, "stale pending media")
	}
	out := make([]domain.Media, 0, len(rows))
	for _, row := range rows {
		out = append(out, mediaFromRow(row))
	}
	return out, nil
}

func mediaFromRow(row db.Media) domain.Media {
	return domain.Media{
		ID:                row.ID,
		ArtisanID:         row.ArtisanID,
		ProductID:         row.ProductID,
		Bucket:            row.Bucket,
		ObjectKey:         row.ObjectKey,
		EnhancedObjectKey: row.EnhancedObjectKey,
		Kind:              domain.MediaKind(row.Kind),
		MimeType:          row.MimeType,
		SizeBytes:         row.SizeBytes,
		SHA256Hex:         row.Sha256Hex,
		WidthPx:           row.WidthPx,
		HeightPx:          row.HeightPx,
		DurationMs:        row.DurationMs,
		Source:            row.Source,
		State:             domain.MediaState(row.State),
		ModelVersion:      row.ModelVersion,
		FailureReason:     row.FailureReason,
		UploadedAt:        row.UploadedAt,
		ConfirmedAt:       row.ConfirmedAt,
		CreatedAt:         row.CreatedAt,
	}
}
