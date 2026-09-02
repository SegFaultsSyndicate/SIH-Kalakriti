// services/bff/internal/bff/client/client_test.go
package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"
	insightv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/insight/v1"
	searchv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/search/v1"
)

// Fakes embed the real client interface (left nil) and override only the
// methods a test needs — same pattern as channel-svc's client tests.

type fakeCatalogService struct {
	catalogv1.CatalogServiceClient
	getListing               func(ctx context.Context, in *catalogv1.GetListingRequest, opts ...grpc.CallOption) (*catalogv1.GetListingResponse, error)
	createProduct            func(ctx context.Context, in *catalogv1.CreateProductRequest, opts ...grpc.CallOption) (*catalogv1.CreateProductResponse, error)
	upsertListing            func(ctx context.Context, in *catalogv1.UpsertListingRequest, opts ...grpc.CallOption) (*catalogv1.UpsertListingResponse, error)
	submitForApproval        func(ctx context.Context, in *catalogv1.SubmitForApprovalRequest, opts ...grpc.CallOption) (*catalogv1.SubmitForApprovalResponse, error)
	approveListing           func(ctx context.Context, in *catalogv1.ApproveListingRequest, opts ...grpc.CallOption) (*catalogv1.ApproveListingResponse, error)
	listListings             func(ctx context.Context, in *catalogv1.ListListingsRequest, opts ...grpc.CallOption) (*catalogv1.ListListingsResponse, error)
	getProvenanceByShortCode func(ctx context.Context, in *catalogv1.GetProvenanceByShortCodeRequest, opts ...grpc.CallOption) (*catalogv1.GetProvenanceByShortCodeResponse, error)
}

func (f *fakeCatalogService) GetListing(ctx context.Context, in *catalogv1.GetListingRequest, opts ...grpc.CallOption) (*catalogv1.GetListingResponse, error) {
	return f.getListing(ctx, in, opts...)
}

func (f *fakeCatalogService) CreateProduct(ctx context.Context, in *catalogv1.CreateProductRequest, opts ...grpc.CallOption) (*catalogv1.CreateProductResponse, error) {
	return f.createProduct(ctx, in, opts...)
}

func (f *fakeCatalogService) UpsertListing(ctx context.Context, in *catalogv1.UpsertListingRequest, opts ...grpc.CallOption) (*catalogv1.UpsertListingResponse, error) {
	return f.upsertListing(ctx, in, opts...)
}

func (f *fakeCatalogService) SubmitForApproval(ctx context.Context, in *catalogv1.SubmitForApprovalRequest, opts ...grpc.CallOption) (*catalogv1.SubmitForApprovalResponse, error) {
	return f.submitForApproval(ctx, in, opts...)
}

func (f *fakeCatalogService) ApproveListing(ctx context.Context, in *catalogv1.ApproveListingRequest, opts ...grpc.CallOption) (*catalogv1.ApproveListingResponse, error) {
	return f.approveListing(ctx, in, opts...)
}

func (f *fakeCatalogService) ListListings(ctx context.Context, in *catalogv1.ListListingsRequest, opts ...grpc.CallOption) (*catalogv1.ListListingsResponse, error) {
	return f.listListings(ctx, in, opts...)
}

func (f *fakeCatalogService) GetProvenanceByShortCode(ctx context.Context, in *catalogv1.GetProvenanceByShortCodeRequest, opts ...grpc.CallOption) (*catalogv1.GetProvenanceByShortCodeResponse, error) {
	return f.getProvenanceByShortCode(ctx, in, opts...)
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
	getArtisan      func(ctx context.Context, in *identityv1.GetArtisanRequest, opts ...grpc.CallOption) (*identityv1.GetArtisanResponse, error)
	registerArtisan func(ctx context.Context, in *identityv1.RegisterArtisanRequest, opts ...grpc.CallOption) (*identityv1.RegisterArtisanResponse, error)
}

func (f *fakeIdentityService) GetArtisan(ctx context.Context, in *identityv1.GetArtisanRequest, opts ...grpc.CallOption) (*identityv1.GetArtisanResponse, error) {
	return f.getArtisan(ctx, in, opts...)
}

