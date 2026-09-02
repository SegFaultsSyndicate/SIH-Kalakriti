// services/bff/internal/bff/client/catalog.go
package client

import (
	"context"
	"strings"

	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"

	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/handler"
)

// Catalog is bff's view of core-svc's catalog, ontology and identity
// services, satisfying handler.CatalogService.
type Catalog struct {
	catalog  catalogv1.CatalogServiceClient
	ontology catalogv1.OntologyServiceClient
	identity identityv1.IdentityServiceClient
}

// NewCatalog builds the catalog client, sharing conn with core-svc's other services.
func NewCatalog(conn grpc.ClientConnInterface) *Catalog {
	return &Catalog{
		catalog:  catalogv1.NewCatalogServiceClient(conn),
		ontology: catalogv1.NewOntologyServiceClient(conn),
		identity: identityv1.NewIdentityServiceClient(conn),
	}
}

func (c *Catalog) GetProvenanceByShortCode(ctx context.Context, code string) (handler.ProvenanceRecord, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.catalog.GetProvenanceByShortCode(ctx, &catalogv1.GetProvenanceByShortCodeRequest{ShortCode: code})
	if err != nil {
		return handler.ProvenanceRecord{}, grpcErr(err)
	}
	rec := resp.GetRecord()
	return handler.ProvenanceRecord{
		ID:               rec.GetId(),
		ListingID:        rec.GetListingId(),
		ArtisanID:        rec.GetArtisanId(),
		CraftID:          rec.GetCraftId(),
		ContentHash:      rec.GetContentHash(),
		PreviousHash:     rec.PreviousHash,
		Signature:        rec.GetSignature(),
		SignatureAlgo:    rec.GetSignatureAlgorithm(),
		PublicKeyID:      rec.GetPublicKeyId(),
		ShortCode:        rec.GetShortCode(),
		TechniqueMatched: rec.GetTechniqueMatched(),
		MediaHashes:      rec.GetMediaHashes(),
		SealedAt:         rec.GetSealedAt().AsTime(),
	}, nil
}

