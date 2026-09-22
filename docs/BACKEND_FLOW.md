# Backend Flow: End-to-End Request Traces

**Last Updated:** 2026-09-15
**How this doc was built:** traced by reading the actual handler/service/repo
call chains, not inferred from proto comments or older docs. File:line
references point at the worktree as of this writing; line numbers drift as the
code changes, but the call chain and table/topic names don't.

This complements `docs/PORTS_AND_APIS.md` (what's reachable and on what port)
and `docs/MICROSERVICES.md` (what each service owns). This doc is "how a
request actually gets from the client to Postgres and back," for the flows
that matter most.

---

## 1. Auth / OTP login

```
Client → POST /auth/otp/request → bff:api.go RequestOTP
       → bff:client/auth.go Auth.RequestOTP (gRPC) → core-svc identity.v1/RequestOtp
       → core-svc service/otp.go OTPService.Request
           - generates 6-digit code (crypto/rand)
           - stores {phone, code} in Redis "otp:challenge:<id>", TTL 5min
           - OTPSender.Send (LoggingOTPSender — logs only, no real SMS)
       ← returns challenge_id
       ← bff caches challenge_id in Redis keyed by phone (works around the
         two-step REST flow having nowhere else to carry it)

Client → POST /auth/otp/verify {phone, otp} → bff:api.go VerifyOTP
       → bff pulls cached challenge_id for phone from Redis
       → core-svc identity.v1/VerifyOtp
       → OTPService.Verify (constant-time compare, burns challenge on success,
         5 attempts max before burning outright)
       → looks up ArtisanExistsByPhone
           - registered  → Subject{Role: RoleArtisan, ID: <artisan_id>, Phone}
           - unregistered → Subject{Role: RoleArtisan, ID: "", Phone} — a
             pre-registration token, only valid for hitting POST /artisans
       → Issuer.Issue mints {access, refresh} JWT pair (HS256)
```

**Dev bypass:** `AUTH_DEV_OTP_ENABLED=true` (set in `docker-compose.yml`, refused
by core-svc at startup if `ENV=production`) makes `OTPService.codeMatches` also
accept the literal `000000` alongside the real generated code — lets local/demo
logins skip reading Redis or logs.

**Tokens:** HS256 only, `JWT_SECRET` must be ≥32 bytes or core-svc refuses to
start. `JWT_ACCESS_TTL` default 15m, `JWT_REFRESH_TTL` default 720h (30 days).
Claims carry `role`, `language`, `kind` (`access`/`refresh`, prevents refresh
tokens being replayed as access tokens), and `phone`.

**Public gRPC methods** (no bearer token required at core-svc's auth
interceptor): `RequestOtp`, `VerifyOtp`, `RefreshToken`, plus the read-only
catalog/ontology RPCs backing bff's anonymous browsing routes (`ListListings`,
`GetListing`, `ListCrafts`, `GetCraft`, `GetProvenanceByShortCode`, `GetArtisan`).

---

## 2. Artisan registration

```
Client → POST /artisans (JWT = pre-registration token from OTP verify, idem key)
       → bff:client/artisan.go Artisan.Register — validates before ever
         reaching core-svc:
           - craft_ids: required, non-empty, ≤20
           - languages: names mapped to commonv1.Language enum
           - region.state_code: required (ISO 3166-2:IN, e.g. "IN-UP")
           - optional: cluster_id, pehchan_id, pm_vishwakarma_id,
             years_of_experience, bio, social_category
       → core-svc identity.v1/RegisterArtisan
       → service/artisan.go Identity.RegisterArtisan
           - re-validates everything server-side (domain.RegisterArtisanInput.Validate)
           - enforces: caller may only register the phone they proved control
             of at login — EXCEPT RoleClusterOfficer/RoleMinistry, who may
             register by proxy
           - one transaction:
               tx.CreateArtisan            → artisan table
               (one row per craft)         → artisan_craft table
               (if cluster_id given)       → cluster_member table
               outbox.Enqueue(ArtisanRegistered)
```