func (f *fakeIdentityService) RegisterArtisan(ctx context.Context, in *identityv1.RegisterArtisanRequest, opts ...grpc.CallOption) (*identityv1.RegisterArtisanResponse, error) {
	return f.registerArtisan(ctx, in, opts...)
}

func strPtr(s string) *string { return &s }

func TestGrpcErrMapsStatusCodesToDomainErrors(t *testing.T) {
	cases := []struct {
		code codes.Code
		is   func(error) bool
	}{
		{codes.NotFound, domain.IsNotFound},
		{codes.AlreadyExists, domain.IsConflict},
		{codes.PermissionDenied, domain.IsForbidden},
	}
	for _, c := range cases {
		err := grpcErr(status.Error(c.code, "boom"))
		assert.True(t, c.is(err), "code %s should map to a domain error IsX() recognises", c.code)
	}

	// InvalidArgument, Unauthenticated, Unavailable are checked directly via
	// domain.HTTPStatus since domain has no IsX helper for them.
	assert.Equal(t, 400, domain.HTTPStatus(grpcErr(status.Error(codes.InvalidArgument, "bad"))))
	assert.Equal(t, 401, domain.HTTPStatus(grpcErr(status.Error(codes.Unauthenticated, "who"))))
	assert.Equal(t, 503, domain.HTTPStatus(grpcErr(status.Error(codes.Unavailable, "down"))))

	// An unmapped code, and a non-status error, both fall through to 500 —
	// the same as an error grpcErr never saw.
	assert.Equal(t, 500, domain.HTTPStatus(grpcErr(status.Error(codes.Internal, "oops"))))
	assert.Equal(t, 500, domain.HTTPStatus(grpcErr(errors.New("not a grpc status"))))

	assert.NoError(t, grpcErr(nil))
}

func TestCatalogGetListingPicksEnglishTranslation(t *testing.T) {
	c := &Catalog{catalog: &fakeCatalogService{
		getListing: func(ctx context.Context, in *catalogv1.GetListingRequest, opts ...grpc.CallOption) (*catalogv1.GetListingResponse, error) {
			require.Equal(t, "lst-1", in.GetListingId())
			return &catalogv1.GetListingResponse{Listing: &catalogv1.Listing{
				Id: "lst-1", ProductId: "prod-1", ArtisanId: "art-1",
				Price: &commonv1.Money{AmountPaise: 250000, CurrencyCode: "INR"},
				Translations: []*catalogv1.ListingTranslation{
					{Language: commonv1.Language_LANGUAGE_HINDI, Title: "hi-title"},
					{Language: commonv1.Language_LANGUAGE_ENGLISH, Title: "en-title"},
				},
			}}, nil
		},
	}}

	got, err := c.GetListing(context.Background(), "lst-1")
	require.NoError(t, err)
	assert.Equal(t, "en-title", got.Title)
	assert.Equal(t, int64(250000), got.Price)
	assert.Equal(t, "INR", got.Currency)
}

func TestCatalogGetListingMapsNotFound(t *testing.T) {
	c := &Catalog{catalog: &fakeCatalogService{
		getListing: func(ctx context.Context, in *catalogv1.GetListingRequest, opts ...grpc.CallOption) (*catalogv1.GetListingResponse, error) {
			return nil, status.Error(codes.NotFound, "no such listing")
		},
	}}

	_, err := c.GetListing(context.Background(), "missing")
	assert.True(t, domain.IsNotFound(err))
}

func TestCatalogGetArtisanPassesClusterIDPointerThrough(t *testing.T) {
	c := &Catalog{identity: &fakeIdentityService{
		getArtisan: func(ctx context.Context, in *identityv1.GetArtisanRequest, opts ...grpc.CallOption) (*identityv1.GetArtisanResponse, error) {
			return &identityv1.GetArtisanResponse{Artisan: &catalogv1.Artisan{
				Id: "art-1", DisplayName: "Lakshmi", ClusterId: strPtr("cluster-1"),
			}}, nil
		},
	}}

	got, err := c.GetArtisan(context.Background(), "art-1")
	require.NoError(t, err)
	assert.Equal(t, "Lakshmi", got.DisplayName)
	require.NotNil(t, got.ClusterID)
	assert.Equal(t, "cluster-1", *got.ClusterID)
}

