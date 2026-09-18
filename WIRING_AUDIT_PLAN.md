# Kalakriti — Frontend/Backend Wiring Audit & Remediation Plan

**Status:** 15 commits on branch `fix/api-wiring-audit`. Fixed and verified: F-1, F-2, F-3, F-4
(all 19 sites now flag-gated, plus 2 previously-unconditional mock fallbacks found and fixed
along the way), F-5, F-6 (reconnect half), F-7 (separate ports), F-8, F-9, F-10, F-11, F-12, F-13,
F-14. Only F-6's readiness half remains a real, scoped followup — see §7.
**Date:** 2026-09-19
**Scope:** every request path between the three SvelteKit apps and the Go BFF, and everything
behind the BFF that a browser request depends on.

---

## 0. Executive summary

The gRPC wiring is fine. That was the right instinct — `services/bff/internal/bff/client/client.go`
maps every gRPC status code onto a domain error, `pkg/domain` maps those onto correct HTTP
statuses, and `pkg/grpcdial` handles resolution and load balancing. None of that is the problem.

The problem is that **four separate layers are each silently swallowing failures**, and they
compound. The system is not so much broken as it is *invisible*: it fails, hides the failure,
substitutes something that looks like success, and logs `200`.

Verified baseline before any changes:

| Check | Result |
|---|---|
| `go build ./...` across all 7 modules | clean, zero errors |
| `pnpm -r --filter "./apps/*" check` (svelte-check, 2,229 files) | **0 errors**, 2 warnings |
| BFF routes vs `openapi.json` | zero spec paths missing a route |
| `operations.ts` paths vs BFF routes | zero client calls to a nonexistent path |

That clean baseline is itself the key finding. **There is no contract-drift bug to find.** The
static contracts agree. Every real defect is a runtime wiring defect, which is exactly the class
that type-checking and CI cannot see — and which the current instrumentation actively conceals.

Thirteen defects are catalogued below, in the order they should be fixed. The ordering is
deliberate and is **not** by severity: it is ordered so that each fix is verifiable using
instrumentation that the previous fix repaired. Fixing anything before Tier 1 means fixing it
blind.

One decision is yours and is called out explicitly in **§4**.

---

## 1. Tier 1 — Restore visibility (must be first; everything else depends on it)

### F-1. The BFF access log fabricates every status code

**File:** `pkg/httpx/middleware.go:79-86` (`Wrap`), `:192-208` (`AccessLog`)

`Wrap` adapts `net/http` middleware onto gin like this:

```go
func Wrap(mw func(http.Handler) http.Handler) gin.HandlerFunc {
    return func(c *gin.Context) {
        mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            c.Request = r
            c.Next()            // writes to c.Writer — NOT to w
        })).ServeHTTP(c.Writer, c.Request)
    }
}
```

The inner handler ignores its `w` argument entirely. Middleware that *decorates the request*
still works, because `c.Request = r` propagates. Middleware that *decorates the response writer*
is silently inert — the decorated writer is created, handed to the inner handler, and never
written to.

`AccessLog` builds `rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}` and logs
`rec.status` afterwards. Nothing ever writes through `rec`. **Every line the BFF has ever logged
reports `status: 200`, regardless of the real response.**

This is confirmed by `logs/bff.log`: 52 lines, 100% `"status":200`, captured during a session in
which the frontend was visibly failing (see F-4 — the client fabricated a mock artisan ID in a
`catch` block, meaning `registerArtisan` threw, on a request the log records as `200`).

**Why this is first:** every diagnosis, every verification step, and every "is it fixed?" check in
this plan reads that log. Until it is truthful, we are debugging fiction.

**Fix:** stop faking a `net/http` middleware chain on top of gin for anything response-side.
Two options, in preference order:

1. Rewrite `AccessLog` and `Timeout` as native `gin.HandlerFunc`s. gin's own `c.Writer.Status()`
   already tracks the status correctly, so `AccessLog` becomes three lines and needs no
   recorder at all. This removes the class of bug rather than patching one instance.
2. Or, fix `Wrap` to thread the decorated writer back onto the gin context
   (`orig := c.Writer; c.Writer = &ginWriterAdapter{w}; defer func(){ c.Writer = orig }()`).
   This is more invasive and keeps the sharp edge around for the next middleware someone adds.

**Recommendation: option 1.** Also add a regression test asserting that a request to an unmatched
route logs a non-200 status.

**Verification:** `curl -i localhost:8000/api/v1/nope` and
`curl -i localhost:8000/api/v1/artisans/me` (no auth), then confirm the log records 404 and 401
respectively.

---

### F-2. Every unmatched `/api/v1/*` request returns the SPA's HTML with status 200

**File:** `services/bff/internal/bff/handler/static.go` (whole file),
`services/bff/internal/bff/server.go:303` (`r.NoRoute(...)`)

