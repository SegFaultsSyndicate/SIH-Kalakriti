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
	ClusterID   *string
}

// Craft is the craft ontology node.
type Craft struct {
	ID               string
	DisplayName      string
	GIRegistrationNo *string
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

// ArtisanProfile is the artisan data for SEO pages.
type ArtisanProfile struct {
	ID          string
	Slug        string
	DisplayName string
	Bio         string
	Location    string
	ImageURL    string
	CraftName   string
}

// CatalogService is the unified interface for catalog operations (SEO + verification).
type CatalogService interface {
	GetListingBySlug(ctx context.Context, slug string) (*ListingDetail, error)
	GetArtisanBySlug(ctx context.Context, slug string) (*ArtisanProfile, error)
	ListPublishedListings(ctx context.Context, limit, offset int32) ([]ListingDetail, error)
	GetProvenanceByShortCode(ctx context.Context, code string) (ProvenanceRecord, error)
	GetListing(ctx context.Context, listingID string) (Listing, error)
	GetArtisan(ctx context.Context, artisanID string) (Artisan, error)
	GetCraft(ctx context.Context, craftID string) (Craft, error)
}