// TestWithTimeoutForwardsBearerTokenAsOutgoingMetadata is the check for the
// bug that hid across three sessions: every client call went out on
// context.Background(), so core-svc's auth interceptor rejected all of them
// as Unauthenticated. A fake grpc client can't see that — it never goes
// through the interceptor — so this asserts directly on the metadata
// withTimeout attaches, which is the one thing a fake response can't fake.
func TestWithTimeoutForwardsBearerTokenAsOutgoingMetadata(t *testing.T) {
	var sawToken []string
	c := &Catalog{identity: &fakeIdentityService{
		getArtisan: func(ctx context.Context, in *identityv1.GetArtisanRequest, opts ...grpc.CallOption) (*identityv1.GetArtisanResponse, error) {
			if md, ok := metadata.FromOutgoingContext(ctx); ok {
				sawToken = md.Get("authorization")
			}
			return &identityv1.GetArtisanResponse{Artisan: &catalogv1.Artisan{Id: "art-1"}}, nil
		},
	}}

	ctx := auth.ContextWithToken(context.Background(), "tok-abc")
	_, err := c.GetArtisan(ctx, "art-1")
	require.NoError(t, err)
	require.Equal(t, []string{"Bearer tok-abc"}, sawToken)
}

func TestWithTimeoutSendsNoAuthorizationMetadataWhenCtxCarriesNoToken(t *testing.T) {
	var mdSeen metadata.MD
	var hadMD bool
	c := &Catalog{identity: &fakeIdentityService{
		getArtisan: func(ctx context.Context, in *identityv1.GetArtisanRequest, opts ...grpc.CallOption) (*identityv1.GetArtisanResponse, error) {
			mdSeen, hadMD = metadata.FromOutgoingContext(ctx)
			return &identityv1.GetArtisanResponse{Artisan: &catalogv1.Artisan{Id: "art-1"}}, nil
		},
	}}

	_, err := c.GetArtisan(context.Background(), "art-1")
	require.NoError(t, err)
	if hadMD {
		assert.Empty(t, mdSeen.Get("authorization"))
	}
}

func TestCatalogGetCraft(t *testing.T) {
	c := &Catalog{ontology: &fakeOntologyService{
		getCraft: func(ctx context.Context, in *catalogv1.GetCraftRequest, opts ...grpc.CallOption) (*catalogv1.GetCraftResponse, error) {
			require.Equal(t, "craft-1", in.GetCraftId())
			return &catalogv1.GetCraftResponse{Craft: &catalogv1.Craft{
				Id: "craft-1", DisplayName: "Ajrakh", GiRegistrationNo: strPtr("GI-9"),
			}}, nil
		},
	}}

	got, err := c.GetCraft(context.Background(), "craft-1")
	require.NoError(t, err)
	assert.Equal(t, "Ajrakh", got.DisplayName)
	require.NotNil(t, got.GIRegistrationNo)
	assert.Equal(t, "GI-9", *got.GIRegistrationNo)
}

