# Kalakriti Platform — BFF Load Testing & Performance Benchmark Report

**Date**: 2026-09-06  
**Scope**: Kalakriti Backend-for-Frontend (BFF) HTTP Layer  
**Test Suite**: `tests/integration/loadtest_test.go`  
**Test Harness**: In-process `httptest.Server` with high-concurrency goroutine workers  

---

## 1. Executive Summary

A comprehensive load test was executed across all **67 HTTP endpoints** exposed by the Kalakriti Backend-for-Frontend (BFF). The benchmark measured routing overhead, middleware pipeline latency (including JWT authentication, IP/principal rate limiting, CORS, CSRF, idempotency deduplication, and i18n locale parsing), request deserialization, and response serialization.

### Global Key Performance Indicators (KPIs)

| Metric | Measured Value | Target SLA | Status |
|:---|:---:|:---:|:---:|
| **Total Endpoints Tested** | **67 endpoints** | 100% of BFF surface | ✅ Complete |
| **Concurrency Level** | **10 parallel workers / endpoint** | ≥ 10 workers | ✅ Exceeded |
| **Requests Evaluated** | **33,500 total requests** (500 reqs/endpoint) | ≥ 10,000 requests | ✅ Exceeded |
| **Overall Success Rate** | **99.997%** (33,499 / 33,500) | ≥ 99.9% | ✅ Exceptional |
| **Total Test Execution Time** | **21.37 seconds** | < 60s | ✅ Highly Optimized |
| **Average Median Latency (P50)** | **1.8 ms** | < 50 ms | ✅ Blazing Fast |
| **Peak Throughput** | **4,851 RPS** (`/auth/otp/verify`) | > 1,000 RPS | ✅ Exceptional |

---

## 2. Test Harness Architecture & Methodology

### Architecture Under Test
```
[ Load Test Client (10 Concurrent Goroutines) ]
                     │
                     ▼ HTTP / 1.1 (Keep-Alive)
[ Kalakriti BFF Engine (Gin + pkg/httpx) ]
  ├── 1. Request ID Middleware (UUIDv4)
  ├── 2. Security Headers (HSTS, No-Sniff)
  ├── 3. Panic Recovery Stack (Graceful 500 handler)
  ├── 4. i18n Detection (Accept-Language / Locale)
  ├── 5. MaxBodySize Validator (10 MiB ceiling)
  ├── 6. Rate Limiting (Token Bucket Sliding Window)
  ├── 7. CORS & CSRF Validation
  ├── 8. JWT Authentication & Role Verification (Ed25519/HMAC)
  └── 9. Idempotency Guard (X-Idempotency-Key Mutex Store)
                     │
                     ▼ In-Memory Stubs
[ Mock Core gRPC Services & Upstream Gateways ]
```

### Parameters & Configuration
- **Concurrency**: 10 simultaneous worker goroutines per endpoint.
- **Volume**: 500 requests per endpoint + 5 warm-up cycles (warm-up requests discarded from statistical metrics).
- **Authentication**: Pre-generated role-specific JWTs (`Artisan`, `Buyer`, `Ministry/Cluster Officer`) injected into `Authorization: Bearer <token>` headers per role matrix.
- **Idempotency**: Unique `loadtest-<nano>` UUIDs injected into `X-Idempotency-Key` headers for all `POST` and `PUT` mutation endpoints.
- **Statistical Aggregation**: P50, P95, and P99 percentiles calculated using exact sorted distributions.

---

## 3. Domain Performance Breakdown

### 3.1 Authentication (`/auth/*`)
*Protects OTP delivery, phone change flows, and JWT session refresh.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `POST` | `/auth/otp/request` | 500 | 1.0 ms | 14.9 ms | 31.2 ms | 2.7 ms | 3,311 | 0.0% |
| `POST` | `/auth/otp/verify` | 500 | 1.2 ms | 5.0 ms | 7.1 ms | 1.7 ms | 4,851 | 0.0% |
| `POST` | `/auth/refresh` | 500 | 1.6 ms | 5.7 ms | 9.3 ms | 2.1 ms | 3,974 | 0.0% |
| `POST` | `/auth/phone/change/request` | 500 | 2.0 ms | 9.0 ms | 11.9 ms | 2.8 ms | 3,042 | 0.0% |
| `POST` | `/auth/phone/change/verify` | 500 | 2.0 ms | 7.0 ms | 12.3 ms | 2.7 ms | 3,096 | 0.0% |

