// services/core-svc/internal/core/wiring/wiring.go

// Package wiring binds core-svc's concrete repository and infrastructure clients
// to the interfaces the service layer declares. It exists as its own package so
// that the service package itself never imports the repository — which is what
// keeps the service tests free of any database dependency — while main.go still
// gets one constructor per port to call.
package wiring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/storage"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Store binds a repository to service.Store. Every read method is promoted from
// the embedded *repo.Repo; only InTx needs adapting, because Go has no variance
// on function parameters: *repo.Repo.InTx hands its callback a *repo.Tx, while
// service.Store.InTx is declared over the service.Tx interface.
type Store struct {
	*repo.Repo
}

// NewStore wraps a repository as the identity service's persistence port.
func NewStore(r *repo.Repo) Store { return Store{Repo: r} }

// Store satisfies the identity service's persistence contract.
var _ service.Store = Store{}

// InTx adapts the repository's concrete transaction type to the Tx interface.
func (s Store) InTx(ctx context.Context, fn func(ctx context.Context, tx service.Tx) error) error {
	return s.Repo.InTx(ctx, func(ctx context.Context, tx *repo.Tx) error {
		return fn(ctx, tx)
	})
}

// CatalogStore binds the same repository to service.CatalogStore. It is a
// separate type only because the two ports declare InTx over different
// transaction interfaces, and one Go type cannot have both methods.
type CatalogStore struct {
	*repo.Repo
}

// NewCatalogStore wraps a repository as the catalog service's persistence port.
func NewCatalogStore(r *repo.Repo) CatalogStore { return CatalogStore{Repo: r} }

// CatalogStore satisfies the catalog service's persistence contract.
var _ service.CatalogStore = CatalogStore{}

// InTx adapts the repository's concrete transaction type to CatalogTx.
func (s CatalogStore) InTx(ctx context.Context, fn func(ctx context.Context, tx service.CatalogTx) error) error {
	return s.Repo.InTx(ctx, func(ctx context.Context, tx *repo.Tx) error {
		return fn(ctx, tx)
	})
}

// MediaStore binds the same repository to service.MediaStore.
type MediaStore struct {
	*repo.Repo
}

// NewMediaStore wraps a repository as the media service's persistence port.
func NewMediaStore(r *repo.Repo) MediaStore { return MediaStore{Repo: r} }

// MediaStore satisfies the media service's persistence contract.
var _ service.MediaStore = MediaStore{}

// PricingStore binds the same repository to service.PricingStore. It is
// read-only end to end, so — unlike Store and CatalogStore — it needs no InTx
// adapter at all; the embedded *repo.Repo methods satisfy the interface as-is.
type PricingStore struct {
	*repo.Repo
}

// NewPricingStore wraps a repository as the pricing service's persistence port.
func NewPricingStore(r *repo.Repo) PricingStore { return PricingStore{Repo: r} }

// PricingStore satisfies the pricing service's persistence contract.
var _ service.PricingStore = PricingStore{}

// InTx adapts the repository's concrete transaction type to MediaTx.
func (s MediaStore) InTx(ctx context.Context, fn func(ctx context.Context, tx service.MediaTx) error) error {
	return s.Repo.InTx(ctx, func(ctx context.Context, tx *repo.Tx) error {
		return fn(ctx, tx)
	})
}

// PipelineStore binds the same repository to service.PipelineStore. The pipeline
// reads through the repo and writes through the catalog and media services, so
// this port carries only the handful of reads those two do not expose.
type PipelineStore struct {
	*repo.Repo
}

// NewPipelineStore wraps a repository as the pipeline's persistence port.
func NewPipelineStore(r *repo.Repo) PipelineStore { return PipelineStore{Repo: r} }

// PipelineStore satisfies the pipeline's persistence contract.
var _ service.PipelineStore = PipelineStore{}

// GetOrCreateProductForMedia runs the insert-if-absent in its own transaction,
// which is all the atomicity the unique index needs.
func (s PipelineStore) GetOrCreateProductForMedia(
	ctx context.Context,
	in domain.CreateProductInput,
	mediaID uuid.UUID,
) (product domain.Product, created bool, err error) {
	err = s.Repo.InTx(ctx, func(ctx context.Context, tx *repo.Tx) error {
		product, created, err = tx.GetOrCreateProductForMedia(ctx, in, mediaID)
		return err
	})
	return product, created, err
}

// ObjectStore binds pkg/storage's MinIO client to service.ObjectStore. The
// service declares its own ObjectInfo so it does not import a MinIO type; this
// adapter is the one place the two meet.
type ObjectStore struct {
	client *storage.Client
}

// NewObjectStore wraps a storage client as the media service's bucket port.
func NewObjectStore(client *storage.Client) ObjectStore { return ObjectStore{client: client} }

// ObjectStore satisfies the media service's bucket contract.
var _ service.ObjectStore = ObjectStore{}

// PresignedPutURL mints a time-limited upload URL.
func (o ObjectStore) PresignedPutURL(ctx context.Context, objectKey, contentType string, expiry time.Duration) (string, error) {
	return o.client.PresignedPutURL(ctx, objectKey, contentType, expiry)
}

// PresignedGetURL mints a time-limited download URL.
func (o ObjectStore) PresignedGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	return o.client.PresignedGetURL(ctx, objectKey, expiry)
}