func TestCatalogGetProvenanceByShortCodeReturnsTheStoredRecord(t *testing.T) {
	var sawReq *catalogv1.GetProvenanceByShortCodeRequest
	previousHash := "prev-hash"
	c := &Catalog{catalog: &fakeCatalogService{
		getProvenanceByShortCode: func(ctx context.Context, in *catalogv1.GetProvenanceByShortCodeRequest, opts ...grpc.CallOption) (*catalogv1.GetProvenanceByShortCodeResponse, error) {
			sawReq = in
			return &catalogv1.GetProvenanceByShortCodeResponse{Record: &catalogv1.SealedProvenance{
				Id:                 "prov-1",
				ListingId:          "listing-1",
				ArtisanId:          "artisan-1",
				CraftId:            "craft-1",
				ContentHash:        "hash-1",
				PreviousHash:       &previousHash,
				SignatureAlgorithm: "ed25519",
				PublicKeyId:        "key-1",
				ShortCode:          "SOMECODE1",
				TechniqueMatched:   true,
				MediaHashes:        []string{"h1", "h2"},
			}}, nil
		},
	}}

	rec, err := c.GetProvenanceByShortCode(context.Background(), "SOMECODE1")
	require.NoError(t, err)
	require.NotNil(t, sawReq)
	assert.Equal(t, "SOMECODE1", sawReq.GetShortCode())

	assert.Equal(t, "prov-1", rec.ID)
	assert.Equal(t, "listing-1", rec.ListingID)
	assert.Equal(t, "SOMECODE1", rec.ShortCode)
	assert.True(t, rec.TechniqueMatched)
	require.NotNil(t, rec.PreviousHash)
	assert.Equal(t, "prev-hash", *rec.PreviousHash)
	assert.Equal(t, []string{"h1", "h2"}, rec.MediaHashes)
}