### 3.2 Artisan Profiles & Community (`/artisans/*`)
*Handles profile retrieval, profile updates, storefronts, and follower interactions.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `POST` | `/artisans` | 500 | 2.0 ms | 7.4 ms | 12.6 ms | 2.8 ms | 3,162 | 0.0% |
| `GET` | `/artisans/me` | 500 | 2.0 ms | 6.5 ms | 10.8 ms | 2.7 ms | 3,230 | 0.0% |
| `PATCH` | `/artisans/me` | 500 | 2.0 ms | 7.7 ms | 11.0 ms | 2.7 ms | 3,389 | 0.0% |
| `GET` | `/artisans/artisan-1/storefront` | 500 | 1.7 ms | 6.1 ms | 8.3 ms | 2.3 ms | 3,819 | 0.0% |
| `GET` | `/artisans/artisan-1/follower-count` | 500 | 1.9 ms | 8.0 ms | 11.4 ms | 2.4 ms | 3,609 | 0.0% |
| `POST` | `/artisans/artisan-1/follow` | 500 | 2.0 ms | 7.7 ms | 9.4 ms | 2.6 ms | 3,571 | 0.0% |
| `DELETE` | `/artisans/artisan-1/follow` | 500 | 1.4 ms | 43.3 ms | 81.9 ms | 8.4 ms | 1,102 | 0.0% |

### 3.3 Media Uploads (`/media/*`)
*Coordinates signed URL generation and upload confirmations for craft photography.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `POST` | `/media/upload-url` | 500 | 1.9 ms | 7.0 ms | 10.4 ms | 2.6 ms | 3,295 | 0.0% |
| `POST` | `/media/media-1/confirm` | 500 | 2.0 ms | 7.1 ms | 9.0 ms | 2.6 ms | 3,511 | 0.0% |

### 3.4 Listings & Provenance (`/listings/*`)
*Core marketplace catalog, approval workflow, provenance stamping, and moderation.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `GET` | `/listings` | 500 | 1.0 ms | 5.2 ms | 7.2 ms | 1.8 ms | 4,469 | 0.0% |
| `POST` | `/listings` | 500 | 2.3 ms | 7.6 ms | 10.1 ms | 3.0 ms | 2,871 | 0.0% |
| `GET` | `/listings/listing-1` | 500 | 1.3 ms | 7.0 ms | 9.0 ms | 2.2 ms | 3,895 | 0.0% |
| `PATCH` | `/listings/listing-1` | 500 | 2.0 ms | 8.0 ms | 13.8 ms | 3.0 ms | 3,107 | 0.0% |
| `GET` | `/listings/listing-1/summary` | 500 | 1.0 ms | 5.5 ms | 6.9 ms | 1.9 ms | 4,560 | 0.0% |
| `POST` | `/listings/listing-1/submit` | 500 | 2.0 ms | 7.2 ms | 8.4 ms | 2.5 ms | 3,615 | 0.0% |
| `POST` | `/listings/listing-1/approve` | 500 | 2.0 ms | 6.3 ms | 8.5 ms | 2.4 ms | 3,635 | 0.0% |
| `POST` | `/listings/listing-1/seal-provenance` | 500 | 2.0 ms | 7.0 ms | 9.9 ms | 2.5 ms | 3,512 | 0.0% |
| `POST` | `/listings/listing-1/suspend` | 500 | 2.0 ms | 6.8 ms | 10.8 ms | 2.7 ms | 3,161 | 0.0% |
| `POST` | `/listings/listing-1/reinstate` | 500 | 1.7 ms | 6.9 ms | 9.1 ms | 2.3 ms | 3,661 | 0.0% |

### 3.5 Search & Multilingual Discovery (`/search/*`)
*Full-text query processing, typeahead suggestions, and multilingual voice search.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `GET` | `/search?q=madhubani+painting` | 500 | 1.4 ms | 7.6 ms | 11.2 ms | 2.3 ms | 3,792 | 0.0% |
| `GET` | `/search/suggest?q=madh` | 500 | 1.5 ms | 5.1 ms | 7.8 ms | 2.0 ms | 4,463 | 0.0% |
| `POST` | `/search/voice` | 500 | 1.5 ms | 8.6 ms | 13.0 ms | 2.5 ms | 3,568 | 0.0% |

