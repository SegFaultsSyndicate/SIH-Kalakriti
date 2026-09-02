// services/channel-svc/internal/channel/indiapost/indiapost_test.go
package indiapost

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/pkg/breaker"
)

func TestCheckServiceabilityStub(t *testing.T) {
	client := NewClient(Config{Stub: true}, nil)

	result, err := client.CheckServiceability(context.Background(), "560001")
	require.NoError(t, err)
	assert.Equal(t, "560001", result.Pincode)
	assert.True(t, result.Serviceable)
	assert.Equal(t, 3, result.EstDays)
}

func TestCheckServiceabilityInvalidPincode(t *testing.T) {
	client := NewClient(Config{Stub: true}, nil)

	result, err := client.CheckServiceability(context.Background(), "abc")
	require.NoError(t, err)
	assert.False(t, result.Serviceable)
}

func TestEstimateRateStub(t *testing.T) {
	client := NewClient(Config{Stub: true}, nil)

	rate, err := client.EstimateRate(context.Background(), "560001", "110001", 500)
	require.NoError(t, err)
	assert.Equal(t, "Speed Post", rate.ServiceType)
	assert.Equal(t, int64(5000+5*200), rate.RatePaise) // ₹50 base + ₹2/100g
	assert.Equal(t, 3, rate.EstDays)
}

func TestStubModeWhenNoKey(t *testing.T) {
	client := NewClient(Config{APIKey: "", Stub: false}, nil)
	assert.True(t, client.stub)
}

// TestCheckServiceabilityOpensCircuitAfterRepeatedFailures proves the
// breaker added to guard the not-yet-implemented real API path actually
// trips, even though nothing reachable through NewClient can flip stub off
// today without an API key — white-box construction is the only way to
// exercise it before that integration lands.
func TestCheckServiceabilityOpensCircuitAfterRepeatedFailures(t *testing.T) {
	client := &Client{log: slog.Default(), stub: false, breaker: breaker.New(2, time.Minute)}

	_, err1 := client.CheckServiceability(context.Background(), "560001")
	require.Error(t, err1)
	_, err2 := client.CheckServiceability(context.Background(), "560001")
	require.Error(t, err2)

	_, err := client.CheckServiceability(context.Background(), "560001")
	require.True(t, errors.Is(err, breaker.ErrCircuitOpen), "expected circuit to be open, got %v", err)
}