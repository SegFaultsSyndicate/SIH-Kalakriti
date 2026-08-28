// services/channel-svc/internal/channel/indiapost/indiapost_test.go
package indiapost

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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