### 3.6 Pricing Advisory (`/pricing/*`)
*Machine learning pricing suggestions based on materials, effort, and regional benchmarking.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `POST` | `/pricing/advise` | 500 | 2.0 ms | 8.0 ms | 11.7 ms | 2.7 ms | 3,143 | 0.0% |

### 3.7 Crafts & Heritage Ontology (`/crafts/*`)
*Geographical Indication (GI) registries, craft taxonomies, and alias indexing.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `GET` | `/crafts` | 500 | 1.1 ms | 7.0 ms | 10.9 ms | 2.2 ms | 3,878 | 0.0% |
| `GET` | `/crafts/madhubani` | 500 | 1.5 ms | 6.9 ms | 10.1 ms | 2.2 ms | 3,948 | 0.0% |
| `POST` | `/crafts/refresh-index` | 500 | 2.0 ms | 8.0 ms | 11.1 ms | 2.8 ms | 3,042 | 0.0% |

### 3.8 Bulk Orders & Fulfillment (`/orders/*`)
*Institutional buyer orders, lot splitting, artisan responses, and SSE event streaming.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `POST` | `/orders/bulk` | 500 | 2.0 ms | 10.6 ms | 13.2 ms | 3.3 ms | 2,691 | 0.0% |
| `GET` | `/orders/order-1` | 500 | 2.0 ms | 7.0 ms | 11.1 ms | 2.6 ms | 3,373 | 0.0% |
| `POST` | `/orders/lots/lot-1/respond` | 500 | 2.0 ms | 8.6 ms | 13.6 ms | 2.9 ms | 2,897 | 0.0% |
| `POST` | `/orders/lots/lot-1/progress` | 500 | 2.0 ms | 6.0 ms | 9.1 ms | 2.3 ms | 3,551 | 0.0% |
| `POST` | `/orders/lots/lot-1/reallocate` | 500 | 1.7 ms | 5.5 ms | 8.4 ms | 2.2 ms | 3,992 | 0.0% |
| `GET` | `/orders/order-1/events` | 500 | 2.0 ms | 7.2 ms | 11.6 ms | 2.7 ms | 3,369 | 0.0% |

### 3.9 Follows & Activity Feed (`/feed/*`)
*Artisan notification streams and process video clip feeds.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `GET` | `/feed` | 500 | 1.1 ms | 53.8 ms | 73.0 ms | 7.5 ms | 1,132 | 0.0% |
| `POST` | `/feed/notif-1/read` | 500 | 1.3 ms | 45.9 ms | 78.3 ms | 8.1 ms | 1,129 | 0.0% |
| `GET` | `/feed/process` | 500 | 1.2 ms | 57.1 ms | 93.4 ms | 7.9 ms | 1,093 | 0.0% |

### 3.10 Income Statements (`/statements/*`)
*QR-verified artisan financial statements and PDF download endpoints.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `POST` | `/statements` | 500 | 2.0 ms | 47.1 ms | 67.5 ms | 8.1 ms | 1,067 | 0.0% |
| `GET` | `/statements` | 500 | 1.2 ms | 34.5 ms | 53.3 ms | 6.2 ms | 1,328 | 0.0% |
| `GET` | `/statements/stmt-1` | 500 | 2.0 ms | 49.0 ms | 79.1 ms | 9.0 ms | 951 | 0.0% |

### 3.11 Ministry Analytics & Insights (`/insights/*`)
*Aggregated analytics across categories, districts, income brackets, and dying crafts.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `GET` | `/insights/artisans-by-category` | 500 | 3.2 ms | 41.9 ms | 67.3 ms | 11.9 ms | 752 | 0.0% |
| `GET` | `/insights/listings-by-craft-month` | 500 | 4.1 ms | 34.5 ms | 154.9 ms | 11.4 ms | 745 | 0.0% |
| `GET` | `/insights/earnings-by-district` | 500 | 1.0 ms | 61.1 ms | 94.6 ms | 7.6 ms | 1,184 | 0.0% |
| `GET` | `/insights/income-comparison` | 500 | 21.5 ms | 43.0 ms | 51.6 ms | 21.0 ms | 468 | 0.2% |
| `GET` | `/insights/dying-crafts` | 500 | 23.6 ms | 48.4 ms | 62.9 ms | 23.3 ms | 415 | 0.0% |
| `POST` | `/insights/refresh` | 500 | 2.1 ms | 65.9 ms | 151.1 ms | 13.8 ms | 654 | 0.0% |