**Kafka:** emits `artisan.registered`. **No consumer exists for this topic** —
it's produced and never read anywhere in the codebase today. Treat it as an
audit trail / future integration point, not a live trigger for anything.

---

## 3. Listing creation → publish → search index

```
POST /listings          → core-svc Catalog.UpsertListing   (draft, not visible)
POST /listings/:id/submit → Catalog.SubmitForApproval        (cluster officer/ministry only,
                                                                 requires ≥1 language's copy)
POST /listings/:id/approve → Catalog.ApproveListing           (owning artisan/SHG only)
    - one transaction: upsert any edited translations, transition → PUBLISHED,
      outbox.Enqueue(CatalogListingPublished)
```

**`catalog.listing.published` fans out to 4 independent consumer groups:**

| Consumer | What it does |
|---|---|
| search-svc | `ListingPublishedHandler` → `Indexer.Index` (or `.Remove` if the event carries `Suspended`) |
| core-svc | translation fan-out consumer |
| core-svc | badge-on-publish consumer |
| channel-svc | ONDC publisher (only runs if `ONDC_PRIVATE_KEY` set) + generic fanout (follower notifications) |

**Indexing is hybrid, not one or the other:** `search-svc/internal/search/service/index.go`
calls ml-svc's `Embed` RPC to get a vector, then writes to `listing_search`
(migration `006_search.sql`), which has *both* a `document_tsv tsvector` column
with a GIN index (lexical) *and* an `embedding vector(768)` column with an HNSW
cosine index (semantic). If the `Embed` call fails, the row is indexed
lexically-only and a background reindex sweep is expected to fill the vector
in later — a failed embed never blocks the listing from becoming searchable.

`ReinstateListing` (un-suspending) re-emits the same `catalog.listing.published`
topic, which is why the suspend/reinstate routes are also idempotency-wrapped
mutations even though they don't "create" anything.

---

## 4. Bulk order (collective fulfilment) — lives entirely in collab-svc

```
POST /orders/bulk → collab-svc Fulfilment.CreateBulkOrder
    - validates quantity/buyer_id/idempotency_key, checks deadline feasibility
    - bulk_order row, state=ALLOCATING
    - outbox.Enqueue(order.bulk.requested)
```

**`bulk_order_state` enum** (migration `007_orders.sql`): `ALLOCATING` →
`PARTIALLY_ALLOCATED` → `CONFIRMED` → `IN_PRODUCTION` → `COMPLETED`, with
`AMENDMENT_PENDING` as a side-branch and `CANCELLED` reachable from any
pre-`IN_PRODUCTION` state (not directly from `IN_PRODUCTION` — must go through
an amendment). Legal transitions are enforced explicitly in
`services/collab-svc/internal/collab/domain/fulfilment.go`.

```
Fulfilment.ProposeAllocation (gRPC only — no BFF route exists for this RPC)
    - ranks candidate artisans by craft/region
    - decomposes remaining quantity into lots (respecting MinLotSize)
    - reserves capacity (capacity_reservation, state=HELD)
    - persists lots as OFFERED, emits order.lot.offered per lot
    - order → PARTIALLY_ALLOCATED if quantity remains unallocated
```

> **Known gap:** nothing in this codebase consumes `order.bulk.requested` to
> auto-trigger the *first* allocation pass after order creation, and
> `ProposeAllocation` has no BFF route. Re-offers after a decline *do* happen
> automatically (`RunReservationReaper` for expired reservations,
> `HandleLotDeclined` — a Kafka consumer on `order.lot.declined` — for active
> declines), so the saga is self-healing once started, but the very first
> allocation of a brand-new order isn't automatically triggered anywhere in
> this snapshot of the code.

