// services/channel-svc/internal/channel/whatsapp/whatsapp_test.go
package whatsapp

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/pkg/breaker"
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

// TestSendMessageOpensCircuitAfterRepeatedFailures proves the breaker added
// to guard the not-yet-implemented real send path actually trips, even
// though nothing reachable through NewClient can flip dryRun off today —
// white-box construction is the only way to exercise it before whatsmeow
// sending lands.
func TestSendMessageOpensCircuitAfterRepeatedFailures(t *testing.T) {
	client := &Client{log: slog.Default(), dryRun: false, breaker: breaker.New(2, time.Minute)}

	require.Error(t, client.SendMessage(context.Background(), "+91", "hi"))
	require.Error(t, client.SendMessage(context.Background(), "+91", "hi"))

	err := client.SendMessage(context.Background(), "+91", "hi")
	require.True(t, errors.Is(err, breaker.ErrCircuitOpen), "expected circuit to be open, got %v", err)
}