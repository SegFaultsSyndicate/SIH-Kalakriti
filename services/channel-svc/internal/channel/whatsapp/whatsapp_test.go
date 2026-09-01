// services/channel-svc/internal/channel/whatsapp/whatsapp_test.go
package whatsapp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSendMessageDryRun(t *testing.T) {
	client := NewClient(Config{DryRun: true}, nil)

	err := client.SendMessage(context.Background(), "+919876543210", "Test message")
	require.NoError(t, err)
}

func TestSendTemplateDryRun(t *testing.T) {
	client := NewClient(Config{DryRun: true}, nil)

	err := client.SendTemplate(context.Background(), "+919876543210", "order_update", map[string]string{
		"order_id": "123",
	}, "en")
	require.NoError(t, err)
}