func TestCatalogGetProvenanceByShortCodePropagatesGRPCError(t *testing.T) {
	c := &Catalog{catalog: &fakeCatalogService{
		getProvenanceByShortCode: func(ctx context.Context, in *catalogv1.GetProvenanceByShortCodeRequest, opts ...grpc.CallOption) (*catalogv1.GetProvenanceByShortCodeResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}}
	_, err := c.GetProvenanceByShortCode(context.Background(), "missing-code")
	assert.Error(t, err)
}

type fakeMediaService struct {
	catalogv1.MediaServiceClient
	requestUpload func(ctx context.Context, in *catalogv1.RequestUploadRequest, opts ...grpc.CallOption) (*catalogv1.RequestUploadResponse, error)
}

func (f *fakeMediaService) RequestUpload(ctx context.Context, in *catalogv1.RequestUploadRequest, opts ...grpc.CallOption) (*catalogv1.RequestUploadResponse, error) {
	return f.requestUpload(ctx, in, opts...)
}

func TestMediaGenerateUploadURL(t *testing.T) {
	m := &Media{media: &fakeMediaService{
		requestUpload: func(ctx context.Context, in *catalogv1.RequestUploadRequest, opts ...grpc.CallOption) (*catalogv1.RequestUploadResponse, error) {
			assert.Equal(t, "art-1", in.GetArtisanId())
			assert.Equal(t, "image/jpeg", in.GetContentType())
			return &catalogv1.RequestUploadResponse{MediaId: "media-1", UploadUrl: "https://upload"}, nil
		},
	}}

	mediaID, uploadURL, err := m.GenerateUploadURL(context.Background(), "art-1", "image/jpeg", 1024)
	require.NoError(t, err)
	assert.Equal(t, "media-1", mediaID)
	assert.Equal(t, "https://upload", uploadURL)
}

type fakeInsightService struct {
	insightv1.InsightServiceClient
	getDyingCrafts          func(ctx context.Context, in *insightv1.GetDyingCraftsRequest, opts ...grpc.CallOption) (*insightv1.GetDyingCraftsResponse, error)
	generateIncomeStatement func(ctx context.Context, in *insightv1.GenerateIncomeStatementRequest, opts ...grpc.CallOption) (*insightv1.GenerateIncomeStatementResponse, error)
}

func (f *fakeInsightService) GetDyingCrafts(ctx context.Context, in *insightv1.GetDyingCraftsRequest, opts ...grpc.CallOption) (*insightv1.GetDyingCraftsResponse, error) {
	return f.getDyingCrafts(ctx, in, opts...)
}

func (f *fakeInsightService) GenerateIncomeStatement(ctx context.Context, in *insightv1.GenerateIncomeStatementRequest, opts ...grpc.CallOption) (*insightv1.GenerateIncomeStatementResponse, error) {
	return f.generateIncomeStatement(ctx, in, opts...)
}

func TestInsightGetDyingCrafts(t *testing.T) {
	in := &Insight{insight: &fakeInsightService{
		getDyingCrafts: func(ctx context.Context, req *insightv1.GetDyingCraftsRequest, opts ...grpc.CallOption) (*insightv1.GetDyingCraftsResponse, error) {
			assert.Equal(t, int32(5), req.GetLimit())
			return &insightv1.GetDyingCraftsResponse{Rows: []*insightv1.DyingCraftRow{
				{CraftId: "craft-1", CraftName: "Kota Doria", DeclineRate: 0.4, PeakArtisans: 500, CurrentArtisans: 200},
			}}, nil
		},
	}}

	rows, err := in.GetDyingCrafts(context.Background(), 5)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Kota Doria", rows[0]["craft_name"])
	assert.Equal(t, int32(200), rows[0]["current_artisans"])
}

func TestInsightGenerateStatementRollsABareEndDateToIncludeTheWholeDay(t *testing.T) {
	// insight-svc bounds settled_at with `< period_end` (exclusive). A bare
	// "2024-01-31" end date must arrive as 2024-02-01T00:00:00Z, not midnight
	// on the 31st itself, or every order settled that day silently drops off
	// the statement.
	in := &Insight{insight: &fakeInsightService{
		generateIncomeStatement: func(ctx context.Context, req *insightv1.GenerateIncomeStatementRequest, opts ...grpc.CallOption) (*insightv1.GenerateIncomeStatementResponse, error) {
			assert.Equal(t, "2024-01-01T00:00:00Z", req.GetPeriodStart().AsTime().Format(time.RFC3339))
			assert.Equal(t, "2024-02-01T00:00:00Z", req.GetPeriodEnd().AsTime().Format(time.RFC3339))
			return &insightv1.GenerateIncomeStatementResponse{StatementId: "stmt-1"}, nil
		},
	}}

	id, err := in.GenerateStatement(context.Background(), "art-1", "2024-01-01", "2024-01-31")
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", id)
}

func TestInsightGenerateStatementTakesAnRFC3339EndExactlyAsGiven(t *testing.T) {
	in := &Insight{insight: &fakeInsightService{
		generateIncomeStatement: func(ctx context.Context, req *insightv1.GenerateIncomeStatementRequest, opts ...grpc.CallOption) (*insightv1.GenerateIncomeStatementResponse, error) {
			assert.Equal(t, "2024-01-31T23:59:59Z", req.GetPeriodEnd().AsTime().Format(time.RFC3339))
			return &insightv1.GenerateIncomeStatementResponse{StatementId: "stmt-1"}, nil
		},
	}}

	_, err := in.GenerateStatement(context.Background(), "art-1", "2024-01-01T00:00:00Z", "2024-01-31T23:59:59Z")
	require.NoError(t, err)
}

func TestInsightGenerateStatementRejectsAnUnparseableDate(t *testing.T) {
	in := &Insight{insight: &fakeInsightService{}}
	_, err := in.GenerateStatement(context.Background(), "art-1", "not-a-date", "2024-01-31")
	assert.True(t, domain.HTTPStatus(err) == 400)
}

func TestInsightGetStatementReturnsAnErrorNotAPanic(t *testing.T) {
	in := &Insight{}
	_, err := in.GetStatement(context.Background(), "stmt-1")
	assert.Error(t, err)
}

func validRegisterFields() map[string]any {
	return map[string]any{
		"display_name": "Lakshmi",
		"craft_ids":    []any{"craft-1"},
		"languages":    []any{"HINDI"},
		"region":       map[string]any{"state_code": "IN-AS"},
	}
}

func TestArtisanRegisterSendsCraftLanguageAndRegionThrough(t *testing.T) {
	a := &Artisan{identity: &fakeIdentityService{
		registerArtisan: func(ctx context.Context, in *identityv1.RegisterArtisanRequest, opts ...grpc.CallOption) (*identityv1.RegisterArtisanResponse, error) {
			require.Equal(t, "Lakshmi", in.GetDisplayName())
			require.Equal(t, "+919900011234", in.GetPhoneE164())
			require.Equal(t, []string{"craft-1"}, in.GetCraftIds())
			require.Equal(t, []commonv1.Language{commonv1.Language_LANGUAGE_HINDI}, in.GetLanguages())
			require.Equal(t, "IN-AS", in.GetRegion().GetStateCode())
			require.Equal(t, "idem-1", in.GetIdempotencyKey())
			return &identityv1.RegisterArtisanResponse{Artisan: &catalogv1.Artisan{Id: "art-1"}}, nil
		},
	}}

	id, err := a.Register(context.Background(), "+919900011234", "idem-1", validRegisterFields())
	require.NoError(t, err)
	assert.Equal(t, "art-1", id)
}

func TestArtisanRegisterRejectsMissingCraftIDs(t *testing.T) {
	a := &Artisan{}
	fields := validRegisterFields()
	delete(fields, "craft_ids")
	_, err := a.Register(context.Background(), "+919900011234", "idem-1", fields)
	assert.Equal(t, 400, domain.HTTPStatus(err))
}

func TestArtisanRegisterRejectsMissingRegion(t *testing.T) {
	a := &Artisan{}
	fields := validRegisterFields()
	delete(fields, "region")
	_, err := a.Register(context.Background(), "+919900011234", "idem-1", fields)
	assert.Equal(t, 400, domain.HTTPStatus(err))
}

func TestArtisanRegisterRejectsNonStringCraftID(t *testing.T) {
	a := &Artisan{}
	fields := validRegisterFields()
	fields["craft_ids"] = []any{42}
	_, err := a.Register(context.Background(), "+919900011234", "idem-1", fields)
	assert.Equal(t, 400, domain.HTTPStatus(err))
}

type fakeSearchService struct {
	searchv1.SearchServiceClient
	search  func(ctx context.Context, in *searchv1.SearchRequest, opts ...grpc.CallOption) (*searchv1.SearchResponse, error)
	suggest func(ctx context.Context, in *searchv1.SuggestRequest, opts ...grpc.CallOption) (*searchv1.SuggestResponse, error)
}

func (f *fakeSearchService) Search(ctx context.Context, in *searchv1.SearchRequest, opts ...grpc.CallOption) (*searchv1.SearchResponse, error) {
	return f.search(ctx, in, opts...)
}
func (f *fakeSearchService) Suggest(ctx context.Context, in *searchv1.SuggestRequest, opts ...grpc.CallOption) (*searchv1.SuggestResponse, error) {
	return f.suggest(ctx, in, opts...)
}

func TestSearchSearchAppliesCraftFilterAndMapsHits(t *testing.T) {
	s := &Search{search: &fakeSearchService{
		search: func(ctx context.Context, in *searchv1.SearchRequest, opts ...grpc.CallOption) (*searchv1.SearchResponse, error) {
			assert.Equal(t, "ajrakh", in.GetQuery())
			require.Equal(t, []string{"craft-1"}, in.GetFilters().GetCraftIds())
			return &searchv1.SearchResponse{Hits: []*searchv1.SearchHit{
				{ListingId: "lst-1", ArtisanId: "art-1", Score: 0.9},
			}}, nil
		},
	}}

	results, err := s.Search(context.Background(), "ajrakh", map[string]any{"craft_id": "craft-1"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "lst-1", results[0]["listing_id"])
}

func TestSearchSuggestReturnsTextOnly(t *testing.T) {
	s := &Search{search: &fakeSearchService{
		suggest: func(ctx context.Context, in *searchv1.SuggestRequest, opts ...grpc.CallOption) (*searchv1.SuggestResponse, error) {
			assert.Equal(t, "ajr", in.GetPrefix())
			return &searchv1.SuggestResponse{Suggestions: []*searchv1.Suggestion{{Text: "ajrakh"}, {Text: "ajrak block print"}}}, nil
		},
	}}

	got, err := s.Suggest(context.Background(), "ajr")
	require.NoError(t, err)
	assert.Equal(t, []string{"ajrakh", "ajrak block print"}, got)
}
