package handler

import (
	"context"
	"time"
)

// Shared types used by both SEO and verification handlers.

// ProvenanceRecord mirrors the domain type for the BFF.
type ProvenanceRecord struct {
	ID               string
	ListingID        string
	ArtisanID        string
	CraftID          string
	ContentHash      string
	PreviousHash     *string
	Signature        []byte
	SignatureAlgo    string
	PublicKeyID      string
	ShortCode        string
	TechniqueMatched bool
	MediaHashes      []string
	SealedAt         time.Time
}

// Listing is the catalog listing.
type Listing struct {
	ID        string
	ProductID string
	ArtisanID string
	Title     string
	Price     int64
	Currency  string
}

// Artisan is the maker.
type Artisan struct {
	ID          string
	DisplayName string
	District    string
	ClusterID   *string
}

// Craft is the craft ontology node.
type Craft struct {
	ID               string
	Slug             string
	DisplayName      string
	GIRegistrationNo *string
	Regions          []string
	Techniques       []string
	Materials        []string
}

// ProcessClip is one craft-in-progress video, derived from a listing's own
// product media rather than a dedicated feed store -- see ListProcessClips.
type ProcessClip struct {
	ListingID   string
	ListingSlug string
	Title       string
	ArtisanID   string
	ArtisanName string
	VideoURL    string
	DurationMs  *int32
}

// ListingDetail is the full listing data for SEO pages.
type ListingDetail struct {
	ID          string
	Slug        string
	Title       string
	Description string
	PricePaise  int64
	Currency    string
	ImageURL    string
	ArtisanName string
	CraftName   string
	Available   bool
}

// ArtisanProfile is the artisan data for SEO pages and the buyer storefront.
type ArtisanProfile struct {
	ID              string
	Slug            string
	DisplayName     string
	Bio             string
	Location        string
	District        *string
	StateCode       string
	ClusterID       *string
	ImageURL        string
	CraftName       string
	CraftSlug       string
	Verified        bool
	YearsExperience *int32
}

// CatalogService is the unified interface for catalog operations (SEO,
// verification, and the buyer-facing storefront/craft/feed reads).
type CatalogService interface {
	GetListingBySlug(ctx context.Context, slug string) (*ListingDetail, error)
	GetArtisanBySlug(ctx context.Context, slug string) (*ArtisanProfile, error)
	ListPublishedListings(ctx context.Context, limit, offset int32) ([]ListingDetail, error)
	GetProvenanceByShortCode(ctx context.Context, code string) (ProvenanceRecord, error)
	GetListing(ctx context.Context, listingID string) (Listing, error)
	GetArtisan(ctx context.Context, artisanID string) (Artisan, error)
	GetCraft(ctx context.Context, craftID string) (Craft, error)
	// ListCrafts returns the whole craft ontology, ordered by slug.
	ListCrafts(ctx context.Context) ([]Craft, error)
	// GetCraftBySlug resolves a craft landing page.
	GetCraftBySlug(ctx context.Context, slug string) (*Craft, error)
	// ListArtisansByCraft returns the makers with a published listing under
	// this craft, deduplicated, up to limit.
	ListArtisansByCraft(ctx context.Context, craftID string, limit int32) ([]Artisan, error)
	// ListProcessClips returns the most recent craft-in-progress videos across
	// published listings, newest first. See client/catalog.go for how it's
	// derived rather than read from a dedicated feed store.
	ListProcessClips(ctx context.Context, limit int32) ([]ProcessClip, error)

	// Cluster and self-help-group administration (Batch 13, /clusters).
	CreateCluster(ctx context.Context, name, stateCode string, district, coordinatorPhone *string, idempotencyKey string) (map[string]any, error)
	GetCluster(ctx context.Context, clusterID string) (map[string]any, error)
	ListClusterMembers(ctx context.Context, clusterID string) ([]map[string]any, error)
	AddClusterMember(ctx context.Context, clusterID, artisanID, role, idempotencyKey string) (map[string]any, error)
	RemoveClusterMember(ctx context.Context, clusterID, artisanID, idempotencyKey string) (bool, error)
	CreateSelfHelpGroup(ctx context.Context, name, registrationNo string, clusterID *string, members []map[string]any, idempotencyKey string) (map[string]any, error)
	GetSelfHelpGroup(ctx context.Context, shgID string) (map[string]any, error)
	SetSelfHelpGroupMembers(ctx context.Context, shgID string, members []map[string]any, idempotencyKey string) (map[string]any, error)

	// Moderation (Batch 13, /moderation): human-decided listing suspension.
	SuspendListing(ctx context.Context, listingID, reason, idempotencyKey string) (Listing, error)
	ReinstateListing(ctx context.Context, listingID, idempotencyKey string) (Listing, error)

	// RefreshCraftIndex rebuilds the ontology alias index (Batch 13, /crafts).
	RefreshCraftIndex(ctx context.Context, idempotencyKey string) (map[string]any, error)
}
