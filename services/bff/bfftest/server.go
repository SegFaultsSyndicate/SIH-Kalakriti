package bfftest

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/handler"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/middleware"
)

type TestServer struct {
	Server        *httptest.Server
	ChannelServer *httptest.Server
	URL           string
	Issuer        *auth.Issuer
	ArtisanToken  string
	BuyerToken    string
	AdminToken    string
}

func (ts *TestServer) Close() {
	if ts.Server != nil {
		ts.Server.Close()
	}
	if ts.ChannelServer != nil {
		ts.ChannelServer.Close()
	}
}

func Start() (*TestServer, error) {
	issuer, err := auth.NewIssuer(auth.Config{
		Secret:     "test-secret-at-least-32-bytes-long-for-jwt-signing",
		Issuer:     "test-issuer",
		AccessTTL:  time.Hour,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	if err != nil {
		return nil, err
	}

	artisanPair, err := issuer.Issue(auth.Subject{ID: "artisan-1", Role: auth.RoleArtisan})
	if err != nil {
		return nil, err
	}
	buyerPair, err := issuer.Issue(auth.Subject{ID: "buyer-1", Role: auth.RoleBuyer})
	if err != nil {
		return nil, err
	}
	adminPair, err := issuer.Issue(auth.Subject{ID: "admin-1", Role: auth.RoleMinistry})
	if err != nil {
		return nil, err
	}

	mockChannelSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))

	rdb := redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		DialTimeout:  time.Millisecond,
		ReadTimeout:  time.Millisecond,
		WriteTimeout: time.Millisecond,
		MaxRetries:   -1,
	})
	idempStore := &MockIdempStore{Responses: make(map[string][]byte)}

	srv, err := bff.NewServer(bff.Config{
		Addr:                  ":0",
		BaseURL:               "http://localhost",
		WebDist:               os.TempDir(),
		ChannelSvcAddr:        mockChannelSrv.URL,
		Logger:                slog.New(slog.NewTextHandler(io.Discard, nil)),
		Issuer:                issuer,
		Redis:                 rdb,
		IdempStore:            idempStore,
		RateLimitPerIP:        100000,
		RateLimitPerPrincipal: 100000,
		RateLimitWindow:       time.Minute,
		AuthSvc:               &StubAuthSvc{},
		ArtisanSvc:            &StubArtisanSvc{},
		MediaSvc:              &StubMediaSvc{},
		ListingSvc:            &StubListingSvc{},
		SearchSvc:             &StubSearchSvc{},
		PricingSvc:            &StubPricingSvc{},
		OrderSvc:              &StubOrderSvc{},
		FollowSvc:             &StubFollowSvc{},
		StmtSvc:               &StubStmtSvc{},
		InsightSvc:            &StubInsightSvc{},
		CatalogSvc:            &StubCatalogSvc{},
	})
	if err != nil {
		mockChannelSrv.Close()
		return nil, err
	}

	ts := httptest.NewServer(srv)
	return &TestServer{
		Server:        ts,
		ChannelServer: mockChannelSrv,
		URL:           ts.URL,
		Issuer:        issuer,
		ArtisanToken:  artisanPair.AccessToken,
		BuyerToken:    buyerPair.AccessToken,
		AdminToken:    adminPair.AccessToken,
	}, nil
}

type MockIdempStore struct {
	mu        sync.RWMutex
	Responses map[string][]byte
}

func (m *MockIdempStore) GetOrInsert(ctx context.Context, scope, key, requestHash string, expiresAt time.Time) (rec middleware.IdempotencyRecord, inserted bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stored, ok := m.Responses[scope+":"+key]
	if !ok {
		return middleware.IdempotencyRecord{RequestHash: requestHash}, true, nil
	}
	return middleware.IdempotencyRecord{RequestHash: requestHash, Response: stored}, false, nil
}

func (m *MockIdempStore) SaveResponse(ctx context.Context, scope, key string, response []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Responses[scope+":"+key] = response
	return nil
}

type StubArtisanSvc struct{}

func (s *StubArtisanSvc) Register(ctx context.Context, phone, idempotencyKey string, fields map[string]any) (string, error) {
	return "artisan-1", nil
}
func (s *StubArtisanSvc) GetProfile(ctx context.Context, artisanID string) (map[string]any, error) {
	return map[string]any{
		"id":           artisanID,
		"display_name": "Lakshmi Devi",
		"phone_e164":   "+919876543210",
		"craft_ids":    []any{"madhubani"},
		"verified":     true,
	}, nil
}
func (s *StubArtisanSvc) UpdateProfile(ctx context.Context, artisanID string, updates map[string]any) error {
	return nil
}

