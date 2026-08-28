// services/channel-svc/internal/channel/ondc/adapter_test.go

package ondc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOnSearch(t *testing.T) {
	adapter, err := NewAdapter(Config{
		PrivateKeyHex:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		KeyID:          "test-key-1",
		SubscriberID:   "kalakriti.in",
		SubscriberURL:  "https://api.kalakriti.in",
		DryRun:         true,
	})
	require.NoError(t, err)

	listings := []Listing{
		{
			ID:          "lst-001",
			Title:       "Handwoven Silk Saree",
			Description: "Pure silk saree from Varanasi",
			PriceCents:  1500000, // ₹15,000
			Currency:    "INR",
			CategoryID:  "fashion-apparel",
			ImageURLs:   []string{"https://example.com/img1.jpg"},
		},
	}

	payload, err := adapter.BuildOnSearch(context.Background(), listings)
	require.NoError(t, err)

	assert.Equal(t, "retail", payload.Context.Domain)
	assert.Equal(t, "on_search", payload.Context.Action)
	assert.Equal(t, "kalakriti.in", payload.Context.BppID)
	assert.Len(t, payload.Message.Catalog.Providers, 1)
	assert.Len(t, payload.Message.Catalog.Providers[0].Items, 1)

	item := payload.Message.Catalog.Providers[0].Items[0]
	assert.Equal(t, "lst-001", item.ID)
	assert.Equal(t, "Handwoven Silk Saree", item.Descriptor.Name)
	assert.Equal(t, "INR", item.Price.Currency)
	assert.Equal(t, "15000.00", item.Price.Value)
}

func TestOnSearchPayloadStructure(t *testing.T) {
	adapter, err := NewAdapter(Config{
		PrivateKeyHex:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		KeyID:          "test-key-1",
		SubscriberID:   "kalakriti.in",
		SubscriberURL:  "https://api.kalakriti.in",
		DryRun:         true,
	})
	require.NoError(t, err)

	listings := []Listing{
		{ID: "lst-001", Title: "Test Product", PriceCents: 100000, Currency: "INR", CategoryID: "test"},
	}

	payload, err := adapter.BuildOnSearch(context.Background(), listings)
	require.NoError(t, err)

	// Validate that it marshals to valid JSON.
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	// Validate schema structure.
	var decoded map[string]any
	err = json.Unmarshal(payloadBytes, &decoded)
	require.NoError(t, err)

	// Check required top-level fields.
	assert.Contains(t, decoded, "context")
	assert.Contains(t, decoded, "message")

	context := decoded["context"].(map[string]any)
	assert.Equal(t, "retail", context["domain"])
	assert.Equal(t, "on_search", context["action"])
	assert.Contains(t, context, "bpp_id")
	assert.Contains(t, context, "bpp_uri")
	assert.Contains(t, context, "transaction_id")
	assert.Contains(t, context, "message_id")
	assert.Contains(t, context, "timestamp")

	message := decoded["message"].(map[string]any)
	assert.Contains(t, message, "catalog")

	catalog := message["catalog"].(map[string]any)
	assert.Contains(t, catalog, "providers")

	providers := catalog["providers"].([]any)
	assert.Len(t, providers, 1)

	provider := providers[0].(map[string]any)
	assert.Contains(t, provider, "id")
	assert.Contains(t, provider, "descriptor")
	assert.Contains(t, provider, "items")

	items := provider["items"].([]any)
	assert.Len(t, items, 1)

	item := items[0].(map[string]any)
	assert.Equal(t, "lst-001", item["id"])
	assert.Contains(t, item, "descriptor")
	assert.Contains(t, item, "price")
	assert.Contains(t, item, "category_id")

	descriptor := item["descriptor"].(map[string]any)
	assert.Equal(t, "Test Product", descriptor["name"])

	price := item["price"].(map[string]any)
	assert.Equal(t, "INR", price["currency"])
	assert.Equal(t, "1000.00", price["value"])
}

func TestSign(t *testing.T) {
	adapter, err := NewAdapter(Config{
		PrivateKeyHex:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		KeyID:          "test-key-1",
		SubscriberID:   "kalakriti.in",
		SubscriberURL:  "https://api.kalakriti.in",
		DryRun:         true,
	})
	require.NoError(t, err)

	payload := []byte(`{"test":"data"}`)
	signature, err := adapter.Sign(payload)
	require.NoError(t, err)
	assert.NotEmpty(t, signature)
	assert.Equal(t, 128, len(signature)) // 64 bytes hex-encoded = 128 chars
}