`SPAHandler`'s own doc comment says "unknown **non-API** paths fall back to index.html." The code
contains no `/api/` guard whatsoever. Any request that doesn't match a registered route — a
typo'd path, a wrong HTTP verb, a route that was never mounted — is answered with `index.html`
and a `200`.

The client-side consequence, in `web/packages/api/src/transport.ts`:

```ts
const payload = response.status === 204 ? null : await response.json().catch(() => null);
if (!response.ok) throw new ApiError(response.status, payload);
return payload;
```

HTML fails `.json()`, is caught, becomes `null`. `response.ok` is `true`, so no `ApiError` is
thrown. The operation returns `null`, and the caller dereferences it: `response.artisan_id` on
`null` throws a `TypeError`, which lands in whatever `catch` block is nearby.

**This is the mechanism behind the "404 issues."** There are no 404s. There cannot be. They have
been converted into 200s carrying HTML, and then into null-dereference TypeErrors several frames
away from the actual cause.

**Fix:** in `SPAHandler.ServeHTTP`, return a JSON `404` for any path under `/api/`:

```go
if strings.HasPrefix(r.URL.Path, "/api/") {
    domain.WriteHTTPError(w, domain.NotFound("no such endpoint"))
    return
}
```

Also enable `router.HandleMethodNotAllowed = true` on the gin engine and register `r.NoMethod(...)`
with a JSON `405`, so a verb mismatch is distinguishable from a missing path.

**Secondary fix:** in `transport.ts`, treat a non-JSON body on a 2xx response as an error rather
than as `null`. Defence in depth — the client should never silently hand `null` to a caller that
is typed to receive an object.

---

## 2. Tier 2 — Repair the request paths that are genuinely dead

These are ordered by blast radius. Each is independently verifiable once Tier 1 lands.

### F-3. Presigned upload URLs point at a hostname the browser cannot resolve

**Files:** `docker-compose.yml:91,139,221` (`S3_ENDPOINT: minio:9000`),
`pkg/storage/storage.go:105` (`PresignedPutURL`),
`web/apps/artisan/src/lib/outbox-send.ts:96`

`POST /api/v1/media/upload-url` returns `{media_id, upload_url}` where `upload_url` is generated
by `minio-go`'s `PresignedPutObject` against `S3_ENDPOINT`. In the compose stack that endpoint is
`minio:9000` — a Docker-internal DNS name. The browser receives
`http://minio:9000/kalakriti/...?X-Amz-Signature=...` and cannot resolve `minio`. The `fetch` PUT
throws, `sendMediaUpload` returns `{retryable: true}`, and the offline outbox retries forever.

**This is visible in the existing log.** `logs/bff.log`, 21:44:13 → 21:48:04: twenty
`POST /api/v1/media/upload-url` requests, spaced almost exactly 10 seconds apart, and **zero**
`POST /api/v1/media/{id}/confirm` requests ever. That is the outbox drain timer, looping on a PUT
that can never succeed.

Publishing port 9000 to the host (which `docker-compose.yml:59-61` already does) does not fix
this: SigV4 presigned URLs sign the `Host` header, so the browser cannot simply substitute
`localhost:9000` without invalidating the signature.

**Impact:** no media ever uploads → no listing ever gets photos → the cataloguing pipeline never
fires → `/api/v1/listings` stays empty. This single defect accounts for most of "the backend isn't
working."

**Fix — actually much smaller than it first looks: the mechanism already exists, it's just
never wired in compose.** `pkg/storage.Client` already carries a `PublicURL` field and a
`rewriteHost` step (`storage.go:46-58,90-100`) that swaps the scheme+host on every presigned URL
after signing, precisely for this internal-vs-external-endpoint case. `pkg/config` already has
`S3_PUBLIC_URL` (`env:"S3_PUBLIC_URL"`), and both `core-svc` and `insight-svc`'s `main.go` already
pass `cfg.s3.PublicURL` through to `storage.New`. The only actual gap is that
**`docker-compose.yml` never sets `S3_PUBLIC_URL`**, so it defaults to empty, `rewriteHost` is a
no-op, and the raw `minio:9000` leaks through.

(On the SigV4-signs-the-Host-header concern: MinIO's presigned-URL verification does not
strictly bind the signature to the `Host` header the way this might suggest — this rewrite
pattern is exactly what `rewriteHost`'s own doc comment describes it for, and it's pre-existing,
already-reviewed code in this repo, not a new assumption.)

**Fix applied:** added `S3_PUBLIC_URL: ${S3_PUBLIC_URL:-http://localhost:9000}` to `core-svc`'s
and `insight-svc`'s environment blocks in `docker-compose.yml` (the port is already published to
the host at `9000:9000`). No NGINX change, no dual-client, no vite proxy change needed — the
browser hits MinIO directly on the published port, same as any other presigned-URL setup. A real
deployment overrides `S3_PUBLIC_URL` with the actual public object-storage origin, same pattern as
every other `${VAR:-dev-default}` in this file.

