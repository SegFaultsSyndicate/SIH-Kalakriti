// services/channel-svc/internal/channel/ondc/client.go

package ondc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/breaker"
)

// Client sends ONDC payloads to the gateway.
type Client struct {
	adapter    *Adapter
	httpClient *http.Client
	gatewayURL string
	breaker    *breaker.Breaker
	log        *slog.Logger
}

// NewClient creates an ONDC client.
func NewClient(adapter *Adapter, gatewayURL string, log *slog.Logger) *Client {
	return &Client{
		adapter:    adapter,
		gatewayURL: gatewayURL,
		log:        log,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		breaker:    breaker.New(5, 60*time.Second), // Open after 5 failures, retry after 60s
	}
}

// PublishOnSearch posts an on_search payload to the ONDC gateway.
func (c *Client) PublishOnSearch(ctx context.Context, listings []Listing) error {
	payload, err := c.adapter.BuildOnSearch(ctx, listings)
	if err != nil {
		return fmt.Errorf("build on_search: %w", err)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	if c.adapter.dryRun {
		c.log.Info("ondc_dry_run", "payload_size", len(payloadBytes))
		fmt.Println(string(payloadBytes))
		return nil
	}

	signature, err := c.adapter.Sign(payloadBytes)
	if err != nil {
		return fmt.Errorf("sign payload: %w", err)
	}

	authHeader := fmt.Sprintf(
		"Signature keyId=\"%s\",algorithm=\"ed25519\",headers=\"(created) (expires) digest\",signature=\"%s\"",
		c.adapter.signer.KeyID(),
		signature,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.gatewayURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	// Wrap HTTP call with circuit breaker
	err = c.breaker.Call(func() error {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("http post: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("ondc gateway: %d %s", resp.StatusCode, string(body))
		}

		c.log.Info("ondc_published", "status", resp.StatusCode, "payload_size", len(payloadBytes))
		return nil
	})

	if err == breaker.ErrCircuitOpen {
		c.log.Warn("ondc circuit breaker open, skipping request")
		return fmt.Errorf("ondc service unavailable (circuit open): %w", err)
	}

	return err
}
