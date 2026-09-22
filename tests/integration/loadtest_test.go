// tests/integration/loadtest_test.go
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// endpointSpec defines a single endpoint to load-test.
type endpointSpec struct {
	Method  string
	Path    string            // relative to /api/v1 unless IsRoot is true
	IsRoot  bool              // true for SEO/verification/sitemap routes mounted at /
	Body    map[string]any    // nil for GET/DELETE
	Headers map[string]string // extra headers beyond Content-Type and Auth
	Auth    string            // "artisan", "buyer", "admin", "none"
}

// loadResult captures per-request outcome.
type loadResult struct {
	Endpoint   string
	StatusCode int
	Latency    time.Duration
	Err        error
}

// endpointStats holds aggregated stats for one endpoint.
type endpointStats struct {
	Endpoint    string
	Method      string
	Path        string
	Requests    int
	Successes   int
	Errors      int
	Non2xx      int
	P50         time.Duration
	P95         time.Duration
	P99         time.Duration
	Mean        time.Duration
	Min         time.Duration
	Max         time.Duration
	TotalTime   time.Duration
	RPS         float64
	ErrorRate   float64
}

func TestLoadAllEndpoints(t *testing.T) {
	if testServer == nil {
		t.Skip("test server not initialized")
	}

	const (
		concurrency     = 10   // goroutines per endpoint
		requestsPerUser = 50   // requests each goroutine sends per endpoint (500 requests per endpoint)
		warmupRequests  = 5    // warm-up requests (discarded from stats)
	)

	specs := allEndpointSpecs()

	t.Logf("Load testing %d endpoints | concurrency=%d | rps_target=unlimited | requests_per_endpoint=%d",
		len(specs), concurrency, concurrency*requestsPerUser)

	var allStats []endpointStats

	for _, spec := range specs {
		stats := benchmarkEndpoint(t, spec, concurrency, requestsPerUser, warmupRequests)
		allStats = append(allStats, stats)
	}

	// Print summary table.
	printResultsTable(t, allStats)

	// Print markdown report to stdout and write to file.
	report := generateMarkdownReport(allStats, concurrency, requestsPerUser)
	_ = os.WriteFile("loadtest_results.md", []byte(report), 0644)
	_ = os.WriteFile("../../loadtest_results.md", []byte(report), 0644)
	t.Log("\n" + report)
}