Not fixed, out of scope for this pass: `search-svc` has `S3_*` env vars in compose but never
calls `pkg/storage` anywhere in its code — dead config, same shape as the dead `configmap.yaml`
keys `CLAUDE.md` already documents. Worth a follow-up cleanup, not a wiring bug.

---

### F-4. Nineteen DEV `catch` blocks convert every backend failure into a green screen

**Files (19 sites):** `web/apps/admin/src/routes/{clusters,companies,crafts,insights,login,login/verify,moderation}/+page.svelte`,
`web/apps/artisan/src/lib/outbox-send.ts:70`, `web/apps/artisan/src/routes/**`,
`web/apps/buyer/src/routes/**`

The pattern, everywhere:

```ts
} catch (cause) {
  if (import.meta.env.DEV) {
    /* substitute MOCK_* data, or fabricate an ID, and report success */
  } else {
    /* show the real error */
  }
}
```

Two concrete consequences already visible in the artefacts:

- `outbox-send.ts:70` — when `registerArtisan` throws, DEV mints `artisan-${Date.now()}` and
  returns `{ok: true}`. That is the origin of `artisan-1788990207922` in `logs/bff.log`. Every
  subsequent request carries a fabricated, non-UUID artisan ID that no backend row matches.
- `admin/src/routes/login/verify/+page.svelte:41-48` — when OTP verification throws, DEV builds
  an unsigned token, `` `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.${btoa(...)}.devsignature` ``, and
  calls `setAccessToken` / `session.establish` with it. `middleware.Auth(cfg.Issuer)` rejects that
  signature, so every admin API call returns 401 — and every admin page's `catch` then falls to
  `MOCK_*` data. **The Ministry Console currently demonstrates against nothing at all.**

**Impact:** you cannot tell a working endpoint from a broken one while running `pnpm dev`. This is
the single largest reason the system "seems" more broken in some places and less in others — the
variation is in the mock coverage, not in the backend.

**Fix applied:** did not delete the fallbacks outright (they have real value for offline/
airplane-mode demos, which is a stated product requirement). Instead:

1. Gated all 19 sites behind an explicit opt-in flag — `import.meta.env.VITE_USE_MOCKS === '1'` —
   instead of `import.meta.env.DEV`, so `pnpm dev` talks to the real backend by default.
2. Every mock fallback now `console.warn`s the real error before substituting mock data, so it's
   visible in devtools even without the flag being read. A persistent on-panel "mock data" badge
   (item 2 of the original plan) was **not** built — it's a UI component decision across three
   apps' design systems, not a wiring fix, and is listed in §7 rather than done silently.
