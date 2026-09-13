# Kalakriti REST API

**Base URL (local):** `http://localhost:8000/api/v1` (direct to bff), or
`http://localhost/api/v1` (through the NGINX `web` container, same routes).

**Authoritative schema:** the bff embeds and self-serves its OpenAPI 3 spec at
`GET /api/v1/openapi.json` — generated from the actual route table, so it
never drifts the way a hand-maintained doc can. This page is a map of what
exists and why; treat the spec as ground truth for exact request/response
shapes, and `services/bff/internal/bff/handler/*.go` as ground truth for
behavior.

All responses are JSON. gRPC is used between the bff and every backend
service (`core-svc`, `search-svc`, `collab-svc`, `channel-svc`, `insight-svc`)
— the bff is the only HTTP surface of the whole system.

---

## Auth

Three unauthenticated endpoints carry the whole login handshake — there is no
separate "challenge ID" round trip at the REST layer (that detail lives only
in the internal `identity.v1.IdentityService` gRPC contract the bff calls on
your behalf).

### `POST /auth/otp/request`
```json
{ "phone": "+919876543210" }
```
→ `200`
```json
{ "status": "otp_sent", "message": "If this account exists, an OTP has been dispatched." }
```
Always returns this same shape whether or not the phone is registered, to
avoid leaking which numbers exist. Rate-limited to 5 requests/10min per
caller. With `AUTH_DEV_OTP_ENABLED=true` (the local-dev default), the OTP is
always `000000`.

### `POST /auth/otp/verify`
```json
{ "phone": "+919876543210", "otp": "000000" }
```
→ `200`
```json
{ "access_token": "eyJ...", "refresh_token": "eyJ..." }
```
Repeated failures lock the account temporarily (`pkg` account-lockout via
Redis); a locked attempt returns `503` with a `retry in N seconds` message.

### `POST /auth/refresh`
```json
{ "refresh_token": "eyJ..." }
```
→ `200`
```json
{ "access_token": "eyJ..." }
```

Every other route below requires `Authorization: Bearer <access_token>`
unless marked **public**.

---

## Route map

Grouped as `server.go` mounts them. "Public" = no bearer token required —
see `services/bff/internal/bff/server.go` for the exact list; this mirrors
it, not the other way around, so re-check that file if the two disagree.

### Public reads (browsing, no auth)
| Method | Path | Purpose |
|---|---|---|
| GET | `/search` | Hybrid BM25 + vector search |
| GET | `/search/suggest` | Autocomplete |
| POST | `/search/voice` | Voice search (audio in, transcript + results out) |
| GET | `/listings` | Page listings (buyer/anonymous view is clamped to published only) |
| GET | `/listings/:id` | One listing (non-published requires owner/officer) |
| GET | `/listings/:id/summary` | Card-shape summary for search/home results |
| GET | `/crafts` | Full craft ontology |
| GET | `/crafts/:slug` | One craft, by slug or id |
| GET | `/artisans/:id/storefront` | Artisan public profile |
| GET | `/artisans/:id/follower-count` | Follower count |
| GET | `/feed/process` | Vertical process-provenance clip feed |

### Server-rendered / SEO (public, HTML or JSON)
| Method | Path | Purpose |
|---|---|---|
| GET | `/listing/:slug` | SEO listing page |
| GET | `/artisan/:slug` | SEO artisan page |
| GET | `/v/:code` | Public provenance verification page |
| GET | `/v/:code/verify.json` | Same, as JSON |
| GET | `/export/indiahandmade` | Outbound catalog feed |
| GET | `/sitemap.xml`, `/robots.txt` | Standard crawler files |

### Auth & profile (bearer required unless noted)
| Method | Path | Purpose |
|---|---|---|
| POST | `/artisans` | Register artisan profile (idempotent) |
| GET / PATCH | `/artisans/me` | Own profile |
| POST | `/auth/phone/change/request` \| `/verify` | Change registered phone |

### Media (3-step upload: ticket → PUT to MinIO → confirm)
| Method | Path | Purpose |
|---|---|---|
| POST | `/media/upload-url` | Request a presigned PUT URL |
| POST | `/media/:id/confirm` | Confirm upload; triggers the AI cataloging pipeline |

### Listings (mutations)
| Method | Path | Purpose |
|---|---|---|
| POST | `/listings` | Create (idempotent) |
| PATCH | `/listings/:id` | Update a draft |
| POST | `/listings/:id/submit` | Move to pending-approval |
| POST | `/listings/:id/approve` | Artisan accepts AI-generated copy → published |
| POST | `/listings/:id/seal-provenance` | Freeze provenance evidence, mint certificate |

