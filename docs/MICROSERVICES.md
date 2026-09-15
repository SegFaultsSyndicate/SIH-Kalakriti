# Microservices Overview

**Last Updated:** 2026-09-15

Seven services: six Go, one Python. Every service except bff talks gRPC only;
bff is the sole REST/JSON edge and the only thing meant to be reachable from
outside the Docker network. See `docs/PORTS_AND_APIS.md` for exact ports/env
vars and `docs/BACKEND_FLOW.md` for how they call each other on real requests.

Common config, not repeated per service below: `APP_ENV`, `LOG_LEVEL`,
`SHUTDOWN_TIMEOUT` (server basics); `POSTGRES_DSN`, `POSTGRES_MAX_CONNS`,
`POSTGRES_MIN_CONNS` (every service except ml-svc); `JWT_SECRET`,
`JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`, `JWT_ISSUER` (core-svc issues, others
would need it to verify — only core-svc currently checks tokens at all).

---

## bff

**Purpose:** The single REST/JSON gateway for the buyer, artisan, and admin
SvelteKit apps. Translates HTTP requests into gRPC calls, verifies JWTs issued
by core-svc, and additionally handles idempotency-key replay, per-IP/
per-principal rate limiting, SEO pages (SSR listing/artisan landing pages,
sitemap), and QR provenance verification pages.

**Requests it handles:** ~90 REST routes — see `docs/PORTS_AND_APIS.md` §4 for
the full table. It registers no gRPC service of its own; it's purely a client.

**Depends on:** Postgres (only for its own `idempotency_key` table), Redis
(rate limiting, OTP challenge-id cache), and gRPC clients to every other
backend service (core-svc, search-svc, collab-svc, channel-svc, insight-svc).
No Kafka, no S3, no ml-svc (never calls it directly).

**Required env vars beyond the shared set:** `JWT_SECRET`, `BASE_URL`, `ADDR`
(default `:8080`, but always `:8000` in `docker-compose.yml` — treat `:8000` as
canonical), `REDIS_ADDR`, `WEB_DIST`, `CORS_ALLOWED_ORIGINS`,
`PROVENANCE_PUBLIC_KEY`, and one `*_ADDR` per backend service it dials
(`CORE_SVC_ADDR`, `SEARCH_SVC_ADDR`, `COLLAB_SVC_ADDR`,
`CHANNEL_SVC_GRPC_ADDR`, `CHANNEL_SVC_ADDR` — the last is an HTTP base URL,
not gRPC — `INSIGHT_SVC_ADDR`). Note: bff hand-rolls env parsing in `main.go`;
it does not use `pkg/config`'s struct-tag loading like the other six services.

**Called by:** end users, via the three web apps or any external API client.

---

## core-svc

**Purpose:** The domain monolith. Owns identity/auth (OTP login, artisan/
cluster/SHG registration and management), catalog (products, listings, the
draft→published state machine, provenance sealing), the media upload pipeline
(orchestrates ml-svc calls to enhance/tag/describe uploaded photos), craft
ontology, pricing advisories, B2B (company registration, boutique matching,
leads), trend links, badges, and government scheme matching.

**gRPC services (9, all in one process):** `identity.v1.IdentityService`,
`catalog.v1.CatalogService`, `catalog.v1.CurationService`,
`catalog.v1.OntologyService`, `catalog.v1.MediaService`,
`pricing.v1.PricingService`, `b2b.v1.B2BService`, `trends.v1.TrendService`,
`badges.v1.BadgeService`, `schemes.v1.SchemeService`.

**Depends on:** Postgres (by far the largest schema footprint of any service),
Redis (OTP challenge store, ontology snapshot cache, media-URL cache), S3/MinIO
(media objects), ml-svc (dialled for the cataloguing pipeline — image
enhance/extract/describe, plus technique/handloom verification for
provenance), Kafka (**produces** `media.uploaded`, `catalog.listing.*`,
`catalog.provenance.sealed`, `artisan.registered`, `company.*`,
`supply.partnership.created`; **consumes** its own `media.uploaded`/
`media.enhanced` to drive the pipeline, plus `catalog.listing.published` and
`order.lot.accepted`/`.completed` for badge-granting).

**Required env vars beyond the shared set:** all `S3_*`, `MEDIA_*` (upload
limits), Pipeline vars (`ML_SVC_ADDR`, `PIPELINE_BUYER_LANGUAGES`,
`PIPELINE_STEP_TIMEOUT`), `CORE_SVC_GRPC_ADDR`/`CORE_SVC_HTTP_ADDR`,
`AUTH_DEV_OTP_ENABLED` (must be unset/false in production — enforced at
startup), `PROVENANCE_PRIVATE_KEY`/`PROVENANCE_KEY_ID` (an ephemeral key is
generated with a startup warning if unset — dev-only), `BASE_URL`.