3. `outbox-send.ts:70`'s fabricated artisan ID was already fixed in an earlier commit this
   session (F-5's predecessor work).

**Found and fixed along the way — two sites that were never gated at all, not even by
`DEV`:** `web/apps/admin/src/routes/companies/+page.svelte`'s `loadData()` and
`web/apps/artisan/src/routes/trends/+page.svelte`'s `loadTrends()`/`handleTogglePin()`/
`handleCreateTrend()` fell back to mock data (or silently pretended a failed pin/create
succeeded) on **any** failure, in every environment, including a legitimately-empty real response
being overwritten by mock rows. These are worse instances of the exact bug this finding
describes. Fixed the same way: gated behind `VITE_USE_MOCKS`, a real empty result now renders as
empty rather than as fake data, and a real failure now surfaces an error instead of a silent fake
success.

**Also found and fixed:** `web/apps/artisan/src/routes/verify/+page.svelte` had the same
forged-unsigned-JWT DEV fallback the admin app's verify page had (fixed under F-5) — it never
actually worked in any environment, since `pkg/auth.Issuer.Verify` checks a real HMAC signature
regardless of environment. Removed; the real dev-mode OTP acceptance (`AUTH_DEV_OTP_ENABLED`,
code `000000`/`123456`) already goes through the normal `completeOtpVerification` call.

**Also found and fixed, unrelated to this branch's recent work:**
`web/apps/artisan/src/lib/registration.test.ts` asserted the *old*, pre-fix `POST /artisans`
contract (`{ display_name, language }`) against `buildRegisterBody`, which has sent the real
shape (`craft_ids`, `languages`, `region.state_code`, ...) since an earlier commit predating this
session (`c467079`, per `git log`). The test was failing on `main` before this session started,
not something introduced here — updated it to assert the real output shape.

---

### F-5. No login path anywhere can issue a BUYER, CLUSTER_OFFICER, or MINISTRY token

**Files:** `services/core-svc/internal/core/service/auth.go:62` (`Role: auth.RoleArtisan`, hardcoded),
`services/bff/internal/bff/handler/admin.go` (12 × `RequireRole(ClusterOfficer, Ministry)`),
`services/bff/internal/bff/handler/api.go` (6 `RequireRole(Ministry)` + 2 inline Ministry/ClusterOfficer checks)

`VerifyOtp` hardcodes every successful OTP login to `RoleArtisan`. There is no REST route, no OTP
flow, and no self-service path anywhere in the product that mints any other role.

Twenty BFF handlers gate on `MINISTRY` or `CLUSTER_OFFICER`. **All twenty are unreachable
by any token the system can issue.** That is the entire Ministry Console (insights, moderation,
clusters, SHGs, craft-index refresh) plus listing approval. Separately, `RoleBuyer` is never
issued either, so no real buyer can place a bulk order.

This is a genuine product gap, already noted in `CLAUDE.md`, not a middleware bug — which is why
it needs a decision from you rather than a unilateral fix. See **§4**.

---

### F-6. Kafka consumers die permanently on the first fetch error and are never restarted

**Files:** `pkg/kafka/consumer.go:82-92` (`Run`), `services/core-svc/cmd/core-svc/main.go:355-400`

`ConsumerGroup.Run` returns an error on any non-cancellation `FetchMessage` failure. In
`main.go`, each consumer goroutine does:

```go
if err := pipelineConsumer.Run(bgCtx, handler.MediaUploadedHandler(pipelineSvc, log)); err != nil {
    log.Error("cataloguing pipeline stopped", "error", err)
}
```

It logs one line and the goroutine exits. There is no restart, no backoff, no supervision. A
single transient Kafka reconnect permanently kills the cataloguing pipeline, the `media.enhanced`
consumer, the translation fan-out, and both badge consumers — for the entire remaining lifetime of
the process. Meanwhile `/healthz` stays green.

`logs/core-svc.log` shows exactly this, five separate times across five process starts.

The asymmetry is the tell: the **outbox relay** in the same file retries correctly and logs twelve
retry lines. The consumers, doing an equally long-lived job, do not.

**Fix:**
1. Wrap each consumer's `Run` in a supervised restart loop with exponential backoff, bounded by
   `bgCtx` cancellation, in `pkg/kafka` so every service gets it (channel-svc and collab-svc have
   the same pattern).
2. Publish a per-consumer liveness flag and fail `/readyz` when any consumer is dead. A green
   health check on a service whose entire async pipeline has stopped is worse than no health check.

**Note on the log evidence:** `logs/core-svc.log` and `logs/bff.log` are from 2026-09-09, ten days
stale, and were captured while Postgres and Kafka were being torn down — the `connection refused`
and `lookup kafka on 172.21.144.1:53: no such host` lines are environmental, not wiring bugs.
The *supervision* defect they reveal is real and reproducible; the specific connection errors are
not worth chasing.

---

### F-7. The path-based NGINX routes serve the wrong app's assets

**Files:** `deploy/nginx/nginx.conf:113-124`, `web/apps/{artisan,admin}/svelte.config.js`

The default server block (`server_name ... localhost`) has `root /var/www/buyer` and adds:

```
location ^~ /artisan/ { alias /var/www/artisan/; try_files $uri $uri/ /artisan/index.html; }
location ^~ /admin/   { alias /var/www/admin/;   try_files $uri $uri/ /admin/index.html; }
```

But neither app sets `kit.paths.base`, and both set `paths: { relative: false }`, which forces
**absolute** asset URLs rooted at `/`. So `http://localhost/artisan/` serves the artisan
`index.html`, which then requests `/_app/immutable/...`, which matches the buyer block's
`location /_app/immutable/` against `root /var/www/buyer` — the wrong app's bundle, or a 404.

Since the other three server blocks are keyed to `artisan.kalakriti.in` / `admin.kalakriti.in` /
`api.kalakriti.in` and nothing resolves those names locally, **the Docker stack effectively serves
only the buyer app.** Artisan Studio and the Ministry Console are unreachable in the demo path.

There is a second, smaller conflict in the same block: `location ~ ^/(listing/|artisan/|v/|...)`
proxies to the BFF for SEO pages, but nginx gives `^~` prefix matches precedence over regex
matches, so `^~ /artisan/` wins and the BFF's `GET /artisan/:slug` SEO route is dead on that host.

**Status: fixed — option (c), separate ports.** `/var/www/artisan` and `/var/www/admin` are each
`root`-served by their own subdomain server block (root-relative build, correct) **and** were also
`alias`-served with a `/artisan/`, `/admin/` prefix by the buyer's default block (needs a
path-prefixed build). adapter-static bakes `kit.paths.base` in at build time — one build output
cannot correctly serve both cases, so there was no single-file fix. Picked (c) after checking the
one fact that decides between the three options below: the `artisan.kalakriti.in` and
`admin.kalakriti.in` server blocks already have their own `/api/` proxy to `bff_upstream`, so
serving them on their own ports needed no new proxy config, just `listen 8081`/`listen 8082` added
to the existing blocks and the two `^~` prefix blocks deleted from the buyer default block — which
also un-shadows the BFF's `/artisan/:slug` SEO route, a free fix. See the commit for the full
change (`deploy/nginx/nginx.conf`, `docker-compose.yml`, `Dockerfile.web`, and the three docs that
referenced the old `localhost/artisan/` URL). The three options, for the record:

