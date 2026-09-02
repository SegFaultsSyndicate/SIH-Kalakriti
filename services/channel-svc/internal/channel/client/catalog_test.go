// services/channel-svc/internal/channel/client/catalog_test.go
package client

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"
)

// Each fake embeds the real client interface (left nil) and overrides only
// the one or two methods the test needs — calling an unoverridden method
// would panic on the nil embedded interface, which is exactly the signal a
// test exercising an unexpected call path should get.

type fakeCatalogService struct {
	catalogv1.CatalogServiceClient
	listListings func(ctx context.Context, in *catalogv1.ListListingsRequest, opts ...grpc.CallOption) (*catalogv1.ListListingsResponse, error)
	getListing   func(ctx context.Context, in *catalogv1.GetListingRequest, opts ...grpc.CallOption) (*catalogv1.GetListingResponse, error)
}

func (f *fakeCatalogService) ListListings(ctx context.Context, in *catalogv1.ListListingsRequest, opts ...grpc.CallOption) (*catalogv1.ListListingsResponse, error) {
	return f.listListings(ctx, in, opts...)
}
func (f *fakeCatalogService) GetListing(ctx context.Context, in *catalogv1.GetListingRequest, opts ...grpc.CallOption) (*catalogv1.GetListingResponse, error) {
	return f.getListing(ctx, in, opts...)
}

type fakeMediaService struct {
	catalogv1.MediaServiceClient
	getMediaURL func(ctx context.Context, in *catalogv1.GetMediaURLRequest, opts ...grpc.CallOption) (*catalogv1.GetMediaURLResponse, error)
}

func (f *fakeMediaService) GetMediaURL(ctx context.Context, in *catalogv1.GetMediaURLRequest, opts ...grpc.CallOption) (*catalogv1.GetMediaURLResponse, error) {
	return f.getMediaURL(ctx, in, opts...)
}

type fakeOntologyService struct {
	catalogv1.OntologyServiceClient
	getCraft func(ctx context.Context, in *catalogv1.GetCraftRequest, opts ...grpc.CallOption) (*catalogv1.GetCraftResponse, error)
}

func (f *fakeOntologyService) GetCraft(ctx context.Context, in *catalogv1.GetCraftRequest, opts ...grpc.CallOption) (*catalogv1.GetCraftResponse, error) {
	return f.getCraft(ctx, in, opts...)
}

type fakeIdentityService struct {
	identityv1.IdentityServiceClient
	getArtisan func(ctx context.Context, in *identityv1.GetArtisanRequest, opts ...grpc.CallOption) (*identityv1.GetArtisanResponse, error)
}

func (f *fakeIdentityService) GetArtisan(ctx context.Context, in *identityv1.GetArtisanRequest, opts ...grpc.CallOption) (*identityv1.GetArtisanResponse, error) {
	return f.getArtisan(ctx, in, opts...)
}

func giStr(s string) *string { return &s }