**Called by:** bff (primary caller for everything), search-svc (ontology
lookups only), channel-svc (catalog/media hydration for ONDC publish and the
IndiaHandmade export). It is the **only** service with a gRPC auth
interceptor — every other backend service trusts the network perimeter.

---

## search-svc

**Purpose:** Buyer-facing discovery — hybrid lexical + semantic search over
listings, autosuggest, and voice search — kept fresh by consuming core-svc's
own `catalog.listing.published` events rather than querying core-svc
synchronously per search.

**gRPC service:** `search.v1.SearchService` (`Search`, `Suggest`,
`VoiceSearch` client-streaming).

**Depends on:** Postgres (the `listing_search` table: a `tsvector` column with
a GIN index for lexical search, a pgvector `vector(768)` column with an HNSW
cosine index for semantic search — both populated together, see
`docs/BACKEND_FLOW.md` §3), Redis (query/result and transliteration caching),
ml-svc (`Embed` for indexing and query vectors, `Rerank`, `Transcribe` for
voice), core-svc (ontology client, craft-name resolution only). Kafka:
**consumes only** `catalog.listing.published`; produces nothing.

**Required env vars beyond the shared set:** `SEARCH_SVC_GRPC_ADDR`/
`SEARCH_SVC_HTTP_ADDR`, `CORE_SVC_ADDR`, `ML_SVC_ADDR`, `S3_*` (for building
media URLs the inference client needs), Redis vars.

**Called by:** bff only. Runs with **zero gRPC interceptors** — no auth check,
not even panic recovery.

---

## collab-svc

**Purpose:** Runs the collective-fulfilment saga for bulk B2B orders —
proposing/allocating order "lots" across artisan clusters, tracking accept/
decline/progress/QC, automatic reallocation on decline, amendment negotiation,
and a live order-watch stream. The directory is named "collab" but the domain
is `fulfilment` — don't expect a generic collaboration/chat feature here.

**gRPC service:** `fulfilment.v1.FulfilmentService` (`CreateBulkOrder`,
`ProposeAllocation`, `RespondToLot`, `ReportProgress`, `SubmitQC`,
`RequestReallocation`, `WatchOrder` server-streaming, `CancelBulkOrder`).