func benchmarkEndpoint(t *testing.T, spec endpointSpec, concurrency, reqsPerWorker, warmup int) endpointStats {
	t.Helper()

	var url string
	if spec.IsRoot {
		url = testServer.URL + spec.Path
	} else {
		url = testServer.URL + "/api/v1" + spec.Path
	}
	token := tokenForRole(spec.Auth)
	label := spec.Method + " " + spec.Path

	// Warm up.
	for i := 0; i < warmup; i++ {
		doRequest(url, spec.Method, token, spec.Body, spec.Headers)
	}

	var (
		mu       sync.Mutex
		results  []loadResult
		wg       sync.WaitGroup
		inflight int64
	)

	start := time.Now()

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < reqsPerWorker; i++ {
				atomic.AddInt64(&inflight, 1)
				res := doRequest(url, spec.Method, token, spec.Body, spec.Headers)
				res.Endpoint = label
				atomic.AddInt64(&inflight, -1)

				mu.Lock()
				results = append(results, res)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	totalTime := time.Since(start)

	// Compute stats.
	latencies := make([]time.Duration, 0, len(results))
	successes, errors, non2xx := 0, 0, 0
	for _, r := range results {
		latencies = append(latencies, r.Latency)
		if r.Err != nil {
			errors++
		} else if r.StatusCode >= 200 && r.StatusCode < 300 || r.StatusCode == 409 || r.StatusCode == 401 || r.StatusCode == 403 || r.StatusCode == 404 {
			// Count expected error codes as successes (the endpoint handled it).
			successes++
		} else {
			non2xx++
		}
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	stats := endpointStats{
		Endpoint:  label,
		Method:    spec.Method,
		Path:      spec.Path,
		Requests:  len(results),
		Successes: successes,
		Errors:    errors,
		Non2xx:    non2xx,
		P50:       percentile(latencies, 0.50),
		P95:       percentile(latencies, 0.95),
		P99:       percentile(latencies, 0.99),
		Mean:      mean(latencies),
		Min:       latencies[0],
		Max:       latencies[len(latencies)-1],
		TotalTime: totalTime,
	}
	if totalTime > 0 {
		stats.RPS = float64(len(results)) / totalTime.Seconds()
	}
	if len(results) > 0 {
		stats.ErrorRate = float64(errors+non2xx) / float64(len(results)) * 100
	}

	t.Logf("  %-45s %5d reqs | p50=%6s p95=%6s p99=%6s | RPS=%8.1f | err=%.1f%%",
		label, len(results),
		fmtDuration(stats.P50), fmtDuration(stats.P95), fmtDuration(stats.P99),
		stats.RPS, stats.ErrorRate)

	return stats
}

func doRequest(url, method, token string, body map[string]any, headers map[string]string) loadResult {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return loadResult{Err: err}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// Always send an idempotency key for POST/PUT endpoints.
	if method == "POST" || method == "PUT" {
		req.Header.Set("Idempotency-Key", fmt.Sprintf("loadtest-%d", time.Now().UnixNano()))
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		return loadResult{Latency: latency, Err: err}
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	return loadResult{StatusCode: resp.StatusCode, Latency: latency}
}

func tokenForRole(role string) string {
	switch role {
	case "artisan":
		return artisanToken
	case "buyer":
		return buyerToken
	case "admin":
		return adminToken
	default:
		return ""
	}
}

// allEndpointSpecs returns every endpoint in the BFF, grouped by domain.
func allEndpointSpecs() []endpointSpec {
	return []endpointSpec{
		// ── Auth ──
		{Method: "POST", Path: "/auth/otp/request", Auth: "none", Body: map[string]any{
			"phone": "+919876543210",
		}},
		{Method: "POST", Path: "/auth/otp/verify", Auth: "none", Body: map[string]any{
			"phone": "+919876543210",
			"otp":   "000000",
		}},
		{Method: "POST", Path: "/auth/refresh", Auth: "none", Body: map[string]any{
			"refresh_token": "dummy-refresh-token",
		}},
		{Method: "POST", Path: "/auth/phone/change/request", Auth: "artisan", Body: map[string]any{
			"new_phone": "+919876543299",
		}},
		{Method: "POST", Path: "/auth/phone/change/verify", Auth: "artisan", Body: map[string]any{
			"new_phone": "+919876543299",
			"otp":       "000000",
		}},

		// ── Artisan ──
		{Method: "POST", Path: "/artisans", Auth: "artisan", Body: map[string]any{
			"display_name": "Load Test Artisan",
			"language":     "hi",
		}},
		{Method: "GET", Path: "/artisans/me", Auth: "artisan"},
		{Method: "PATCH", Path: "/artisans/me", Auth: "artisan", Body: map[string]any{
			"display_name": "Updated Name",
		}},
		{Method: "GET", Path: "/artisans/artisan-1/storefront", Auth: "none"},
		{Method: "GET", Path: "/artisans/artisan-1/follower-count", Auth: "none"},

		// ── Media ──
		{Method: "POST", Path: "/media/upload-url", Auth: "artisan", Body: map[string]any{
			"content_type": "image/jpeg",
			"size_bytes":   1024000,
		}},
		{Method: "POST", Path: "/media/media-1/confirm", Auth: "artisan"},

		// ── Listings ──
		{Method: "GET", Path: "/listings", Auth: "none"},
		{Method: "POST", Path: "/listings", Auth: "artisan", Body: map[string]any{
			"title":      "Load Test Listing",
			"craft_id":   "madhubani",
			"media_ids":  []string{"media-1"},
			"type":       "READY_STOCK",
			"price":      map[string]any{"amount_paise": 100000, "currency_code": "INR"},
		}},
		{Method: "GET", Path: "/listings/listing-1", Auth: "none"},
		{Method: "PATCH", Path: "/listings/listing-1", Auth: "artisan", Body: map[string]any{
			"title": "Updated Title",
		}},
		{Method: "GET", Path: "/listings/listing-1/summary", Auth: "none"},
		{Method: "POST", Path: "/listings/listing-1/submit", Auth: "artisan"},
		{Method: "POST", Path: "/listings/listing-1/approve", Auth: "artisan", Body: map[string]any{}},
		{Method: "POST", Path: "/listings/listing-1/seal-provenance", Auth: "artisan", Body: map[string]any{
			"media":             []string{"media-1"},
			"claimed_technique": "hand-painted",
		}},
		{Method: "POST", Path: "/listings/listing-1/suspend", Auth: "admin", Body: map[string]any{
			"reason": "load test suspension",
		}},
		{Method: "POST", Path: "/listings/listing-1/reinstate", Auth: "admin"},

		// ── Search ──
		{Method: "GET", Path: "/search?q=madhubani+painting", Auth: "none"},
		{Method: "GET", Path: "/search/suggest?q=madh", Auth: "none"},
		{Method: "POST", Path: "/search/voice", Auth: "none", Body: nil, Headers: map[string]string{
			"Content-Type":    "application/octet-stream",
			"Accept-Language": "hi-IN",
		}},

		// ── Pricing ──
		{Method: "POST", Path: "/pricing/advise", Auth: "artisan", Body: map[string]any{
			"listing_id":    "listing-1",
			"material_cost": map[string]any{"amount_paise": 50000, "currency_code": "INR"},
			"hours":         40,
		}},

		// ── Crafts ──
		{Method: "GET", Path: "/crafts", Auth: "none"},
		{Method: "GET", Path: "/crafts/madhubani", Auth: "none"},
		{Method: "POST", Path: "/crafts/refresh-index", Auth: "admin"},

		// ── Orders ──
		{Method: "POST", Path: "/orders/bulk", Auth: "buyer", Body: map[string]any{
			"listing_id":  "listing-1",
			"quantity":    100,
			"required_by": "2026-12-01T00:00:00Z",
		}},
		{Method: "GET", Path: "/orders/order-1", Auth: "artisan"},
		{Method: "POST", Path: "/orders/lots/lot-1/respond", Auth: "artisan", Body: map[string]any{
			"accept":             true,
			"promised_ship_date": "2026-11-01T00:00:00Z",
		}},
		{Method: "POST", Path: "/orders/lots/lot-1/progress", Auth: "artisan", Body: map[string]any{
			"progress_pct": 50,
			"note":         "halfway done",
		}},
		{Method: "POST", Path: "/orders/lots/lot-1/reallocate", Auth: "artisan", Body: map[string]any{
			"reason": "unable to complete",
		}},
		{Method: "GET", Path: "/orders/order-1/events", Auth: "artisan"},

		// ── Follow / Feed ──
		{Method: "POST", Path: "/artisans/artisan-1/follow", Auth: "buyer"},
		{Method: "DELETE", Path: "/artisans/artisan-1/follow", Auth: "buyer"},
		{Method: "GET", Path: "/feed", Auth: "artisan"},
		{Method: "POST", Path: "/feed/notif-1/read", Auth: "artisan"},
		{Method: "GET", Path: "/feed/process", Auth: "none"},

		// ── Statements ──
		{Method: "POST", Path: "/statements", Auth: "artisan", Body: map[string]any{
			"start": "2026-08-01",
			"end":   "2026-08-31",
		}},
		{Method: "GET", Path: "/statements", Auth: "artisan"},
		{Method: "GET", Path: "/statements/stmt-1", Auth: "artisan"},

		// ── Insights (admin/ministry) ──
		{Method: "GET", Path: "/insights/artisans-by-category", Auth: "admin"},
		{Method: "GET", Path: "/insights/listings-by-craft-month", Auth: "admin"},
		{Method: "GET", Path: "/insights/earnings-by-district", Auth: "admin"},
		{Method: "GET", Path: "/insights/income-comparison", Auth: "admin"},
		{Method: "GET", Path: "/insights/dying-crafts", Auth: "admin"},
		{Method: "POST", Path: "/insights/refresh", Auth: "admin"},

		// ── Clusters ──
		{Method: "POST", Path: "/clusters", Auth: "admin", Body: map[string]any{
			"name":       "Test Cluster",
			"state_code": "BR",
		}},
		{Method: "GET", Path: "/clusters/cluster-1", Auth: "admin"},
		{Method: "GET", Path: "/clusters/cluster-1/members", Auth: "admin"},
		{Method: "POST", Path: "/clusters/cluster-1/members", Auth: "admin", Body: map[string]any{
			"artisan_id": "artisan-1",
			"role":       "MEMBER",
		}},
		{Method: "DELETE", Path: "/clusters/cluster-1/members/artisan-1", Auth: "admin"},
		{Method: "POST", Path: "/clusters/cluster-1/onboard", Auth: "admin", Body: map[string]any{
			"display_name": "Onboarded Artisan",
			"phone_e164":   "+919876543211",
			"craft_ids":    []string{"madhubani"},
			"languages":    []string{"hi"},
			"region":       map[string]any{"state_code": "BR", "district": "Madhubani"},
		}},

		// ── Self-Help Groups ──
		{Method: "POST", Path: "/self-help-groups", Auth: "admin", Body: map[string]any{
			"name":            "Test SHG",
			"registration_no": "SHG-001",
			"members": []map[string]any{
				{"artisan_id": "artisan-1", "share_pct": 100},
			},
		}},
		{Method: "GET", Path: "/self-help-groups/shg-1", Auth: "admin"},
		{Method: "PUT", Path: "/self-help-groups/shg-1/members", Auth: "admin", Body: map[string]any{
			"members": []map[string]any{
				{"artisan_id": "artisan-1", "share_pct": 100},
			},
		}},

		// ── Payments ──
		{Method: "POST", Path: "/payments/webhook", Auth: "none", Body: map[string]any{
			"event": "payment.captured",
		}},

		// ── Public Web / SEO / Verification ──
		{Method: "GET", Path: "/listing/madhubani-fish-painting", IsRoot: true, Auth: "none"},
		{Method: "GET", Path: "/artisan/lakshmi-devi", IsRoot: true, Auth: "none"},
		{Method: "GET", Path: "/v/QR_CODE_PROV_123", IsRoot: true, Auth: "none"},
		{Method: "GET", Path: "/v/QR_CODE_PROV_123/verify.json", IsRoot: true, Auth: "none"},
		{Method: "GET", Path: "/export/indiahandmade", IsRoot: true, Auth: "none"},
		{Method: "GET", Path: "/sitemap.xml", IsRoot: true, Auth: "none"},
		{Method: "GET", Path: "/robots.txt", IsRoot: true, Auth: "none"},

		// ── OpenAPI spec ──
		{Method: "GET", Path: "/openapi.json", Auth: "none"},
	}
}

func percentile(sorted []time.Duration, pct float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(pct*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func mean(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	return sum / time.Duration(len(durations))
}

func fmtDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.0fµs", float64(d.Microseconds()))
	}
	return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
}

func printResultsTable(t *testing.T, stats []endpointStats) {
	t.Helper()
	t.Log("\n=== LOAD TEST RESULTS ===")
	t.Logf("%-45s %7s %7s %7s %7s %10s %6s", "ENDPOINT", "REQS", "P50", "P95", "P99", "RPS", "ERR%")
	t.Log(strings.Repeat("-", 100))
	for _, s := range stats {
		t.Logf("%-45s %7d %7s %7s %7s %10.1f %5.1f%%",
			s.Endpoint, s.Requests,
			fmtDuration(s.P50), fmtDuration(s.P95), fmtDuration(s.P99),
			s.RPS, s.ErrorRate)
	}
}

func generateMarkdownReport(stats []endpointStats, concurrency, reqsPerWorker int) string {
	var b strings.Builder

	totalReqs := 0
	totalErrors := 0
	var allLatencies []time.Duration
	var overallStart time.Duration

	for _, s := range stats {
		totalReqs += s.Requests
		totalErrors += s.Errors + s.Non2xx
		overallStart += s.TotalTime
		// Approximate: add the mean as a representative.
		for i := 0; i < s.Requests; i++ {
			allLatencies = append(allLatencies, s.Mean)
		}
	}

	sort.Slice(allLatencies, func(i, j int) bool { return allLatencies[i] < allLatencies[j] })

	b.WriteString("# Kalakriti BFF Load Test Results\n\n")
	b.WriteString(fmt.Sprintf("**Date**: %s  \n", time.Now().Format("2006-01-02 15:04:05 MST")))
	b.WriteString(fmt.Sprintf("**Endpoints tested**: %d  \n", len(stats)))
	b.WriteString(fmt.Sprintf("**Concurrency**: %d workers per endpoint  \n", concurrency))
	b.WriteString(fmt.Sprintf("**Requests per worker**: %d  \n", reqsPerWorker))
	b.WriteString(fmt.Sprintf("**Total requests**: %d  \n", totalReqs))
	b.WriteString(fmt.Sprintf("**Total errors**: %d (%.2f%%)  \n", totalErrors, float64(totalErrors)/float64(totalReqs)*100))
	b.WriteString(fmt.Sprintf("**Test mode**: In-process httptest.Server with stubbed gRPC backends  \n"))
	b.WriteString("\n---\n\n")

	// Group endpoints by domain.
	groups := map[string][]endpointStats{}
	groupOrder := []string{
		"Auth", "Artisan", "Media", "Listings", "Search",
		"Pricing", "Crafts", "Orders", "Follow / Feed",
		"Statements", "Insights", "Clusters", "Self-Help Groups",
		"Payments", "Public Web / SEO / Verification", "Other",
	}
	for _, s := range stats {
		group := classifyEndpoint(s.Path)
		groups[group] = append(groups[group], s)
	}

	for _, group := range groupOrder {
		endpoints, ok := groups[group]
		if !ok || len(endpoints) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("## %s\n\n", group))
		b.WriteString("| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |\n")
		b.WriteString("|--------|----------|------|-----|-----|-----|------|-----|------|\n")
		for _, s := range endpoints {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | %d | %s | %s | %s | %s | %.0f | %.1f%% |\n",
				s.Method, s.Path, s.Requests,
				fmtDuration(s.P50), fmtDuration(s.P95), fmtDuration(s.P99),
				fmtDuration(s.Mean), s.RPS, s.ErrorRate))
		}
		b.WriteString("\n")
	}

	// Top 10 slowest endpoints.
	sorted := make([]endpointStats, len(stats))
	copy(sorted, stats)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].P99 > sorted[j].P99 })

	b.WriteString("## 🐢 Top 10 Slowest Endpoints (by P99)\n\n")
	b.WriteString("| Rank | Endpoint | P99 | P95 | Mean | RPS |\n")
	b.WriteString("|------|----------|-----|-----|------|-----|\n")
	for i, s := range sorted {
		if i >= 10 {
			break
		}
		b.WriteString(fmt.Sprintf("| %d | `%s %s` | %s | %s | %s | %.0f |\n",
			i+1, s.Method, s.Path,
			fmtDuration(s.P99), fmtDuration(s.P95), fmtDuration(s.Mean), s.RPS))
	}
	b.WriteString("\n")

	// Top 10 fastest.
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].P50 < sorted[j].P50 })
	b.WriteString("## ⚡ Top 10 Fastest Endpoints (by P50)\n\n")
	b.WriteString("| Rank | Endpoint | P50 | P95 | Mean | RPS |\n")
	b.WriteString("|------|----------|-----|-----|------|-----|\n")
	for i, s := range sorted {
		if i >= 10 {
			break
		}
		b.WriteString(fmt.Sprintf("| %d | `%s %s` | %s | %s | %s | %.0f |\n",
			i+1, s.Method, s.Path,
			fmtDuration(s.P50), fmtDuration(s.P95), fmtDuration(s.Mean), s.RPS))
	}
	b.WriteString("\n")

	// Any endpoints with errors.
	var errorEndpoints []endpointStats
	for _, s := range stats {
		if s.ErrorRate > 0 {
			errorEndpoints = append(errorEndpoints, s)
		}
	}
	if len(errorEndpoints) > 0 {
		b.WriteString("## ⚠️ Endpoints With Errors\n\n")
		b.WriteString("| Endpoint | Reqs | Errors | Non-2xx | Error Rate |\n")
		b.WriteString("|----------|------|--------|---------|------------|\n")
		for _, s := range errorEndpoints {
			b.WriteString(fmt.Sprintf("| `%s %s` | %d | %d | %d | %.1f%% |\n",
				s.Method, s.Path, s.Requests, s.Errors, s.Non2xx, s.ErrorRate))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Notes\n\n")
	b.WriteString("- **Test environment**: In-process `httptest.Server` with stubbed gRPC backend services.\n")
	b.WriteString("- **What's measured**: BFF HTTP layer — routing, middleware (auth, rate-limiting, CORS, i18n, idempotency), JSON serialization/deserialization, and response assembly.\n")
	b.WriteString("- **What's NOT measured**: Database queries, gRPC round-trips to real services, object storage, Kafka producers — all stubbed.\n")
	b.WriteString("- **Error codes 401/403/404/409**: Counted as handled successes — the BFF correctly rejected the request.\n")
	b.WriteString("- **Idempotency**: POST/PUT endpoints send unique `Idempotency-Key` per request.\n")

	return b.String()
}

func classifyEndpoint(path string) string {
	path = strings.Split(path, "?")[0]
	switch {
	case strings.HasPrefix(path, "/auth"):
		return "Auth"
	case strings.HasPrefix(path, "/artisans"):
		return "Artisan"
	case strings.HasPrefix(path, "/media"):
		return "Media"
	case strings.HasPrefix(path, "/listings"):
		return "Listings"
	case strings.HasPrefix(path, "/search"):
		return "Search"
	case strings.HasPrefix(path, "/pricing"):
		return "Pricing"
	case strings.HasPrefix(path, "/crafts"):
		return "Crafts"
	case strings.HasPrefix(path, "/orders"):
		return "Orders"
	case strings.HasPrefix(path, "/feed") || strings.Contains(path, "/follow"):
		return "Follow / Feed"
	case strings.HasPrefix(path, "/statements"):
		return "Statements"
	case strings.HasPrefix(path, "/insights"):
		return "Insights"
	case strings.HasPrefix(path, "/clusters"):
		return "Clusters"
	case strings.HasPrefix(path, "/self-help-groups"):
		return "Self-Help Groups"
	case strings.HasPrefix(path, "/payments"):
		return "Payments"
	case strings.HasPrefix(path, "/listing/") || strings.HasPrefix(path, "/artisan/") || strings.HasPrefix(path, "/v/") || strings.HasPrefix(path, "/export") || strings.HasPrefix(path, "/sitemap") || strings.HasPrefix(path, "/robots"):
		return "Public Web / SEO / Verification"
	default:
		return "Other"
	}
}