type StubMediaSvc struct{}

func (s *StubMediaSvc) GenerateUploadURL(ctx context.Context, artisanID, contentType string, sizeBytes int64) (string, string, error) {
	return "media-1", "http://storage/upload/media-1", nil
}
func (s *StubMediaSvc) ConfirmUpload(ctx context.Context, mediaID string) error {
	return nil
}

type StubListingSvc struct{}

func (s *StubListingSvc) CreateListing(ctx context.Context, artisanID, idempotencyKey string, listing map[string]any) (string, error) {
	return "listing-1", nil
}
func (s *StubListingSvc) UpdateListing(ctx context.Context, listingID, idempotencyKey string, updates map[string]any) error {
	return nil
}
func (s *StubListingSvc) SubmitForReview(ctx context.Context, listingID, idempotencyKey string) error {
	return nil
}
func (s *StubListingSvc) ApproveListing(ctx context.Context, listingID, reviewerID, idempotencyKey string, editedTranslations []map[string]any) error {
	return nil
}
func (s *StubListingSvc) GetListing(ctx context.Context, listingID string) (map[string]any, error) {
	return map[string]any{
		"id":     listingID,
		"status": "published",
		"state":  "PUBLISHED",
		"translations": []any{
			map[string]any{
				"language": "en",
				"title":    "Madhubani Fish Painting",
			},
		},
	}, nil
}
func (s *StubListingSvc) GetListingSummary(ctx context.Context, listingID string) (map[string]any, error) {
	return map[string]any{
		"id":    listingID,
		"state": "PUBLISHED",
		"title": "Madhubani Fish Painting",
	}, nil
}
func (s *StubListingSvc) BatchGetListingSummaries(ctx context.Context, ids []string) ([]map[string]any, error) {
	out := make([]map[string]any, len(ids))
	for i, id := range ids {
		out[i] = map[string]any{"id": id, "state": "PUBLISHED", "title": "Madhubani Fish Painting"}
	}
	return out, nil
}
func (s *StubListingSvc) ListListings(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	return []map[string]any{{"id": "listing-1", "state": "PUBLISHED"}}, nil
}
func (s *StubListingSvc) SealProvenance(ctx context.Context, listingID, idempotencyKey string, fields map[string]any) (map[string]any, error) {
	return map[string]any{
		"provenance_id": "prov-1",
		"qr_code":       "QR_CODE_PROV_123",
		"listing_id":    listingID,
	}, nil
}

type StubSearchSvc struct{}

func (s *StubSearchSvc) Search(ctx context.Context, query string, filters map[string]any) (map[string]any, error) {
	return map[string]any{
		"results": []any{
			map[string]any{
				"id":         "listing-1",
				"listing_id": "listing-1",
			},
		},
	}, nil
}
func (s *StubSearchSvc) Suggest(ctx context.Context, prefix string) ([]string, error) {
	return []string{"Madhubani fish painting"}, nil
}
func (s *StubSearchSvc) SearchVoice(ctx context.Context, audioData []byte, language string) (map[string]any, error) {
	return map[string]any{"results": []any{map[string]any{"listing_id": "listing-1"}}}, nil
}

type StubPricingSvc struct{}

func (s *StubPricingSvc) AdvisePricing(ctx context.Context, listingID string, inputs map[string]any) (map[string]any, error) {
	return map[string]any{"suggested_price": map[string]any{"amount_paise": 100000}}, nil
}

type StubOrderSvc struct{}

