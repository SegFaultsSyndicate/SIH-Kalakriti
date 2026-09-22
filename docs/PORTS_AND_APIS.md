# Ports & APIs Reference

**Last Updated:** 2026-09-15
**Source of truth used to write this doc:** `docker-compose.yml`, each service's
`cmd/<svc>/main.go`, `services/bff/internal/bff/server.go`, `services/bff/openapi.json`.

This is the map to reach for when you need "what port is X on" or "what does
route Y actually require." For exact request/response JSON shapes, don't hand-copy
examples from old docs — hit the live spec at `GET /api/v1/openapi.json` (served by
the running bff) instead; it's generated from the real handler code and won't drift.

`docker-compose.full.yml` and `deploy/k8s/configmap.yaml` are **not** sources of
truth — both are known-stale (wrong ports, wrong env var names, services that don't
exist in this repo). See `CLAUDE.md` for the history. Only `docker-compose.yml` is
maintained.

---

## 1. Port map (`docker-compose.yml`)

| Service | Host port(s) | Container port(s) | Protocol |
|---|---|---|---|
| **web** (NGINX) | `80` | `80` | HTTP — serves all 3 SvelteKit apps + proxies `/api/v1` to bff |
| **bff** | `8000` | `8000` | HTTP (REST/JSON) |
| **core-svc** | `50051`, `8081` | `50051`, `8081` | gRPC, HTTP (health only) |
| **search-svc** | `50052`, `8082` | `50052`, `8082` | gRPC, HTTP (health only) |
| **collab-svc** | `50053`, `8083` | `50053`, `8083` | gRPC, HTTP (health only: `/healthz`, `/readyz`) |
| **channel-svc** | `9096`, `8084` | `9096`, `8084` | gRPC, HTTP (health **and** the `/export/indiahandmade` feed bff proxies to) |
| **insight-svc** | `8085` | `8085` | gRPC only — **no HTTP listener at all**, not even health |
| **ml-svc** | `50055`, `9097` | `50055`, `9095` | gRPC, Prometheus metrics (host `9097` deliberately offset from bff's reserved `9095`) |
| **postgres** | `5432` | `5432` | Postgres 18 + pgvector |
| **redis** | `6379` | `6379` | Redis 7 |
| **kafka** | `9092` | `9092` | KRaft single-broker (controller port `9093` is internal-only) |
| **minio** | `9000`, `9001` | `9000`, `9001` | S3 API, web console |
| **minio-init** | — | — | one-shot `mc mb` job that creates the `kalakriti` bucket, then exits 0 |
| **jaeger** | `16686` | `16686` | Trace UI only — OTLP gRPC (`4317`) is internal-only, reachable as `jaeger:4317` |
| **prometheus** | `9090` | `9090` | Metrics UI/API |

**Note on host-port naming collisions in code (not a bug, just easy to misread):**
collab-svc's and channel-svc's own HTTP-health default ports collide at the Go
level (`:8083` is the code default for *both* — docker-compose disambiguates by
explicitly setting channel-svc's to `:8084`). Don't "fix" the code defaults to
look consistent; they're overridden correctly where it matters.

`docker-compose.gpu.yml` is an **optional, non-merged override** — it is never
applied by `make up`/`make demo-up`. Use it explicitly when you have an NVIDIA
GPU + nvidia-container-toolkit:
```sh
docker compose -f docker-compose.yml -f docker-compose.gpu.yml up -d ml-svc
```
It adds a GPU device reservation for ml-svc and pins the embedding/reranking
models to CPU (`ML_SVC_TEXT_EMBEDDING_DEVICE=cpu`, `ML_SVC_TEXT_RERANKING_DEVICE=cpu`)
so the VLM has the card to itself. No port changes.

---

## 2. Service address environment variables

Each service's *own* listen address and the env var another service uses to
*dial* it are, confusingly, not always named the same way. This table is the
ground truth — don't infer a naming convention and apply it to a service that
doesn't follow it.

| Service | Its own listen env var(s) | Default | What calls it, using which env var |
|---|---|---|---|
| core-svc | `CORE_SVC_GRPC_ADDR` / `CORE_SVC_HTTP_ADDR` | `:50051` / `:8081` | bff, search-svc, channel-svc all dial `CORE_SVC_ADDR` |
| search-svc | `SEARCH_SVC_GRPC_ADDR` / `SEARCH_SVC_HTTP_ADDR` | `:50052` / `:8082` | bff dials `SEARCH_SVC_ADDR` |
| collab-svc | `COLLAB_GRPC_ADDR` / `COLLAB_HTTP_ADDR` (**no `_SVC_` infix — real, not a typo**) | `:50053` / `:8083` | bff dials `COLLAB_SVC_ADDR` (different name than collab-svc's own var) |
| channel-svc | `CHANNEL_SVC_GRPC_ADDR` / `CHANNEL_SVC_HTTP_ADDR` | `:9096` / `:8083` (compose overrides HTTP to `:8084`) | bff dials `CHANNEL_SVC_GRPC_ADDR` for gRPC (FollowService) and `CHANNEL_SVC_ADDR` (an **HTTP** base URL, e.g. `http://channel-svc:8084`) for the export proxy — same-looking name, different protocol, don't confuse the two |
| insight-svc | `GRPC_ADDR` (**generic, no service prefix at all — real**) | `:8085` | bff dials `INSIGHT_SVC_ADDR` |
| bff | `ADDR` (**generic — real**) | `:8080` in code, but **`docker-compose.yml` always sets it to `:8000`** — treat `:8000` as canonical | n/a — bff is the entry point, nothing dials it internally |
| ml-svc | `ML_SVC_GRPC_PORT` / `ML_SVC_METRICS_PORT` | `50055` / `9095` | core-svc and search-svc both dial `ML_SVC_ADDR` (`pkg/config.Pipeline.MLSvcAddr`, default `localhost:50055`) |

**Dead env vars** — present in `.env.example` but read by no Go code, don't bother
setting them: `CORE_SVC_HTTP_PORT`, `SEARCH_SVC_HTTP_PORT`, `COLLAB_SVC_HTTP_PORT`,
`CHANNEL_SVC_HTTP_PORT`, `ML_SVC_HTTP_PORT`, `BFF_HTTP_PORT`, `BFF_ADDR` (bff's real
var is the bare `ADDR`).

---

## 3. gRPC services registered per process

| Service | Proto services registered | Interceptors |
|---|---|---|
| **core-svc** | `identity.v1.IdentityService`, `catalog.v1.CatalogService`, `catalog.v1.CurationService`, `catalog.v1.OntologyService`, `catalog.v1.MediaService`, `pricing.v1.PricingService`, `b2b.v1.B2BService`, `trends.v1.TrendService`, `badges.v1.BadgeService`, `schemes.v1.SchemeService` | **Auth interceptor** (`auth.UnaryServerInterceptor`/`StreamServerInterceptor` + `PublicMethods()` allowlist) — the only service in the mesh that checks JWTs at the gRPC layer |
| **search-svc** | `search.v1.SearchService` | none (bare `grpc.NewServer()`) |
| **collab-svc** | `fulfilment.v1.FulfilmentService` | panic-recovery only |
| **channel-svc** | `social.v1.FollowService` | panic-recovery only |
| **insight-svc** | `insight.v1.InsightService` | none |
| **ml-svc** (Python) | `inference.v1.InferenceService` | none (grpc.aio, no interceptor chain) |
| **bff** | *(none — bff registers no gRPC server, it's purely a client)* | n/a |

**Trust model, stated plainly:** bff is the sole caller of search-svc, collab-svc,
channel-svc, and insight-svc — none of the four run an auth interceptor, so
whatever reaches their gRPC port unauthenticated is trusted. This is enforced
by network topology (Docker network / k8s NetworkPolicy), not application code.
core-svc has 3 legitimate callers: bff, search-svc (ontology lookups only), and
channel-svc (catalog/media hydration for ONDC + IndiaHandmade export). ml-svc has
2 callers: core-svc and search-svc. If you add a new internal caller of any
service, this trust model — not an app-level auth check — is what currently
stops an unrelated container on the same network from doing the same thing.

---

## 4. BFF HTTP route table

All routes below are mounted by `services/bff/internal/bff/server.go`. Every
`/api/v1/*` route is rate-limited (Redis-backed: 100 req/min per IP, 1000 req/min
per authenticated principal — see `docs/RATE_LIMITING.md`). Routes marked
**idem** are wrapped in idempotency middleware (client sends `Idempotency-Key`
header; replay within the key's TTL returns the original response instead of
re-executing).

### Non-API routes (no `/api/v1` prefix, no rate limit, no auth)

| Method | Path | Purpose |
|---|---|---|
| GET | `/listing/:slug` | Server-rendered SEO landing page for a listing |
| GET | `/artisan/:slug` | Server-rendered SEO landing page for an artisan |
| GET | `/v/:code` | QR provenance verification page (human-facing) |
| GET | `/v/:code/verify.json` | Same, machine-readable |
| GET | `/export/indiahandmade` | Proxies to channel-svc's own `/export/indiahandmade` (see §5) |
| GET | `/sitemap.xml`, `/robots.txt` | SEO |
| GET | `/*` (fallback) | Serves the SPA build |

### `/api/v1` — public (no JWT required)

| Method | Path | idem | Notes |
|---|---|---|---|
| POST | `/auth/otp/request` | | extra limit: 5/10min per phone |
| POST | `/auth/otp/verify` | | |
| POST | `/auth/refresh` | | |
| POST | `/payments/webhook` | | inbound payment-gateway callback, HMAC-verified — unrelated to the outbound webhook subsystem in `docs/WEBHOOKS.md` |
| GET | `/search`, `/search/suggest` | | |
| POST | `/search/voice` | | |
| GET | `/listings`, `/listings/summaries`, `/listings/:id`, `/listings/:id/summary` | | anonymous callers only ever see `PUBLISHED` listings |
| GET | `/crafts`, `/crafts/:slug` | | |
| GET | `/artisans/:id/storefront`, `/artisans/:id/follower-count` | | |
| GET | `/feed/process` | | |
| POST | `/companies` | ✅ | |
| GET | `/companies`, `/companies/:id` | | |
| GET | `/trends`, `/badges`, `/schemes` | | |
| GET | `/artisans/:id/badges` | | |
| GET | `/boutiques/nearby` | | |
| GET | `/openapi.json` | | the live spec — see §0 above |

### `/api/v1` — authenticated (JWT required; role checks happen **inside** the handler, not the router)

| Method | Path | idem | Role note |
|---|---|---|---|
| GET/PATCH | `/companies/me` | | |
| POST | `/companies/:id/verify` | | |
| GET | `/companies/commission-stats` | | |
| POST | `/companies/sales/settle` | ✅ | |
| GET | `/companies/:id/sales` | | |
| POST | `/companies/:id/interest` | ✅ | |
| POST | `/leads/:id/respond` | | |
| GET | `/artisans/me/leads`, `/artisans/me/boutique-matches` | | |
| POST/GET | `/partnerships` | ✅ (POST) | **not in `openapi.json`** — real gap, see §6 |
| POST/DELETE | `/trends`, `/trends/:id`, `/trends/:id/pin` | ✅ (POST) | |
| GET | `/badges/me/progress` | | |
| GET | `/schemes/match` | | |
| POST/PATCH/DELETE | `/schemes`, `/schemes/:id` | ✅ | |
| POST/DELETE | `/artisans/:id/badges`, `/artisans/:id/badges/:code` | ✅ (POST) | |
| POST | `/artisans` | ✅ | registration — pre-registration JWT (phone only, no artisan ID yet) is enough |
| GET/PATCH | `/artisans/me` | | |
| POST | `/auth/phone/change/request`, `/auth/phone/change/verify` | | |
| POST | `/media/upload-url`, `/media/:id/confirm` | | |
| POST/PATCH | `/listings`, `/listings/:id` | ✅ (POST) | |
| POST | `/listings/:id/submit` | | cluster officer / ministry only |
| POST | `/listings/:id/approve` | | owning artisan / SHG signatory only |
| POST | `/listings/:id/seal-provenance` | | |
| POST | `/listings/:id/suspend`, `/listings/:id/reinstate` | ✅ | cluster officer / ministry only |
| POST | `/pricing/advise` | | |
| POST | `/orders/bulk` | ✅ | creates a `bulk_order` in collab-svc |
| GET | `/orders/:id` | | |
| GET | `/orders/:id/events` | | SSE stream of order lifecycle events |
| POST | `/orders/lots/:id/respond`, `/orders/lots/:id/progress`, `/orders/lots/:id/reallocate` | | |
| POST/DELETE | `/artisans/:id/follow` | | |
| GET | `/feed`, POST `/feed/:id/read` | | |
| POST/GET | `/statements`, `/statements/:id` | ✅ (POST) | |
| GET | `/insights/*` (5 report endpoints), POST `/insights/refresh` | | ministry role only |
| POST/GET/DELETE | `/clusters`, `/clusters/:id`, `/clusters/:id/members`, `/clusters/:id/onboard` | ✅ (creates) | cluster officer / ministry only |
| POST/GET/PUT | `/self-help-groups`, `/self-help-groups/:id`, `/self-help-groups/:id/members` | ✅ | |
| POST | `/crafts/refresh-index` | | ministry only |

### OpenAPI spec drift

`services/bff/openapi.json` (served live at `GET /api/v1/openapi.json`) documents
75 paths. Two real routes are **missing** from it — no compile-time check catches
this (`route-parity.test.ts` on the frontend only checks the reverse direction):

- `POST /partnerships`, `GET /partnerships`
- `POST /payments/webhook` (plausibly intentional — it's a server-to-server callback, not a client-facing route)

Every path *in* the spec has a matching route in `server.go` — no orphaned spec
entries.

---

## 5. Two routes that leave bff's usual "sole public surface" pattern

- **`GET /export/indiahandmade`** exists on *both* bff and channel-svc. bff's
  is a thin reverse proxy (`services/bff/internal/bff/handler/export.go`) that
  forwards the query string and streams channel-svc's response straight through.
  channel-svc's own copy of this route is real HTTP served on its own port
  (`:8084` per compose) — reachable directly inside the Docker network, but not
  meant to be hit from outside it. If you're calling this from outside the
  cluster, use the bff path; the channel-svc path is an implementation detail.

---

## 6. Known unwired/incomplete surfaces (don't assume these work end-to-end)

- **Outbound webhook subscriptions** (`pkg/webhook.Manager`, migration `027_webhooks.sql`)
  have no BFF route to create/list/delete a subscription — see `docs/WEBHOOKS.md`.
  Only `POST /api/v1/payments/webhook` exists, and it's the unrelated *inbound*
  payment-gateway callback.
- **`POST /partnerships` / `GET /partnerships`** work but aren't documented in
  `openapi.json` — frontend code hitting them needs a hand-rolled request shape
  until someone regenerates the spec.
- Role enforcement for ministry/cluster-officer-gated routes lives entirely in
  handler code, not in the router — don't assume a route's presence in the
  `authed` group tells you who can call it; check the handler.
