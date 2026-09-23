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
| GET | `/verify/certificate/:short_code` | Public Digital Ready certificate check (MoSJE tier 4) |

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
| Method | Path | Purpose |
|---|---|---|
| POST | `/payments/webhook` | Inbound payment callback, HMAC-SHA256 verified, public |
| POST / GET | `/webhooks/subscriptions` | Create / list your outbound webhook subscriptions |
| DELETE | `/webhooks/subscriptions/:id` | Remove one (filters on the caller's own subscriber id) |

Outbound delivery (HMAC signing, retry-with-backoff) is `pkg/webhook.Manager` /
`cmd/webhook-worker`; these three routes are its only REST surface.

### MoSJE tier 4 (`services/bff/internal/bff/mosje/`)
Finance-corporation loan linkage, self-reported income, the ministry impact
dashboard, staff accounts, assisted ("helping") mode and digital-literacy
lessons/certificates. Every route below is under `/api/v1` and requires a
bearer token except the one marked public; see the route map's comments in
`mosje/routes.go` for exactly who (artisan / FIELD_AGENT / CLUSTER_OFFICER /
MINISTRY) may call each one — that file is the source of truth, not this list.

| Method | Path | Purpose |
|---|---|---|
| GET / POST | `/finance/links` | List own / link a loan (consent_given must be true) |
| PATCH / DELETE | `/finance/links/:id` | Update / remove (delete = consent withdrawal) |
| GET | `/finance/coverage` | Whether this month's sales cover the EMI |
| GET | `/admin/finance/links` | Review queue (officer/ministry) |
| POST | `/admin/finance/links/:id/review` | Mark verified or rejected |
| GET / PUT | `/income/baseline` | Self-reported pre-Kalakriti monthly income |
| GET / POST | `/income/sales` | Offline (fair/market/direct) sale log |
| DELETE | `/income/sales/:id` | Remove one logged sale |
| GET | `/income/summary` | Own income card (now vs. before, insufficient-data reason) |
| GET | `/impact/summary` \| `/by-group` \| `/sales-mix` \| `/finance-coverage` \| `/literacy-funnel` | Ministry dashboard (MINISTRY / scoped CLUSTER_OFFICER); groups of fewer than 5 artisans are suppressed, never shown as 0 |
| GET | `/impact/export.csv` | CSV export of the current filtered view |
| GET | `/staff/me` | Own staff account |
| GET / POST | `/admin/staff` | List / create staff accounts (MINISTRY) |
| POST | `/admin/staff/:id/active` | Activate / deactivate |
| GET | `/admin/staff/productivity` | Field-agent onboarding/listing counts |
| POST | `/assisted/consent/start` \| `/link` \| `/voice-consent` | Agent-assisted onboarding: OTP or recorded-voice consent |
| GET | `/assisted/artisans` | Artisans a FIELD_AGENT currently helps |
| GET | `/assisted/review` \| POST `/assisted/review/:id` | Voice-consent review queue |
| GET | `/media/:id/url` | Time-limited download URL (e.g. to listen to a voice consent); owner or officer/ministry only |
| GET | `/helpers` \| DELETE `/helpers/:id` | An artisan's own list of helpers / revoke one |
| GET | `/listings/:id/helper` | Which agent (if any) last touched this listing |
| GET | `/learn/lessons` | The 8 digital-skills lessons |
| POST | `/learn/lessons/:code/progress` | Record practice/quiz progress |
| GET / POST | `/learn/certificate` | Fetch / issue the Digital Ready certificate |

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
