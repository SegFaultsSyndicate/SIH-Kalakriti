// services/channel-svc/internal/channel/consumer/ondc_publisher_test.go
package consumer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/ondc"
)

const testONDCPrivateKeyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeCatalog is a test double for CatalogClient.
type fakeCatalog struct {
	byID map[string]ondc.Listing
}

func (f *fakeCatalog) GetListingForONDC(_ context.Context, listingID string) (ondc.Listing, error) {
	l, ok := f.byID[listingID]
	if !ok {
		return ondc.Listing{}, errors.New("listing not found")
	}
	return l, nil
}

func newDryRunONDCClient(t *testing.T) *ondc.Client {
	t.Helper()
	adapter, err := ondc.NewAdapter(ondc.Config{
		PrivateKeyHex: testONDCPrivateKeyHex,
		KeyID:         "test-key-1",
		SubscriberID:  "kalakriti.in",
		SubscriberURL: "https://api.kalakriti.in",
		DryRun:        true,
	})
	require.NoError(t, err)
	return ondc.NewClient(adapter, "https://gateway.example.com", testLogger())
}

func TestONDCPublisherHandlePublishesTheListing(t *testing.T) {
	catalog := &fakeCatalog{byID: map[string]ondc.Listing{
		"lst-1": {ID: "lst-1", Title: "Handwoven Saree", PriceCents: 150000, Currency: "INR"},
	}}
	p := NewONDCPublisher(catalog, newDryRunONDCClient(t), testLogger())

	err := p.Handle(context.Background(), []byte(`{"payload":{"listing_id":"lst-1"}}`))
	assert.NoError(t, err)
}

func TestONDCPublisherHandleRejectsMissingListingID(t *testing.T) {
	p := NewONDCPublisher(&fakeCatalog{}, newDryRunONDCClient(t), testLogger())

	err := p.Handle(context.Background(), []byte(`{"payload":{}}`))
	assert.Error(t, err)
}

func TestONDCPublisherHandlePropagatesCatalogFetchError(t *testing.T) {
	p := NewONDCPublisher(&fakeCatalog{byID: map[string]ondc.Listing{}}, newDryRunONDCClient(t), testLogger())

	err := p.Handle(context.Background(), []byte(`{"payload":{"listing_id":"unknown"}}`))
	assert.Error(t, err)
}

func TestONDCPublisherHandleRejectsMalformedJSON(t *testing.T) {
	p := NewONDCPublisher(&fakeCatalog{}, newDryRunONDCClient(t), testLogger())

	err := p.Handle(context.Background(), []byte(`not json`))
	assert.Error(t, err)
}
