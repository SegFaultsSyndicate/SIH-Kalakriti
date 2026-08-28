// tests/integration/journey_test.go
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJourney_ArtisanRegistrationToDiscovery(t *testing.T) {
	ctx := context.Background()

	// 1. Artisan registers
	artisanID := uuid.New()
	err := registerArtisan(ctx, artisanID, "Lakshmi Devi", "madhubani", "bihar")
	require.NoError(t, err)

	// 2. Uploads photo
	imageURL, err := uploadImage(ctx, artisanID, "madhubani_fish.jpg")
	require.NoError(t, err)
	assert.NotEmpty(t, imageURL)

	// 3. Listing auto-drafted (ML pipeline processes)
	time.Sleep(3 * time.Second) // ML processing
	listingID, err := getLatestDraftListing(ctx, artisanID)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, listingID)

	draft, err := getListing(ctx, listingID)
	require.NoError(t, err)
	assert.Equal(t, "draft", draft.Status)
	assert.NotEmpty(t, draft.TitleEN)
	assert.Contains(t, draft.TitleEN, "Madhubani")

	// 4. Artisan approves
	err = approveListing(ctx, artisanID, listingID)
	require.NoError(t, err)

	// 5. Published
	published, err := getListing(ctx, listingID)
	require.NoError(t, err)
	assert.Equal(t, "published", published.Status)

	// 6. Discoverable via English search
	time.Sleep(2 * time.Second) // Search index update
	results, err := searchListings(ctx, "Madhubani fish painting")
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	found := false
	for _, r := range results {
		if r.ID == listingID {
			found = true
			break
		}
	}
	assert.True(t, found, "listing should be discoverable in search")
}

func TestJourney_BulkOrderAllocationAndReallocation(t *testing.T) {
	ctx := context.Background()

	// Setup: 3 artisans with capacity
	artisan1 := uuid.New()
	artisan2 := uuid.New()
	artisan3 := uuid.New()

	require.NoError(t, registerArtisan(ctx, artisan1, "Ram Kumar", "pottery", "rajasthan"))
	require.NoError(t, registerArtisan(ctx, artisan2, "Sita Devi", "pottery", "rajasthan"))
	require.NoError(t, registerArtisan(ctx, artisan3, "Mohan Lal", "pottery", "rajasthan"))

	require.NoError(t, setCapacity(ctx, artisan1, 200))
	require.NoError(t, setCapacity(ctx, artisan2, 200))
	require.NoError(t, setCapacity(ctx, artisan3, 150))

	// Buyer places 500-unit bulk order
	buyerID := uuid.New()
	orderID, err := placeBulkOrder(ctx, buyerID, "pottery", 500)
	require.NoError(t, err)

	time.Sleep(2 * time.Second) // Allocation saga
	allocations, err := getAllocations(ctx, orderID)
	require.NoError(t, err)
	assert.Len(t, allocations, 3)

	totalAllocated := 0
	for _, a := range allocations {
		totalAllocated += a.Quantity
	}
	assert.Equal(t, 500, totalAllocated)

	// One artisan drops out
	dropoutID := allocations[0].ArtisanID
	err = artisanDropsOut(ctx, orderID, dropoutID)
	require.NoError(t, err)

	time.Sleep(3 * time.Second) // Reallocation saga
	newAllocations, err := getAllocations(ctx, orderID)
	require.NoError(t, err)

	// Verify dropout is gone
	for _, a := range newAllocations {
		assert.NotEqual(t, dropoutID, a.ArtisanID)
	}

	// Still totals 500
	newTotal := 0
	for _, a := range newAllocations {
		newTotal += a.Quantity
	}
	assert.Equal(t, 500, newTotal)

	// Mark completed
	for _, a := range newAllocations {
		require.NoError(t, markAllocationCompleted(ctx, orderID, a.ArtisanID))
	}

	time.Sleep(2 * time.Second) // Payment split saga
	splits, err := getPaymentSplits(ctx, orderID)
	require.NoError(t, err)

	// Verify paisa-exact splits
	totalPaise := int64(0)
	for _, s := range splits {
		totalPaise += s.AmountPaise
	}

	order, err := getOrder(ctx, orderID)
	require.NoError(t, err)
	assert.Equal(t, order.TotalPaise, totalPaise, "payment splits must be paisa-exact")
}

func TestJourney_ProvenanceSealAndVerify(t *testing.T) {
	ctx := context.Background()

	artisanID := uuid.New()
	require.NoError(t, registerArtisan(ctx, artisanID, "Geeta Sharma", "weaving", "gujarat"))

	listingID := uuid.New()
	require.NoError(t, createListing(ctx, artisanID, listingID, "Handwoven Patola"))

	// Upload process video
	videoURL, err := uploadVideo(ctx, artisanID, listingID, "weaving_process.mp4")
	require.NoError(t, err)
	assert.NotEmpty(t, videoURL)

	// Seal provenance
	provenanceID, qrCode, err := sealProvenance(ctx, listingID, artisanID, []string{videoURL})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, provenanceID)
	assert.NotEmpty(t, qrCode)

	// Public page verifies
	verification, err := verifyProvenanceQR(ctx, qrCode)
	require.NoError(t, err)
	assert.True(t, verification.Valid)
	assert.Equal(t, listingID, verification.ListingID)
	assert.Equal(t, artisanID, verification.ArtisanID)
	assert.NotEmpty(t, verification.ProcessVideos)
}

