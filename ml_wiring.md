# ML wiring — what Batch 8 mocked and what has to replace it

Batch 8 (the listing-creation wizard, `web/apps/artisan/src/routes/listing/new`) needed a
photo quality gate, a multi-stage processing pipeline, generated attributes/description, and
sentence-level claim traceability. **None of that has an HTTP-reachable surface in
`services/bff` today** — no attributes, no claims, no confidence, no per-step pipeline state,
anywhere in `services/bff/openapi.json`. `core-svc`'s ML pipeline exists internally but isn't
exposed over the BFF's REST API.

Per instruction, the wizard was built end to end anyway, with every ML-shaped gap filled by
fabricated data, isolated in one file: `web/apps/artisan/src/lib/ml-mock.ts`. Nothing else in
the app imports mock data directly — every other module only sees the mock's output through
that file's exported types. **Delete `ml-mock.ts` and its two call sites once real endpoints
exist**; nothing else should need to change.

This document is the list of what to replace it with.

## 1. Photo quality gate

- **Mock**: `assessPhotoQuality(blob)` in `ml-mock.ts`. Looks only at `blob.size` — under
  20KB is treated as "no subject", under 60KB as "blurry". This is not real image analysis;
  it exists so the wizard's retake flow has *something* to trigger on.
- **Called from**: the capture step, immediately after each photo is added via
  `addCapturedMedia` in `$lib/listing-draft.ts`.
- **What's needed**: either an on-device model (works offline, matches the app's
  airplane-mode requirement) or a new BFF endpoint like `POST /media/{id}/quality` that
  returns issue codes after upload. If it's server-side, the capture step's quality check
  has to move from "immediately on capture" to "after that photo's `media.upload` outbox
  entry lands" — capture-time feedback would need the on-device path instead.
- **Shape to design**: `{ passed: boolean, issues: { code: string, messageKey: string }[] }`.
  Keep the issue codes as an enum the client can map to a translated retake reason — that
  part of the mock's shape is worth keeping as-is.

## 2. Pipeline stages (the processing screen)

- **Mock**: `runMockPipeline()` in `ml-mock.ts` drives a 5-stage state machine
  (`upload → enhance → attributes → describe → translate`), each stage taking ~700ms. The
  `upload` stage is the **one real signal** in the mock: it polls the draft's `media.upload`
  outbox entries and only advances once they've actually drained. The other four stages are
  fabricated delays.
- **Called from**: the processing step route.
- **What's needed**: a way for the client to observe `core-svc`'s actual pipeline progress
  for a listing — either a polling endpoint (`GET /listings/{id}/pipeline`) or an SSE stream
  (this app already has one pattern for that: `orderEventsUrl` / `watchOrderEvents` in
  `packages/api/src/sse.svelte.ts`, built for order status — the same shape would work here).
  Whatever stage keys the real pipeline uses, keep them as an explicit enum like
  `PipelineStageKey`, not inferred from booleans, for the same reason `OutboxStatus` in
  `packages/offline/src/db.ts` is a closed union.

## 3. Attributes, description, translations

- **Mock**: `buildMockResult()` in `ml-mock.ts` fabricates three attributes (craft, material,
  technique), a two-sentence English description, and an English + Hindi
  `ListingTranslation` pair, all derived from whatever craft/working-title the artisan
  already typed.
- **The one part of this that is real**: `ListingTranslation` (`language`, `title`,
  `description`, `highlights`, `machine_generated`) is an actual field on
  `CreateListingRequest`/`UpdateListingRequest` in `services/bff/openapi.json` today. The
  mock's translations are sent through the real `queueListingUpdate()` →
  `PATCH /listings/{id}` path in `outbox-send.ts`, marked `machine_generated: true`, exactly
  as a real pipeline's output would be. **No new endpoint is needed to land translations** —
  only to generate better ones.
- **Attributes have no home yet**: `MockAttribute[]` is never sent anywhere; it's read
  straight off the pipeline result and stored in the draft's local `fields` for the review
  screen to render. There is no `attributes` field on `Listing` or `CreateListingRequest`. If
  attributes should persist server-side (for search facets, provenance, etc.), the spec needs
  one; until then this stays a local-only, per-device fabrication that a second attempt at the
  same draft (say, a fresh tab) would regenerate differently.

## 4. Claim traceability (tap a sentence, see the attribute it's based on)

- **Mock**: `MockClaim[]` — `{ sentenceIndex, attributeKey }` pairs, hand-matched to the two
  fabricated sentences in `buildMockResult()`.
