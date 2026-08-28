// services/core-svc/internal/core/service/provenance.go
package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/segfaultsyndicate/kalakriti/pkg/auth"
	"github.com/segfaultsyndicate/kalakriti/pkg/canonical"
	"github.com/segfaultsyndicate/kalakriti/pkg/crypto"
	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
	"github.com/segfaultsyndicate/kalakriti/pkg/ids"
	"github.com/segfaultsyndicate/kalakriti/pkg/outbox"
	"github.com/segfaultsyndicate/kalakriti/pkg/shortcode"
	"github.com/segfaultsyndicate/kalakriti/pkg/topics"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/domain"
)

// ProvenanceTx is what SealProvenance needs from the transaction.
type ProvenanceTx interface {
	outbox.Enqueuer
	InsertProvenanceRecord(ctx context.Context, rec domain.ProvenanceRecord) (domain.ProvenanceRecord, error)
	SetListingProvenance(ctx context.Context, listingID, provenanceID uuid.UUID) error
	ShortCodeExists(ctx context.Context, code string) (bool, error)
	GetLatestProvenanceForArtisan(ctx context.Context, artisanID uuid.UUID) (*domain.ProvenanceRecord, error)
}

// ProvenanceStore is the provenance read surface plus transaction entry point.
type ProvenanceStore interface {
	InTx(ctx context.Context, fn func(ctx context.Context, tx ProvenanceTx) error) error
	GetProvenanceRecord(ctx context.Context, id uuid.UUID) (domain.ProvenanceRecord, error)
	GetProvenanceByShortCode(ctx context.Context, code string) (domain.ProvenanceRecord, error)
	GetProvenanceByListing(ctx context.Context, listingID uuid.UUID) (domain.ProvenanceRecord, error)
}

// MediaHasher computes SHA-256 hashes of stored media.
type MediaHasher interface {
	HashMedia(ctx context.Context, mediaID uuid.UUID) (string, error)
}

// TechniqueVerifier checks a claimed technique against inference results.
type TechniqueVerifier interface {
	Verify(ctx context.Context, listingID uuid.UUID, claimedTechnique string) (bool, error)
}

// Provenance is core-svc's provenance sealing service.
type Provenance struct {
	store     ProvenanceStore
	catalog   CatalogStore
	hasher    MediaHasher
	verifier  TechniqueVerifier
	signer    *crypto.Signer
	shortCode *shortcode.Generator
	now       func() time.Time
}

// NewProvenance builds the provenance service.
func NewProvenance(
	store ProvenanceStore,
	catalog CatalogStore,
	hasher MediaHasher,
	verifier TechniqueVerifier,
	signer *crypto.Signer,
) *Provenance {
	return &Provenance{
		store:    store,
		catalog:  catalog,
		hasher:   hasher,
		verifier: verifier,
		signer:   signer,
		now:      time.Now,
	}
}