func (s *StubOrderSvc) CreateBulkOrder(ctx context.Context, buyerID, idempotencyKey string, fields map[string]any) (string, error) {
	return "order-1", nil
}
func (s *StubOrderSvc) GetOrder(ctx context.Context, orderID string) (map[string]any, error) {
	return map[string]any{
		"id":          orderID,
		"total_paise": int64(200000),
		"lots": []any{
			map[string]any{"id": "lot-1", "artisan_id": "artisan-1", "quantity": 200, "state": "IN_PRODUCTION"},
			map[string]any{"id": "lot-2", "artisan_id": "artisan-2", "quantity": 200, "state": "IN_PRODUCTION"},
			map[string]any{"id": "lot-3", "artisan_id": "artisan-3", "quantity": 100, "state": "IN_PRODUCTION"},
		},
	}, nil
}
func (s *StubOrderSvc) RespondToLot(ctx context.Context, lotID, artisanID, idempotencyKey string, accept bool, fields map[string]any) error {
	return nil
}
func (s *StubOrderSvc) ReportProgress(ctx context.Context, lotID, artisanID, idempotencyKey string, fields map[string]any) (map[string]any, error) {
	return map[string]any{"id": lotID, "status": "completed"}, nil
}
func (s *StubOrderSvc) RequestReallocation(ctx context.Context, lotID, artisanID, idempotencyKey string, fields map[string]any) (map[string]any, error) {
	return map[string]any{"id": lotID, "reallocated": true}, nil
}
func (s *StubOrderSvc) WatchOrder(ctx context.Context, orderID string, since *time.Time) (<-chan map[string]any, error) {
	ch := make(chan map[string]any)
	close(ch)
	return ch, nil
}

type StubFollowSvc struct{}

func (s *StubFollowSvc) FollowArtisan(ctx context.Context, followerID, artisanID string) error {
	return nil
}
func (s *StubFollowSvc) UnfollowArtisan(ctx context.Context, followerID, artisanID string) error {
	return nil
}
func (s *StubFollowSvc) GetFeed(ctx context.Context, userID string, limit, offset int32) ([]map[string]any, error) {
	return nil, nil
}
func (s *StubFollowSvc) MarkFeedItemRead(ctx context.Context, notificationID, userID string) error {
	return nil
}
func (s *StubFollowSvc) GetFollowerCount(ctx context.Context, artisanID string) (int32, error) {
	return 10, nil
}

type StubStmtSvc struct{}

func (s *StubStmtSvc) GenerateStatement(ctx context.Context, artisanID string, start, end string) (map[string]any, error) {
	return map[string]any{
		"statement_id":     "stmt-1",
		"short_code":       "QR_CODE_INCOME_123",
		"download_url":     "https://minio/statements/stmt.pdf",
		"verification_url": "http://localhost/v/QR_CODE_INCOME_123",
	}, nil
}
func (s *StubStmtSvc) GetStatement(ctx context.Context, statementID string) (map[string]any, error) {
	return map[string]any{
		"statement_id": statementID,
		"short_code":   "QR_CODE_INCOME_123",
	}, nil
}
func (s *StubStmtSvc) ListIncomeStatements(ctx context.Context, artisanID string, limit, offset int32) ([]map[string]any, error) {
	return []map[string]any{
		{
			"statement_id": "stmt-1",
			"short_code":   "QR_CODE_INCOME_123",
		},
	}, nil
}

type StubInsightSvc struct{}

func (s *StubInsightSvc) GetArtisansByCategory(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	return nil, nil
}
func (s *StubInsightSvc) GetListingsByCraftMonth(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	return nil, nil
}
func (s *StubInsightSvc) GetEarningsByDistrict(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	return nil, nil
}
func (s *StubInsightSvc) GetIncomeComparison(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	return nil, nil
}
func (s *StubInsightSvc) GetDyingCrafts(ctx context.Context, limit int32) ([]map[string]any, error) {
	return nil, nil
}
func (s *StubInsightSvc) RefreshMaterializedViews(ctx context.Context) (map[string]any, error) {
	return nil, nil
}

type StubCatalogSvc struct {
	handler.CatalogService
}

