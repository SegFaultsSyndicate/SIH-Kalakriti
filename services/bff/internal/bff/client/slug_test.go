// services/bff/internal/bff/client/slug_test.go
package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"
)

func TestSlugifyLowercasesAndCollapsesNonAlnumRuns(t *testing.T) {
	assert.Equal(t, "blue-dhurrie-9x9", slugify("  Blue!! Dhurrie -- 9x9  "))
	assert.Equal(t, "", slugify("!!!"))
}

func TestBuildSlugAndParseSlugIDRoundTrip(t *testing.T) {
	slug := buildSlug("Blue Dhurrie", "lst-1")
	assert.Equal(t, "blue-dhurrie--lst-1", slug)

	id, ok := parseSlugID(slug)
	require.True(t, ok)
	assert.Equal(t, "lst-1", id)
}

func TestBuildSlugWithEmptyNameIsJustTheID(t *testing.T) {
	slug := buildSlug("", "lst-1")
	assert.Equal(t, "lst-1", slug)

	id, ok := parseSlugID(slug)
	require.True(t, ok)
	assert.Equal(t, "lst-1", id)
}

func TestParseSlugIDRejectsEmptyString(t *testing.T) {
	_, ok := parseSlugID("")
	assert.False(t, ok)
}

func TestCatalogGetListingBySlugParsesIDAndHydratesArtisanAndCraftNames(t *testing.T) {
	c := &Catalog{
		catalog: &fakeCatalogService{
			getListing: func(ctx context.Context, in *catalogv1.GetListingRequest, opts ...grpc.CallOption) (*catalogv1.GetListingResponse, error) {
				require.Equal(t, "lst-1", in.GetListingId())
				require.True(t, in.GetIncludeProduct())
				return &catalogv1.GetListingResponse{
					Listing: &catalogv1.Listing{
						Id: "lst-1", ArtisanId: "art-1",
						State: catalogv1.ListingState_LISTING_STATE_PUBLISHED,
						Price: &commonv1.Money{AmountPaise: 100000, CurrencyCode: "INR"},
						Translations: []*catalogv1.ListingTranslation{
							{Language: commonv1.Language_LANGUAGE_ENGLISH, Title: "Blue dhurrie", Description: "Hand-woven"},
						},
					},
					Product: &catalogv1.Product{CraftId: "craft-1"},
				}, nil
			},
		},
		identity: &fakeIdentityService{
			getArtisan: func(ctx context.Context, in *identityv1.GetArtisanRequest, opts ...grpc.CallOption) (*identityv1.GetArtisanResponse, error) {
				return &identityv1.GetArtisanResponse{Artisan: &catalogv1.Artisan{Id: "art-1", DisplayName: "Lakshmi"}}, nil
			},
		},
		ontology: &fakeOntologyService{
			getCraft: func(ctx context.Context, in *catalogv1.GetCraftRequest, opts ...grpc.CallOption) (*catalogv1.GetCraftResponse, error) {
				return &catalogv1.GetCraftResponse{Craft: &catalogv1.Craft{Id: "craft-1", DisplayName: "Ajrakh"}}, nil
			},
		},
	}

	got, err := c.GetListingBySlug(context.Background(), "some-old-title--lst-1")
	require.NoError(t, err)
	assert.Equal(t, "blue-dhurrie--lst-1", got.Slug)
	assert.Equal(t, "Blue dhurrie", got.Title)
	assert.Equal(t, "Hand-woven", got.Description)
	assert.Equal(t, "Lakshmi", got.ArtisanName)
	assert.Equal(t, "Ajrakh", got.CraftName)
	assert.True(t, got.Available)
}

func TestCatalogGetListingBySlugRejectsAnUnparseableSlug(t *testing.T) {
	c := &Catalog{}
	_, err := c.GetListingBySlug(context.Background(), "")
	assert.True(t, domain.IsNotFound(err))
}

func TestCatalogGetArtisanBySlugHydratesCraftAndLocation(t *testing.T) {
	district := "Kutch"
	c := &Catalog{
		identity: &fakeIdentityService{
			getArtisan: func(ctx context.Context, in *identityv1.GetArtisanRequest, opts ...grpc.CallOption) (*identityv1.GetArtisanResponse, error) {
				require.Equal(t, "art-1", in.GetArtisanId())
				return &identityv1.GetArtisanResponse{Artisan: &catalogv1.Artisan{
					Id: "art-1", DisplayName: "Lakshmi", CraftIds: []string{"craft-1"},
					Region: &commonv1.GeoRegion{StateCode: "IN-GJ", District: &district},
				}}, nil
			},
		},
		ontology: &fakeOntologyService{
			getCraft: func(ctx context.Context, in *catalogv1.GetCraftRequest, opts ...grpc.CallOption) (*catalogv1.GetCraftResponse, error) {
				return &catalogv1.GetCraftResponse{Craft: &catalogv1.Craft{Id: "craft-1", DisplayName: "Ajrakh"}}, nil
			},
		},
	}

	got, err := c.GetArtisanBySlug(context.Background(), "art-1")
	require.NoError(t, err)
	assert.Equal(t, "lakshmi--art-1", got.Slug)
	assert.Equal(t, "Ajrakh", got.CraftName)
	assert.Equal(t, "Kutch, IN-GJ", got.Location)
}

func TestCatalogListPublishedListingsReturnsEmptyForNonZeroOffset(t *testing.T) {
	c := &Catalog{}
	got, err := c.ListPublishedListings(context.Background(), 100, 20)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestCatalogListPublishedListingsFiltersToPublishedState(t *testing.T) {
	var sawReq *catalogv1.ListListingsRequest
	c := &Catalog{catalog: &fakeCatalogService{
		listListings: func(ctx context.Context, in *catalogv1.ListListingsRequest, opts ...grpc.CallOption) (*catalogv1.ListListingsResponse, error) {
			sawReq = in
			return &catalogv1.ListListingsResponse{Listings: []*catalogv1.Listing{
				{Id: "lst-1", Translations: []*catalogv1.ListingTranslation{
					{Language: commonv1.Language_LANGUAGE_ENGLISH, Title: "Blue dhurrie"},
				}},
			}}, nil
		},
	}}

	got, err := c.ListPublishedListings(context.Background(), 50, 0)
	require.NoError(t, err)
	assert.Equal(t, catalogv1.ListingState_LISTING_STATE_PUBLISHED, sawReq.GetState())
	require.Len(t, got, 1)
	assert.Equal(t, "blue-dhurrie--lst-1", got[0].Slug)
}