**Depends on:** Postgres only — no Redis, no S3, no ml-svc. Kafka: produces
the full `order.*`/`payment.*`/`escrow.*` event set (see
`docs/BACKEND_FLOW.md` §7 for which of those actually have a consumer — most
of the payment/escrow events currently don't); consumes its own
`order.lot.declined` to drive the reoffer-after-decline step, plus its own
lifecycle topics again to feed 6 separate `WatchOrder`/SSE consumer groups.

**Required env vars beyond the shared set:** `FULFILMENT_RESERVATION_TTL`
(default 48h), `FULFILMENT_REAPER_INTERVAL` (default 5m),
`FULFILMENT_REAPER_BATCH_SIZE` (default 200), `COLLAB_GRPC_ADDR`/
`COLLAB_HTTP_ADDR` (note: no `_SVC_` infix, unlike every other service's own
listen vars).

**Called by:** bff only, as "OrderSvc". No auth interceptor, panic-recovery
only.

---

## channel-svc

**Purpose:** Two distinct responsibilities under one roof: (1) the buyer-side
follow/feed social graph, and (2) outbound multi-channel distribution —
publishing the catalog to ONDC (India's Open Network for Digital Commerce),
serving an IndiaHandmade catalog export feed, and stubbed WhatsApp/India Post
integrations that are not wired to anything real yet.

**gRPC service:** `social.v1.FollowService` (`FollowArtisan`,
`UnfollowArtisan`, `GetFeed`, `MarkFeedItemRead`, `GetFollowerCount`). ONDC/
IndiaHandmade are not exposed as gRPC — they're Kafka-consumer-driven and
plain HTTP respectively.

**Depends on:** Postgres, Redis, core-svc (catalog/media hydration for ONDC
and the IndiaHandmade export — no ml-svc, no direct S3). Kafka: **consumes
only** `catalog.listing.published`, via two independent consumer groups on the
same topic (`-fanout` for follower notifications, `-ondc` for ONDC publishing —
the latter only starts if `ONDC_PRIVATE_KEY` is set). Produces nothing (an
outbox-relay code path exists but is commented out — channel-svc has no
outbox tables yet).

**Required env vars beyond the shared set:** `CHANNEL_SVC_GRPC_ADDR`/
`CHANNEL_SVC_HTTP_ADDR` (code default `:8083` collides with collab-svc's own
default — `docker-compose.yml` disambiguates to `:8084`, don't "fix" the code
defaults to match), `CORE_SVC_ADDR`, `ONDC_PRIVATE_KEY`/`ONDC_KEY_ID`/
`ONDC_SUBSCRIBER_ID`/`ONDC_SUBSCRIBER_URL` (ONDC publishing silently disabled
if the private key is unset), `INDIAPOST_API_URL`/`INDIAPOST_API_KEY`
(currently unused — the India Post client is an uninstantiated stub).

**Called by:** bff (gRPC, as "FollowSvc") for the social graph. Its **own HTTP
port** also serves `GET /export/indiahandmade` directly — bff's identically-
named route is a thin reverse proxy to this one (see
`docs/PORTS_AND_APIS.md` §5). No auth interceptor, panic-recovery only.

---

## insight-svc

**Purpose:** Analytics and government-facing reporting. Two unrelated
features: (1) ministry dashboard aggregates (artisans by category, listings by
craft/month, earnings by district, income comparison, "dying crafts"
detection) computed from materialized views; (2) signed, QR-verifiable PDF
income statements per artisan/SHG, used for loan/scheme applications. It is
**not** a fraud or pricing-anomaly service despite the name suggesting
"insights" broadly — see `docs/BACKEND_FLOW.md` §5 for where those actually
live.

**gRPC service:** `insight.v1.InsightService` (5 ministry-only report RPCs,
`RefreshMaterializedViews`, `GenerateIncomeStatement`, `GetIncomeStatements`,
`VerifyIncomeStatement`).

**Depends on:** Postgres (materialized views + income statement records),
S3/MinIO (statement PDF storage). **No Kafka at all** — the only service in
the mesh with zero Kafka usage; reports are refreshed via the explicit
`RefreshMaterializedViews` RPC, not event-driven. No Redis, no ml-svc.

**Required env vars beyond the shared set:** `SIGNING_PRIVATE_KEY`/
`SIGNING_KEY_ID` (Ed25519 signer for statements — an ephemeral key is
generated with a warning if unset, dev-only, same pattern as core-svc's
provenance signer), `BASE_URL` (used to build QR verify URLs), `GRPC_ADDR`
(generic, no `INSIGHT_SVC_` prefix — real, not a typo).

**Called by:** bff only, doubling as both "InsightSvc" and "StmtSvc" (one
client satisfies both interfaces). **Has no HTTP listener at all** — not even
a health endpoint — and **no gRPC interceptors of any kind**, not even panic
recovery.

---

## ml-svc (Python — the one non-Go service)

**Purpose:** All ML inference: image enhancement, attribute extraction,
description generation, craft-technique/handloom verification, text
embedding/reranking for search, and voice transcription/translation. Called
synchronously by core-svc (cataloguing pipeline) and search-svc (embed/rerank/
voice) — bff never calls it directly.

**gRPC service:** `inference.v1.InferenceService` — `EnhanceImage`,
`ExtractAttributes`, `GenerateDescription`, `VerifyTechnique`,
`DetectHandloom`, `Embed`, `Rerank`, `Transcribe` (server-streaming),
`Translate`. Built on `grpc.aio`; health check reports `NOT_SERVING` until
`registry.load_all(cfg)` finishes loading models off the event loop.

**Architecture:** `server.py` (protobuf boundary) → `features/<rpc>.py`
(backend-agnostic business logic) → `models/<component>/` — 8 swappable
components (storage, image background/lighting, VLM, embedding, reranking,
handloom texture, voice transcription), each with a `mock.py`/`real.py` pair
selected by `ML_SVC_MOCK_MODE` (**default `true`** — mock mode by default, so
Go services are never blocked on model weights being present).

**Depends on:** MinIO/S3 (`storage` component), Bhashini's external HTTP API
(voice transcription, real mode only), HuggingFace Hub (model downloads,
cached in a named `hf-cache` Docker volume so re-creates don't re-download).
No Postgres, no Kafka, no Redis — stateless.

**Required env vars (real mode; mock mode needs almost none):**
`ML_SVC_GRPC_PORT`/`ML_SVC_METRICS_PORT`, `ML_SVC_MOCK_MODE`,
`ML_SVC_MAX_BATCH_SIZE`/`ML_SVC_MAX_WAIT_MS` (micro-batching),
`ML_SVC_CRAFT_ALLOWLIST` (path to `scripts/data/crafts.csv` — must be set
explicitly in the Docker image; the flattened container layout breaks the
auto-discovery walk-up that works in a local checkout), `ML_SVC_OBJECT_STORAGE_*`,
and per-component model env vars (`ML_SVC_IMAGE_BACKGROUND_MODEL`,
`ML_SVC_TEXT_EMBEDDING_MODEL`/`_BACKEND`/`_DEVICE`, etc.).

**GPU:** `docker-compose.gpu.yml` is a separate, non-merged override (kept
separate because a hard GPU device reservation makes `docker compose up` fail
outright on hosts without an NVIDIA GPU). It reserves the GPU for the VLM
alone and pins embedding/reranking to CPU to avoid VRAM contention on small
cards. `resolve_device()` degrades CUDA→CPU/float32 gracefully on its own if
no GPU override is used. `Dockerfile.ml-svc` builds a mock-mode image with no
torch dependency by default; `--build-arg EXTRAS=real` pulls in the real
model stack.

**Called by:** core-svc and search-svc only.