### Pricing & orders
| Method | Path | Purpose |
|---|---|---|
| POST | `/pricing/advise` | ML pricing suggestion |
| POST | `/orders/bulk` | Create a bulk order (idempotent) |
| GET | `/orders/:id` | Order + lot allocation detail |
| GET | `/orders/:id/events` | `text/event-stream` live order updates (SSE) |
| POST | `/orders/lots/:id/respond` | Artisan accepts/declines a lot invitation |
| POST | `/orders/lots/:id/progress` | Report production progress |
| POST | `/orders/lots/:id/reallocate` | Request reallocation |

### Social
| Method | Path | Purpose |
|---|---|---|
| POST / DELETE | `/artisans/:id/follow` | Follow / unfollow |
| GET | `/feed` | Own notification/follow feed |
| POST | `/feed/:id/read` | Mark one feed item read |

### Income statements
| Method | Path | Purpose |
|---|---|---|
| POST | `/statements` | Generate a signed PDF income statement (idempotent) |
| GET | `/statements/:id` | Fetch one |
| GET | `/statements` | List own statements |

There is no separate `/statements/{code}/verify` route — a statement's QR
resolves through the same provenance-style verification path
(`GET /v/:code`), not a statements-specific endpoint.

### Ministry insights (`MINISTRY` role, enforced in-handler)
| Method | Path |
|---|---|
| GET | `/insights/artisans-by-category` |
| GET | `/insights/listings-by-craft-month` |
| GET | `/insights/earnings-by-district` |
| GET | `/insights/income-comparison` |
| GET | `/insights/dying-crafts` |
| POST | `/insights/refresh` |

### Cluster / SHG administration (`CLUSTER_OFFICER` / `MINISTRY`)
| Method | Path |
|---|---|
| POST | `/clusters` |
| GET | `/clusters/:id` |
| GET / POST | `/clusters/:id/members` |
| DELETE | `/clusters/:id/members/:artisanId` |
| POST | `/clusters/:id/onboard` |
| POST | `/self-help-groups` |
| GET | `/self-help-groups/:id` |
| PUT | `/self-help-groups/:id/members` |

### Moderation (`CLUSTER_OFFICER` / `MINISTRY`)
| Method | Path |
|---|---|
| POST | `/listings/:id/suspend` |
| POST | `/listings/:id/reinstate` |

### Craft ontology admin (`MINISTRY` only)
| Method | Path |
|---|---|
| POST | `/crafts/refresh-index` |

### Webhooks
| Method | Path |
|---|---|
| POST | `/payments/webhook` | HMAC-SHA256 verified, public endpoint |

---

## Error format

```json
{ "error": "unauthenticated", "message": "invalid or missing access token: unauthenticated" }
```
Two fields only — `error` (a short machine-readable code) and `message`. No
`request_id` or `timestamp` field in the body (trace correlation is via
OpenTelemetry, not the error payload). A `500` always renders `message` as
the generic `"internal error"`, never the real internal error text.

Status codes follow the usual mapping: `400` invalid input, `401`
unauthenticated, `403` forbidden, `404` not found, `409` conflict, `429` rate
limited, `503` a downstream dependency is unavailable.

---

## Idempotency

The server reads the key from **`Idempotency-Key`**, on every route wrapped
with `withIdempotency` in `server.go` (artisan registration, listing/order/
statement/cluster/SHG creation, moderation actions). A repeated key within
the configured TTL replays the original response instead of re-executing the
write. `web/packages/api/src/transport.ts` sends the same header name.

---

## Rate limiting

Two layers, both Redis-backed:
- A global per-IP / per-authenticated-principal limiter on the whole
  `/api/v1` group (`middleware.RateLimit`, configured via
  `RateLimitPerIP`/`RateLimitPerPrincipal`/`RateLimitWindow`).
- An additional strict limit on `POST /auth/otp/request` specifically: 5
  requests per 10 minutes per caller (`middleware.RateLimitEndpoint`).

A `429` does not currently document a `Retry-After` header or
`X-RateLimit-*` response headers in the handler code — check
`pkg/httpx`/`middleware.RateLimit` directly if a client needs to react to
those rather than assuming they're present.

---

## Trying it locally

```sh
# health
curl http://localhost:8000/healthz

# auth
curl -X POST http://localhost:8000/api/v1/auth/otp/request \
  -H 'Content-Type: application/json' -d '{"phone":"+919876543210"}'

curl -X POST http://localhost:8000/api/v1/auth/otp/verify \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+919876543210","otp":"000000"}'
# {"access_token":"...","refresh_token":"..."}

export TOKEN="<access_token from above>"

# public browsing, no token needed
curl http://localhost:8000/api/v1/crafts
curl "http://localhost:8000/api/v1/listings?page_size=10"

# the live, generated spec
curl http://localhost:8000/api/v1/openapi.json | jq .
```
