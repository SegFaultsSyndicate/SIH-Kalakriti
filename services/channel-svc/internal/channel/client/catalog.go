// services/channel-svc/internal/channel/client/catalog.go

// Package client holds channel-svc's outbound gRPC clients: core-svc's
// catalog and media services, needed to hydrate a bare
// catalog.listing.published event into the full listing detail an outbound
// channel (ONDC today) publishes.
package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"

	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/export"
	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/ondc"
)

// Catalog is channel-svc's view of core-svc: fetch one listing, fully
// hydrated with its product and presigned image URLs, shaped for an outbound
// channel adapter.
type Catalog struct {
	catalog  catalogv1.CatalogServiceClient
	media    catalogv1.MediaServiceClient
	ontology catalogv1.OntologyServiceClient
	identity identityv1.IdentityServiceClient
}

// NewCatalog builds the catalog client. conn is shared with any other client
// dialled against the same core-svc address — every service used here is
// registered on that one gRPC server.
func NewCatalog(conn grpc.ClientConnInterface) *Catalog {
	return &Catalog{
		catalog:  catalogv1.NewCatalogServiceClient(conn),
		media:    catalogv1.NewMediaServiceClient(conn),
		ontology: catalogv1.NewOntologyServiceClient(conn),
		identity: identityv1.NewIdentityServiceClient(conn),
	}
}

// GetListingForONDC fetches listingID hydrated with its product, and returns
// it as an ondc.Listing ready for Client.PublishOnSearch. Only PUBLISHED
// listings are meaningful here, but that state check belongs to the caller
// deciding whether to publish, not to this fetch.
func (c *Catalog) GetListingForONDC(ctx context.Context, listingID string) (ondc.Listing, error) {
	resp, err := c.catalog.GetListing(ctx, &catalogv1.GetListingRequest{
		ListingId:      listingID,
		Language:       commonv1.Language_LANGUAGE_ENGLISH,
		IncludeProduct: true,
	})
	if err != nil {
		return ondc.Listing{}, fmt.Errorf("fetching listing %s: %w", listingID, err)
	}
	listing := resp.GetListing()
	if listing == nil {
		return ondc.Listing{}, fmt.Errorf("listing %s: empty response", listingID)
	}

	title, description := listingCopy(listing)

	out := ondc.Listing{
		ID:          listing.GetId(),
		Title:       title,
		Description: description,
		PriceCents:  listing.GetPrice().GetAmountPaise(),
		Currency:    listing.GetPrice().GetCurrencyCode(),
		// ponytail: ONDC's own category taxonomy isn't modelled anywhere in
		// this schema yet; craft_id is the closest existing grouping. Swap in
		// a real category mapping if/when ONDC listing approval starts
		// rejecting on this field.
		CategoryID: resp.GetProduct().GetCraftId(),
	}

	for _, m := range resp.GetProduct().GetMedia() {
		if m.GetKind() != commonv1.MediaKind_MEDIA_KIND_IMAGE {
			continue
		}
		urlResp, err := c.media.GetMediaURL(ctx, &catalogv1.GetMediaURLRequest{MediaId: m.GetId()})
		if err != nil {
			// One broken image link shouldn't drop the whole listing from the
			// feed — publish with whatever images did resolve.
			continue
		}
		out.ImageURLs = append(out.ImageURLs, urlResp.GetUrl())
	}

	return out, nil
}

// ListPublishedForIndiaHandmade pages up to limit PUBLISHED listings,
// hydrated into IndiaHandmade export rows. One listing's detail failing to
// resolve (a craft or artisan lookup, an image URL) is not fatal to the
// whole export — that row just carries whatever it could gather rather than
// dropping the listing from the feed entirely.
func (c *Catalog) ListPublishedForIndiaHandmade(ctx context.Context, limit int32) ([]export.IndiaHandmadeRecord, error) {
	listResp, err := c.catalog.ListListings(ctx, &catalogv1.ListListingsRequest{
		State: catalogv1.ListingState_LISTING_STATE_PUBLISHED,
		Page:  &commonv1.PageRequest{PageSize: limit},
	})
	if err != nil {
		return nil, fmt.Errorf("listing published catalog: %w", err)
	}

	records := make([]export.IndiaHandmadeRecord, 0, len(listResp.GetListings()))
	for _, l := range listResp.GetListings() {
		detail, err := c.catalog.GetListing(ctx, &catalogv1.GetListingRequest{
			ListingId:      l.GetId(),
			Language:       commonv1.Language_LANGUAGE_ENGLISH,
			IncludeProduct: true,
		})
		if err != nil {
			continue
		}
		title, description := listingCopy(detail.GetListing())

		var craftName, giNumber string
		if craftID := detail.GetProduct().GetCraftId(); craftID != "" {
			if craftResp, err := c.ontology.GetCraft(ctx, &catalogv1.GetCraftRequest{CraftId: craftID}); err == nil {
				craftName = craftResp.GetCraft().GetDisplayName()
				giNumber = craftResp.GetCraft().GetGiRegistrationNo()
			}
		}

		var artisanName string
		if artisanID := l.GetArtisanId(); artisanID != "" {
			if artResp, err := c.identity.GetArtisan(ctx, &identityv1.GetArtisanRequest{ArtisanId: artisanID}); err == nil {
				artisanName = artResp.GetArtisan().GetDisplayName()
			}
		}

		var imageURL string
		for _, m := range detail.GetProduct().GetMedia() {
			if m.GetKind() != commonv1.MediaKind_MEDIA_KIND_IMAGE {
				continue
			}
			if urlResp, err := c.media.GetMediaURL(ctx, &catalogv1.GetMediaURLRequest{MediaId: m.GetId()}); err == nil {
				imageURL = urlResp.GetUrl()
				break // IndiaHandmadeRecord carries one image, not a gallery.
			}
		}

		records = append(records, export.IndiaHandmadeRecord{
			ArtisanID:    l.GetArtisanId(),
			ArtisanName:  artisanName,
			ProductID:    l.GetProductId(),
			ProductTitle: title,
			Description:  description,
			Craft:        craftName,
			Price:        fmt.Sprintf("%.2f", float64(l.GetPrice().GetAmountPaise())/100.0),
			Currency:     l.GetPrice().GetCurrencyCode(),
			ImageURL:     imageURL,
			GINumber:     giNumber,
		})
	}
	return records, nil
}

// listingCopy picks the listing's English translation, falling back to
// whichever translation came back first if English isn't among them (a
// listing must have at least one to have reached PUBLISHED).
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