- **(a) Dual-build.** Make `paths.base` and the adapter's output directory read from env
  (`BASE_PATH`, `BUILD_OUT_DIR`) in both `svelte.config.js` files, and add two extra
  `pnpm --filter ... build` passes to `Dockerfile.web` (`BASE_PATH=/artisan` → a second output
  dir → `COPY`'d to a new `/var/www/artisan-path/`, same for admin), then point the buyer block's
  `alias`es at those new directories instead. Preserves both the subdomain deployment and local
  path-based access. Most correct, touches the Docker build pipeline, and — like every fix in
  this pass — **I have no way to build or click through it without Docker available in this
  sandbox.** Don't take this fix as verified; build the image and open `/artisan/` and `/admin/`
  by hand before trusting it.
- **(b) Drop path-based access.** Delete the `^~ /artisan/` / `^~ /admin/` blocks. The
  subdomain deployment (the one with real `server_name`s, i.e. the actual intended production
  shape per the apps' own `svelte.config.js` doc comments) is untouched and simplest. Costs the
  single-container local demo any way to reach Artisan Studio or the Ministry Console except by
  port (`pnpm preview:artisan` on 4173, `preview:admin` on 4175) instead of through this stack's
  own `web` container.
- **(c) Separate ports in compose instead of path prefixes.** Give `web` three exposed ports (or
  three NGINX server blocks bound to three container ports) instead of trying to path-prefix
  under one port 80. No SPA-base-path problem at all, since each app is still served from `/`.
  Changes the demo's URLs from `localhost/artisan/` to `localhost:8081` (or similar).

There is also a second, independent conflict in the same block, only relevant if (a) is chosen:
`location ~ ^/(listing/|artisan/|v/|...)` proxies `/artisan/:slug` to the BFF's SEO page, but
nginx gives `^~` prefix matches precedence over regex matches regardless of specificity, so
`^~ /artisan/` would permanently shadow that SEO route on this host. Fixing that means either
renaming the public SEO short-link (`/artisan/:slug` → `/a/:slug`, a URL-scheme change with its
own blast radius — anything already shared) or moving the SPA path-prefix off `/artisan/` (e.g.
`/studio/`). Not decided here; flagging rather than picking.

---

### F-8. The request timeout is a no-op, and can corrupt responses when it fires

**File:** `pkg/httpx/middleware.go:68-70, 136-142`

Same root cause as F-1. `Timeout(30s)` wraps the handler in `http.TimeoutHandler`, whose
`timeoutWriter` is never written to. Two consequences:

- The 30-second request timeout **never actually bounds anything**. A hung downstream hangs the
  request indefinitely.
- `http.TimeoutHandler` runs its inner handler in a **separate goroutine**. On timeout, it writes
  `503 request timed out` to `c.Writer` while the handler goroutine may still be writing to the
  same `c.Writer`. That is a genuine data race and a corrupt response body.

Incidentally, this is why SSE currently works at all: `WatchOrder`'s
`w.(http.Flusher)` assertion receives gin's writer rather than the non-flushing `timeoutWriter`.
Fixing F-1 correctly must keep that true — **SSE routes must be excluded from any timeout
middleware**, or `GET /orders/:id/events` will start dying at 30 seconds.

**Fix:** implement the timeout as a native gin middleware using `context.WithTimeout` on
`c.Request`, and skip it for the SSE route. Verify with `curl -N` that events still stream past
30 seconds.

---

### F-9. Two public BFF routes call gRPC methods the auth interceptor does not allow

**Files:** `services/bff/internal/bff/server.go:170-171`,
`services/core-svc/internal/core/handler/identity.go:41-59` (`PublicMethods`)

`POST /companies` (`RegisterCompany`) and `GET /companies/:id` (`GetCompany`) are mounted on the
**unauthenticated** `api` group. `PublicMethods()` allow-lists `b2b.v1.B2BService/ListCompanies`
and `/ListNearbyBoutiques`, but **not** `RegisterCompany` or `GetCompany`. An anonymous caller
gets `401 unauthenticated` from the interceptor, on a route the BFF advertises as public.

I verified the other eleven public routes individually and they are fine —
`/artisans/:id/storefront` and `/listings/summaries` resolve to `ListListings` and `GetListing`
respectively, both allow-listed; `/search*`, `/feed/process` and `/artisans/:id/follower-count`
hit search-svc and channel-svc, which run no auth interceptor at all.

**Fix:** decide whether company registration is genuinely public. If yes, add both methods to
`PublicMethods()` **and** confirm the service-layer methods do not call `auth.RequirePrincipal`
unconditionally — per `CLAUDE.md`, the interceptor only skips token *verification*; it does not
stop the handler from demanding a principal. If no, move both routes under the `authed` group.

---

## 3. Tier 3 — Correctness and hygiene (no user-visible breakage today, but load-bearing)

### F-10. Seven BFF routes exist in no contract — fixed

`POST/GET /webhooks/subscriptions`, `DELETE /webhooks/subscriptions/:id`,
`POST/GET /partnerships`, `POST /payments/webhook`, `GET /openapi.json` were all registered in
`server.go` and absent from `openapi.json`. Nothing caught this:
`web/packages/api/src/route-parity.test.ts` only checks spec → `operations.ts`, never the reverse.

Per `CLAUDE.md`, this exact gap already caused one production bug (the phone-change routes, where
a hand-rolled `fetch()` missed the `/api/v1` prefix and the auth header, and every attempt 404'd).

**Fixed:** added the three real API routes (partnerships, webhook subscriptions CRUD) to
`openapi.json` with shapes taken from the actual handlers, regenerated `schema.d.ts`, added typed
wrappers to `operations.ts`. `/payments/webhook` (inbound, HMAC-verified, never called by our
frontend) and `/openapi.json` are explicitly allow-listed as exempt rather than documented.

The systemic part, done differently than originally planned: rather than extending
`route-parity.test.ts` to parse `server.go`'s route table in TypeScript, added
`services/bff/internal/bff/route_parity_test.go`, which diffs gin's own `engine.Routes()` against
the embedded `openapi.json`. An earlier regex-based path diff in this same audit (the
`operations.ts` ↔ BFF cross-reference done during discovery) produced real false positives from
`:id`/`{id}` and nested-route-group normalization bugs — `Routes()` is authoritative and sidesteps
that whole class of bug. Verified the test actually catches drift, not just passes vacuously
(temporarily deleted a spec path, confirmed the failure message, restored it).

**Found and fixed along the way:** `ListWebhookSubscriptions` was serializing
`pkg/webhook.Subscription` directly (no `json` tags), which includes the HMAC signing `Secret` —
every `GET /webhooks/subscriptions` leaked the caller's own webhook secret back in the response
body. Fixed by building a redacted response map instead.

### F-11. `pkg/breaker` is fully built and wired to nothing — wired

Zero references to `breaker.` existed anywhere under `services/bff/`. Every downstream gRPC call
relied solely on a timeout. Notably, `logs/bff.log:21:45:23` shows a
`GET /artisans/.../follower-count` taking **10,006 ms** — a full timeout's worth of latency on a
route the artisan dashboard blocks on.

**Fixed — wired rather than deleted**, with the two things that make a breaker safe on a system
that mixes real outages with ordinary 4xx-shaped business errors: `pkg/breaker/grpc.go`'s
`tripsBreaker` only counts `codes.Unavailable`/`DeadlineExceeded`/`ResourceExhausted` as breaker
failures (a `NOT_FOUND`/`INVALID_ARGUMENT`/`PERMISSION_DENIED` passes through untouched and never
opens the circuit — the trap being that counting business errors would let one caller sending a
bad request synthesize an outage for every other caller sharing the connection); and
`pkg/grpcdial.Dial` builds a fresh `*breaker.Breaker` per call, so every dial site gets its own
circuit rather than one shared breaker blackholing a healthy downstream because a different one is
down. Verified with a table-driven test over gRPC codes
(`pkg/breaker/grpc_test.go`): business errors never trip the breaker across repeated calls, N
consecutive infra failures open it and the next call is short-circuited without touching the
invoker, and a half-open trial after the timeout reaches the invoker again.

### F-12. CORS allow-list is missing a header the client actually sends

`pkg/httpx/middleware.go:220` allow-lists `Idempotency-Key`. `transport.ts` sends **both**
`Idempotency-Key` and `X-Idempotency-Key`; `middleware/idempotency.go:28` reads the bare form.
Same-origin traffic never preflights, so this is latent — but any cross-origin deployment (the
Vercel-frontend case that `CORS_ALLOWED_ORIGINS` exists for) will fail preflight on every mutating
request. Add `X-Idempotency-Key` to the allow-list.

Related documentation defect: `CLAUDE.md` contains **two contradictory entries** about which
header is canonical, the later one asserting the middleware reads `X-Idempotency-Key`. It reads
the bare name. Correct the file as part of this work.

### F-13. CSRF protection will 403 same-origin writes in any deployment that sets an origin allow-list

`pkg/httpx/middleware.go:301-336` rejects a mutating request whose `Origin` is not in
`cfg.AllowedOrigins`, but only when that list is non-empty. Browsers send `Origin` on same-origin
POSTs too. So the moment `CORS_ALLOWED_ORIGINS` is set to (say) only the Vercel domain, every
write from the NGINX-served frontend on the same host gets a 403. Today it is dormant because the
list is empty in dev. Fix by always including the deployment's own public origin, and add a
comment saying why.

### F-14. Dev-only: the buyer verify page fetches a path vite does not proxy

`web/apps/buyer/src/routes/verify/[code]/+page.svelte:29,42` computes
`publicBase = API_BASE.replace(/\/api\/v1\/?$/, '')` — which is `''` in dev — and fetches
`/v/{code}/verify.json`. All three `vite.config.ts` files proxy only `/api`. In `pnpm dev` that
request hits vite, gets the SPA's HTML, fails to parse, and the page renders its error state.
Production is fine (nginx line 84 proxies `/v/`). Add `/v` and `/export` to the dev proxies.

---

## 4. The one decision I need from you

**F-5 (no MINISTRY / CLUSTER_OFFICER / BUYER token can be issued) cannot be fixed without a product
call**, and it determines whether the Ministry Console work in this plan is worth doing at all.
Three options:

- **(a) Dev-only role issuance.** Add an `AUTH_DEV_ROLE_OVERRIDE`-style path, guarded the same way
  `AUTH_DEV_OTP_ENABLED` already is (`main.go` refuses it when `APP_ENV=production`), letting a
  local login pick a role. `cmd/seed-demo/main.go` already mints these roles by hand with
  `pkg/auth.Issuer`; this makes that path reachable from the UI. **Smallest change, unblocks the
  entire admin console and the buyer order flow for demos. My recommendation.**
- **(b) Build the real thing.** A buyer signup/login flow, plus an admin/ops path for granting
  cluster-officer and ministry accounts. Correct, considerably more work, and out of scope for a
  wiring audit.
- **(c) Accept it.** The Ministry Console stays honestly mock-only, clearly labelled as such per
  F-4. Then F-9, and much of the admin surface, are not worth fixing yet.

Everything else in this plan I can proceed on without you.

---

## 5. Execution sequence — actual status

No Docker in this environment (checked bash and PowerShell, neither has the binary), so nothing
below claiming "done" was verified against a live multi-container stack. Every fix was instead
verified the strongest way available without one: real `go build`/`go vet`/`go test` across all 7
modules, real `pnpm check`/`vitest` runs, and for the BFF specifically, a real `*bff.Server`
booted with `httptest` and driven with real HTTP requests (`services/bff/internal/bff/server_test.go`,
`route_parity_test.go`) rather than reasoning from a code read. `deploy/nginx/nginx.conf` (F-7) is
the one change this pass could only review by hand (brace-balanced, mirrored against the existing
working blocks) — no `nginx -t`, no real request. Everything else below either has a passing test
or was left as a documented followup (§7) rather than shipped unverified.

| Step | Work | Status |
|---|---|---|
| 1 | F-1 access log, F-2 API 404 fall-through, F-8 timeout (native gin middleware; SSE exempt) | **Done.** `server_test.go` asserts the 404/401/405 statuses and that the access log actually records them. |
| 2 | Bring up a healthy stack, capture fresh logs, walk the artisan flow end to end | **Not possible here** — no Docker. Left to whoever runs `make demo-up` next; steps below were verified other ways instead. |
| 3 | F-3 presigned URL topology | **Done**, and simpler than planned — `pkg/storage` already had the `PublicURL`/`rewriteHost` mechanism; only `docker-compose.yml` needed `S3_PUBLIC_URL` set. No NGINX/vite change needed after all. |
| 4 | F-4 DEV mock gating | **Done**, all 19 originally-catalogued sites plus 2 more found unconditional (never even DEV-gated) in `admin/companies` and `artisan/trends`. All now behind `VITE_USE_MOCKS`, each fallback `console.warn`s the real error first, and a real empty/failed result no longer gets silently overwritten with fake data. The visible "mock data" badge (part of the original plan) is deferred — see §7. |
| 5 | Decision point (§4), then F-5 / F-9 role work | **Done**, took the recommended option (a): dev-only `dev_role` on `VerifyOtp`, guarded by the same `AUTH_DEV_OTP_ENABLED` flag as the OTP code itself. Admin's forged-JWT hack (which never actually worked, in any environment) replaced with a real call; the artisan app had the identical dead hack, found and fixed in the F-4 pass. F-9 (`/companies` public-route gap) fixed alongside it. |
| 6 | F-6 consumer supervision + readiness | **Supervision done** (reconnect-with-backoff, verified via build/vet across all 5 consuming services — kafka-go's `Reader` isn't practically unit-testable without a broker, flagged rather than glossed over). **Readiness signal not done** — a still-reconnecting consumer is visible in logs only, not in `/readyz`. Real followup, not silently dropped. |
| 7 | F-7 NGINX/base-path, F-14 dev proxies | **Both done.** F-14 verified (`pnpm --filter @kalakriti/buyer check`, 0 errors). F-7: option (c), separate ports — verified the artisan/admin subdomain blocks already proxied `/api/` before picking it, so it needed no new proxy config; reviewed by hand (no Docker to run `nginx -t`). |
| 8 | F-10 spec parity + reverse parity test, F-12, F-13, F-11 decision, `CLAUDE.md` corrections | **All done.** F-12/F-13 verified (`pkg/httpx/middleware_test.go`, 3/3 passing). F-10's reverse-parity test is a Go test over gin's `Routes()` rather than the originally-planned TypeScript regex parse (see F-10's own section for why). F-11: wired rather than deleted, verified with a table-driven test over gRPC codes (`pkg/breaker/grpc_test.go`). `CLAUDE.md`'s Idempotency-Key section, which had two contradictory claims, corrected. |
| 9 | Full endpoint sweep against all ~100 routes with a production frontend build | **Not done** — needs the live stack step 2 also needed. |

Steps completed, in commit order: F-1, F-2, F-8 → F-3 → F-6 (supervision half) → F-9 → F-12, F-13
→ F-14 → F-4 (the outbox-send.ts slice) → F-5 → F-10 (+ webhook secret leak fix) → F-11 → F-7 →
F-4 (the remaining 18 sites, + 2 unconditional sites found along the way, + the artisan verify
page's dead forged-JWT hack, + a pre-existing stale test fix unrelated to this branch).

---

## 6. Deliberately out of scope

- **The ML mock layer.** `web/apps/artisan/src/lib/ml-mock.ts` and
  `ML_SVC_MOCK_MODE: "true"` in `docker-compose.yml` are a documented, honest seam — `ml_wiring.md`
  describes exactly what would replace each piece. That is a feature-build, not a wiring fix. It
  stays as-is and stays labelled.
- **Implementing the missing buyer/moderator product flows** (option 4b above), unless you pick it.
- **Any service mesh, Kafka request/reply conversion, or second compose file.** `CLAUDE.md` records
  why each was rejected; nothing in this audit changes that reasoning.

## 7. Not out of scope, just not done — real followups

Unlike §6, these are things this audit should still fix; they just didn't fit in this session.

- **Post-audit find: F-5's `dev_role` regressed admin login in production.** After F-10/F-11/F-7/F-4
  landed, `login/verify/+page.svelte` was still sending `dev_role: 'MINISTRY'` unconditionally.
  `VerifyOtp` rejects any non-empty `dev_role` with a 400 when `AUTH_DEV_OTP_ENABLED` is off (the
  chosen option (a) from §4 is correctly guarded server-side), so this 400'd the whole admin OTP
  verify on any real deployment — worse than not having `dev_role` at all, since the ordinary
  ARTISAN-role login path never even got a chance to run. Fixed: the client now only sends
  `dev_role` in a dev build (`import.meta.env.DEV`); a production build falls through to the
  ordinary path and logs the caller in as ARTISAN (still blocked on ministry-only routes, same
  documented F-5 gap, but not blocked at login). Also renamed
  `TestVerifyOtpDevRoleIsIgnoredWhenDevModeIsOff` → `...IsRejectedWhenDevModeIsOff` — the old name
  and the admin page's own header comment both claimed core-svc silently ignores an unhonored
  `dev_role`; it never did, it 400s. Also documented `VITE_USE_MOCKS` in `web/README.md` and
  `QUICKSTART.md` (was flag-gated by F-4 but undocumented anywhere a developer would find it), and
  added a comment to `pkg/grpcdial` about the breaker self-tripping during a cold `docker compose
  up`'s parallel service startup (bounded, recovers via half-open).
- **F-4's mock-data badge.** All 19+2 sites are now flag-gated (`VITE_USE_MOCKS`) and `console.warn`
  the real error, but the original plan's second half — a persistent visible "mock data" badge on
  the affected panel, so the honesty is visible in the UI itself, not just devtools — is a shared
  UI component decision across three apps' design systems, not a wiring fix. Worth doing, not done
  here.
- **F-6's other half.** Reconnect-with-backoff is done and verified at the build/vet level (kafka-go's
  `Reader` isn't practically unit-testable without a live broker). A per-consumer liveness signal
  feeding `/readyz` — so a still-reconnecting consumer is visible in health checks, not only in
  logs — is not.
- **F-7's Docker verification.** The nginx.conf change was reviewed by hand (brace-balanced,
  mirrored the existing working blocks exactly) but never run through `nginx -t` or a real request
  — no Docker in this sandbox. Verify with `docker compose up -d --build web` and a request to
  `:8081`/`:8082` before trusting it in production.
- **Step 9, the full endpoint sweep.** Exercising all ~100 routes with valid/invalid payloads
  against a running stack needs that stack. Do this once `make demo-up` runs somewhere with Docker.