func TestListPublishedForIndiaHandmadeHydratesEveryField(t *testing.T) {
	c := &Catalog{
		catalog: &fakeCatalogService{
			listListings: func(ctx context.Context, in *catalogv1.ListListingsRequest, opts ...grpc.CallOption) (*catalogv1.ListListingsResponse, error) {
				assert.Equal(t, catalogv1.ListingState_LISTING_STATE_PUBLISHED, in.GetState())
				return &catalogv1.ListListingsResponse{Listings: []*catalogv1.Listing{
					{Id: "lst-1", ArtisanId: "art-1", ProductId: "prod-1", Price: &commonv1.Money{AmountPaise: 500000, CurrencyCode: "INR"}},
				}}, nil
			},
			getListing: func(ctx context.Context, in *catalogv1.GetListingRequest, opts ...grpc.CallOption) (*catalogv1.GetListingResponse, error) {
				require.Equal(t, "lst-1", in.GetListingId())
				require.True(t, in.GetIncludeProduct())
				return &catalogv1.GetListingResponse{
					Listing: &catalogv1.Listing{
						Id: "lst-1",
						Translations: []*catalogv1.ListingTranslation{
							{Language: commonv1.Language_LANGUAGE_ENGLISH, Title: "Handwoven Saree", Description: "Pure silk"},
						},
					},
					Product: &catalogv1.Product{
						CraftId: "craft-1",
						Media:   []*commonv1.MediaRef{{Id: "media-1", Kind: commonv1.MediaKind_MEDIA_KIND_IMAGE}},
					},
				}, nil
			},
		},
		media: &fakeMediaService{
			getMediaURL: func(ctx context.Context, in *catalogv1.GetMediaURLRequest, opts ...grpc.CallOption) (*catalogv1.GetMediaURLResponse, error) {
				require.Equal(t, "media-1", in.GetMediaId())
				return &catalogv1.GetMediaURLResponse{Url: "https://cdn.example.com/media-1.jpg"}, nil
			},
		},
		ontology: &fakeOntologyService{
			getCraft: func(ctx context.Context, in *catalogv1.GetCraftRequest, opts ...grpc.CallOption) (*catalogv1.GetCraftResponse, error) {
				require.Equal(t, "craft-1", in.GetCraftId())
				return &catalogv1.GetCraftResponse{Craft: &catalogv1.Craft{DisplayName: "Kanchipuram Weaving", GiRegistrationNo: giStr("GI-123")}}, nil
			},
		},
		identity: &fakeIdentityService{
			getArtisan: func(ctx context.Context, in *identityv1.GetArtisanRequest, opts ...grpc.CallOption) (*identityv1.GetArtisanResponse, error) {
				require.Equal(t, "art-1", in.GetArtisanId())
				return &identityv1.GetArtisanResponse{Artisan: &catalogv1.Artisan{DisplayName: "Lakshmi Devi"}}, nil
			},
		},
	}

	records, err := c.ListPublishedForIndiaHandmade(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, records, 1)

	r := records[0]
	assert.Equal(t, "art-1", r.ArtisanID)
	assert.Equal(t, "Lakshmi Devi", r.ArtisanName)
	assert.Equal(t, "prod-1", r.ProductID)
	assert.Equal(t, "Handwoven Saree", r.ProductTitle)
	assert.Equal(t, "Pure silk", r.Description)
	assert.Equal(t, "Kanchipuram Weaving", r.Craft)
	assert.Equal(t, "5000.00", r.Price)
	assert.Equal(t, "INR", r.Currency)
	assert.Equal(t, "https://cdn.example.com/media-1.jpg", r.ImageURL)
	assert.Equal(t, "GI-123", r.GINumber)
}

func TestListPublishedForIndiaHandmadeSkipsAListingWhoseDetailFails(t *testing.T) {
	c := &Catalog{
		catalog: &fakeCatalogService{
			listListings: func(ctx context.Context, in *catalogv1.ListListingsRequest, opts ...grpc.CallOption) (*catalogv1.ListListingsResponse, error) {
				return &catalogv1.ListListingsResponse{Listings: []*catalogv1.Listing{
					{Id: "lst-broken"}, {Id: "lst-ok"},
				}}, nil
			},
			getListing: func(ctx context.Context, in *catalogv1.GetListingRequest, opts ...grpc.CallOption) (*catalogv1.GetListingResponse, error) {
				if in.GetListingId() == "lst-broken" {
					return nil, errors.New("core-svc unavailable")
				}
				return &catalogv1.GetListingResponse{Listing: &catalogv1.Listing{Id: "lst-ok"}}, nil
			},
		},
		media:    &fakeMediaService{},
		ontology: &fakeOntologyService{},
		identity: &fakeIdentityService{},
	}

	records, err := c.ListPublishedForIndiaHandmade(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, records, 1, "the broken listing should be skipped, not fail the whole export")
}

func TestListPublishedForIndiaHandmadeToleratesCraftAndArtisanLookupFailures(t *testing.T) {
	c := &Catalog{
		catalog: &fakeCatalogService{
			listListings: func(ctx context.Context, in *catalogv1.ListListingsRequest, opts ...grpc.CallOption) (*catalogv1.ListListingsResponse, error) {
				return &catalogv1.ListListingsResponse{Listings: []*catalogv1.Listing{{Id: "lst-1", ArtisanId: "art-1"}}}, nil
			},
			getListing: func(ctx context.Context, in *catalogv1.GetListingRequest, opts ...grpc.CallOption) (*catalogv1.GetListingResponse, error) {
				return &catalogv1.GetListingResponse{
					Listing: &catalogv1.Listing{Id: "lst-1"},
					Product: &catalogv1.Product{CraftId: "craft-1"},
				}, nil
			},
		},
		media: &fakeMediaService{},
		ontology: &fakeOntologyService{
			getCraft: func(ctx context.Context, in *catalogv1.GetCraftRequest, opts ...grpc.CallOption) (*catalogv1.GetCraftResponse, error) {
				return nil, errors.New("ontology unavailable")
			},
		},
		identity: &fakeIdentityService{
			getArtisan: func(ctx context.Context, in *identityv1.GetArtisanRequest, opts ...grpc.CallOption) (*identityv1.GetArtisanResponse, error) {
				return nil, errors.New("identity unavailable")
			},
		},
	}

	records, err := c.ListPublishedForIndiaHandmade(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, records, 1, "the row must still be present with what did resolve")
	assert.Equal(t, "", records[0].Craft)
	assert.Equal(t, "", records[0].ArtisanName)
}