```
POST /orders/lots/:id/respond → RespondToLot
    accept: reservation HELD→CONSUMED, emits order.lot.accepted
            if all lots answered + quantity covered: order → CONFIRMED,
            emits order.fulfilment.completed (reused for "confirmed" too,
            not just final completion — check the payload, not just the topic)
    decline: releases reservation, emits order.lot.declined, reoffers to next candidate

POST /orders/lots/:id/progress → ReportProgress
    lot → IN_PRODUCTION; first one moves order CONFIRMED→IN_PRODUCTION
    emits order.lot.progressed

Fulfilment.SubmitQC (human inspection, see §5)
    CRITICAL defect → lot → REALLOCATED
    pass → lot → COMPLETED; once every lot settled, order → COMPLETED,
           emits order.lot.completed / order.fulfilment.completed

Fulfilment.CancelBulkOrder — pre-IN_PRODUCTION only → order → CANCELLED,
           emits order.bulk.cancelled

Amendments (service/compensation.go) — when accepted lots can't cover the full
quantity, buyer must decide: ProposeAmendment → AMENDMENT_PENDING, emits
order.amendment.proposed; DecideAmendment emits order.amendment.decided,
routes back to ALLOCATING/PARTIALLY_ALLOCATED/CONFIRMED, or CANCELLED.
```

`WatchOrder` (server-streaming gRPC, exposed at `GET /orders/:id/events` via
SSE) is fed by 6 dedicated consumer groups on collab-svc's own
`order.lot.*`/`order.fulfilment.completed`/`order.bulk.cancelled` topics —
that's what makes the order-detail page update live.

---

## 5. QC, fraud, and pricing anomaly — three unrelated mechanisms, easy to conflate

These get lumped together in casual conversation ("the risk stuff") but they
are three separate systems with no code path between them:

**QC is human inspection, not ML.** `Fulfilment.SubmitQC` takes an inspector's
pass/fail plus defects and writes to `qc_result`/`qc_defect`
(migration `020_qc_results.sql`). No call to ml-svc or insight-svc anywhere in
this path.

**ml-svc is called synchronously via gRPC, never via Kafka.** Callers:
- core-svc's cataloguing pipeline (triggered by consuming its own
  `media.uploaded` event) calls `EnhanceImage`, `ExtractAttributes`,
  `GenerateDescription`; provenance sealing calls `VerifyTechnique`/`DetectHandloom`.
- search-svc calls `Embed` (indexing, §3) and `Rerank`/`Transcribe` (voice search).

There is no Kafka usage anywhere in `services/insight-svc` or `services/ml-svc`.

**Fraud detection is pure Postgres triggers**, matching `docs/FRAUD_DETECTION.md`'s
rule descriptions exactly (migration `026_fraud_detection.sql`, 4 trigger
functions on `bulk_order`/`artisan`/`listing` inserts/updates, writing to
`fraud_flags`). **However**, `pkg/fraud.Detector` — the Go API for listing/
resolving flags that the doc's "admin review workflow" section describes — is
imported by no service. Flags get created; nothing in the running stack
currently reads them back out via an API.

**Pricing anomaly detection is a separate, synchronous, non-ML feature in
core-svc** — `pricing.v1.PricingService.GetAdvisory`
(`services/core-svc/internal/core/service/pricing.go`). It reads the listing's
already-computed embedding out of `listing_search` (populated by search-svc's
indexer, §3), computes a cost floor from state minimum wage, a market band from
cosine-similar comparables, and a seasonality multiplier — pure SQL/statistics
at request time, no live ml-svc or insight-svc call.

`insight-svc` itself is **not** fraud or pricing — it's ministry-facing
analytics (materialized-view reports) and income-statement PDF generation. See
`docs/MICROSERVICES.md`.

---

## 6. Outbound webhooks — built, but not reachable from any live path

