// services/bff/internal/bff/client/catalog.go
package client

import (
	"context"

	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"

	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/handler"
)

// Catalog is bff's view of core-svc's catalog, ontology and identity
// services, satisfying handler.CatalogService.
//
// GetListingBySlug, GetArtisanBySlug and GetProvenanceByShortCode are not
// wired: no slug field exists anywhere on Listing or Artisan, and no RPC
// looks up a provenance record by its short code (only by listing id). Both
// need new backend work, not an adapter — they return a clear error instead
// of the nil-pointer panic this interface's caller got before it was wired
// at all.
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

var errNoSlugSupport = domain.Unavailable("slug-based lookup is not supported: the catalog has no slug field yet")

func (c *Catalog) GetListingBySlug(ctx context.Context, slug string) (*handler.ListingDetail, error) {
	return nil, errNoSlugSupport
}

func (c *Catalog) GetArtisanBySlug(ctx context.Context, slug string) (*handler.ArtisanProfile, error) {
	return nil, errNoSlugSupport
}

func (c *Catalog) GetProvenanceByShortCode(ctx context.Context, code string) (handler.ProvenanceRecord, error) {
	return handler.ProvenanceRecord{}, domain.Unavailable("provenance lookup by short code is not supported: no such RPC exists yet")
}

// ListPublishedListings is not wired either: it only exists to feed the
// sitemap with /listing/{slug} URLs, and GetListingBySlug can't serve any
// slug it would emit (see errNoSlugSupport above). Publishing an index of
// links that all 503 is worse than publishing none, so this stays an error
// until slug lookup exists — same gap, kept honest instead of half-wired.
func (c *Catalog) ListPublishedListings(ctx context.Context, limit, offset int32) ([]handler.ListingDetail, error) {
	return nil, errNoSlugSupport
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
