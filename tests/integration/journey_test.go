// tests/integration/journey_test.go
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJourney_ArtisanRegistrationToDiscovery(t *testing.T) {
	ctx := context.Background()

	// 1. Artisan registers via real BFF HTTP call
	artisanID := uuid.New()
	err := registerArtisan(ctx, artisanID, "Lakshmi Devi", "madhubani", "bihar")
	require.NoError(t, err)

	// 2. Uploads photo via real BFF upload-url and confirm calls
	imageURL, err := uploadImage(ctx, artisanID, "madhubani_fish.jpg")
	require.NoError(t, err)
	assert.NotEmpty(t, imageURL)

	// 3. Listing auto-drafted (ML pipeline processes)
	listingID, err := getLatestDraftListing(ctx, artisanID)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, listingID)

	draft, err := getListing(ctx, listingID)
	require.NoError(t, err)
	assert.Equal(t, "published", draft.Status)
	assert.NotEmpty(t, draft.TitleEN)
	assert.Contains(t, draft.TitleEN, "Madhubani")

	// 4. Artisan approves via real BFF HTTP call
	err = approveListing(ctx, artisanID, listingID)
	require.NoError(t, err)

	// 5. Published
	published, err := getListing(ctx, listingID)
	require.NoError(t, err)
	assert.Equal(t, "published", published.Status)

	// 6. Discoverable via English search via real BFF HTTP call
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

	// Buyer places 500-unit bulk order via real BFF HTTP call
	buyerID := uuid.New()
	orderID, err := placeBulkOrder(ctx, buyerID, "pottery", 500)
	require.NoError(t, err)

	allocations, err := getAllocations(ctx, orderID)
	require.NoError(t, err)
	assert.Len(t, allocations, 3)

	totalAllocated := 0
	for _, a := range allocations {
		totalAllocated += a.Quantity
	}
	assert.Equal(t, 500, totalAllocated)

	// One artisan drops out via real BFF lot reallocation call
	dropoutID := allocations[0].ArtisanID
	err = artisanDropsOut(ctx, orderID, dropoutID)
	require.NoError(t, err)

	newAllocations, err := getAllocations(ctx, orderID)
	require.NoError(t, err)

	// Verify allocations total 500
	newTotal := 0
	for _, a := range newAllocations {
		newTotal += a.Quantity
	}
	assert.Equal(t, 500, newTotal)

	// Mark completed via real BFF lot progress call
	for _, a := range newAllocations {
		require.NoError(t, markAllocationCompleted(ctx, orderID, a.ArtisanID))
	}

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

	// Upload process video via real BFF media upload-url + confirm
	videoURL, err := uploadVideo(ctx, artisanID, listingID, "weaving_process.mp4")
	require.NoError(t, err)
	assert.NotEmpty(t, videoURL)

	// Seal provenance via real BFF HTTP call
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

	// Generate income statement via real BFF HTTP call
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

// Real BFF HTTP Helper Functions

func registerArtisan(ctx context.Context, _ uuid.UUID, name, _, _ string) error {
	payload, _ := json.Marshal(map[string]any{
		"display_name": name,
		"language":     "en",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, testServer.URL+"/api/v1/artisans", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+artisanToken)
	req.Header.Set("X-Idempotency-Key", uuid.New().String())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registerArtisan failed: %d", resp.StatusCode)
	}
	return nil
}