Migration `027_webhooks.sql` (`webhook_subscriptions`/`webhook_deliveries`
tables + Postgres triggers that auto-enqueue a delivery on `bulk_order`
insert/update) and `pkg/webhook.Manager`/`Worker` (HMAC-SHA256 signing,
exponential backoff, `FOR UPDATE SKIP LOCKED` polling) are a complete,
internally consistent subsystem. But:

1. `Manager.CreateSubscription`/`ListSubscriptions`/`DeleteSubscription` are
   called from nowhere in `services/` or `web/` — no BFF route exists to
   register a subscriber. A `webhook_subscriptions` row can only be created by
   a direct SQL insert today.
2. `cmd/webhook-worker` (the binary that actually delivers queued webhooks) is
   **not listed as a service in `docker-compose.yml`** — even a manually
   inserted subscription won't get delivered unless someone runs the binary
   by hand.
3. The only *used* piece of `pkg/webhook` is `VerifySignature`, and only for
   the unrelated **inbound** payment-gateway callback at
   `POST /api/v1/payments/webhook` — that's Kalakriti *receiving* a signed
   webhook, not the outbound delivery system above.

This is a fully-built but currently inert subsystem — see `docs/WEBHOOKS.md`
for the mechanism details, corrected to reflect this status.

---

## 7. Kafka topic inventory — producer/consumer, including dead ones

All topic name constants live in `pkg/topics/topics.go`. Every producer here
goes through the transactional outbox (`pkg/outbox`) **except** the webhook
triggers and fraud-flag triggers, which are bespoke Postgres triggers that
bypass the app-level event bus entirely.

| Topic | Producer | Consumer(s) |
|---|---|---|
| `media.uploaded` | core-svc (`ConfirmUpload`) | core-svc (cataloguing pipeline) |
| `media.enhanced` | *(ml-svc side, not traced in Go code)* | core-svc |
| `catalog.listing.published` | core-svc (`ApproveListing`, `ReinstateListing`) | search-svc, core-svc (×2), channel-svc (×2) — see §3 |
| `catalog.provenance.sealed` | core-svc | core-svc (badge consumer) |
| `order.bulk.requested` | collab-svc | **none** — see the allocation gap in §4 |
| `order.lot.offered` | collab-svc | collab-svc (watch/SSE) |
| `order.lot.accepted` | collab-svc | collab-svc (watch), core-svc (badge) |
| `order.lot.declined` | collab-svc | collab-svc (watch + reoffer saga) |
| `order.lot.progressed` | collab-svc | collab-svc (watch) |
| `order.lot.completed` | collab-svc | core-svc (badge) |
| `order.fulfilment.completed` | collab-svc (reused for confirm + complete) | collab-svc (watch) |
| `order.bulk.cancelled` | collab-svc | collab-svc (watch) |
| `order.amendment.proposed` / `.decided` | collab-svc | **none** |
| `payment.split.requested`, `payment.settled`, `escrow.milestone.released` | collab-svc | **none** |
| `artisan.registered` | core-svc | **none** |
| `company.*` (registered/verified/rejected/sale.settled/interest.*) | core-svc | **none** |
| `supply.partnership.created` | core-svc | **none** |
| `catalog.attributes.extracted`, `catalog.listing.drafted`, `search.index.requested`, `order.lot.expired`, `artisan.followed`, `dispute.raised`, `dispute.resolved`, `shipment.dispatched`, `shipment.delivered`, `boutique.match.found` | **none** | **none** — declared as constants, never produced or consumed anywhere |

Roughly a third of declared topics are produced-but-unconsumed (a fire-and-forget
audit trail, mostly the b2b/payment/escrow events) or declared-but-entirely-dead
code. `FollowArtisan` writes directly to the DB and does not go through
`artisan.followed` despite that topic existing — don't assume the topic name
implies a live wiring; check the producer/consumer columns above.

`order.lot.expired` has no producer because expiry is handled by
`RunReservationReaper` polling capacity reservations directly, not by an event.