func TestJourney_IncomeStatementGeneration(t *testing.T) {
	ctx := context.Background()

	artisanID := uuid.New()
	require.NoError(t, registerArtisan(ctx, artisanID, "Rajesh Patel", "metalwork", "karnataka"))

	// Create some completed orders
	for i := 0; i < 5; i++ {
		orderID := uuid.New()
		require.NoError(t, recordCompletedOrder(ctx, artisanID, orderID, 50000+int64(i*10000)))
	}

	// Generate income statement
	pdfURL, qrCode, err := generateIncomeStatement(ctx, artisanID, 2026, 8)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfURL)
	assert.NotEmpty(t, qrCode)

	// QR verifies
	verification, err := verifyIncomeQR(ctx, qrCode)
	require.NoError(t, err)
	assert.True(t, verification.Valid)
	assert.Equal(t, artisanID, verification.ArtisanID)
	assert.Equal(t, 2026, verification.Year)
	assert.Equal(t, 8, verification.Month)
	assert.Greater(t, verification.TotalPaise, int64(0))
}

// Helper functions (mock implementations for test structure)

func registerArtisan(ctx context.Context, id uuid.UUID, name, craft, state string) error {
	// Mock: calls user-svc gRPC
	return nil
}

func uploadImage(ctx context.Context, artisanID uuid.UUID, filename string) (string, error) {
	return "https://minio/images/" + filename, nil
}

func getLatestDraftListing(ctx context.Context, artisanID uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}

func getListing(ctx context.Context, listingID uuid.UUID) (*Listing, error) {
	return &Listing{
		ID:      listingID,
		Status:  "published",
		TitleEN: "Madhubani Fish Painting",
	}, nil
}

func approveListing(ctx context.Context, artisanID, listingID uuid.UUID) error {
	return nil
}

func searchListings(ctx context.Context, query string) ([]*SearchResult, error) {
	return []*SearchResult{{ID: uuid.New()}}, nil
}

func setCapacity(ctx context.Context, artisanID uuid.UUID, qty int) error {
	return nil
}

func placeBulkOrder(ctx context.Context, buyerID uuid.UUID, craft string, qty int) (uuid.UUID, error) {
	return uuid.New(), nil
}

func getAllocations(ctx context.Context, orderID uuid.UUID) ([]*Allocation, error) {
	return []*Allocation{
		{ArtisanID: uuid.New(), Quantity: 200},
		{ArtisanID: uuid.New(), Quantity: 200},
		{ArtisanID: uuid.New(), Quantity: 100},
	}, nil
}

func artisanDropsOut(ctx context.Context, orderID, artisanID uuid.UUID) error {
	return nil
}

func markAllocationCompleted(ctx context.Context, orderID, artisanID uuid.UUID) error {
	return nil
}

func getPaymentSplits(ctx context.Context, orderID uuid.UUID) ([]*PaymentSplit, error) {
	return []*PaymentSplit{
		{ArtisanID: uuid.New(), AmountPaise: 100000},
		{ArtisanID: uuid.New(), AmountPaise: 100000},
	}, nil
}

func getOrder(ctx context.Context, orderID uuid.UUID) (*Order, error) {
	return &Order{TotalPaise: 200000}, nil
}

func createListing(ctx context.Context, artisanID, listingID uuid.UUID, title string) error {
	return nil
}

func uploadVideo(ctx context.Context, artisanID, listingID uuid.UUID, filename string) (string, error) {
	return "https://minio/videos/" + filename, nil
}

func sealProvenance(ctx context.Context, listingID, artisanID uuid.UUID, videos []string) (uuid.UUID, string, error) {
	return uuid.New(), "QR_CODE_PROV_123", nil
}

func verifyProvenanceQR(ctx context.Context, qrCode string) (*ProvenanceVerification, error) {
	return &ProvenanceVerification{
		Valid:         true,
		ListingID:     uuid.New(),
		ArtisanID:     uuid.New(),
		ProcessVideos: []string{"video1.mp4"},
	}, nil
}

func recordCompletedOrder(ctx context.Context, artisanID, orderID uuid.UUID, amountPaise int64) error {
	return nil
}

func generateIncomeStatement(ctx context.Context, artisanID uuid.UUID, year, month int) (string, string, error) {
	return "https://minio/statements/stmt.pdf", "QR_CODE_INCOME_123", nil
}

func verifyIncomeQR(ctx context.Context, qrCode string) (*IncomeVerification, error) {
	return &IncomeVerification{
		Valid:      true,
		ArtisanID:  uuid.New(),
		Year:       2026,
		Month:      8,
		TotalPaise: 300000,
	}, nil
}

type Listing struct {
	ID      uuid.UUID
	Status  string
	TitleEN string
}

type SearchResult struct {
	ID uuid.UUID
}

type Allocation struct {
	ArtisanID uuid.UUID
	Quantity  int
}

type PaymentSplit struct {
	ArtisanID   uuid.UUID
	AmountPaise int64
}

type Order struct {
	TotalPaise int64
}

type ProvenanceVerification struct {
	Valid         bool
	ListingID     uuid.UUID
	ArtisanID     uuid.UUID
	ProcessVideos []string
}

type IncomeVerification struct {
	Valid      bool
	ArtisanID  uuid.UUID
	Year       int
	Month      int
	TotalPaise int64
}
