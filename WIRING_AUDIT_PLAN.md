# Kalakriti — Frontend/Backend Wiring Audit & Remediation Plan

**Status:** proposed, not executed. Nothing in this document has been changed in the codebase.
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

**Fix (recommended): route object storage through NGINX and presign against the public origin.**

1. Add to `deploy/nginx/nginx.conf`, in each server block:
   ```
   location /s3/ {
       proxy_pass http://minio:9000/;
       proxy_set_header Host $host;
       client_max_body_size 100m;
   }
   ```
2. Add a `S3_PUBLIC_ENDPOINT` config field to `pkg/config`, defaulting to `S3_ENDPOINT` when
   unset. `pkg/storage` keeps one `minio.Client` for its own server-side operations (internal
   endpoint) and a second, presign-only client bound to the public endpoint.
3. Set `S3_PUBLIC_ENDPOINT: localhost/s3` in `docker-compose.yml`, and add
   `'/s3': { target: 'http://localhost:9000', rewrite: p => p.replace(/^\/s3/, '') }` to all three
   `vite.config.ts` proxies so the dev path works too.

**Alternative (simpler, worse):** proxy the upload bytes through a new BFF route
`PUT /api/v1/media/{id}/blob`. This removes the hostname problem entirely but puts up to 100 MB
of video through the BFF's memory and defeats the point of presigning. Only worth it if the NGINX
route proves impractical.

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

**Fix:** do not delete the fallbacks outright (they have real value for offline/airplane-mode
demos, which is a stated product requirement). Instead:

1. Gate them behind an explicit opt-in flag — `VITE_USE_MOCKS=1` — rather than `import.meta.env.DEV`,
   so `pnpm dev` talks to the real backend by default.
2. Whenever a mock fallback fires, log the real error to the console **and** render a persistent
   visible "mock data" badge on the affected panel. The project's own stated value is honesty about
   mock-vs-real; a silent substitution violates it.
3. Never fabricate an identifier that will be sent back to the server. `outbox-send.ts:70` should
   leave the entry in the outbox as retryable, not invent an artisan ID.

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

**Fix:** set `kit.paths.base = '/artisan'` and `'/admin'` in the respective `svelte.config.js`
(behind an env flag if the subdomain deployment is also wanted), rebuild, and re-check the
`location` precedence. Rename the SEO artisan route to `/a/:slug` to remove the collision, or
scope the `^~` blocks to hostnames that don't also serve SEO pages.

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

### F-10. Seven BFF routes exist in no contract

`POST/GET /webhooks/subscriptions`, `DELETE /webhooks/subscriptions/:id`,
`POST/GET /partnerships`, `POST /payments/webhook`, `GET /openapi.json` are all registered in
`server.go` and absent from `openapi.json`. Nothing catches this:
`web/packages/api/src/route-parity.test.ts` only checks spec → `operations.ts`, never the reverse.

Per `CLAUDE.md`, this exact gap already caused one production bug (the phone-change routes, where
a hand-rolled `fetch()` missed the `/api/v1` prefix and the auth header, and every attempt 404'd).

**Fix:** add the five real API routes to `openapi.json` with shapes taken from the actual handlers,
regenerate `schema.d.ts`, add typed wrappers to `operations.ts`, and — the systemic part — extend
`route-parity.test.ts` to assert **route → spec** parity by parsing `server.go`'s route table.
`/payments/webhook` (inbound, HMAC-verified, never called by our frontend) and `/openapi.json` can
be explicitly allow-listed as exempt.

### F-11. `pkg/breaker` is fully built and wired to nothing

Zero references to `breaker.` exist anywhere under `services/bff/`. Every downstream gRPC call
relies solely on a timeout. Notably, `logs/bff.log:21:45:23` shows a
`GET /artisans/.../follower-count` taking **10,006 ms** — a full timeout's worth of latency on a
route the artisan dashboard blocks on. Either wire the circuit breaker into the BFF's gRPC
clients, or delete it; a half-built resilience layer is worse than an honest absence.

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

## 5. Execution sequence

| Step | Work | Gate before moving on |
|---|---|---|
| 1 | F-1 access log, F-2 API 404 fall-through, F-8 timeout (native gin middleware; SSE exempt) | `curl` a bad route and an unauthenticated route; log shows 404 and 401 |
| 2 | Bring up a healthy `make demo-up` stack, capture fresh logs, walk the artisan flow end to end | A log that now contains real non-200 statuses |
| 3 | F-3 presigned URL topology | Photo upload completes; `POST /media/{id}/confirm` appears in the log |
| 4 | F-4 DEV mock gating (`VITE_USE_MOCKS`, visible badge, no fabricated IDs) | `pnpm dev` surfaces real errors instead of mock data |
| 5 | **Decision point (§4)**, then F-5 / F-9 role work as chosen | An admin login that reaches a real 200, or a documented decision not to |
| 6 | F-6 consumer supervision + readiness | Kill and restart Kafka; consumers recover; `/readyz` went red meanwhile |
| 7 | F-7 NGINX/base-path, F-14 dev proxies | All three apps load with their own assets on `localhost` |
| 8 | F-10 spec parity + reverse parity test, F-12, F-13, F-11 decision, `CLAUDE.md` corrections | `pnpm test` green; per-module `go build` and svelte-check still clean |
| 9 | Full endpoint sweep: exercise all 100 routes with a valid and an invalid payload, against a **production** frontend build | Filled-in status column for every route |

Steps 1–2 are strictly sequential. Steps 3, 6, and 7 are independent of one another and can be
parallelised. Step 9 is the acceptance gate and should not be started until 1–8 are complete.

---

## 6. Deliberately out of scope

- **The ML mock layer.** `web/apps/artisan/src/lib/ml-mock.ts` and
  `ML_SVC_MOCK_MODE: "true"` in `docker-compose.yml` are a documented, honest seam — `ml_wiring.md`
  describes exactly what would replace each piece. That is a feature-build, not a wiring fix. It
  stays as-is and stays labelled.
- **Implementing the missing buyer/moderator product flows** (option 4b above), unless you pick it.
- **Any service mesh, Kafka request/reply conversion, or second compose file.** `CLAUDE.md` records
  why each was rejected; nothing in this audit changes that reasoning.
