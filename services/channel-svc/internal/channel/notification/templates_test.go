// services/channel-svc/internal/channel/notification/templates_test.go
package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTemplate(t *testing.T) {
	tpl, err := GetTemplate(LotOffered, English)
	require.NoError(t, err)
	assert.Equal(t, "New Lot Offer", tpl.Title)
	assert.Contains(t, tpl.Body, "new lot offer")
}

func TestGetTemplateHindi(t *testing.T) {
	tpl, err := GetTemplate(LotOffered, Hindi)
	require.NoError(t, err)
	assert.Equal(t, "नया लॉट ऑफ़र", tpl.Title)
	assert.Contains(t, tpl.Body, "लॉट ऑफ़र")
}

func TestGetTemplateUnknownKind(t *testing.T) {
	_, err := GetTemplate("UNKNOWN_KIND", English)
	require.Error(t, err)
}

func TestGetTemplateFallbackToEnglish(t *testing.T) {
	// Request a language that doesn't exist for this kind, should fallback to English.
	tpl, err := GetTemplate(LotOffered, "GUJARATI")
	require.NoError(t, err)
	assert.Equal(t, "New Lot Offer", tpl.Title)
}

func TestRenderTemplate(t *testing.T) {
	tpl, err := GetTemplate(LotOffered, English)
	require.NoError(t, err)

	rendered := tpl.Render(map[string]string{
		"product_name": "Handwoven Saree",
		"deadline":     "2026-08-30",
	})

	assert.Contains(t, rendered.Body, "Handwoven Saree")
	assert.Contains(t, rendered.Body, "2026-08-30")
	assert.NotContains(t, rendered.Body, "{product_name}")
	assert.NotContains(t, rendered.Body, "{deadline}")
}

func TestAllTemplatesHaveContent(t *testing.T) {
	kinds := []NotificationKind{
		LotOffered, LotExpiring, ListingApprovalDue, PaymentSettled,
		QCFailed, DisputeRaised, ShipmentDelivered, ArtisanFollowed,
	}

	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			tpl, err := GetTemplate(kind, English)
			require.NoError(t, err)
			assert.NotEmpty(t, tpl.Title)
			assert.NotEmpty(t, tpl.Body)
		})
	}
}