- **What's needed**: nothing in `core-svc` or the BFF exposes *why* a generated sentence says
  what it says, i.e. no confidence score or source attribute per sentence. A real
  implementation would need the generation step itself to emit this alignment (sentence →
  attribute → confidence), then a field to carry it to the client — there's no natural home
  for it in `ListingTranslation` today (highlights is close but is plain strings, not an
  attribute reference). This is the deepest gap in the batch: everything else above is
  "add an endpoint"; this one needs the generation pipeline itself to produce alignment data
  it may not currently track.

## 5. What's NOT mocked (already real, use as-is)

- **Pricing advisory** — `POST /pricing/advise` is a real, already-implemented BFF endpoint
  (`packages/api/src/operations.ts`'s `advisePricing`). The pricing step calls it directly
  (not through the outbox — it's a read, not a mutation) and only when online; offline, the
  artisan just enters a price with no advisory shown. No mock involved.
- **Media upload** — `POST /media/upload-url` → raw `PUT` → `POST /media/{id}/confirm` is the
  real three-step flow, fully wired in `outbox-send.ts`'s `sendMediaUpload`.
- **Listing create/update/submit/approve** — all four are real BFF endpoints, wired in
  `outbox-send.ts`. `SubmitForApproval` (`services/core-svc/internal/core/service/catalog.go`)
  rejects a listing with zero translations — this is why the processing step's mock
  translations must land (via a `listing.update`) before the wizard's final publish action can
  succeed even in the mock world; a real pipeline has the same constraint.

## 6. Batch 9 additions (`/listings`, `/listings/{id}`, provenance sealing, HandloomVerdict)

Batch 9 needed real backend work too, not just UI — done as part of this batch, not deferred:

- **`POST /listings/{id}/seal-provenance` is now real and wired end to end.** `core-svc`'s
  `SealProvenance` RPC (`services/core-svc/internal/core/service/provenance.go`) already existed
  and does real work — this batch only added the missing BFF facade (`client/listing.go`'s
  `SealProvenance` + `provenanceRecordToMap`, the REST handler, the openapi.json entry) mirroring
  `client/pricing.go`'s existing pattern. Nothing here is mocked.
- **`GET /listings?artisan_id=` is now real.** `core-svc`'s `ListListings` already filtered by
  artisan and already fell back to published-only for a non-matching id
  (`service/catalog.go:494-509`) — the BFF handler just wasn't passing the query param through.
  One-line fix (`handler/api.go`'s `ListListings`), no mock involved.

Two real, but genuinely incomplete, surfaces this batch had to design around:

- **`TechniqueVerdict` has no third outcome.** The proto (`proto/inference/v1/inference.proto`)
  and the mock model (`services/ml-svc/app/models/mock.py`) only ever return `matches: bool` +
  `confidence: float` — there is no `INSUFFICIENT_EVIDENCE` anywhere upstream. The UI
  (`$lib/provenance.ts`'s `deriveVerdictOutcome`) derives it client-side: `confidence < 0.55`
  (either direction) is shown as inconclusive rather than as a match or a mismatch, regardless
  of what `matches` says. **This is a UI-side judgement call standing in for a real model
  capability** — when ml-svc adds a genuine third output (or a calibrated confidence the product
  team trusts at a different threshold), delete `deriveVerdictOutcome` and read `matches`
  directly, and move the 0.55 threshold decision to wherever that confidence is calibrated.
- **Listing media has no read path.** `CreateListingRequest` accepts `media` (confirmed media
  ids) on create, but neither `catalogv1.Listing` (the proto message) nor
  `GET /listings`/`GET /listings/{id}`'s response ever return it back — confirmed at the proto
  level (`proto/catalog/v1/catalog.proto`'s `message Listing` has no media field at all), not
  just an unwired BFF gap like the two above. The `/listings` catalog and "duplicate as draft"
  in this batch both work around it: the catalog row's photo falls back to whatever this device
  has cached locally in `@kalakriti/offline`'s `db.media` (real for a listing created on this
  device; blank elsewhere), and duplicate-as-draft honestly does not carry photos into the copy
  and says so in the UI. Fixing this for real needs a proto change (`repeated
  common.v1.MediaRef media` on `Listing`, populated from `AttachListingMedia`'s already-real
  storage) — out of scope for a frontend-focused batch, flagged here rather than worked around
  silently.
- **Attributes still have no server-side read path either**, confirmed at the same proto level
  as media (no `attributes` field on `Listing`) — batch 8's finding in section 3 above wasn't a
  BFF gap to close, it's an architectural one. `core-svc`'s `UpsertListingAttributes`
  (`service/catalog.go:520-601`) is real, with real artisan-wins precedence
  (`domain.AttributeSource`: `MODEL`/`ARTISAN`/`CURATOR`), but nothing calls it from the BFF and
  nothing reads attributes back. Batch 9's `/listings/{id}` edit screen keeps attributes as the
  same local-only, per-device construct batch 8 established, now explicitly labelled
  "AI-guessed" vs "you confirmed" using the same three-way `AttributeSource` vocabulary the real
  enum already uses — so wiring the real thing later is a data-source swap, not a UI redesign.

One non-ML backend finding worth recording here since this is the file whoever picks up backend
work will read first: **an artisan cannot suspend or reinstate their own listing.**
`SuspendListing`/`ReinstateListing` (`service/catalog.go:373-465`) require
`RoleClusterOfficer`/`RoleMinistry` — the code comment is explicit: *"Suspension is an operator
action: an artisan who wants to stop selling pauses orders instead."* The batch 9 brief asked
for artisan-facing "suspend" and "republish" bulk actions; built as **pause/resume selling**
instead, using the real, artisan-permitted `accepting_orders` field
(`made_to_order_terms.accepting_orders`, already wired via `PATCH /listings/{id}` since batch 8)
and `stock_quantity: 0` for READY_STOCK. This is not a mock or a stand-in for a missing
endpoint — it is the actual capability the backend grants an artisan, under honest UI copy
instead of copy borrowed from the admin-only state machine.

## 7. Batch 10 additions (orders, lots, earnings, notifications, follower count)

Batch 10 needed more real backend work than any prior batch — most of what the brief asked for
had no RPC anywhere, not just no BFF facade. Done as part of this batch:

- **`POST /orders/lots/{id}/progress` and `/reallocate` are new, real, full-stack additions.**
  `fulfilment.proto`'s domain transition table already allowed `QC_FAILED -> QC_PENDING` (rework)
  and `ACCEPTED/IN_PRODUCTION/QC_FAILED -> REALLOCATED` (give-up) — nothing called either edge
  from those states. `ReportProgress` (`collab-svc/internal/collab/service/fulfilment.go`) had
  its guard relaxed to also accept `QC_FAILED` as a starting state (a rework resubmission is just
  another progress report); a new `RequestReallocation` RPC (proto, service, handler, BFF client,
  BFF handler, openapi.json) mirrors `SubmitQC`'s existing give-up branch exactly, including its
  documented gap: no outbox topic yet for a REALLOCATED transition, so a lot given up by an
  artisan (like one given up by QC) only reaches a currently-connected SSE watcher on
  reconnect/replay, not live. Upgrade path is the same one already noted for `SubmitQC`'s branch.
- **`GET /orders/{id}`, `POST /orders/lots/{id}/respond` and `GET /orders/{id}/events` were real
  but undocumented** (missing from `openapi.json` entirely, except `/events`) — documented this
  batch, no code change needed. `order.respond` was already a reserved `OutboxKind`
  (`packages/offline/src/db.ts`) with no sender wired to it; `outbox-send.ts` now sends it, so a
  lot accept/decline queues and syncs offline the same way every other batch 8 write does.
- **`GET /feed` already was channel-svc's notification feed** (its own doc comment: "a follower's
  feed is exactly the notifications addressed to them") — this batch only added
  `POST /feed/{id}/read` (channel-svc's `notification.Service.MarkRead` existed but nothing
  called it with an ownership check) and `GET /artisans/{id}/follower-count` (reuses the existing
  `GetFollowers` query, just never had an RPC). Both needed a `social.proto` addition + regen,
  same weight as batch 9's `SealProvenance` facade.
- **`POST /statements` returned only `statement_id`, discarding `download_url`.** Fixed to return
  the full response, since `GetStatement`-by-id is permanently unusable (`insight-svc` has no
  such RPC — only a list scoped to an artisan or a lookup by public short code, per
  `client/insight.go`'s own doc comment, already true before this batch). Added
  `GET /statements` (`ListIncomeStatements`, wrapping the real, previously-unwired
  `GetIncomeStatements` RPC) as the earnings-over-time read model.

Two real gaps this batch could not close, worked around honestly rather than faked:

- **No `GET /orders` (list) exists anywhere** — not in the BFF, not in collab-svc's service
  layer, not even as a repo-level SQL query keyed by artisan. `/orders` therefore shows a
  **local, per-device index of every order id this device has ever fetched** (`$lib/orders.ts`'s
  `touchedOrderIds`), refreshed from the real `GET /orders/{id}` when online — same
  local-cache-first pattern batch 9 used for the listings-media gap. In practice this means an
  artisan reaches a new lot offer through its notification's deep link (which is real — `payload`
  on a `LOT_OFFERED` `FeedItem` carries `order_id`/`lot_id`), not by browsing a list that doesn't
  exist server-side.
- **No platform-fee threshold field exists anywhere** — checked `collab-svc`'s `compensation.go`
  and `payment.go`; commission is a flat calculation with no minimum-order exemption. The batch
  10 brief asks for the fee shown as zero below a stated threshold; `/earnings` shows the real
  `fee_amount` unqualified rather than inventing a threshold number for a money-correctness-
  sensitive figure. If a real threshold is added to the commission calc later, surface it next to
  `earnings.history.fee` — the row is already there, it just has nothing to compare against yet.

## 8. Batch 11 additions (buyer marketplace: home, search, craft, artisan storefront, process feed)

Batch 11 needed real backend work in three services, not just BFF facades — search-svc's query
understander and business rules already existed and worked end to end; they were simply never
reachable from the buyer app.

- **Query understanding was already computed and thrown away.** `search-svc`'s
  `service.Result.Understood` (parsed craft spans + `domain.Filters`) existed before this batch,
  but `searchv1.SearchResponse` had no field to carry it and `handler.responseToProto` never
  built one. Added `SearchResponse.understood_filters`/`craft_spans` to `proto/search/v1/search.proto`,
  `filtersToProto` (the inverse of the existing `filtersFromProto`) in `search-svc`'s handler, and
  wired the BFF's `GET /search` to return `understood` — this is what `/search`'s removable
  filter chips render, not a client-side re-parse.
- **Cross-lingual match was a real, derivable signal, not a heuristic.** `listing_translation`
  already has a `machine_generated` column (an artisan's own words vs. a translation) and
  search-svc's `HydrateSearchHits` query already joined that table — it just never selected the
  column. Added `machine_generated` to the query, `domain.Hit`, and `SearchHit` proto; the BFF
  reports `machine_generated: true` only when it's also true that `matched_terms` is non-empty
  (the query matched *inside* the translated copy), which is what `/search`'s "matched across
  languages" badge is gated on.
- **Filters/facets were computed correctly in SQL and dropped at the BFF.** `search-svc`'s
  `SearchLexical`/`SearchVector` already applied colour/material/price/GI/sealed/lead-time
  filters (`services/search-svc/internal/search/repo/repo.go`); only `craft_id`/`region` ever
  reached them from `client/search.go`'s `structuredFilters`. Extended it to forward the rest,
  and the BFF's `GET /search` handler to read the corresponding query params — a facet in the UI
  now narrows the real result set end to end, not a subset of a fixed page.
- **No public REST for craft ontology or artisan storefront existed.** `client/catalog.go`
  already had `GetListingBySlug`/`GetArtisanBySlug` (built for the SEO crawler pages) with an
  `ImageURL` field that nothing ever populated. Added `GetCraftBySlug`, `ListCrafts`,
  `ListArtisansByCraft`, and wired `ImageURL`/`Verified`/`District`/`ClusterID` through
  `MediaService.GetMediaURL` and `catalog.v1.Artisan`'s real `photo`/`verified`/`region` fields —
  new public routes `GET /crafts`, `GET /crafts/{slug}`, `GET /artisans/{id}/storefront`.
- **Listing cards needed craft/material/image data `GET /listings/{id}` never returned.**
  `catalog.v1.Listing` has no `craft_id` or media field at all (that lives on `Product`, joined
  via `GetListing(include_product: true)`, already used by `GetListingBySlug`). Added
  `GetListingSummary`/`GET /listings/{id}/summary`, used by every card-rendering surface (home
  sections, search results, storefronts) instead of extending the plain `GetListing` shape every
  caller pays for.
- **The process-provenance feed (`GET /feed/process`) has no dedicated store.** Neither
  `search-svc` nor `catalog-svc` indexes by media asset; `ListingMediaRole.PROCESS_VIDEO` exists
  on `CurationService.AttachListingMedia` but there's no matching read RPC. `ListProcessClips`
  scans recent published listings and keeps the first `MEDIA_KIND_VIDEO` ref each `Product`
  carries — real video, honestly derived, not a fabricated feed; documented as an `O(n)` scan in
  a `ponytail:` comment, not an index, since nothing here paginates past the first screen.
- **Search-svc had no real pagination to page.** `SearchResponse.page.next_page_token` is always
  `""` — there is no cursor anywhere in the retrieval path. `/search`'s "load more" therefore
  re-asks for a bigger `limit` (a new bounded query param threaded through `client/search.go`)
  rather than paging a cursor that doesn't exist; each click replaces the result set with a
  larger one instead of appending an unverifiable next page.
- **`/artisans/{id}/follower-count` and `POST /search/voice` were needlessly behind auth.**
  Both are read-only and carry no PII beyond a count; moved to the public route group so an
  anonymous buyer's storefront view and voice search both work without a login prompt the brief
  never asked for.
- **No backend field exists for ODOP or "awardee" status anywhere in the repo** (grepped every
  `.proto`/`.go`/SQL file). Not rendered on the home page's One District One Product section
  (the section itself is skipped; i18n keys were added and are unused, kept for whenever the
  field exists) or on the artisan storefront — a fabricated award badge is worse than an honestly
  missing one on a government platform.
- **"Artisan of the month" is a deterministic pick over real data, not a backend ranking.** The
  home page walks `GET /crafts` in order and takes the first craft with a published listing,
  then that listing's artisan's real storefront profile. Documented in the home page's own file
  header — do not read this as the platform's editorial choice; it is a placeholder selection
  rule until a real "featured artisan" concept exists.
- **Native-script craft names weren't retrievable.** `ontology.proto`'s `Craft` message carries
  one `display_name` (its canonical English form); non-English spellings exist only as
  `CraftAlias` rows matched *from* free text (`ResolveCraftAlias`), with no "list aliases for
  this craft" read RPC. The shop-by-craft tiles show the single `display_name` only — the batch
  11 brief's "craft name in both the buyer's language and its native script" is not met for the
  second half, and isn't faked with a lookup table that would silently go stale against the real
  ontology.

## 9. Batch 12 additions (conversion: product page, purchase, allocation view, provenance page, orders)

- **SSE reconnect/backfill fix, not new backend logic.** `GET /orders/{id}/events` always
  existed, but ignored `Last-Event-ID` and never emitted an `id:` line, so `EventSource`'s (and
  `packages/api/src/sse.svelte.ts`'s own hand-rolled fetch reader's) reconnect couldn't resume —
  every reconnect replayed from the start with no dedup key. Fixed by making the SSE `id:` line
  the event's own `occurred_at` (RFC3339Nano) instead of an opaque counter: `WatchOrderRequest`
  already had a `since *Timestamp` field with nothing setting it, and a timestamp round-trips
  through `Last-Event-ID` with no separate id→timestamp store needed. See
  `services/bff/internal/bff/handler/api.go`'s `WatchOrder`/`parseSince` and
  `client/order.go`'s `WatchOrder(ctx, orderID, since)`. Client-side dedup by `event_id` (already
  the established pattern in `apps/artisan/src/lib/order-timeline.ts`, mirrored in
  `apps/buyer/src/lib/allocation.ts`) is the actual duplicate-row guarantee; the timestamp resume
  just keeps a long-disconnected client from replaying the entire order history every time.
- **`GET /listings/{id}/summary` extended, not a new endpoint.** Added `made_to_order_terms`
  (real: `MadeToOrderTerms.lead_time_days/capacity_per_month/advance_pct/customisation_options`),
  `provenance` (via `GetListing(include_provenance: true)`, previously never set), the artisan's
  `bio`/`district`/`verified` and `story_audio_url` (the product's own `voice_note` media,
  resolved to a signed URL). All real fields that already existed on the proto messages and were
  simply never read into the summary map before this batch.
- **No payment/checkout backend exists anywhere in this repo.** `fulfilment.proto`'s
  `CreateBulkOrder` is the only order-placing RPC in the whole system — there is no separate
  cart/checkout service, no UPI or any other payment-initiation RPC, and no field for a delivery
  address on `CreateBulkOrderRequest`. Consequences, all deliberate:
  - A single-item purchase is `quantity: 1` through the same `CreateBulkOrder` call a bulk order
    uses (`POST /orders/bulk`, newly documented in `openapi.json` — the route existed server-side
    since batch 10 but was never in the spec, so no frontend code could legally call it).
  - Delivery details go into the one real free-text channel the backend has, `notes` — labelled
    honestly (`purchase.deliveryNotesHint`) rather than presented as a structured address field
    that would silently drop.
  - No UPI deep link is built. Fabricating a `upi://pay?pa=...` would need a payee VPA, and
    nothing in the proto tree carries one before settlement (`PaymentSplitLine.payout_ref` only
    exists post-fact). `purchase.noPaymentGateway` says plainly that placing the order collects
    no payment.
  - The artisan's *actual* payout has no pre-order source either: `commission_pct` is an internal
    settlement input (`services/collab-svc/internal/collab/service/payment.go`), never exposed to
    a buyer-facing RPC. The "radical transparency" earnings display the brief asks for is real,
    but it lands on the allocation view (`routes/orders/[id]`) when the real `payment_settled`
    event actually fires with real `PaymentSplitLine` amounts — not fabricated at checkout time.
- **No feasibility-before-submission RPC.** `ProposeAllocation`'s `dry_run: true` is the only
  allocation-preview mechanism, and it requires an existing `bulk_order_id` — there's no way to
  dry-run allocation before `CreateBulkOrder` has already created the order. The bulk order
  wizard's inline feedback is instead a real but more modest signal: how many artisans practise
  the chosen craft at all (`GET /crafts/{slug}` → `artisans`, already wired in batch 11), labelled
  as a headcount, not a completion promise.
- **No buyer-scoped order list RPC.** `GetOrder` only fetches one order by id; there is no
  `ListOrders(buyer_id)` anywhere in `fulfilment.proto`. `/orders` rehydrates a per-device
  remembered list of order ids (`lib/order-store.ts`, `localStorage`) through the real `GetOrder`
  — said plainly in `orders.empty.hint` as per-device, not a synced account history.
- **No dispute RPC.** The "dispute entry point" the brief asks for is a `mailto:` link carrying
  the real order id (`lib/DisputeDialog.svelte`) — not a submission to a backend that doesn't
  exist. A real in-app flow needs an RPC first.
- **Amendments have a state but no data or RPC.** `BulkOrderState.AMENDMENT_PENDING` exists and
  `BulkOrder` can be observed sitting in it, but nothing on the wire carries what quantity or date
  is being proposed, and there is no propose/accept/reject RPC for a buyer to act on one. The
  allocation view renders a read-only banner when it sees that state rather than fabricating
  accept/reject controls with no data behind them.
- **Provenance page restyled in place, not spec'd for a handoff.** `services/bff/internal/bff/handler/verification.go`'s
  `html/template` strings *are* the deliverable — token values (from `web/packages/tokens/src/palette.css`
  and `scale.css`) copied literally into an inlined `<style>` block since a Go template has no CSS
  custom-property pipeline of its own. Zero `<script>` anywhere on the page, so "renders correctly
  with JavaScript disabled" holds by construction. The "we cannot verify this tag" state
  (`notFoundPageTemplate`) already existed and was restyled in place rather than rebuilt.
- **Process video vs. stills "equal prominence"** on the product gallery is real: the same
  `media` array `GET /listings/{id}/summary` returns for images carries `MEDIA_KIND_VIDEO` items
  too (same limitation noted in the batch 11 section — there's still no dedicated
  `ListingMediaRole.PROCESS_VIDEO` read path, so this is Product-level media, not a curated
  process reel).
- **JSON-LD Product shape mirrors the Go-rendered SEO page exactly** (`name`/`description`/
  `image`/`brand`/`offers`), read directly from `services/bff/internal/bff/handler/seo.go`'s
  existing `jsonLD` map rather than re-derived — the two variants a crawler might hit produce the
  same structured data by construction, not by two people agreeing on a shape.

## 10. Batch 13 additions (ministry dashboard, cluster/SHG admin, moderation, craft ontology)

Batch 13 is desktop-first admin tooling (`apps/admin`): a sidebar/breadcrumb/command-palette shell,
`/insights`, `/clusters`, `/moderation`, `/crafts`. Wiring status per surface:

- **App shell, login**: real. `apps/admin` had no auth flow at all before this batch — `/login` and
  `/login/verify` reuse the exact same `POST /auth/otp/request` / `POST /auth/otp/verify` the
  artisan and buyer apps use (`requestOtp`/`completeOtpVerification`), a desktop text-input form
  instead of the artisan app's Keypad. A phone number logs in as whatever role its account carries
  in the JWT's `role` claim; there is no admin-specific auth backend to build. Role-gating is done
  by hiding page content behind an "access restricted" message, never by redirecting to a login
  that cannot grant a role a phone number's account doesn't have.

- **`/insights`**: `GetEarningsByDistrict`, `GetIncomeComparison`, `GetDyingCrafts` existed from
  earlier batches but had **zero rows in `openapi.json`** (checked directly — the earlier
  assumption that they were already documented was wrong) — added here.
  `GetArtisansByCategory`/`GetListingsByCraftMonth`/`RefreshMaterializedViews` existed in
  `insight.proto` and `insight-svc`'s repo/service layers but had no BFF client method, handler, or
  route at all — added (`client/insight.go`, `handler/api.go`, `server.go`). Fixed a real bug along
  the way: `GetIncomeComparison`'s handler dropped `state_code`/`district` query params entirely
  (`filters := map[string]any{}`), so the "filters throughout" requirement was silently unmet for
  that one endpoint — the client and RPC always supported the filter, only the handler ignored it.
  **Suppression**: confirmed by reading `services/insight-svc/internal/insight/repo/repo.go` — there
  is no `suppressed: bool` field anywhere in `insight.proto`. `GetEarningsByDistrict` and
  `GetIncomeComparison` suppress via a SQL `HAVING artisan_count >= minBucket` clause, i.e. silent
  row omission; `GetArtisansByCategory`, `GetListingsByCraftMonth` and `GetDyingCrafts` apply no
  suppression at all. The dashboard resolves this by treating `GetArtisansByCategory`'s response
  (unsuppressed) as the full district roster, and rendering any roster district missing from an
  earnings/income response as the literal text "Suppressed for privacy" — never a zero, never a
  blank cell — in both the chart and its table equivalent. This is a real, honest reconstruction of
  a signal the backend doesn't emit directly, not a guess: a district is either in the earnings
  response (real number) or in the roster-but-not-earnings (suppressed) or in neither (never seen).
  One open edge the roster diff cannot resolve: a district with real artisans whose earnings are
  *all zero* is indistinguishable in the UI from "not yet seen" if it never appears in the roster
  either — this can't happen given how the roster is built (roster comes from the same artisan
  table earnings aggregate over), but it's worth naming as the assumption this diff rests on. CSV
  export is a dependency-free `Blob`/`URL.createObjectURL` download (`lib/csv.ts`). Charts are
  hand-rolled SVG (`lib/BarChart.svelte`; LayerChart was suggested but never installed and adding
  a new dependency for this was not worth it under Ponytail) with a `Tabs`-toggled real `<table>`
  sibling for every chart, values as text (not colour-only).

- **`/clusters`**: all 10 cluster/SHG RPCs (`CreateCluster`, `GetCluster`, `AddClusterMember`,
  `RemoveClusterMember`, `ListClusterMembers`, `CreateSelfHelpGroup`, `GetSelfHelpGroup`,
  `SetSelfHelpGroupMembers`, `ListSelfHelpGroupMembers`) were fully implemented in
  `services/core-svc/internal/core/handler/identity.go` with **zero BFF wiring** — added client
  methods (`client/catalog.go`), handlers (`handler/admin.go`), routes and `openapi.json` entries
  for all of them. **Real gap, not worked around**: there is no `ListClusters` or
  `ListSelfHelpGroups` RPC anywhere in `identity.proto` — only fetch-by-id and create. The page
  reflects this plainly: a newly created cluster/SHG is shown immediately with its id; loading an
  existing one means typing its id in. Share-pct-sums-to-100 is validated live in the UI and
  enforced server-side in `core-svc` (`shgSharesFromProto`) — the UI surfaces the server's rejection
  rather than assuming its own check is the only guard.
  **Bulk CSV onboarding**: `POST /artisans` (`RegisterArtisan`) is self-registration only — the
  handler takes the phone number from the *caller's own* authenticated principal
  (`p.PhoneE164`), ignoring anything in the body, because that's what the artisan app's own
  onboarding needs. The underlying `identity.v1.RegisterArtisan` RPC always accepted an arbitrary
  `phone_e164`; there was just no REST path exposing that to a caller registering someone else. Added
  `POST /clusters/{id}/onboard` (`OnboardClusterArtisan`), scoped to `CLUSTER_OFFICER`/`MINISTRY`,
  as that path — one call per validated CSV row, not a new self-registration bypass. There is no
  batch/bulk RPC, so a 500-row sheet is 500 sequential calls; validation (required columns,
  E.164 phone shape, at least one craft id and language, required state code) happens entirely
  client-side (`lib/bulk-onboard.ts`, unit-tested) before any network call, and the full per-row
  error report is shown before the commit button is even enabled.

- **`/moderation`**: **no backend flagging, counterfeit-detection, or duplicate-detection RPC
  exists anywhere** — `catalog.proto`/`curation.proto` have no such call. This was not built as a
  queue reading from a fabricated endpoint. What is real: `SuspendListing`/`ReinstateListing`
  (`services/core-svc/internal/core/handler/curation.go`) were fully implemented with zero BFF
  wiring — added (`client/catalog.go`, `handler/admin.go`, routes, openapi). And
  `technique_verdict.matches`/`loom_verdict.is_handloom` on a listing's sealed provenance (added to
  `ListingSummary` in Batch 12, already shown on the buyer product page) are genuine model outputs —
  the review queue scans up to 30 published listings (`getListingSummary` per listing; ponytail —
  a scan, not an index, see the page's own comment), sorts model-flagged ones first, and shows the
  claimed/observed technique, match confidence and explanation side by side as the "evidence" a
  human is deciding over. Every suspend requires a typed reason and a click; there is no bulk
  action anywhere on the page, matching the "Do not allow bulk auto-moderation" constraint exactly
  because there is no automated signal to act on in bulk in the first place.

- **`/crafts`**: `ListCrafts`/`GetCraft` (browse) were already wired from earlier batches.
  `RefreshCraftIndex` existed in `ontology.proto` and `ontology-svc` with no BFF wiring — added
  (`client/catalog.go`, `handler/admin.go`, `POST /crafts/refresh-index`, MINISTRY only). **Real
  gap, not worked around**: alias editing across scripts and merging duplicate crafts have no
  backend at all. `OntologyService.ResolveCraftAlias` looks like it might be alias CRUD from its
  name, but reading `ontology.proto` shows it's a free-text mention *resolver* (NER over a search
  query or transcript, returning byte-offset matches) — there is no alias-write or craft-merge RPC
  anywhere. The page states this directly rather than building a form with nowhere to send its
  data.

- **RBAC**: `pkg/auth.RequireRole` already supported multiple accepted roles
  (`RequireRole(ctx, roles ...Role)`); every new cluster/SHG/moderation/craft-index handler in
  `handler/admin.go` gates on `RoleClusterOfficer, RoleMinistry` (moderation and cluster tooling —
  ministry has read/oversight access too) or `RoleMinistry` alone (insights, craft-index refresh —
  matching the brief's "MINISTRY role only" for `/insights`). `RoleClusterOfficer` existed in the
  enum since an earlier batch with no handler checking it anywhere; this batch is its first real
  server-side gate.

- **Idempotency header naming**: `packages/api/src/transport.ts` sends the caller's dedup key as
  `Idempotency-Key`; `services/bff/internal/bff/middleware/idempotency.go`'s replay-cache
  middleware reads `X-Idempotency-Key` — a pre-existing mismatch across the *entire* API client,
  not something introduced here (confirmed via `git blame`-equivalent: both names predate this
  batch). Because an empty/missing key just skips replay protection rather than erroring, every
  existing POST/PATCH/DELETE flow in the app still works, just without the replay-dedup guarantee
  the header was meant to provide. `handler/admin.go`'s own `idempotencyKey()` helper reads
  `Idempotency-Key` (matching what the client actually sends) to forward into each RPC's own
  `idempotency_key` field, which is a separate mechanism from the middleware's replay cache. The
  middleware/client header-name mismatch itself is out of this batch's scope to fix (it touches
  every existing mutating endpoint, not just Batch 13's) and is flagged here for whoever picks it
  up next — the one-line fix is changing `transport.ts` to send `X-Idempotency-Key` or the
  middleware to also accept `Idempotency-Key`.

## For whoever wires the real endpoints

Search the codebase for `MOCK:` (all in `ml-mock.ts`) to find every fabricated shape and its
call site. Each one names what it stands in for. The intended replacement sequence:

1. Design and ship the missing endpoints in `services/bff/openapi.json` (hand-edit — it's
   `//go:embed`'d, not generated) and their handlers.
2. Run `pnpm -F @kalakriti/api api:gen` to regenerate `schema.d.ts`.
3. Add typed wrappers to `packages/api/src/operations.ts` and export them from `src/index.ts`.
4. Replace the corresponding `ml-mock.ts` call in the wizard route with the real call.
5. Delete the now-unused export from `ml-mock.ts`; delete the file once nothing imports it.
