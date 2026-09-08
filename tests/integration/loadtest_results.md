# Kalakriti BFF Load Test Results

**Date**: 2026-09-06 20:33:21 IST  
**Endpoints tested**: 67  
**Concurrency**: 10 workers per endpoint  
**Requests per worker**: 50  
**Total requests**: 33500  
**Total errors**: 1 (0.00%)  
**Test mode**: In-process httptest.Server with stubbed gRPC backends  

---

## Auth

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `POST` | `/auth/otp/request` | 500 | 1.0ms | 14.9ms | 31.2ms | 2.7ms | 3311 | 0.0% |
| `POST` | `/auth/otp/verify` | 500 | 1.2ms | 5.0ms | 7.1ms | 1.7ms | 4851 | 0.0% |
| `POST` | `/auth/refresh` | 500 | 1.6ms | 5.7ms | 9.3ms | 2.1ms | 3974 | 0.0% |
| `POST` | `/auth/phone/change/request` | 500 | 2.0ms | 9.0ms | 11.9ms | 2.8ms | 3042 | 0.0% |
| `POST` | `/auth/phone/change/verify` | 500 | 2.0ms | 7.0ms | 12.3ms | 2.7ms | 3096 | 0.0% |

## Artisan

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `POST` | `/artisans` | 500 | 2.0ms | 7.4ms | 12.6ms | 2.8ms | 3162 | 0.0% |
| `GET` | `/artisans/me` | 500 | 2.0ms | 6.5ms | 10.8ms | 2.7ms | 3230 | 0.0% |
| `PATCH` | `/artisans/me` | 500 | 2.0ms | 7.7ms | 11.0ms | 2.7ms | 3389 | 0.0% |
| `GET` | `/artisans/artisan-1/storefront` | 500 | 1.7ms | 6.1ms | 8.3ms | 2.3ms | 3819 | 0.0% |
| `GET` | `/artisans/artisan-1/follower-count` | 500 | 1.9ms | 8.0ms | 11.4ms | 2.4ms | 3609 | 0.0% |
| `POST` | `/artisans/artisan-1/follow` | 500 | 2.0ms | 7.7ms | 9.4ms | 2.6ms | 3571 | 0.0% |
| `DELETE` | `/artisans/artisan-1/follow` | 500 | 1.4ms | 43.3ms | 81.9ms | 8.4ms | 1102 | 0.0% |

## Media

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `POST` | `/media/upload-url` | 500 | 1.9ms | 7.0ms | 10.4ms | 2.6ms | 3295 | 0.0% |
| `POST` | `/media/media-1/confirm` | 500 | 2.0ms | 7.1ms | 9.0ms | 2.6ms | 3511 | 0.0% |

## Listings

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `GET` | `/listings` | 500 | 1.0ms | 5.2ms | 7.2ms | 1.8ms | 4469 | 0.0% |
| `POST` | `/listings` | 500 | 2.3ms | 7.6ms | 10.1ms | 3.0ms | 2871 | 0.0% |
| `GET` | `/listings/listing-1` | 500 | 1.3ms | 7.0ms | 9.0ms | 2.2ms | 3895 | 0.0% |
| `PATCH` | `/listings/listing-1` | 500 | 2.0ms | 8.0ms | 13.8ms | 3.0ms | 3107 | 0.0% |
| `GET` | `/listings/listing-1/summary` | 500 | 1.0ms | 5.5ms | 6.9ms | 1.9ms | 4560 | 0.0% |
| `POST` | `/listings/listing-1/submit` | 500 | 2.0ms | 7.2ms | 8.4ms | 2.5ms | 3615 | 0.0% |
| `POST` | `/listings/listing-1/approve` | 500 | 2.0ms | 6.3ms | 8.5ms | 2.4ms | 3635 | 0.0% |
| `POST` | `/listings/listing-1/seal-provenance` | 500 | 2.0ms | 7.0ms | 9.9ms | 2.5ms | 3512 | 0.0% |
| `POST` | `/listings/listing-1/suspend` | 500 | 2.0ms | 6.8ms | 10.8ms | 2.7ms | 3161 | 0.0% |
| `POST` | `/listings/listing-1/reinstate` | 500 | 1.7ms | 6.9ms | 9.1ms | 2.3ms | 3661 | 0.0% |

## Search

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `GET` | `/search?q=madhubani+painting` | 500 | 1.4ms | 7.6ms | 11.2ms | 2.3ms | 3792 | 0.0% |
| `GET` | `/search/suggest?q=madh` | 500 | 1.5ms | 5.1ms | 7.8ms | 2.0ms | 4463 | 0.0% |
| `POST` | `/search/voice` | 500 | 1.5ms | 8.6ms | 13.0ms | 2.5ms | 3568 | 0.0% |

## Pricing

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `POST` | `/pricing/advise` | 500 | 2.0ms | 8.0ms | 11.7ms | 2.7ms | 3143 | 0.0% |

