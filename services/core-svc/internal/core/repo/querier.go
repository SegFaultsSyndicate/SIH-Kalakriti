// services/core-svc/internal/core/repo/querier.go
package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/repo/db"
)

// Querier is the exact slice of the sqlc-generated query set that core-svc uses.
// It exists for two reasons: the repo depends on a narrow, readable contract
// rather than the whole generated surface, and the compile-time assertion below
// turns any drift between these expectations and what `make sqlc` actually emits
// into one clear error here, rather than a scatter of errors across every call
// site.
type Querier interface {
	CreateArtisan(ctx context.Context, arg db.CreateArtisanParams) (db.Artisan, error)
	GetArtisan(ctx context.Context, id uuid.UUID) (db.Artisan, error)
	GetArtisanByPhone(ctx context.Context, phoneE164 string) (db.Artisan, error)
	UpdateArtisan(ctx context.Context, arg db.UpdateArtisanParams) (db.Artisan, error)
	ListArtisansByCluster(ctx context.Context, arg db.ListArtisansByClusterParams) ([]db.Artisan, error)
	AddArtisanCraft(ctx context.Context, arg db.AddArtisanCraftParams) error
	ListArtisanCrafts(ctx context.Context, artisanID uuid.UUID) ([]db.Craft, error)
	RemoveArtisanCraft(ctx context.Context, arg db.RemoveArtisanCraftParams) (int64, error)

	CreateCluster(ctx context.Context, arg db.CreateClusterParams) (db.Cluster, error)
	GetCluster(ctx context.Context, id uuid.UUID) (db.Cluster, error)
	UpdateCluster(ctx context.Context, arg db.UpdateClusterParams) (db.Cluster, error)
	CountClusterMembers(ctx context.Context, clusterID uuid.UUID) (int64, error)
	UpsertClusterMember(ctx context.Context, arg db.UpsertClusterMemberParams) error
	RemoveClusterMember(ctx context.Context, arg db.RemoveClusterMemberParams) (int64, error)
	ListClusterMembers(ctx context.Context, arg db.ListClusterMembersParams) ([]db.ListClusterMembersRow, error)
	GetClusterMember(ctx context.Context, arg db.GetClusterMemberParams) (db.GetClusterMemberRow, error)

	CreateShg(ctx context.Context, arg db.CreateShgParams) (db.Shg, error)
	GetShg(ctx context.Context, id uuid.UUID) (db.Shg, error)
	GetShgForArtisan(ctx context.Context, artisanID uuid.UUID) (db.Shg, error)
	UpdateShg(ctx context.Context, arg db.UpdateShgParams) (db.Shg, error)
	DeleteShgMembers(ctx context.Context, shgID uuid.UUID) (int64, error)
	InsertShgMember(ctx context.Context, arg db.InsertShgMemberParams) error
	RemoveShgMember(ctx context.Context, arg db.RemoveShgMemberParams) (int64, error)
	ListShgMembers(ctx context.Context, shgID uuid.UUID) ([]db.ListShgMembersRow, error)

	ListCrafts(ctx context.Context) ([]db.Craft, error)
	GetCraft(ctx context.Context, id uuid.UUID) (db.Craft, error)
	ListAllCraftAliases(ctx context.Context) ([]db.CraftAlias, error)
	UpsertCraft(ctx context.Context, arg db.UpsertCraftParams) (db.Craft, error)
	UpsertCraftAlias(ctx context.Context, arg db.UpsertCraftAliasParams) (db.CraftAlias, error)

	CreateProduct(ctx context.Context, arg db.CreateProductParams) (db.Product, error)
	CreateProductForMedia(ctx context.Context, arg db.CreateProductForMediaParams) (db.Product, error)
	GetProduct(ctx context.Context, id uuid.UUID) (db.Product, error)
	GetProductByMedia(ctx context.Context, sourceMediaID *uuid.UUID) (db.Product, error)

	CreateListing(ctx context.Context, arg db.CreateListingParams) (db.Listing, error)
	UpdateListing(ctx context.Context, arg db.UpdateListingParams) (db.Listing, error)
	TransitionListingState(ctx context.Context, arg db.TransitionListingStateParams) (db.Listing, error)
	GetListing(ctx context.Context, id uuid.UUID) (db.Listing, error)
	ListListings(ctx context.Context, arg db.ListListingsParams) ([]db.Listing, error)
	GetListingByProduct(ctx context.Context, productID uuid.UUID) (db.Listing, error)
	SetListingNeedsDescription(ctx context.Context, arg db.SetListingNeedsDescriptionParams) (db.Listing, error)
	GetListingTranslations(ctx context.Context, listingID uuid.UUID) ([]db.ListingTranslation, error)
	GetListingAttributes(ctx context.Context, listingID uuid.UUID) ([]db.ListingAttribute, error)
	UpsertListingTranslation(ctx context.Context, arg db.UpsertListingTranslationParams) (db.ListingTranslation, error)
	UpsertModelAttribute(ctx context.Context, arg db.UpsertModelAttributeParams) (int64, error)
	UpsertAuthoritativeAttribute(ctx context.Context, arg db.UpsertAuthoritativeAttributeParams) error
	DeleteListingAttributesByName(ctx context.Context, arg db.DeleteListingAttributesByNameParams) (int64, error)
	DeleteListingMedia(ctx context.Context, listingID uuid.UUID) (int64, error)
	InsertListingMedia(ctx context.Context, arg db.InsertListingMediaParams) error
	ListListingMedia(ctx context.Context, listingID uuid.UUID) ([]db.ListListingMediaRow, error)

	CreateMediaPending(ctx context.Context, arg db.CreateMediaPendingParams) (db.Media, error)
	GetMedia(ctx context.Context, id uuid.UUID) (db.Media, error)
	GetMediaByHash(ctx context.Context, arg db.GetMediaByHashParams) (db.Media, error)
	ListMediaByIDs(ctx context.Context, ids []uuid.UUID) ([]db.ListMediaByIDsRow, error)
	AttachMediaToProduct(ctx context.Context, arg db.AttachMediaToProductParams) (int64, error)
	TransitionMediaState(ctx context.Context, arg db.TransitionMediaStateParams) (db.Media, error)
	SetMediaEnhanced(ctx context.Context, arg db.SetMediaEnhancedParams) (db.Media, error)
	ListStalePendingMedia(ctx context.Context, arg db.ListStalePendingMediaParams) ([]db.Media, error)
	DeleteMedia(ctx context.Context, id uuid.UUID) (int64, error)

	GetListingPricingSource(ctx context.Context, arg db.GetListingPricingSourceParams) (db.GetListingPricingSourceRow, error)
	GetMinimumWage(ctx context.Context, arg db.GetMinimumWageParams) (db.GetMinimumWageRow, error)
	ListSeasonalityMultipliers(ctx context.Context, arg db.ListSeasonalityMultipliersParams) ([]db.ListSeasonalityMultipliersRow, error)
	PricingComparableStats(ctx context.Context, arg db.PricingComparableStatsParams) (db.PricingComparableStatsRow, error)

	InsertOutbox(ctx context.Context, arg db.InsertOutboxParams) (int64, error)
	FetchUnpublishedOutbox(ctx context.Context, batchSize int32) ([]db.Outbox, error)
	MarkOutboxPublished(ctx context.Context, ids []uuid.UUID) (int64, error)

	GetOrInsertIdempotencyKey(ctx context.Context, arg db.GetOrInsertIdempotencyKeyParams) (db.GetOrInsertIdempotencyKeyRow, error)
	SetIdempotentResponse(ctx context.Context, arg db.SetIdempotentResponseParams) (int64, error)
}

// Compile-time assertion that the generated *db.Queries satisfies the contract
// above. If `make sqlc` emits a different signature, this line is the single
// place that fails, and the fix is to correct Querier to match the generated code.
var _ Querier = (*db.Queries)(nil)