// SealProvenance freezes provenance evidence, computes the hash chain, signs, and emits the event.
func (s *Provenance) SealProvenance(ctx context.Context, in domain.SealProvenanceInput, idempotencyKey string) (domain.ProvenanceRecord, error) {
	if err := in.Validate(); err != nil {
		return domain.ProvenanceRecord{}, err
	}
	if idempotencyKey == "" {
		return domain.ProvenanceRecord{}, fmt.Errorf("idempotency_key is required: %w", pkgdomain.ErrInvalidInput)
	}

	// Fetch the listing and verify it's published.
	listing, err := s.catalog.GetListingDetail(ctx, in.ListingID)
	if err != nil {
		return domain.ProvenanceRecord{}, err
	}
	if listing.State != domain.StatePublished {
		return domain.ProvenanceRecord{}, fmt.Errorf(
			"listing %s is in state %s, only PUBLISHED listings may be sealed: %w",
			listing.ID, listing.State, pkgdomain.ErrInvalidInput)
	}

	// Authorize: only the listing's artisan may seal.
	principal, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return domain.ProvenanceRecord{}, err
	}
	if principal.ArtisanID == nil || *principal.ArtisanID != listing.ArtisanID {
		return domain.ProvenanceRecord{}, fmt.Errorf(
			"only the listing's artisan may seal provenance: %w", pkgdomain.ErrForbidden)
	}

	// Check if already sealed.
	if listing.ProvenanceID != nil {
		existing, err := s.store.GetProvenanceRecord(ctx, *listing.ProvenanceID)
		if err != nil && !errors.Is(err, pkgdomain.ErrNotFound) {
			return domain.ProvenanceRecord{}, err
		}
		if err == nil {
			return existing, nil // idempotent
		}
	}

	// Fetch the product to get craft_id.
	product, err := s.catalog.GetProduct(ctx, listing.ProductID)
	if err != nil {
		return domain.ProvenanceRecord{}, err
	}

	// Verify the technique claim.
	techniqueMatched, err := s.verifier.Verify(ctx, listing.ID, in.ClaimedTechnique)
	if err != nil {
		return domain.ProvenanceRecord{}, fmt.Errorf("verifying technique: %w", err)
	}

	// Hash the media.
	mediaHashes := make([]string, len(in.MediaIDs))
	for i, mediaID := range in.MediaIDs {
		hash, err := s.hasher.HashMedia(ctx, mediaID)
		if err != nil {
			return domain.ProvenanceRecord{}, fmt.Errorf("hashing media %s: %w", mediaID, err)
		}
		mediaHashes[i] = hash
	}
	sort.Strings(mediaHashes) // deterministic order

	provenanceID := ids.New()
	sealedAt := s.now().UTC()

	// Build canonical form.
	canon := domain.CanonicalProvenance{
		ListingID:        listing.ID.String(),
		ArtisanID:        listing.ArtisanID.String(),
		CraftID:          product.CraftID.String(),
		TechniqueMatched: techniqueMatched,
		MediaHashes:      mediaHashes,
		CreatedAt:        sealedAt.Format(time.RFC3339),
	}

	// Hash it.
	contentHash, err := canonical.HashJSON(canon)
	if err != nil {
		return domain.ProvenanceRecord{}, fmt.Errorf("hashing canonical provenance: %w", err)
	}

	// Sign the hash.
	hashBytes, _ := hex.DecodeString(contentHash)
	signature := s.signer.Sign(hashBytes)

	var stored domain.ProvenanceRecord
	err = s.store.InTx(ctx, func(ctx context.Context, tx ProvenanceTx) error {
		// Initialize shortcode generator with collision checker bound to this tx.
		if s.shortCode == nil {
			s.shortCode = shortcode.NewGenerator(func(ctx context.Context, code string) (bool, error) {
				return tx.ShortCodeExists(ctx, code)
			})
		}

		// Get the artisan's previous record for the hash chain.
		var previousHash *string
		prev, err := tx.GetLatestProvenanceForArtisan(ctx, listing.ArtisanID)
		if err != nil && !errors.Is(err, pkgdomain.ErrNotFound) {
			return err
		}
		if prev != nil {
			previousHash = &prev.ContentHash
		}

		// Generate collision-free short code.
		shortCode, err := s.shortCode.Generate(ctx)
		if err != nil {
			return fmt.Errorf("generating short code: %w", err)
		}

		rec := domain.ProvenanceRecord{
			ID:               provenanceID,
			ListingID:        listing.ID,
			ArtisanID:        listing.ArtisanID,
			CraftID:          product.CraftID,
			ContentHash:      contentHash,
			PreviousHash:     previousHash,
			Signature:        signature,
			SignatureAlgo:    "ed25519",
			PublicKeyID:      s.signer.KeyID(),
			ShortCode:        shortCode,
			TechniqueMatched: techniqueMatched,
			MediaHashes:      mediaHashes,
			SealedAt:         sealedAt,
		}

		stored, err = tx.InsertProvenanceRecord(ctx, rec)
		if err != nil {
			return err
		}

		// Link provenance to listing.
		if err := tx.SetListingProvenance(ctx, listing.ID, provenanceID); err != nil {
			return err
		}

		// Emit event.
		payload := provenanceSealed{
			ProvenanceID:     provenanceID.String(),
			ListingID:        listing.ID.String(),
			ArtisanID:        listing.ArtisanID.String(),
			ContentHash:      contentHash,
			PreviousHash:     previousHash,
			TechniqueMatched: techniqueMatched,
			SealedAt:         sealedAt.Format(time.RFC3339),
		}
		evt := event{
			Header: eventHeader{
				EventID:        ids.New().String(),
				OccurredAt:     sealedAt,
				AggregateID:    provenanceID.String(),
				IdempotencyKey: idempotencyKey,
				SchemaVersion:  schemaVersion,
				Producer:       producerName,
			},
			Payload: payload,
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), provenanceID.String(),
			topics.CatalogProvenanceSealed, idempotencyKey, evt)
	})
	if err != nil {
		return domain.ProvenanceRecord{}, err
	}

	return stored, nil
}

// provenanceSealed mirrors events.v1.CatalogProvenanceSealed's payload fields.
type provenanceSealed struct {
	ProvenanceID     string  `json:"provenance_id"`
	ListingID        string  `json:"listing_id"`
	ArtisanID        string  `json:"artisan_id"`
	ContentHash      string  `json:"content_hash"`
	PreviousHash     *string `json:"previous_hash,omitempty"`
	TechniqueMatched bool    `json:"technique_matched"`
	SealedAt         string  `json:"sealed_at"`
}

// GetProvenanceByShortCode fetches a sealed record by its public short code.
func (s *Provenance) GetProvenanceByShortCode(ctx context.Context, code string) (domain.ProvenanceRecord, error) {
	if code == "" {
		return domain.ProvenanceRecord{}, fmt.Errorf("short_code is required: %w", pkgdomain.ErrInvalidInput)
	}
	return s.store.GetProvenanceByShortCode(ctx, code)
}