## Crafts

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `GET` | `/crafts` | 500 | 1.1ms | 7.0ms | 10.9ms | 2.2ms | 3878 | 0.0% |
| `GET` | `/crafts/madhubani` | 500 | 1.5ms | 6.9ms | 10.1ms | 2.2ms | 3948 | 0.0% |
| `POST` | `/crafts/refresh-index` | 500 | 2.0ms | 8.0ms | 11.1ms | 2.8ms | 3042 | 0.0% |

## Orders

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `POST` | `/orders/bulk` | 500 | 2.0ms | 10.6ms | 13.2ms | 3.3ms | 2691 | 0.0% |
| `GET` | `/orders/order-1` | 500 | 2.0ms | 7.0ms | 11.1ms | 2.6ms | 3373 | 0.0% |
| `POST` | `/orders/lots/lot-1/respond` | 500 | 2.0ms | 8.6ms | 13.6ms | 2.9ms | 2897 | 0.0% |
| `POST` | `/orders/lots/lot-1/progress` | 500 | 2.0ms | 6.0ms | 9.1ms | 2.3ms | 3551 | 0.0% |
| `POST` | `/orders/lots/lot-1/reallocate` | 500 | 1.7ms | 5.5ms | 8.4ms | 2.2ms | 3992 | 0.0% |
| `GET` | `/orders/order-1/events` | 500 | 2.0ms | 7.2ms | 11.6ms | 2.7ms | 3369 | 0.0% |

## Follow / Feed

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `GET` | `/feed` | 500 | 1.1ms | 53.8ms | 73.0ms | 7.5ms | 1132 | 0.0% |
| `POST` | `/feed/notif-1/read` | 500 | 1.3ms | 45.9ms | 78.3ms | 8.1ms | 1129 | 0.0% |
| `GET` | `/feed/process` | 500 | 1.2ms | 57.1ms | 93.4ms | 7.9ms | 1093 | 0.0% |

## Statements

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `POST` | `/statements` | 500 | 2.0ms | 47.1ms | 67.5ms | 8.1ms | 1067 | 0.0% |
| `GET` | `/statements` | 500 | 1.2ms | 34.5ms | 53.3ms | 6.2ms | 1328 | 0.0% |
| `GET` | `/statements/stmt-1` | 500 | 2.0ms | 49.0ms | 79.1ms | 9.0ms | 951 | 0.0% |

## Insights

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `GET` | `/insights/artisans-by-category` | 500 | 3.2ms | 41.9ms | 67.3ms | 11.9ms | 752 | 0.0% |
| `GET` | `/insights/listings-by-craft-month` | 500 | 4.1ms | 34.5ms | 154.9ms | 11.4ms | 745 | 0.0% |
| `GET` | `/insights/earnings-by-district` | 500 | 1.0ms | 61.1ms | 94.6ms | 7.6ms | 1184 | 0.0% |
| `GET` | `/insights/income-comparison` | 500 | 21.5ms | 43.0ms | 51.6ms | 21.0ms | 468 | 0.2% |
| `GET` | `/insights/dying-crafts` | 500 | 23.6ms | 48.4ms | 62.9ms | 23.3ms | 415 | 0.0% |
| `POST` | `/insights/refresh` | 500 | 2.1ms | 65.9ms | 151.1ms | 13.8ms | 654 | 0.0% |

## Clusters

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `POST` | `/clusters` | 500 | 1.8ms | 67.1ms | 135.0ms | 10.3ms | 819 | 0.0% |
| `GET` | `/clusters/cluster-1` | 500 | 1.1ms | 58.0ms | 95.1ms | 9.1ms | 922 | 0.0% |
| `GET` | `/clusters/cluster-1/members` | 500 | 1.0ms | 50.2ms | 101.7ms | 7.8ms | 1128 | 0.0% |
| `POST` | `/clusters/cluster-1/members` | 500 | 1.0ms | 31.3ms | 51.9ms | 5.4ms | 1483 | 0.0% |
| `DELETE` | `/clusters/cluster-1/members/artisan-1` | 500 | 1.0ms | 41.5ms | 65.0ms | 5.7ms | 1501 | 0.0% |
| `POST` | `/clusters/cluster-1/onboard` | 500 | 1.2ms | 51.3ms | 91.4ms | 7.3ms | 1206 | 0.0% |

## Self-Help Groups

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `POST` | `/self-help-groups` | 500 | 1.0ms | 36.8ms | 61.8ms | 5.8ms | 1417 | 0.0% |
| `GET` | `/self-help-groups/shg-1` | 500 | 1.0ms | 31.2ms | 59.0ms | 4.6ms | 1671 | 0.0% |
| `PUT` | `/self-help-groups/shg-1/members` | 500 | 1.2ms | 46.6ms | 81.3ms | 6.6ms | 1246 | 0.0% |

## Payments

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `POST` | `/payments/webhook` | 500 | 999µs | 44.2ms | 76.7ms | 5.9ms | 1401 | 0.0% |

