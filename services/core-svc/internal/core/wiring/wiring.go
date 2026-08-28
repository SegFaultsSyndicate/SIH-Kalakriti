// services/core-svc/internal/core/wiring/wiring.go

// Package wiring binds core-svc's concrete repository and infrastructure clients
// to the interfaces the service layer declares. It exists as its own package so
// that the service package itself never imports the repository — which is what
// keeps the service tests free of any database dependency — while main.go still
// gets one constructor per port to call.
package wiring

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/segfaultsyndicate/kalakriti/pkg/storage"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/domain"
	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/repo"
	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/service"
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
