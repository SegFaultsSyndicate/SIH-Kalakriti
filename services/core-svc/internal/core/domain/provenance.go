// services/core-svc/internal/core/domain/provenance.go
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
)

// ProvenanceRecord is the frozen, hash-chained evidence that a listing is what it claims.
type ProvenanceRecord struct {
	ID               uuid.UUID
	ListingID        uuid.UUID
	ArtisanID        uuid.UUID
	CraftID          uuid.UUID
	ContentHash      string
	PreviousHash     *string
	Signature        []byte
	SignatureAlgo    string
	PublicKeyID      string
	ShortCode        string
	TechniqueMatched bool
	MediaHashes      []string
	SealedAt         time.Time
	CreatedAt        time.Time
}

// SealProvenanceInput is what the service needs to freeze a provenance record.
type SealProvenanceInput struct {
	ListingID        uuid.UUID
	MediaIDs         []uuid.UUID
	ClaimedTechnique string
	CreatedBy        string
}

// Validate checks the seal request before processing.
func (in SealProvenanceInput) Validate() error {
	if in.ListingID == uuid.Nil {
		return fmt.Errorf("listing_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if len(in.MediaIDs) == 0 {
		return fmt.Errorf("at least one media item is required for provenance: %w", pkgdomain.ErrInvalidInput)
	}
	if in.ClaimedTechnique == "" {
		return fmt.Errorf("claimed_technique is required: %w", pkgdomain.ErrInvalidInput)
	}
	return nil
}

// CanonicalProvenance is the subset of fields that are hashed and signed.
type CanonicalProvenance struct {
	ListingID        string   `json:"listing_id"`
	ArtisanID        string   `json:"artisan_id"`
	CraftID          string   `json:"craft_id"`
	TechniqueMatched bool     `json:"technique_matched"`
	MediaHashes      []string `json:"media_hashes"`
	CreatedAt        string   `json:"created_at"`
}