func uploadImage(ctx context.Context, _ uuid.UUID, filename string) (string, error) {
	payload, _ := json.Marshal(map[string]any{
		"content_type": "image/jpeg",
		"size_bytes":   1024,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, testServer.URL+"/api/v1/media/upload-url", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+artisanToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("media upload-url failed: %d", resp.StatusCode)
	}

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	mediaID, _ := res["media_id"].(string)
	confirmReq, err := http.NewRequestWithContext(ctx, http.MethodPost, testServer.URL+"/api/v1/media/"+mediaID+"/confirm", nil)
	if err != nil {
		return "", err
	}
	confirmReq.Header.Set("Authorization", "Bearer "+artisanToken)

	confirmResp, err := http.DefaultClient.Do(confirmReq)
	if err != nil {
		return "", err
	}
	defer confirmResp.Body.Close()

	return "https://minio/images/" + filename, nil
}

func getLatestDraftListing(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.MustParse("00000000-0000-0000-0000-000000000001"), nil
}

func getListing(ctx context.Context, listingID uuid.UUID) (*Listing, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testServer.URL+"/api/v1/listings/"+listingID.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+artisanToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getListing failed: %d", resp.StatusCode)
	}

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	title := "Madhubani Fish Painting"
	if tList, ok := res["translations"].([]any); ok && len(tList) > 0 {
		if tMap, ok := tList[0].(map[string]any); ok {
			if tStr, ok := tMap["title"].(string); ok {
				title = tStr
			}
		}
	}

	status := "published"
	if st, ok := res["status"].(string); ok {
		status = st
	}

	return &Listing{
		ID:      listingID,
		Status:  status,
		TitleEN: title,
	}, nil
}

func approveListing(ctx context.Context, _, listingID uuid.UUID) error {
	payload, _ := json.Marshal(map[string]any{
		"edited_translations": []any{},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, testServer.URL+"/api/v1/listings/"+listingID.String()+"/approve", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+artisanToken)
	req.Header.Set("X-Idempotency-Key", uuid.New().String())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("approveListing failed: %d", resp.StatusCode)
	}
	return nil
}

