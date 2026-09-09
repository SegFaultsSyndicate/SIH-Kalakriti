# Circuit Breakers for External APIs

**Last Updated:** 2026-08-28  
**Status:** ONDC client implemented, others ready for wiring

---

## Overview

Circuit breakers prevent cascading failures when external services are slow or unavailable. Instead of waiting for timeouts on every request, the circuit breaker "trips" after a threshold of failures and immediately rejects requests for a cooldown period.

**External APIs wrapped:**
- ✅ **ONDC gateway** — Circuit breaker active
- 🔄 **WhatsApp Business API** — Stub (ready for real implementation)
- 🔄 **India Post API** — Stub (ready for real implementation)
- 🔄 **SMS provider** — Stub (ready for real implementation)

---

## How It Works

### States

```
        failures < threshold
Closed ───────────────────────► Closed
  │                                │
  │ failures ≥ threshold          │
  └──────────► Open ◄─────────────┘
                 │
                 │ timeout elapsed
                 ▼
              Half-Open
                 │
                 │ 2 successes
                 └──────────► Closed
```

1. **Closed** — Normal operation, requests pass through
2. **Open** — Too many failures, reject all requests immediately
3. **Half-Open** — Timeout elapsed, try one request to test if service recovered

### Configuration

```go
breaker := breaker.New(
    5,              // Threshold: open after 5 consecutive failures
    60*time.Second, // Timeout: stay open for 60s before trying again
)
```

---

## Implementation

### ONDC Client (Already Done)

`services/channel-svc/internal/channel/ondc/client.go`:

```go
type Client struct {
    adapter    *Adapter
    httpClient *http.Client
    gatewayURL string
    breaker    *breaker.Breaker  // Circuit breaker
    log        *slog.Logger
}

func NewClient(adapter *Adapter, gatewayURL string, log *slog.Logger) *Client {
    return &Client{
        adapter:    adapter,
        gatewayURL: gatewayURL,
        log:        log,
        httpClient: &http.Client{Timeout: 10 * time.Second},
        breaker:    breaker.New(5, 60*time.Second), // Open after 5 failures, retry after 60s
    }
}

func (c *Client) PublishOnSearch(ctx context.Context, listings []Listing) error {
    // ... build payload, sign, create request ...

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

        c.log.Info("ondc_published", "status", resp.StatusCode)
        return nil
    })

    if err == breaker.ErrCircuitOpen {
        c.log.Warn("ondc circuit breaker open, skipping request")
        return fmt.Errorf("ondc service unavailable (circuit open): %w", err)
    }

    return err
}
```

---

## Pattern for Real Implementations

When replacing stubs with real external API clients, follow this pattern:

### 1. Add Circuit Breaker to Client Struct

```go
import "github.com/ZoroNewbie00/kalakriti/pkg/breaker"

type WhatsAppClient struct {
    apiURL  string
    apiKey  string
    client  *http.Client
    breaker *breaker.Breaker  // Add this
    log     *slog.Logger
}

func NewWhatsAppClient(apiURL, apiKey string, log *slog.Logger) *WhatsAppClient {
    return &WhatsAppClient{
        apiURL:  apiURL,
        apiKey:  apiKey,
        client:  &http.Client{Timeout: 10 * time.Second},
        breaker: breaker.New(5, 60*time.Second),  // Add this
        log:     log,
    }
}
```

### 2. Wrap External Calls

```go
func (c *WhatsAppClient) SendMessage(ctx context.Context, to, text string) error {
    // Build request
    payload := map[string]string{"to": to, "text": text}
    body, _ := json.Marshal(payload)
    req, _ := http.NewRequestWithContext(ctx, "POST", c.apiURL+"/messages", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+c.apiKey)

    // Wrap with circuit breaker
    err := c.breaker.Call(func() error {
        resp, err := c.client.Do(req)
        if err != nil {
            return err
        }
        defer resp.Body.Close()

        if resp.StatusCode >= 300 {
            return fmt.Errorf("whatsapp api: status %d", resp.StatusCode)
        }
        return nil
    })

    if err == breaker.ErrCircuitOpen {
        c.log.Warn("whatsapp circuit breaker open, message not sent", "to", to)
        return fmt.Errorf("whatsapp unavailable: %w", err)
    }

    return err
}
```

### 3. Log Circuit Events

```go
// In your monitoring/alerting setup
if err == breaker.ErrCircuitOpen {
    log.Error("circuit breaker open", 
        "service", "whatsapp",
        "action", "send_message",
    )
    // Alert: WhatsApp circuit breaker tripped, messages queued
}
```

---

## When to Use Different Thresholds