func (s *StubCatalogSvc) ListCrafts(ctx context.Context) ([]handler.Craft, error) {
	return nil, nil
}
func (s *StubCatalogSvc) GetCraft(ctx context.Context, craftID string) (handler.Craft, error) {
	return handler.Craft{ID: craftID, Slug: "madhubani", DisplayName: "Madhubani"}, nil
}
func (s *StubCatalogSvc) GetCraftBySlug(ctx context.Context, slug string) (*handler.Craft, error) {
	return &handler.Craft{ID: "craft-1", Slug: slug, DisplayName: "Madhubani"}, nil
}
func (s *StubCatalogSvc) GetArtisanStorefront(ctx context.Context, artisanID string) (map[string]any, error) {
	return nil, nil
}
func (s *StubCatalogSvc) ListProcessClips(ctx context.Context, limit int32) ([]handler.ProcessClip, error) {
	return nil, nil
}
func (s *StubCatalogSvc) RefreshCraftIndex(ctx context.Context, idempotencyKey string) (map[string]any, error) {
	return nil, nil
}
func (s *StubCatalogSvc) CreateCluster(ctx context.Context, name, stateCode string, district, coordinatorPhone *string, idempotencyKey string) (map[string]any, error) {
	return map[string]any{"cluster_id": "cluster-1", "name": name, "state_code": stateCode}, nil
}
func (s *StubCatalogSvc) GetCluster(ctx context.Context, clusterID string) (map[string]any, error) {
	return map[string]any{"id": clusterID, "name": "Test Cluster"}, nil
}
func (s *StubCatalogSvc) ListClusterMembers(ctx context.Context, clusterID string) ([]map[string]any, error) {
	return []map[string]any{{"artisan_id": "artisan-1", "role": "MEMBER"}}, nil
}
func (s *StubCatalogSvc) AddClusterMember(ctx context.Context, clusterID, artisanID, role, idempotencyKey string) (map[string]any, error) {
	return map[string]any{"cluster_id": clusterID, "artisan_id": artisanID, "role": role}, nil
}
func (s *StubCatalogSvc) RemoveClusterMember(ctx context.Context, clusterID, artisanID, idempotencyKey string) (bool, error) {
	return true, nil
}
func (s *StubCatalogSvc) CreateSelfHelpGroup(ctx context.Context, name, registrationNo string, clusterID *string, members []map[string]any, idempotencyKey string) (map[string]any, error) {
	return map[string]any{"shg_id": "shg-1", "name": name}, nil
}
func (s *StubCatalogSvc) GetSelfHelpGroup(ctx context.Context, shgID string) (map[string]any, error) {
	return map[string]any{"id": shgID, "name": "Test SHG"}, nil
}
func (s *StubCatalogSvc) SetSelfHelpGroupMembers(ctx context.Context, shgID string, members []map[string]any, idempotencyKey string) (map[string]any, error) {
	return map[string]any{"shg_id": shgID, "members": members}, nil
}
func (s *StubCatalogSvc) SuspendListing(ctx context.Context, listingID, reason, idempotencyKey string) (handler.Listing, error) {
	return handler.Listing{ID: listingID, Title: "Suspended Item"}, nil
}
func (s *StubCatalogSvc) ReinstateListing(ctx context.Context, listingID, idempotencyKey string) (handler.Listing, error) {
	return handler.Listing{ID: listingID, Title: "Reinstated Item"}, nil
}
func (s *StubCatalogSvc) GetListingBySlug(ctx context.Context, slug string) (*handler.ListingDetail, error) {
	return &handler.ListingDetail{ID: "listing-1", Title: "Madhubani Painting"}, nil
}
func (s *StubCatalogSvc) GetArtisanBySlug(ctx context.Context, slug string) (*handler.ArtisanProfile, error) {
	return &handler.ArtisanProfile{ID: "artisan-1", DisplayName: "Lakshmi Devi"}, nil
}
func (s *StubCatalogSvc) ListPublishedListings(ctx context.Context, limit, offset int32) ([]handler.ListingDetail, error) {
	return []handler.ListingDetail{{ID: "listing-1"}}, nil
}
func (s *StubCatalogSvc) GetProvenanceByShortCode(ctx context.Context, code string) (handler.ProvenanceRecord, error) {
	return handler.ProvenanceRecord{ID: "prov-1", ShortCode: code}, nil
}
func (s *StubCatalogSvc) GetListing(ctx context.Context, listingID string) (handler.Listing, error) {
	return handler.Listing{ID: listingID, Title: "Madhubani Painting"}, nil
}
func (s *StubCatalogSvc) GetArtisan(ctx context.Context, artisanID string) (handler.Artisan, error) {
	return handler.Artisan{ID: artisanID, DisplayName: "Lakshmi Devi"}, nil
}
func (s *StubCatalogSvc) ListArtisansByCraft(ctx context.Context, craftID string, limit int32) ([]handler.Artisan, error) {
	return []handler.Artisan{{ID: "artisan-1"}}, nil
}

type StubAuthSvc struct{}

func (s *StubAuthSvc) RequestOTP(ctx context.Context, phone string) error {
	return nil
}
func (s *StubAuthSvc) VerifyOTP(ctx context.Context, phone, otp, devRole string) (string, string, error) {
	return "mock-access-token", "mock-refresh-token", nil
}
func (s *StubAuthSvc) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	return "mock-access-token", nil
}