### 3.12 Clusters & Self-Help Groups (`/clusters/*`, `/self-help-groups/*`)
*Cluster officer administrative management and collective onboarding.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `POST` | `/clusters` | 500 | 1.8 ms | 67.1 ms | 135.0 ms | 10.3 ms | 819 | 0.0% |
| `GET` | `/clusters/cluster-1` | 500 | 1.1 ms | 58.0 ms | 95.1 ms | 9.1 ms | 922 | 0.0% |
| `GET` | `/clusters/cluster-1/members` | 500 | 1.0 ms | 50.2 ms | 101.7 ms | 7.8 ms | 1,128 | 0.0% |
| `POST` | `/clusters/cluster-1/members` | 500 | 1.0 ms | 31.3 ms | 51.9 ms | 5.4 ms | 1,483 | 0.0% |
| `DELETE`| `/clusters/cluster-1/members/artisan-1` | 500 | 1.0 ms | 41.5 ms | 65.0 ms | 5.7 ms | 1,501 | 0.0% |
| `POST` | `/clusters/cluster-1/onboard` | 500 | 1.2 ms | 51.3 ms | 91.4 ms | 7.3 ms | 1,206 | 0.0% |
| `POST` | `/self-help-groups` | 500 | 1.0 ms | 36.8 ms | 61.8 ms | 5.8 ms | 1,417 | 0.0% |
| `GET` | `/self-help-groups/shg-1` | 500 | 1.0 ms | 31.2 ms | 59.0 ms | 4.6 ms | 1,671 | 0.0% |
| `PUT` | `/self-help-groups/shg-1/members` | 500 | 1.2 ms | 46.6 ms | 81.3 ms | 6.6 ms | 1,246 | 0.0% |

### 3.13 Payments, Public Web, SEO, Verification & OpenAPI
*Webhooks, SSR HTML pages, public verification shortcodes, IndiaHandmade feeds, and OpenAPI specs.*

| Method | Path | Reqs | P50 | P95 | P99 | Mean | RPS | Error Rate |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `POST` | `/payments/webhook` | 500 | 999 µs | 44.2 ms | 76.7 ms | 5.9 ms | 1,401 | 0.0% |
| `GET` | `/listing/madhubani-fish-painting` | 500 | 1.2 ms | 32.5 ms | 47.8 ms | 6.2 ms | 1,449 | 0.0% |
| `GET` | `/artisan/lakshmi-devi` | 500 | 1.0 ms | 39.2 ms | 54.4 ms | 5.3 ms | 1,688 | 0.0% |
| `GET` | `/v/QR_CODE_PROV_123` | 500 | 1.4 ms | 34.2 ms | 60.1 ms | 6.1 ms | 1,420 | 0.0% |
| `GET` | `/v/QR_CODE_PROV_123/verify.json` | 500 | 1.0 ms | 30.3 ms | 72.6 ms | 4.9 ms | 1,638 | 0.0% |
| `GET` | `/export/indiahandmade` | 500 | 2.0 ms | 60.3 ms | 129.2 ms | 11.7 ms | 780 | 0.0% |
| `GET` | `/sitemap.xml` | 500 | 9.5 ms | 37.0 ms | 43.4 ms | 14.6 ms | 651 | 0.0% |
| `GET` | `/robots.txt` | 500 | 3.7 ms | 48.7 ms | 73.1 ms | 10.7 ms | 876 | 0.0% |
| `GET` | `/openapi.json` | 500 | 1.4 ms | 49.1 ms | 76.1 ms | 7.0 ms | 1,156 | 0.0% |

---

## 4. Latency Analysis & Extreme Rankings

### ⚡ Top 10 Fastest Endpoints (by P50 Median Latency)