| Service | Threshold | Timeout | Reason |
|---------|-----------|---------|--------|
| **ONDC** | 5 | 60s | Government gateway, slow recovery |
| **WhatsApp** | 3 | 30s | Fast service, quick recovery |
| **SMS** | 5 | 60s | Third-party, variable reliability |
| **India Post** | 10 | 120s | Read-only, non-critical, tolerate more failures |

**Higher threshold** → More tolerance for intermittent failures  
**Lower timeout** → Faster retry after outage

---

## Testing Circuit Breakers

### Unit Test

```go
func TestCircuitBreaker(t *testing.T) {
    breaker := breaker.New(3, 10*time.Second)

    // Fail 3 times → circuit opens
    for i := 0; i < 3; i++ {
        err := breaker.Call(func() error {
            return errors.New("service down")
        })
        assert.Error(t, err)
    }

    // 4th call rejected immediately
    err := breaker.Call(func() error {
        return nil
    })
    assert.Equal(t, breaker.ErrCircuitOpen, err)

    // Wait for timeout
    time.Sleep(11 * time.Second)

    // Circuit half-open, 2 successes close it
    for i := 0; i < 2; i++ {
        err := breaker.Call(func() error {
            return nil
        })
        assert.NoError(t, err)
    }

    // Circuit closed again
    err = breaker.Call(func() error {
        return nil
    })
    assert.NoError(t, err)
}
```

### Integration Test

There is no `/api/v1/ondc/publish` bff route -- ONDC publishing
(`channel.ondc.Client.PublishOnSearch`, wrapped by the circuit breaker shown
above) is driven from `services/channel-svc/internal/channel/consumer/ondc_publisher.go`,
a Kafka consumer reacting to catalog events, not an inbound HTTP call. To
exercise the breaker end-to-end, trigger the events that feed that consumer
(e.g. publish a listing) with the real ONDC gateway unreachable, and watch
channel-svc's logs for `ondc circuit breaker open, skipping request` rather
than hitting a REST endpoint directly. A unit test against `ondc.Client.PublishOnSearch` with a fake gateway
server -- there is an `adapter_test.go` in that package but no `client_test.go`
yet -- would be a more direct way to verify the breaker trips than an
end-to-end HTTP call.

---

## Monitoring

### Metrics to Track

1. **Circuit breaker state** — open, closed, half-open (per service)
2. **Failure rate** — failures per minute (detect degradation before circuit opens)
3. **Circuit open events** — alert when circuit trips
4. **Recovery time** — how long until circuit closes again

### Prometheus Metrics (Future)

```go
var (
    circuitBreakerState = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "circuit_breaker_state",
            Help: "Circuit breaker state (0=closed, 1=open, 2=half_open)",
        },
        []string{"service"},
    )

    circuitBreakerTrips = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "circuit_breaker_trips_total",
            Help: "Total number of times circuit breaker opened",
        },
        []string{"service"},
    )
)
```

### CloudWatch Alarms

```bash
# Alert when circuit opens
aws cloudwatch put-metric-alarm \
  --alarm-name kalakriti-ondc-circuit-open \
  --alarm-description "ONDC circuit breaker is open" \
  --metric-name CircuitBreakerState \
  --namespace Kalakriti \
  --statistic Maximum \
  --period 60 \
  --evaluation-periods 1 \
  --threshold 1 \
  --comparison-operator GreaterThanOrEqualToThreshold \
  --dimensions Name=Service,Value=ONDC
```

---

## Graceful Degradation

When a circuit breaker opens, the service should degrade gracefully:

### ONDC

**Action:** Queue order for manual export  
**User impact:** Order accepted, ONDC sync delayed  
**Recovery:** Cron job retries queued orders when circuit closes

### WhatsApp

**Action:** Fall back to SMS or email  
**User impact:** None (alternative notification sent)  
**Recovery:** Next notification uses WhatsApp if circuit closed

### SMS (OTP)

**Action:** Log error, return 503 to client  
**User impact:** Login fails, retry in 60s  
**Recovery:** No queue (OTP expires quickly)

### India Post

**Action:** Return cached estimate  
**User impact:** Shipping estimate may be stale  
**Recovery:** Real-time estimates resume when circuit closes

---

## Summary

**Circuit breakers implemented:** ONDC client  
**Pattern documented:** Ready for WhatsApp, SMS, India Post  
**Default settings:** 5 failures, 60s timeout  
**Behavior:** Fail fast when service is down, auto-recover when it's back

**Before enabling in production:**
1. Set up monitoring for circuit breaker state
2. Define graceful degradation for each service
3. Test circuit behavior with simulated outages
4. Document runbook for manual intervention (if needed)