## Public Web / SEO / Verification

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `GET` | `/listing/madhubani-fish-painting` | 500 | 1.2ms | 32.5ms | 47.8ms | 6.2ms | 1449 | 0.0% |
| `GET` | `/artisan/lakshmi-devi` | 500 | 1.0ms | 39.2ms | 54.4ms | 5.3ms | 1688 | 0.0% |
| `GET` | `/v/QR_CODE_PROV_123` | 500 | 1.4ms | 34.2ms | 60.1ms | 6.1ms | 1420 | 0.0% |
| `GET` | `/v/QR_CODE_PROV_123/verify.json` | 500 | 1.0ms | 30.3ms | 72.6ms | 4.9ms | 1638 | 0.0% |
| `GET` | `/export/indiahandmade` | 500 | 2.0ms | 60.3ms | 129.2ms | 11.7ms | 780 | 0.0% |
| `GET` | `/sitemap.xml` | 500 | 9.5ms | 37.0ms | 43.4ms | 14.6ms | 651 | 0.0% |
| `GET` | `/robots.txt` | 500 | 3.7ms | 48.7ms | 73.1ms | 10.7ms | 876 | 0.0% |

## Other

| Method | Endpoint | Reqs | P50 | P95 | P99 | Mean | RPS | Err% |
|--------|----------|------|-----|-----|-----|------|-----|------|
| `GET` | `/openapi.json` | 500 | 1.4ms | 49.1ms | 76.1ms | 7.0ms | 1156 | 0.0% |

## 🐢 Top 10 Slowest Endpoints (by P99)

| Rank | Endpoint | P99 | P95 | Mean | RPS |
|------|----------|-----|-----|------|-----|
| 1 | `GET /insights/listings-by-craft-month` | 154.9ms | 34.5ms | 11.4ms | 745 |
| 2 | `POST /insights/refresh` | 151.1ms | 65.9ms | 13.8ms | 654 |
| 3 | `POST /clusters` | 135.0ms | 67.1ms | 10.3ms | 819 |
| 4 | `GET /export/indiahandmade` | 129.2ms | 60.3ms | 11.7ms | 780 |
| 5 | `GET /clusters/cluster-1/members` | 101.7ms | 50.2ms | 7.8ms | 1128 |
| 6 | `GET /clusters/cluster-1` | 95.1ms | 58.0ms | 9.1ms | 922 |
| 7 | `GET /insights/earnings-by-district` | 94.6ms | 61.1ms | 7.6ms | 1184 |
| 8 | `GET /feed/process` | 93.4ms | 57.1ms | 7.9ms | 1093 |
| 9 | `POST /clusters/cluster-1/onboard` | 91.4ms | 51.3ms | 7.3ms | 1206 |
| 10 | `DELETE /artisans/artisan-1/follow` | 81.9ms | 43.3ms | 8.4ms | 1102 |

## ⚡ Top 10 Fastest Endpoints (by P50)

| Rank | Endpoint | P50 | P95 | Mean | RPS |
|------|----------|-----|-----|------|-----|
| 1 | `POST /payments/webhook` | 999µs | 44.2ms | 5.9ms | 1401 |
| 2 | `GET /artisan/lakshmi-devi` | 1.0ms | 39.2ms | 5.3ms | 1688 |
| 3 | `GET /self-help-groups/shg-1` | 1.0ms | 31.2ms | 4.6ms | 1671 |
| 4 | `GET /v/QR_CODE_PROV_123/verify.json` | 1.0ms | 30.3ms | 4.9ms | 1638 |
| 5 | `POST /self-help-groups` | 1.0ms | 36.8ms | 5.8ms | 1417 |
| 6 | `POST /clusters/cluster-1/members` | 1.0ms | 31.3ms | 5.4ms | 1483 |
| 7 | `GET /clusters/cluster-1/members` | 1.0ms | 50.2ms | 7.8ms | 1128 |
| 8 | `POST /auth/otp/request` | 1.0ms | 14.9ms | 2.7ms | 3311 |
| 9 | `GET /listings` | 1.0ms | 5.2ms | 1.8ms | 4469 |
| 10 | `GET /insights/earnings-by-district` | 1.0ms | 61.1ms | 7.6ms | 1184 |

## ⚠️ Endpoints With Errors

| Endpoint | Reqs | Errors | Non-2xx | Error Rate |
|----------|------|--------|---------|------------|
| `GET /insights/income-comparison` | 500 | 1 | 0 | 0.2% |

## Notes

- **Test environment**: In-process `httptest.Server` with stubbed gRPC backend services.
- **What's measured**: BFF HTTP layer — routing, middleware (auth, rate-limiting, CORS, i18n, idempotency), JSON serialization/deserialization, and response assembly.
- **What's NOT measured**: Database queries, gRPC round-trips to real services, object storage, Kafka producers — all stubbed.
- **Error codes 401/403/404/409**: Counted as handled successes — the BFF correctly rejected the request.
- **Idempotency**: POST/PUT endpoints send unique `X-Idempotency-Key` per request.