// GetListingBySlug resolves a slug built by buildSlug back to its id
// (parseSlugID) and fetches the listing. The slug is derived, not stored —
// no proto change, no migration — so it stays valid even if the title later
// changes; the id is the only part that must round-trip.
func (c *Catalog) GetListingBySlug(ctx context.Context, slug string) (*handler.ListingDetail, error) {
	id, ok := parseSlugID(slug)
	if !ok {
		return nil, domain.NotFound("listing not found")
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.catalog.GetListing(ctx, &catalogv1.GetListingRequest{ListingId: id, IncludeProduct: true})
	if err != nil {
		return nil, grpcErr(err)
	}
	listing := resp.GetListing()
	title, description := listingCopy(listing)

	var artisanName string
	if art, err := c.identity.GetArtisan(ctx, &identityv1.GetArtisanRequest{ArtisanId: listing.GetArtisanId()}); err == nil {
		artisanName = art.GetArtisan().GetDisplayName()
	}
	var craftName string
	if product := resp.GetProduct(); product != nil {
		if craft, err := c.ontology.GetCraft(ctx, &catalogv1.GetCraftRequest{CraftId: product.GetCraftId()}); err == nil {
			craftName = craft.GetCraft().GetDisplayName()
		}
	}

	return &handler.ListingDetail{
		ID:          listing.GetId(),
		Slug:        buildSlug(title, listing.GetId()),
		Title:       title,
		Description: description,
		PricePaise:  listing.GetPrice().GetAmountPaise(),
		Currency:    listing.GetPrice().GetCurrencyCode(),
		ArtisanName: artisanName,
		CraftName:   craftName,
		Available:   listing.GetState() == catalogv1.ListingState_LISTING_STATE_PUBLISHED,
	}, nil
}

// GetArtisanBySlug mirrors GetListingBySlug for artisan profile pages.
func (c *Catalog) GetArtisanBySlug(ctx context.Context, slug string) (*handler.ArtisanProfile, error) {
	id, ok := parseSlugID(slug)
	if !ok {
		return nil, domain.NotFound("artisan not found")
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.identity.GetArtisan(ctx, &identityv1.GetArtisanRequest{ArtisanId: id})
	if err != nil {
		return nil, grpcErr(err)
	}
	art := resp.GetArtisan()

	var craftName string
	if craftIDs := art.GetCraftIds(); len(craftIDs) > 0 {
		if craft, err := c.ontology.GetCraft(ctx, &catalogv1.GetCraftRequest{CraftId: craftIDs[0]}); err == nil {
			craftName = craft.GetCraft().GetDisplayName()
		}
	}

	var bio string
	if art.Bio != nil {
		bio = *art.Bio
	}

	return &handler.ArtisanProfile{
		ID:          art.GetId(),
		Slug:        buildSlug(art.GetDisplayName(), art.GetId()),
		DisplayName: art.GetDisplayName(),
		Bio:         bio,
		Location:    regionLocation(art.GetRegion()),
		CraftName:   craftName,
	}, nil
}

// regionLocation renders a GeoRegion as a short human-readable location.
func regionLocation(r *commonv1.GeoRegion) string {
	if r == nil {
		return ""
	}
	var parts []string
	if r.District != nil && *r.District != "" {
		parts = append(parts, *r.District)
	}
	if r.GetStateCode() != "" {
		parts = append(parts, r.GetStateCode())
	}
	return strings.Join(parts, ", ")
}

// ListPublishedListings feeds the sitemap. ListListingsRequest pages by
// opaque cursor, not numeric offset, so only the first page (offset 0) is
// servable without persisting a cursor between calls; anything else comes
// back empty rather than silently returning the first page again under a
// different offset's URL.
func (c *Catalog) ListPublishedListings(ctx context.Context, limit, offset int32) ([]handler.ListingDetail, error) {
	if offset != 0 {
		return nil, nil
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.catalog.ListListings(ctx, &catalogv1.ListListingsRequest{
		State: catalogv1.ListingState_LISTING_STATE_PUBLISHED,
		Page:  &commonv1.PageRequest{PageSize: limit},
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]handler.ListingDetail, 0, len(resp.GetListings()))
	for _, l := range resp.GetListings() {
		title, _ := listingCopy(l)
		out = append(out, handler.ListingDetail{
			ID:         l.GetId(),
			Slug:       buildSlug(title, l.GetId()),
			Title:      title,
			PricePaise: l.GetPrice().GetAmountPaise(),
			Currency:   l.GetPrice().GetCurrencyCode(),
			Available:  true,
		})
	}
	return out, nil
}

func (c *Catalog) GetListing(ctx context.Context, listingID string) (handler.Listing, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.catalog.GetListing(ctx, &catalogv1.GetListingRequest{ListingId: listingID})
	if err != nil {
		return handler.Listing{}, grpcErr(err)
	}

	l := resp.GetListing()
	title, _ := listingCopy(l)
	return handler.Listing{
		ID:        l.GetId(),
		ProductID: l.GetProductId(),
		ArtisanID: l.GetArtisanId(),
		Title:     title,
		Price:     l.GetPrice().GetAmountPaise(),
		Currency:  l.GetPrice().GetCurrencyCode(),
	}, nil
}

func (c *Catalog) GetArtisan(ctx context.Context, artisanID string) (handler.Artisan, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.identity.GetArtisan(ctx, &identityv1.GetArtisanRequest{ArtisanId: artisanID})
	if err != nil {
		return handler.Artisan{}, grpcErr(err)
	}

	art := resp.GetArtisan()
	return handler.Artisan{
		ID:          art.GetId(),
		DisplayName: art.GetDisplayName(),
		ClusterID:   art.ClusterId,
	}, nil
}

func (c *Catalog) GetCraft(ctx context.Context, craftID string) (handler.Craft, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.ontology.GetCraft(ctx, &catalogv1.GetCraftRequest{CraftId: craftID})
	if err != nil {
		return handler.Craft{}, grpcErr(err)
	}

	craft := resp.GetCraft()
	return handler.Craft{
		ID:               craft.GetId(),
		DisplayName:      craft.GetDisplayName(),
		GIRegistrationNo: craft.GiRegistrationNo,
	}, nil
}

// listingCopy picks a listing's English translation, falling back to
// whichever translation came back first if English isn't among them.
func listingCopy(listing *catalogv1.Listing) (title, description string) {
	translations := listing.GetTranslations()
	for _, t := range translations {
		if t.GetLanguage() == commonv1.Language_LANGUAGE_ENGLISH {
			return t.GetTitle(), t.GetDescription()
		}
	}
	if len(translations) > 0 {
		return translations[0].GetTitle(), translations[0].GetDescription()
	}
	return "", ""
}