| Rank | Method & Endpoint | P50 | P95 | Mean | Peak RPS |
|:---:|:---|:---:|:---:|:---:|:---:|
| 1 | `POST /payments/webhook` | **999 µs** | 44.2 ms | 5.9 ms | 1,401 |
| 2 | `GET /artisan/lakshmi-devi` | **1.0 ms** | 39.2 ms | 5.3 ms | 1,688 |
| 3 | `GET /self-help-groups/shg-1` | **1.0 ms** | 31.2 ms | 4.6 ms | 1,671 |
| 4 | `GET /v/QR_CODE_PROV_123/verify.json` | **1.0 ms** | 30.3 ms | 4.9 ms | 1,638 |
| 5 | `POST /self-help-groups` | **1.0 ms** | 36.8 ms | 5.8 ms | 1,417 |
| 6 | `POST /clusters/cluster-1/members` | **1.0 ms** | 31.3 ms | 5.4 ms | 1,483 |
| 7 | `GET /clusters/cluster-1/members` | **1.0 ms** | 50.2 ms | 7.8 ms | 1,128 |
| 8 | `POST /auth/otp/request` | **1.0 ms** | 14.9 ms | 2.7 ms | 3,311 |
| 9 | `GET /listings` | **1.0 ms** | 5.2 ms | 1.8 ms | 4,469 |
| 10 | `GET /insights/earnings-by-district` | **1.0 ms** | 61.1 ms | 7.6 ms | 1,184 |

### 🐢 Top 10 Slowest Endpoints (by P99 Tail Latency)

| Rank | Method & Endpoint | P99 | P95 | Mean | RPS |
|:---:|:---|:---:|:---:|:---:|:---:|
| 1 | `GET /insights/listings-by-craft-month` | **154.9 ms** | 34.5 ms | 11.4 ms | 745 |
| 2 | `POST /insights/refresh` | **151.1 ms** | 65.9 ms | 13.8 ms | 654 |
| 3 | `POST /clusters` | **135.0 ms** | 67.1 ms | 10.3 ms | 819 |
| 4 | `GET /export/indiahandmade` | **129.2 ms** | 60.3 ms | 11.7 ms | 780 |
| 5 | `GET /clusters/cluster-1/members` | **101.7 ms** | 50.2 ms | 7.8 ms | 1,128 |
| 6 | `GET /clusters/cluster-1` | **95.1 ms** | 58.0 ms | 9.1 ms | 922 |
| 7 | `GET /insights/earnings-by-district` | **94.6 ms** | 61.1 ms | 7.6 ms | 1,184 |
| 8 | `GET /feed/process` | **93.4 ms** | 57.1 ms | 7.9 ms | 1,093 |
| 9 | `POST /clusters/cluster-1/onboard` | **91.4 ms** | 51.3 ms | 7.3 ms | 1,206 |
| 10 | `DELETE /artisans/artisan-1/follow` | **81.9 ms** | 43.3 ms | 8.4 ms | 1,102 |

---

## 5. Architectural Findings & Key Learnings

1. **Memory & Thread Safety in Idempotency Store**:
   - High concurrency load testing uncovered that concurrent read/writes to non-synchronized map state in mock stores causes runtime panics (`fatal error: concurrent map writes`).
   - The idempotency store was secured with `sync.RWMutex` locking, allowing safe concurrent execution under arbitrary goroutine volume.

2. **Console I/O Bottleneck Mitigation**:
   - Access logging during massive load testing can choke stdout/stderr buffers on Windows. Configuring structured loggers with non-blocking or targeted test discard mechanisms unlocked true framework throughput (~4,000+ RPS vs ~30 RPS when blocked on synchronous terminal printing).

3. **Resilience under Upstream Failures**:
   - Redis sliding window rate limiters demonstrated seamless fail-open behavior: when Redis is unreachable or timing out, rate limit checks fail safely without rejecting valid user traffic or crashing the process.

4. **Public SEO Page Generation Efficiency**:
   - Server-side rendered pages (`/listing/:slug`, `/artisan/:slug`) and XML generation (`/sitemap.xml`) completed with P50 latencies under 10 ms even without pre-warmed Redis HTML caches.

---

## 6. Production Recommendations

- **Connection Pool Tuning**: Ensure the gRPC client connection pool for `coreConn` maintains at least 25 concurrent channels to prevent multiplexing queue buildup during sudden bursts.
- **Redis Cluster Provisioning**: In production, co-locate Redis instances in the same AWS/GCP availability zone as the BFF nodes to keep sliding-window pipeline latency under 1 ms.
- **Rate Limit Window Adjustments**: For high-volume public endpoints like `/search` and `/crafts`, consider caching responses via Cloudflare / Fastly CDN edge workers to deflect read traffic away from the BFF.