// Stat reports what the bucket actually holds for an object key.
func (o ObjectStore) Stat(ctx context.Context, objectKey string) (service.ObjectInfo, error) {
	info, err := o.client.Stat(ctx, objectKey)
	if err != nil {
		return service.ObjectInfo{}, err
	}
	return service.ObjectInfo{
		SizeBytes:   info.SizeBytes,
		ContentType: info.ContentType,
		ETag:        info.ETag,
	}, nil
}

// Delete removes an object; deleting a missing object is not an error.
func (o ObjectStore) Delete(ctx context.Context, objectKey string) error {
	return o.client.Delete(ctx, objectKey)
}

// ProvenanceStore binds the same repository to service.ProvenanceStore.
type ProvenanceStore struct {
	*repo.Repo
}

// NewProvenanceStore wraps a repository as the provenance service's persistence port.
func NewProvenanceStore(r *repo.Repo) ProvenanceStore { return ProvenanceStore{Repo: r} }

// ProvenanceStore satisfies the provenance service's persistence contract.
var _ service.ProvenanceStore = ProvenanceStore{}

// InTx adapts the repository's concrete transaction type to ProvenanceTx.
func (s ProvenanceStore) InTx(ctx context.Context, fn func(ctx context.Context, tx service.ProvenanceTx) error) error {
	return s.Repo.InTx(ctx, func(ctx context.Context, tx *repo.Tx) error {
		return fn(ctx, tx)
	})
}

// MediaHasher computes a media object's SHA-256 by downloading it fresh from
// object storage rather than trusting the client-supplied hash recorded at
// upload time: the sealed record is a cryptographic attestation, so it must
// hash what the artisan actually uploaded, not what a client claimed.
type MediaHasher struct {
	repo    *repo.Repo
	objects *storage.Client
}

// NewMediaHasher builds the provenance service's media-hashing port.
func NewMediaHasher(r *repo.Repo, objects *storage.Client) MediaHasher {
	return MediaHasher{repo: r, objects: objects}
}

// MediaHasher satisfies the provenance service's hashing contract.
var _ service.MediaHasher = MediaHasher{}

// HashMedia downloads the stored object and returns its lowercase hex SHA-256.
func (m MediaHasher) HashMedia(ctx context.Context, mediaID uuid.UUID) (string, error) {
	media, err := m.repo.GetMedia(ctx, mediaID)
	if err != nil {
		return "", fmt.Errorf("fetching media %s: %w", mediaID, err)
	}
	url, err := m.objects.PresignedGetURL(ctx, media.ServableObjectKey(), 5*time.Minute)
	if err != nil {
		return "", fmt.Errorf("presigning media %s: %w", mediaID, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("building request for media %s: %w", mediaID, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading media %s: %w", mediaID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("downloading media %s: object storage returned %d", mediaID, resp.StatusCode)
	}
	h := sha256.New()
	if _, err := io.Copy(h, resp.Body); err != nil {
		return "", fmt.Errorf("hashing media %s: %w", mediaID, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// TechniqueVerifier checks a listing's claimed craft technique against the
// model's observation, stored as the listing's "technique" attribute by the
// cataloguing pipeline (batch 9).
type TechniqueVerifier struct {
	repo *repo.Repo
}

// NewTechniqueVerifier builds the provenance service's technique-check port.
func NewTechniqueVerifier(r *repo.Repo) TechniqueVerifier { return TechniqueVerifier{repo: r} }

// TechniqueVerifier satisfies the provenance service's verification contract.
var _ service.TechniqueVerifier = TechniqueVerifier{}

// Observe returns the model's observed technique and its confidence, or ("",
// 0, nil) if the pipeline never wrote one for this listing.
func (t TechniqueVerifier) Observe(ctx context.Context, listingID uuid.UUID) (observed string, confidence float32, err error) {
	attrs, err := t.repo.GetListingAttributes(ctx, listingID)
	if err != nil {
		return "", 0, fmt.Errorf("fetching listing attributes for %s: %w", listingID, err)
	}
	for _, a := range attrs {
		if a.Name == "technique" {
			return a.Value, a.Confidence, nil
		}
	}
	return "", 0, nil
}

// Verify reports whether the claimed technique matches the model's
// observation. A listing the pipeline never scored cannot be confirmed.
func (t TechniqueVerifier) Verify(ctx context.Context, listingID uuid.UUID, claimedTechnique string) (bool, error) {
	observed, _, err := t.Observe(ctx, listingID)
	if err != nil {
		return false, err
	}
	if observed == "" {
		return false, nil
	}
	return strings.EqualFold(observed, claimedTechnique), nil
}

// B2BStore binds the repository to service.B2BStore.
type B2BStore struct {
	*repo.Repo
}

// NewB2BStore wraps a repository as the B2B service's persistence port.
func NewB2BStore(r *repo.Repo) B2BStore { return B2BStore{Repo: r} }

// B2BStore satisfies the B2B service's persistence contract.
var _ service.B2BStore = B2BStore{}

// InTx adapts the repository's concrete transaction type to the B2BTx interface.
func (s B2BStore) InTx(ctx context.Context, fn func(ctx context.Context, tx service.B2BTx) error) error {
	return s.Repo.InTx(ctx, func(ctx context.Context, tx *repo.Tx) error {
		return fn(ctx, tx)
	})
}
