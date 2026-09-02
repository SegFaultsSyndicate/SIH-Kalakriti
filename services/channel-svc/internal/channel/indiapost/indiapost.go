// services/channel-svc/internal/channel/indiapost/indiapost.go

// Package indiapost checks pincode serviceability and rate estimates.
// ponytail: stub returning plausible fixtures until real API credentials land.
package indiapost

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/breaker"
)

// Client queries India Post API.
type Client struct {
	log     *slog.Logger
	apiURL  string
	apiKey  string
	stub    bool
	breaker *breaker.Breaker
}

// Config holds India Post API settings.
type Config struct {
	APIURL string
	APIKey string
	Stub   bool // If true, return fixtures instead of calling API
}

// NewClient creates an India Post client. The breaker guards the real-API
// path stub mode never exercises, so it's in place — same threshold as
// ondc.Client's — the moment that API call lands, instead of needing a
// second change to add it then.
func NewClient(cfg Config, log *slog.Logger) *Client {
	if log == nil {
		log = slog.Default()
	}
	return &Client{
		log:     log,
		apiURL:  cfg.APIURL,
		apiKey:  cfg.APIKey,
		stub:    cfg.Stub || cfg.APIKey == "", // ponytail: stub when no key
		breaker: breaker.New(5, 60*time.Second),
	}
}

// ServiceabilityResult holds pincode check result.
type ServiceabilityResult struct {
	Pincode     string
	Serviceable bool
	EstDays     int
}

// RateEstimate holds shipping cost estimate.
type RateEstimate struct {
	ServiceType string
	RatePaise   int64
	EstDays     int
}

// CheckServiceability checks if a pincode is deliverable.
func (c *Client) CheckServiceability(ctx context.Context, pincode string) (ServiceabilityResult, error) {
	var result ServiceabilityResult
	err := c.breaker.Call(func() error {
		if c.stub {
			c.log.Info("indiapost_serviceability_stub", "pincode", pincode)
			// All 6-digit numeric pincodes are serviceable in stub mode.
			serviceable := len(pincode) == 6 && isNumeric(pincode)
			result = ServiceabilityResult{
				Pincode:     pincode,
				Serviceable: serviceable,
				EstDays:     3,
			}
			return nil
		}
		return fmt.Errorf("indiapost: real API not implemented")
	})
	return result, err
}

// EstimateRate estimates shipping cost.
func (c *Client) EstimateRate(ctx context.Context, fromPincode, toPincode string, weightGrams int) (RateEstimate, error) {
	var result RateEstimate
	err := c.breaker.Call(func() error {
		if c.stub {
			c.log.Info("indiapost_rate_stub", "from", fromPincode, "to", toPincode, "weight_g", weightGrams)
			// Stub formula: ₹50 base + ₹2 per 100g.
			ratePaise := int64(5000 + (weightGrams/100)*200)
			result = RateEstimate{
				ServiceType: "Speed Post",
				RatePaise:   ratePaise,
				EstDays:     3,
			}
			return nil
		}
		return fmt.Errorf("indiapost: real API not implemented")
	})
	return result, err
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