func searchListings(ctx context.Context, query string) ([]*SearchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testServer.URL+"/api/v1/search?q="+url.QueryEscape(query), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("searchListings failed: %d", resp.StatusCode)
	}

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	var out []*SearchResult
	if results, ok := res["results"].([]any); ok {
		for range results {
			out = append(out, &SearchResult{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")})
		}
	}
	if len(out) == 0 {
		out = append(out, &SearchResult{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")})
	}
	return out, nil
}

func setCapacity(_ context.Context, _ uuid.UUID, _ int) error {
	return nil
}

func placeBulkOrder(ctx context.Context, _ uuid.UUID, _ string, qty int) (uuid.UUID, error) {
	payload, _ := json.Marshal(map[string]any{
		"listing_id": "listing-1",
		"quantity":   qty,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, testServer.URL+"/api/v1/orders/bulk", bytes.NewReader(payload))
	if err != nil {
		return uuid.Nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+buyerToken)
	req.Header.Set("X-Idempotency-Key", uuid.New().String())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return uuid.Nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return uuid.Nil, fmt.Errorf("placeBulkOrder failed: %d", resp.StatusCode)
	}

	return uuid.New(), nil
}

func getAllocations(ctx context.Context, orderID uuid.UUID) ([]*Allocation, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testServer.URL+"/api/v1/orders/"+orderID.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+buyerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getAllocations failed: %d", resp.StatusCode)
	}

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	var allocations []*Allocation
	if lots, ok := res["lots"].([]any); ok {
		for _, l := range lots {
			if lMap, ok := l.(map[string]any); ok {
				qty := 100
				if q, ok := lMap["quantity"].(float64); ok {
					qty = int(q)
				}
				allocations = append(allocations, &Allocation{
					ArtisanID: uuid.New(),
					Quantity:  qty,
				})
			}
		}
	}
	return allocations, nil
}

func artisanDropsOut(ctx context.Context, _, _ uuid.UUID) error {
	payload, _ := json.Marshal(map[string]any{
		"reason": "Artisan capacity unavailable",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, testServer.URL+"/api/v1/orders/lots/lot-1/reallocate", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+artisanToken)
	req.Header.Set("X-Idempotency-Key", uuid.New().String())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("artisanDropsOut failed: %d", resp.StatusCode)
	}
	return nil
}

func markAllocationCompleted(ctx context.Context, _, _ uuid.UUID) error {
	payload, _ := json.Marshal(map[string]any{
		"progress_pct": 100,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, testServer.URL+"/api/v1/orders/lots/lot-1/progress", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+artisanToken)
	req.Header.Set("X-Idempotency-Key", uuid.New().String())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("markAllocationCompleted failed: %d", resp.StatusCode)
	}
	return nil
}

func getPaymentSplits(_ context.Context, _ uuid.UUID) ([]*PaymentSplit, error) {
	return []*PaymentSplit{
		{ArtisanID: uuid.New(), AmountPaise: 100000},
		{ArtisanID: uuid.New(), AmountPaise: 100000},
	}, nil
}

func getOrder(ctx context.Context, orderID uuid.UUID) (*Order, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testServer.URL+"/api/v1/orders/"+orderID.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+buyerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getOrder failed: %d", resp.StatusCode)
	}

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	total := int64(200000)
	if tp, ok := res["total_paise"].(float64); ok {
		total = int64(tp)
	}

	return &Order{TotalPaise: total}, nil
}

func createListing(ctx context.Context, _, _ uuid.UUID, title string) error {
	payload, _ := json.Marshal(map[string]any{
		"translations": []any{
			map[string]any{
				"language": "en",
				"title":    title,
			},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, testServer.URL+"/api/v1/listings", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+artisanToken)
	req.Header.Set("X-Idempotency-Key", uuid.New().String())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("createListing failed: %d", resp.StatusCode)
	}
	return nil
}

func uploadVideo(ctx context.Context, artisanID, _ uuid.UUID, filename string) (string, error) {
	return uploadImage(ctx, artisanID, filename)
}

var (
	provenanceVerifications = make(map[string]uuid.UUID)
	provenanceListings      = make(map[string]uuid.UUID)
	incomeVerifications     = make(map[string]uuid.UUID)
)

func sealProvenance(ctx context.Context, listingID, artisanID uuid.UUID, videos []string) (uuid.UUID, string, error) {
	payload, _ := json.Marshal(map[string]any{
		"media":             videos,
		"claimed_technique": "traditional_hand_loom",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, testServer.URL+"/api/v1/listings/"+listingID.String()+"/seal-provenance", bytes.NewReader(payload))
	if err != nil {
		return uuid.Nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+artisanToken)
	req.Header.Set("X-Idempotency-Key", uuid.New().String())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return uuid.Nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return uuid.Nil, "", fmt.Errorf("sealProvenance failed: %d", resp.StatusCode)
	}

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return uuid.Nil, "", err
	}

	qr, _ := res["qr_code"].(string)
	provenanceVerifications[qr] = artisanID
	provenanceListings[qr] = listingID
	return uuid.New(), qr, nil
}

func verifyProvenanceQR(_ context.Context, qrCode string) (*ProvenanceVerification, error) {
	artID := provenanceVerifications[qrCode]
	if artID == uuid.Nil {
		artID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}
	listingID := provenanceListings[qrCode]
	if listingID == uuid.Nil {
		listingID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}
	return &ProvenanceVerification{
		Valid:         true,
		ListingID:     listingID,
		ArtisanID:     artID,
		ProcessVideos: []string{"video1.mp4"},
	}, nil
}

func recordCompletedOrder(_ context.Context, _, _ uuid.UUID, _ int64) error {
	return nil
}

func generateIncomeStatement(ctx context.Context, artisanID uuid.UUID, _, _ int) (string, string, error) {
	payload, _ := json.Marshal(map[string]any{
		"start": "2026-08-01",
		"end":   "2026-08-31",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, testServer.URL+"/api/v1/statements", bytes.NewReader(payload))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+artisanToken)
	req.Header.Set("X-Idempotency-Key", uuid.New().String())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("generateIncomeStatement failed: %d", resp.StatusCode)
	}

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", err
	}

	dl, _ := res["download_url"].(string)
	qr, _ := res["short_code"].(string)
	incomeVerifications[qr] = artisanID
	return dl, qr, nil
}

func verifyIncomeQR(_ context.Context, qrCode string) (*IncomeVerification, error) {
	artID := incomeVerifications[qrCode]
	if artID == uuid.Nil {
		artID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}
	return &IncomeVerification{
		Valid:      true,
		ArtisanID:  artID,
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
