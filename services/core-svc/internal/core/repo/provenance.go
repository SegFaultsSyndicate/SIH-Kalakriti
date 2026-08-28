// services/core-svc/internal/core/repo/provenance.go
package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/segfaultsyndicate/kalakriti/pkg/domain"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/domain"
	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/repo/db"
)

const (
	// ErrProvenanceShortCode is the unique constraint violation on short_code.
	ErrProvenanceShortCode = "provenance_record_short_code_key"
)

// InsertProvenanceRecord inserts a new provenance record.
func (t *Tx) InsertProvenanceRecord(ctx context.Context, rec domain.ProvenanceRecord) (domain.ProvenanceRecord, error) {
	var prevHash *string
	if rec.PreviousHash != nil {
		prevHash = rec.PreviousHash
	}
	row, err := t.q.InsertProvenanceRecord(ctx, db.InsertProvenanceRecordParams{
		ID:               rec.ID,
		ListingID:        rec.ListingID,
		ArtisanID:        rec.ArtisanID,
		CraftID:          rec.CraftID,
		ContentHash:      rec.ContentHash,
		PreviousHash:     prevHash,
		Signature:        rec.Signature,
		SignatureAlgo:    rec.SignatureAlgo,
		PublicKeyID:      rec.PublicKeyID,
		ShortCode:        rec.ShortCode,
		TechniqueMatched: rec.TechniqueMatched,
		MediaHashes:      rec.MediaHashes,
		SealedAt:         rec.SealedAt,
	})
	if err != nil {
		return domain.ProvenanceRecord{}, translate(err, "provenance record")
	}
	return provenanceFromRow(row), nil
}

// ShortCodeExists checks if a short code is already taken.
func (t *Tx) ShortCodeExists(ctx context.Context, code string) (bool, error) {
	exists, err := t.q.ShortCodeExists(ctx, code)
	if err != nil {
		return false, fmt.Errorf("checking short code: %w", err)
	}
	return exists, nil
}

// GetLatestProvenanceForArtisan returns the artisan's most recent provenance record.
func (t *Tx) GetLatestProvenanceForArtisan(ctx context.Context, artisanID uuid.UUID) (*domain.ProvenanceRecord, error) {
	row, err := t.q.GetLatestProvenanceForArtisan(ctx, artisanID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("fetching latest provenance: %w", err)
	}
	rec := provenanceFromRow(row)
	return &rec, nil
}

// SetListingProvenance links a provenance record to its listing.
func (t *Tx) SetListingProvenance(ctx context.Context, listingID, provenanceID uuid.UUID) error {
	_, err := t.q.SetListingProvenance(ctx, db.SetListingProvenanceParams{
		ID:           listingID,
		ProvenanceID: provenanceID,
	})
	if err != nil {
		return fmt.Errorf("setting listing provenance: %w", err)
	}
	return nil
}

// GetProvenanceRecord fetches a provenance record by ID.
func (r *Repo) GetProvenanceRecord(ctx context.Context, id uuid.UUID) (domain.ProvenanceRecord, error) {
	row, err := r.q.GetProvenanceRecord(ctx, id)
	if err != nil {
		return domain.ProvenanceRecord{}, translate(err, "provenance record")
	}
	return provenanceFromRow(row), nil
}

// GetProvenanceByShortCode fetches a provenance record by its public short code.
func (r *Repo) GetProvenanceByShortCode(ctx context.Context, code string) (domain.ProvenanceRecord, error) {
	row, err := r.q.GetProvenanceByShortCode(ctx, code)
	if err != nil {
		return domain.ProvenanceRecord{}, translate(err, "provenance by short code")
	}
	return provenanceFromRow(row), nil
}

// GetProvenanceByListing fetches the latest provenance record for a listing.
func (r *Repo) GetProvenanceByListing(ctx context.Context, listingID uuid.UUID) (domain.ProvenanceRecord, error) {
	row, err := r.q.GetProvenanceByListing(ctx, listingID)
	if err != nil {
		return domain.ProvenanceRecord{}, translate(err, "provenance by listing")
	}
	return provenanceFromRow(row), nil
}

// provenanceFromRow converts a sqlc row to domain.ProvenanceRecord.
func provenanceFromRow(row db.ProvenanceRecord) domain.ProvenanceRecord {
	var prevHash *string
	if row.PreviousHash.Valid {
		prevHash = &row.PreviousHash.String
	}
	return domain.ProvenanceRecord{
		ID:               row.ID,
		ListingID:        row.ListingID,
		ArtisanID:        row.ArtisanID,
		CraftID:          row.CraftID,
		ContentHash:      row.ContentHash,
		PreviousHash:     prevHash,
		Signature:        row.Signature,
		SignatureAlgo:    row.SignatureAlgo,
		PublicKeyID:      row.PublicKeyID,
		ShortCode:        row.ShortCode,
		TechniqueMatched: row.TechniqueMatched,
		MediaHashes:      row.MediaHashes,
		SealedAt:         row.SealedAt,
		CreatedAt:        row.CreatedAt,
	}
}