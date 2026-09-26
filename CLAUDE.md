# CLAUDE.md

Project-specific knowledge for working in this repo, accumulated from debugging
`make up` end-to-end. Read this before touching local dev infra, migrations,
or docker-compose.yml.

## Local dev prerequisites (beyond the README table)

`make up` now runs `make proto sqlc` before `docker compose up`. That needs,
on the host, in addition to Go/Docker/Python/uv:

- `protoc-gen-go` and `protoc-gen-go-grpc` installed as local binaries:
  ```sh
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  ```
- `$(go env GOPATH)/bin` (usually `~/go/bin`) on `PATH`. This is the #1 cause
  of a fresh-looking `make up` failure even when the plugins are installed —
  `go install` puts them there, but many shells don't have that dir on `PATH`
  by default. Symptom: `buf generate` fails with
  `exec: "protoc-gen-go": executable file not found in $PATH` (or
  `protoc-gen-go-grpc`). Fix: add `export PATH="$HOME/go/bin:$PATH"` to
  `~/.zshrc`/`~/.bashrc`, then open a new shell (or `source` the rc file).
- `pkg/pb/*` and every service's `internal/*/repo/db/` are `.gitignore`d
  (generated code, correctly not committed). `make up`'s new `proto sqlc`
  dependency regenerates them automatically now — you should never need to
  run `make proto`/`make sqlc` by hand before `make up` anymore. If you ever
  bypass the Makefile and run `docker compose up` directly on a fresh clone,
  this will bite you.

## Disk space

Docker's data root (`/var/lib/docker`) lives on whatever partition is mounted
at `/` — on a tight root partition (common on dual-boot / small-SSD laptops),
building this stack's ~7 Go services + web frontend from scratch can exhaust
it, especially combined with pacman package cache / systemd journal / flatpak
also living on `/`. Symptom: Go build fails mid-compile with
`no space left on device`, or Postgres/other containers fail to start with
the same error even after images built successfully (build cache silently
eats the freed headroom back up).

Fastest safe checks/fixes when this happens:
```sh
df -h /                          # confirm root partition is actually full
docker system df                 # see what Docker is holding
docker builder prune -a -f       # biggest safe win — build cache is fully regenerable
docker container prune -f        # clears stopped containers from failed attempts
sudo paccache -rk1               # Arch: trims pacman package cache (~GBs)
sudo journalctl --vacuum-time=3d # trims systemd journal
```
Don't blanket-run `docker system prune -a --volumes` — it can delete named
volumes (e.g. `hf-cache` for ml-svc) that hold slow-to-regenerate downloaded
data. Prune build cache and containers first; only remove images if still short.

One `docker builder prune -a -f` is not a one-time fix on a genuinely tight
partition (seen recurring on a 46G root with ~1GB headroom after a prune) —
each subsequent multi-service rebuild (e.g. `docker compose up -d --build`
with no service names, rebuilding all ~7) can refill the cache faster than
expected and hit the wall again mid-build, or even crash Postgres itself
(`PANIC: could not write to file "pg_logical/replorigin_checkpoint.tmp": No
space left on device`, which then crash-loops on WAL replay until space is
freed). On a tight partition, prefer rebuilding one or two services at a
time (`docker compose up -d --build <service>`) over a blanket rebuild, and
re-check `df -h /` between them. `paccache`/`journalctl --vacuum` need a
password for `sudo` in a non-interactive shell — can't be run silently on
the user's behalf; ask them to run those two directly if builder+container
pruning alone isn't enough headroom.

## docker-compose.yml gotchas already fixed once (don't reintroduce)

- **Postgres 18+ volume mount**: must be `pg_data:/var/lib/postgresql` (the
  whole data root), not `.../data`. Images 18+ manage a version-specific
  subdirectory themselves (`pg_ctlcluster` convention) and refuse to start
  if something's already mounted at the old `.../data` path.
- **Kafka `CLUSTER_ID`**: must be a base64url-encoded 16-byte UUID (KRaft
  requirement), not an arbitrary string. Generate one with:
  ```sh
  python3 -c "import uuid, base64; print(base64.urlsafe_b64encode(uuid.uuid4().bytes).rstrip(b'=').decode())"
  ```
- **Env var names must match `pkg/config` struct tags exactly.** The Postgres
  DSN env var is `POSTGRES_DSN` (not `DATABASE_URL`), and object storage is
  `S3_ENDPOINT`/`S3_ACCESS_KEY`/`S3_SECRET_KEY`/`S3_BUCKET`/`S3_USE_SSL` (not
  `MINIO_*`) — see `pkg/config/config.go`. `S3_BUCKET` must actually exist in
  MinIO before core-svc/search-svc/insight-svc will start (`pkg/storage`
  errors if the bucket is missing rather than creating it) — the compose file
  has a `minio-init` one-shot service (`minio/mc`) that creates the
  `kalakriti` bucket; services depend on it with
  `condition: service_completed_successfully`.
- **JWT_SECRET must be ≥32 bytes** (`pkg/config` / bff enforce this at
  startup) — a short "obviously fake" dev secret will crash-loop the service.

## Migrations are NOT run by `make up`

`make up` only starts infra containers. Schema (including `CREATE EXTENSION
vector` etc.) requires a separate `make migrate-up`. Symptom if skipped: every
service that touches Postgres crash-loops with
`registering pgvector types: vector type not found in the database`. Run
`make migrate-up` once after `make up` succeeds (or use `make demo-up`, which
bundles infra + migrate + seed).

Migrations `025_audit_log.sql`, `026_fraud_detection.sql`, and
`027_webhooks.sql` had table/column names that didn't match the real schema
(`payment_splits`→`payment_split`, `bulk_orders`→`bulk_order`,
`qc_results`→`qc_result`, `status`→`state` on `bulk_order`/`order_lot`-derived
tables, wrong enum casing, a reference to a `users` table that doesn't exist
— buyer identity is external, `bulk_order.buyer_id` is opaque `text`). These
are fixed as of commit history around "hopefully-1.0"/"hopefully-1.67" — if
you add new migrations with triggers/functions referencing other tables,
double-check column names against the actual `CREATE TABLE` in the migration
that defines them; this schema was largely AI-authored and had several
never-tested mismatches.

## Go workspace vs. Docker build mismatch

The repo root has a `go.work` that pulls hash verification from
`go.work.sum`, which can mask a genuinely incomplete per-service `go.sum`
(missing a `.../go.mod h1:...` line). Local `go build` in workspace mode
succeeds; the Docker build (which only `COPY`s each service's own `go.mod`/
`go.sum`, not `go.work`) fails with `missing go.sum entry for go.mod file`.
To reproduce/fix the real per-service `go.sum` locally:
```sh
cd services/<svc> && GOWORK=off go mod download <missing-module>
```

## sqlc nullable-type pattern

Nullable sqlc-generated params use a `NullX{X Value; Valid bool}` struct
(e.g. `db.NullListingType`, `db.NullListingState`), not a `*X` pointer. Code
that builds these params by hand (not sqlc-generated) has gotten this wrong
before (`search-svc/internal/search/repo/repo.go`,
`core-svc/internal/core/repo/catalog.go`) — always check the actual generated
param struct's field type rather than assuming a pointer.

## `make demo-up` was pointed at a stale, broken compose file (fixed)

There is no separate "production" compose file — `docker-compose.yml` already
includes the NGINX `web` service and is the one true maintained stack.
`docker-compose.full.yml`, which `make demo-up` used to `-f` into, was an
abandoned fork frozen from *before* every fix in the gotchas section above:
invalid Kafka `CLUSTER_ID`, old Postgres `.../data` volume path, `DATABASE_URL`/
`MINIO_*` env vars instead of `POSTGRES_DSN`/`S3_*`, a sub-32-byte
`JWT_SECRET`, wrong gRPC ports, a nonexistent `./kalakriti` build context, and
no `minio-init`. It had zero services `docker-compose.yml` doesn't already
have. Fixed by pointing `demo-up` at `docker-compose.yml` (via `$(COMPOSE)`)
instead, with `proto sqlc` added as a prerequisite like `up` has. If
`docker-compose.full.yml` reappears or gets edited again, it's very likely
someone patching the wrong file — check `docker-compose.yml` first.

`demo-up`'s seed step (`go run scripts/seed/main.go`) was *also* stale — it
inserts into a `users`/`orders`/`order_allocations` schema that predates the
real migrations (no `users` table exists; buyer identity is external, per the
migrations section above). Fixed by having `demo-up` call `make seed` (the
real, working craft-ontology loader) instead. There is still no seed data for
artisans/listings/orders against the current schema — `/api/v1/listings`
correctly returns `{"listings":[]}` until something creates real listings
through the app; this is a documented gap in `web/FRONTEND.md`, not a bug.

## core-svc was rejecting every anonymous public-read RPC (fixed)

The BFF (`services/bff/internal/bff/server.go`) intentionally exposes several
routes with no JWT required: `GET /crafts`, `/crafts/:slug`, `/listings`,
`/listings/:id`, `/artisans/:id/storefront`, `/feed/process`. But core-svc's
gRPC auth interceptor
(`services/core-svc/internal/core/handler/identity.go`'s `PublicMethods()`)
only whitelisted the 3 identity RPCs (`RequestOtp`/`VerifyOtp`/`RefreshToken`)
— every other RPC required a bearer token the BFF has no reason to hold for
an anonymous buyer request. Symptom: any of the "public" BFF routes above
returns `401 {"error":"unauthenticated"}` (interceptor rejects it) or, once
the interceptor allow-lists the method, `403 {"error":"forbidden"}` (the
service-layer code still calls `auth.RequirePrincipal` unconditionally).

Fixed by:
- Adding `catalog.v1.OntologyService/ListCrafts`, `.../GetCraft`,
  `catalog.v1.CatalogService/ListListings`, `.../GetListing`, `.../GetProvenanceByShortCode`,
  and `identity.v1.IdentityService/GetArtisan` to `PublicMethods()`.
- `services/core-svc/internal/core/service/ontology.go`: `GetCraft`/
  `GetCraftByCode`/`ListCrafts` no longer call `auth.RequirePrincipal` at all
  — craft ontology is reference data with no ownership concept, same as
  `GetArtisan`/`GetProvenanceByShortCode` already were.
- `services/core-svc/internal/core/service/catalog.go`: `GetListing` only
  requires a principal for a *non-published* listing (via `authoriseFor`,
  which already handles the anonymous-caller case); `ListListings` uses
  `auth.PrincipalFrom` (ok-pattern, never errors) instead of
  `auth.RequirePrincipal`, so a token-less caller cleanly falls through to the
  existing "buyer browsing may only ever see live listings" clamp instead of
  getting rejected before that logic even runs.

If you add a new public BFF route backed by a core-svc RPC, both layers need
updating: the interceptor's `PublicMethods()` set *and* the service method
itself (don't assume `RequirePrincipal`/`RequireRole` is optional just
because the RPC is in the public-methods list — the interceptor only skips
verifying a token; it doesn't stop the handler from still demanding one).
Note `search-svc` and `collab-svc` currently run no gRPC auth interceptor at
all (`grpc.NewServer()` with no chain) — trust is implicit, resting entirely
on the BFF being the only caller with network access to them.

## Vite dev-server proxy pointed at the wrong BFF port

`web/apps/{buyer,artisan,admin}/vite.config.ts` all hardcode the dev-mode
`/api` proxy target as `http://localhost:8080`. That was core-svc's HTTP port
in a pre-fix compose file; the actual BFF (the only thing dev-mode should ever
proxy to) listens on `:8000` — see `ADDR: ":8000"` in the `bff` gotcha above.
With nothing on 8080, every `pnpm dev:*` API call fails with vite's
`http proxy error: ... AggregateError [ECONNREFUSED]`. This is unrelated to
Kafka or any other docker-compose service — it's purely the Node dev-server
proxy target, checked before the request ever leaves your machine. Fixed by
pointing all three configs' `server.proxy['/api'].target` at `:8000`. Vite
does not hot-reload `vite.config.ts` — restart `pnpm dev:*` after touching it.

## `.env.example` had several required-field name mismatches (fixed)

Same class of bug as the docker-compose gotchas above, just in the template
file people copy to `.env` before running a service directly (not through
`docker compose`, which hardcodes its own correct env vars and never reads
`.env`). `pkg/config` uses `env:"...,required"` tags with no fallback, so a
wrong name here doesn't get ignored — the service refuses to start. Fixed:
`ENV` → `APP_ENV`, `KAFKA_BOOTSTRAP_SERVERS` → `KAFKA_BROKERS`, `MINIO_ENDPOINT`/
`MINIO_BUCKET`/`MINIO_USE_SSL`/`MINIO_REGION` → `S3_ENDPOINT`/`S3_BUCKET`/
`S3_USE_SSL`/`S3_REGION` (plus added the missing `S3_ACCESS_KEY`/`S3_SECRET_KEY`
— `MINIO_ROOT_USER`/`MINIO_ROOT_PASSWORD` only bootstrap the MinIO *server*
container, they're not what a Go service reads), `JWT_ACCESS_TOKEN_TTL`/
`JWT_REFRESH_TOKEN_TTL` → `JWT_ACCESS_TTL`/`JWT_REFRESH_TTL`. Also fixed a
copy-paste bug where `CHANNEL_SVC_ADDR` pointed at collab-svc's HTTP port
(8083) instead of channel-svc's own (8084), and a fictional
`INSIGHT_SVC_HTTP_PORT` (insight-svc has one gRPC-only listener, no HTTP
port at all). None of this affects `docker compose up` — only the
"run a service directly with `go run`" workflow in the README.

## `demo-up`/`demo-reset`/`seed-data` Makefile targets were split-brained

Only `demo-up` got fixed in an earlier pass; `demo-reset` still shelled out to
the stale `docker-compose.full.yml` and `seed-data` still called the broken
`scripts/seed/main.go` (see the `make demo-up` section above for why both are
wrong). Fixed both to match `demo-up`: `demo-reset` now uses `$(COMPOSE)`
(plain `docker-compose.yml`), `seed-data` is now an alias for `make seed`
(craft ontology only — there is still no working artisan/listing/order
seeder against the current schema).

## The BFF's REST auth flow has no `challenge_id` — don't trust docs that show one

Every doc found this session (README, QUICKSTART, docs/API.md, before they
were corrected) described a 3-field OTP flow with a `challenge_id` round
trip, phone as `phone_e164`, and paths `/auth/otp`/`/auth/verify`. The real
BFF routes (`services/bff/internal/bff/server.go`) are
`POST /auth/otp/request` and `POST /auth/otp/verify`, and the actual request
bodies (`services/bff/internal/bff/handler/api.go`) are just
`{"phone": "..."}` and `{"phone": "...", "otp": "..."}` — no challenge id at
all. `challenge_id` is real, but only in the internal
`identity.v1.IdentityService` gRPC contract the bff calls on your behalf; it
never crosses the REST boundary. `POST /auth/otp/request` also always
returns the same `{"status":"otp_sent",...}` body regardless of whether the
phone is registered (deliberate anti-enumeration design) — don't expect a
`challenge_id` back to feed into the verify call.

## Idempotency header mismatch between frontend and bff (fixed)

`services/bff/internal/bff/middleware/idempotency.go` read
**`X-Idempotency-Key`** while `web/packages/api/src/transport.ts` sends
**`Idempotency-Key`** (no `X-` prefix). Every mutating BFF route wrapped in
`withIdempotency` (artisan/listing/order/statement/cluster/SHG creation,
moderation actions) therefore never actually saw the frontend's key — a
retried write from the offline outbox was indistinguishable from a new one
server-side. This was a silent retry-safety gap, not a visible request
failure: `API_BASE` defaults to same-origin `/api/v1`, dev proxies through
Vite to `:8000`, and prod serves web + API from the same compose stack, so
the browser's CORS preflight path (which only allowlisted
`X-Idempotency-Key` in `pkg/httpx/middleware.go`, and is only mounted at all
when `AllowedOrigins` is configured) likely never ran for same-origin
traffic. Fixed by standardizing on the bare `Idempotency-Key` name
everywhere: the replay middleware, `idempotencyKeyFrom` in
`services/bff/internal/bff/handler/api.go`, and the CORS allowlist (the
`admin.go` handlers already read the bare name correctly and needed no
change) — the CORS allowlist was wrong in the same direction and worth
fixing regardless, for any deployment that does cross-origin the browser
frontend. Integration/unit tests that were asserting the old header name
were updated to match — they were passing only because both sides of the
test shared the same wrong name.

## Outbound webhook subscription has no REST endpoint

`pkg/webhook.Manager` (subscription CRUD, HMAC signing, retry-with-backoff
delivery worker) and the `webhook_subscriptions`/`webhook_deliveries` tables
(migration 027) are fully built, but `services/bff/internal/bff/server.go`
never mounts a route for it. The bff's only webhook-related route is the
*inbound* payment callback, `POST /api/v1/payments/webhook` — unrelated.
`docs/WEBHOOKS.md` and `docs/DEPLOYMENT_GUIDE.md` used to show a
`POST /api/webhooks/subscribe` curl example (at two different ports, neither
`:8000`) that has never worked. If a buyer-facing subscribe flow is wanted,
it needs a new bff route calling into the existing, working `pkg/webhook.Manager`.

## Real `bulk_order` schema, for anyone writing SQL against it by hand

Table is `bulk_order` (singular, not `bulk_orders`), `buyer_id` is opaque
`text` (no `users` table exists — see the migrations section above), and the
order-status column is `state` (a `bulk_order_state` enum, e.g.
`'ALLOCATING'`), not `status`. `docs/DEPLOYMENT_GUIDE.md` used to have a
manual fraud-detection test query that got all three wrong; fixed to match
`migrations/007_orders.sql`'s actual `CREATE TABLE bulk_order`.

## Live OpenAPI spec exists — stop hand-documenting request/response shapes

The bff embeds and serves its generated OpenAPI 3 spec at
`GET /api/v1/openapi.json` (`services/bff/internal/bff/server.go`) — it was
still marked "Coming Soon" in `docs/API.md` despite already existing. Any doc
describing exact JSON request/response shapes by hand is a maintenance trap
(see how wrong the auth flow shapes above got) — point at the live spec for
schemas and keep hand-written docs to route existence + auth requirement +
one-line purpose, verified against `server.go`'s actual route table rather
than assumed.

## Phone-change routes existed in the BFF but nowhere in the contract (fixed)

`server.go` has mounted `POST /auth/phone/change/request` and
`.../verify` (both under `authed`, JWT required) since Batch 13, but neither
path ever made it into `services/bff/openapi.json`, so `schema.d.ts` and
`operations.ts` had no types for them either. With no typed operation to
call, `apps/artisan/src/routes/profile/+page.svelte` hand-rolled a raw
`fetch('/auth/phone/change/request', …)` — missing the `/api/v1` prefix
every real route lives under, and missing the `Authorization` bearer header
`call()` normally attaches automatically. Every phone-change attempt 404'd.
Fixed by adding both paths to `openapi.json` (request/response shapes taken
from the actual handler in `services/bff/internal/bff/handler/api.go`),
regenerating `schema.d.ts`, adding `requestPhoneChangeOtp`/
`verifyPhoneChangeOtp` to `operations.ts` + the `index.ts` barrel, and
switching the call site to use them. `verifyPhoneChangeOtp`'s response
carries a fresh token pair (the endpoint revokes the caller's other
sessions), so the call site also needs `session.establish(access_token)`
alongside `setAccessToken`/`setRefreshToken` — see `completeOtpVerification`
in `web/packages/api/src/auth-flow.ts` for the canonical three-step pattern;
don't call `setAccessToken`/`setRefreshToken` alone at a new call site
without it, or `session.claims` goes stale relative to the stored token.

If a BFF route exists in `server.go` but not in `openapi.json`, nothing
catches it at build time — `route-parity.test.ts` only checks the reverse
direction (every spec path has *some* reference in `operations.ts`). A route
missing from the spec has no compile-time signal at all until someone writes
a raw `fetch()` for it by hand and gets the shape wrong.

## Cluster member role enum is intentionally asymmetric — don't "fix" it

`POST /clusters/{id}/members` accepts `role` as the short form (`"MEMBER"`,
`"COORDINATOR"`, `"MASTER"`) but every response containing a cluster member
returns the fully-qualified proto enum name (`"CLUSTER_MEMBER_ROLE_MEMBER"`,
etc.) — see `services/bff/internal/bff/client/catalog.go`:
`clusterMemberRoleFromString` parses the short form on the way in,
`clusterMemberToMap` calls `m.GetRole().String()` (untrimmed) on the way
out. `openapi.json`'s two schemas for this endpoint already reflect that
asymmetry correctly. This one enum is the only place in the codebase that
doesn't run `trimEnumPrefix` on an outbound enum (every other enum — listing
type/state, media kind, order/lot state, defect severity, pricing anomaly
level, search listing_type — is trimmed to match a short-form spec on both
sides). `apps/admin/src/routes/clusters/+page.svelte`'s dev-mode mock/
fallback data had the two forms backwards (short-form values pushed into
`members`, which is response-shaped and needs the long form) — that's a
`svelte-check` failure, not a live request bug, since the one real write
call (`addClusterMember`) already sent the correct short form. Fixed by
correcting the mock data, not by touching the request path or the spec.
## `AUTH_DEV_OTP_ENABLED` was never set in `docker-compose.yml` (fixed)

There is no real SMS provider (`service.LoggingOTPSender` is a stub that only
logs), and with dev mode off the stub doesn't even log the code — so with
this unset (defaults `false`), OTP login was **completely unusable** against
the local stack: `000000` gets rejected and the real code is nowhere a human
can see it. Every doc assumes this is on for local/demo. Fixed by adding
`AUTH_DEV_OTP_ENABLED: "true"` to core-svc's environment in
`docker-compose.yml` (main.go refuses it when `APP_ENV=production`, which is
unset here, so this is safe for this file specifically — never add it to a
real deployment's config).

## Idempotency middleware double-wrote every response (fixed)

`services/bff/internal/bff/middleware/idempotency.go`'s `responseRecorder`
both buffered the handler's response into a byte slice **and** forwarded it
live to the real `http.ResponseWriter` immediately — and then, after
capturing it, the middleware's outer code wrote the same captured response
to that same real writer *again*. Every idempotency-protected mutating route
(artisan registration, listing/order/statement/cluster/SHG creation,
moderation actions) double-wrote its response body on every fresh (non-
replayed) call. Symptom: doubled JSON in the response body
(`Content-Length` exactly 2x what a single copy would be), and — worse —
whichever write actually "won" the status line could be a *different*
status than the handler produced, e.g. an unrelated late failure clobbering
an already-successful write into a client-visible 500. Fixed by making
`responseRecorder.WriteHeader`/`Write` buffer only, never touch the real
`ResponseWriter` — the single real write now happens exactly once, from the
existing post-`idempotency.Do` code, for both a fresh execution and a true
replay alike.

## Masked 500s were completely silent everywhere (fixed) — and the real bug they were hiding

`pkg/domain.WriteHTTPError` replaces any error message on a 500 with the
generic "internal error" **before logging it anywhere** — so an unmapped
error was invisible not just to the client (intentional) but to every
service's own logs too (not intentional). Made worse by
`pkg/domain.GRPCStatus`'s `default` case *also* discarding the real message
before it even left the originating service — an internal service-to-service
boundary (e.g. core-svc → bff) has no reason to mask anything; only the
client-facing edge (`WriteHTTPError`) does. Fixed: `GRPCStatus`'s default
case now keeps `err.Error()` like every other case there, and
`WriteHTTPError` now `slog.Default().Error()`s the real error before masking
it for the client. This is how the next bug was actually found — without it,
this would have stayed a black box:

**pgx never had `language_code` registered, so no code path could ever write
`artisan.languages` (fixed).** `pkg/postgres.New()`'s `AfterConnect` only
registered pgvector's type. `artisan.languages language_code[]` is the
*only* array-of-custom-enum column in the entire schema (checked: every
other one of the 20+ custom enums is a scalar column, which pgx's text-
format fallback already handles fine without registration — only arrays of
an unregistered element type have no encode plan). Real error, once
unmasked: `failed to encode args[N]: unable to encode []db.LanguageCode
{"ENGLISH"} into text format for unknown type (OID ...): cannot find encode
plan`. This made artisan registration (and anything else touching
`languages`) fail with a masked 500 on every attempt, with zero trace in
core-svc's own logs (no panic, no logged error — just a returned error
nothing had printed yet). Fixed by registering `language_code`'s base and
array codecs via `conn.LoadType`/`TypeMap().RegisterType` in the same
`AfterConnect` hook. If a future migration adds another `sometype[]` column,
add its name to the `registerEnumArrayTypes(ctx, conn, "language_code")`
call in `pkg/postgres/postgres.go`.

## Artisan registration's real request contract (frontend was built against a stale spec)

`POST /artisans` requires `display_name`, a non-empty `craft_ids` (real
craft ontology UUIDs from `GET /crafts` — not slugs), and `region.state_code`
(ISO 3166-2:IN, e.g. `"IN-UP"`) — none of which the checked-in
`services/bff/openapi.json` documented (it only listed `display_name` as
required, `language` as an unused optional string the handler never reads).
`web/apps/artisan/src/lib/registration.ts` had a comment explaining it
deliberately sent only `display_name`/`language` because "the ABSOLUTE RULE"
(never invent fields beyond the generated spec) — a correct read of a wrong
spec. Fixed the whole chain: `services/bff/openapi.json`'s `/artisans` POST
schema now matches `services/bff/internal/bff/client/artisan.go`'s actual
`Register()` validation, `web/packages/api/src/generated/schema.d.ts` was
regenerated from it (`pnpm --filter @kalakriti/api api:gen`), and
`buildRegisterBody` in `registration.ts` now assembles the real shape
(`craft_ids`, `languages` as uppercase `commonv1.Language` enum names via
`LOCALES[code].englishName.toUpperCase()`, `region.state_code` via a new
`STATE_CODES` map in `$lib/ontology.ts`). The craft picker
(`/register/craft`) and the listing-creation craft `<Select>`
(`/listing/new/story`) both used to source a static local slug list
(`ontology.ts`'s old `CRAFTS`) that had no relation to real craft UUIDs —
replaced with `loadCrafts()`, which calls the real `GET /crafts` (public,
no auth) and caches the result via `@kalakriti/offline`'s
`getCached`/`setCached` for offline reuse. A free-text ("my district isn't
listed") registration now also needs a state picked from `STATES` (added
alongside `STATE_CODES`), since `state_code` can't be derived from arbitrary
free text. If you touch `POST /artisans` again: check
`client/artisan.go`'s `Register()` for the real required fields, not the
checked-in `openapi.json` — the two have drifted before and will again
unless someone runs `pnpm api:gen` after every bff route change.

## Idempotency header mismatch — now fixed (was: found, not yet fixed)

`services/bff/internal/bff/middleware/idempotency.go` reads the bare
`Idempotency-Key` header — that has not changed. `web/packages/api/src/transport.ts`
sends **both** `Idempotency-Key` and `X-Idempotency-Key` on every mutating
request, so the middleware always sees the one it reads regardless of which
name a future refactor favors. The CORS allow-list
(`pkg/httpx/middleware.go`) was missing `X-Idempotency-Key` — same-origin
traffic never preflights so this was latent, but any cross-origin deployment
would fail preflight on every mutating request. Fixed by adding it alongside
the existing bare-name entry. (The two earlier versions of this entry
contradicted each other on which header name is canonical — this replaces
both; canonical is the bare `Idempotency-Key`.)

## Internationalization (i18n) verification and ratchet convention

Kalakriti supports 20 Eighth Schedule scheduled Indian languages plus English (21 locales total).
The source of truth for message keys is `web/packages/i18n/src/messages/en.ts` (currently 3,727 keys).

### Audit Command
Run the audit script to verify catalogue completeness, script correctness, and placeholder consistency:
```sh
# Inside web/packages/i18n:
npm run audit
# Or audit a specific locale:
node scripts/audit.mjs --locale <code_or_tag>
```

### Ratchet Ceiling & Baseline Convention
- `web/packages/i18n/i18n-baseline.json` defines the ratchet ceiling for allowed issues per locale.
- **As of 2026-09-26, every non-English locale audits at 0 issues / 100% catalogue coverage** across
  3,727 keys. Most recently added UI copy was machine-translated; low-resource languages still need
  native-speaker review before the translations are treated as publish-quality. Always rerun
  `npm run audit` and trust the live `i18n-baseline.json` for current status.
- CI and vitest (`catalogue-audit.test.ts`) enforce that a locale's issues must never exceed its baseline count.
- If you add new keys to `en.ts`, every locale is "missing" them until its own translation batch
  lands — raise that locale's baseline number by the new key count in the same commit that adds the
  keys (with an inline `_comment` note saying why, dated), then lower it back down as real
  translations land. Never raise a number for any other reason, and never round when lowering one.
- All non-English catalogues are typed as `export const <code>: Messages = { ... }`, making missing keys a compile-time type error permanently.
- brx (Bodo), ks (Kashmiri) and sd (Sindhi) got real, machine-assisted translations for the 254
  MoSJE tier-4 keys that needed them (2026-09-24) — these three should still get a native-speaker
  review pass before being treated as publish-quality, but they're no longer English fallback.

## Outbound webhook subscribe route added (pkg/webhook.Manager was unused)

`pkg/webhook.Manager` (CRUD) and the delivery `Worker`/`cmd/webhook-worker`
existed, but nothing in `services/bff/internal/bff/server.go` ever mounted a
route for subscription CRUD (only the *inbound* payment webhook existed).
Fixed by adding `POST/GET /webhooks/subscriptions` and
`DELETE /webhooks/subscriptions/:id` (all under `authed`) to
`handler/api.go` + `server.go`, wired to a `*webhook.Manager` built in
`cmd/bff/main.go` from its own `database/sql`/`lib/pq` connection (pkg/webhook
predates pgxpool and wasn't worth rewriting just for this). Subscriber id
comes from the JWT principal's `Subject`, parsed as a UUID — true for artisan
ids, **not** guaranteed for buyer ids (opaque `text` elsewhere in this
schema); a non-UUID buyer subject gets a 400, not a silent wrong write.
`DeleteSubscription` now takes `subscriberID` too and filters on it, so one
caller can no longer delete another's subscription by guessing a UUID.

## No public path ever issues a BUYER, CLUSTER_OFFICER or MINISTRY token

Found while writing `cmd/seed-demo` (a seeder that creates real
artisans/listings/orders through the live BFF API instead of writing rows
directly). `VerifyOtp` in `services/core-svc/internal/core/service/auth.go`
hardcodes every OTP login to `auth.RoleArtisan` — there is no OTP flow, REST
route, or self-service path anywhere that mints a `RoleBuyer` token, and
`SubmitForApproval` (the listing-moderation step) requires
`RoleClusterOfficer`/`RoleMinistry`, which are equally unreachable. Both
roles appear only in test helpers (`bfftest/server.go`,
`artisan_test.go`) — never in a real request path. Concretely: **nothing in
the current product can create a bulk order as a real buyer, or move a
listing from draft to published, without someone minting a JWT by hand.**
`cmd/seed-demo/main.go` does exactly that with `pkg/auth.Issuer` directly
(same `JWT_SECRET` the bff verifies against) to get demo data in, and says so
in its own doc comment — this is a workaround for a real product gap, not a
fix. Before real buyers or moderators use this in production, something
needs to actually issue those roles: a buyer signup/login flow, and an
admin/ops path for granting cluster-officer or ministry accounts.

## Production secrets: docker-compose.yml now reads `.env`, no second compose file

Every hardcoded dev secret in `docker-compose.yml`
(`POSTGRES_PASSWORD`, `JWT_SECRET`, `S3_ACCESS_KEY`/`S3_SECRET_KEY`,
`MINIO_ROOT_USER`/`PASSWORD`, Kafka `CLUSTER_ID`) is now `${VAR:-dev-default}`
— same pattern the ml-svc block already used for `HF_TOKEN`. A real deploy
generates a repo-root `.env` (`docker compose` reads it automatically) with
`scripts/gen-prod-secrets.sh`, which refuses to run if `.env` already exists
and does not touch `BASE_URL`/`CORS_ALLOWED_ORIGINS` (deploy-specific, edit by
hand). Deliberately did **not** add a `docker-compose.prod.yml` — see the
`docker-compose.full.yml` postmortem above; a second compose file drifts the
moment someone edits only one of them. `bff`'s `CORS_ALLOWED_ORIGINS` env var
also got wired into compose (was previously unset/undocumented there) since a
frontend deployed separately on Vercel calls this bff cross-origin.

## Backend CI added (`.github/workflows/ci.yml` only had a `web` job before)

Added `go` (spins up a `pgvector/pgvector:pg17` service container, installs
`protoc-gen-go`/`protoc-gen-go-grpc`, runs `make proto-go sqlc migrate-up
lint test`) and `ml-svc` (`uv sync --extra dev && pytest`) jobs alongside the
existing `web` job. `buf`/`goose`/`sqlc` need no separate install step — the
Makefile already falls back to `go run .../tool@pinned-version` when the
binary isn't on `PATH`.

## gRPC service-to-service calls had no load-balancing policy (fixed)

Every gRPC client in the repo dialed a bare `host:port` via
`grpc.NewClient(addr, grpc.WithTransportCredentials(...))` with no LB policy
— fine at one replica per service (docker-compose's reality today), but
silently broken the moment any backend scales to multiple pods in k8s: gRPC's
default resolver scheme is `passthrough`, which treats the address as one
opaque target and never re-resolves it, so the connection pins to whichever
single pod it first reached — `round_robin` has nothing to balance across
without also fixing the resolver scheme, and a plain `ClusterIP` Service
doesn't help either, since it hands back only its own virtual IP, not one
address per pod.

Fixed with **pkg/grpcdial** (`pkg/grpcdial/grpcdial.go`): `grpcdial.Dial(addr)`
prefixes the target with `dns:///` (triggers real re-resolution to every `A`
record behind a name) and sets `round_robin` via
`grpc.WithDefaultServiceConfig`. Every dial site now uses it: `services/bff/
cmd/bff/main.go` (→ core-svc, search-svc, insight-svc, collab-svc,
channel-svc), `services/channel-svc/cmd/channel-svc/main.go` (→ core-svc),
`services/core-svc/cmd/core-svc/main.go` (→ ml-svc), `services/search-svc/
cmd/search-svc/main.go` (→ ml-svc, → core-svc). A unit test
(`pkg/grpcdial/grpcdial_test.go`, using grpc-go's manual resolver against 3
fake backends) proves the round_robin service config actually spreads calls
across resolved addresses rather than pinning to one.

`dns:///` alone is not sufficient — it also requires a **headless** k8s
Service (`clusterIP: None`) on the callee, since CoreDNS only returns one A
record per pod for a headless Service; a normal ClusterIP Service still
resolves to just its own virtual IP regardless of the dial-side scheme. Every
Service manifest under `deploy/k8s/` for a service reached over gRPC
(core-svc, collab-svc, channel-svc, insight-svc, ml-svc, search-svc) is now
headless. `bff`'s Service stays plain ClusterIP — it's reached over HTTP via
Ingress, not dialed as gRPC by anything.

Explicitly out of scope, and don't revisit without a real reason: no service
mesh (Istio/Linkerd) — `round_robin` + headless Service solves the actual
problem with zero new infrastructure. No Kafka request/reply conversion for
ml-svc's synchronous, same-request-cycle calls (`Embed`/`Rerank`/`Transcribe`
in search-svc, live at query time) — that would need correlation IDs, a reply
topic, and a blocking wait-with-timeout in an HTTP handler, objectively more
complex than fixing the LB policy. ml-svc's *other* calls
(`EnhanceImage`/`ExtractAttributes`/`GenerateDescription`/`Translate`, all
reachable only from `services/core-svc/internal/core/service/pipeline.go`
inside the `MediaUploadedHandler` Kafka consumer) were already async before
this change and needed no architecture change, only the same `dns:///` +
round_robin fix on their own outbound gRPC hop.

## k8s manifests for core-svc/collab-svc/channel-svc/insight-svc added; `user-svc-deployment.yaml` deleted

`deploy/k8s/` previously had manifests only for `bff`, `ml-svc`, `search-svc`
and a stray `web`, plus a `user-svc-deployment.yaml` that doesn't correspond
to any real service under `services/` (real services are core-svc, search-svc,
collab-svc, channel-svc, insight-svc, bff, ml-svc, web) — its ports (8080/9090),
ownership of `JWT_SECRET`, and general shape strongly suggest it's a stale
pre-rename draft of what's now `core-svc`. Deleted it rather than fixing it,
same reasoning as the `docker-compose.full.yml` postmortem above: keeping two
manifests both claiming to be "the identity service" under different names is
exactly the kind of duplicate that drifts and confuses later, not a safety
net. Added real manifests for the four services that were missing entirely
(`core-svc-deployment.yaml`, `collab-svc-deployment.yaml`,
`channel-svc-deployment.yaml`, `insight-svc-deployment.yaml`), with ports,
env var names, and health-check paths taken from each service's actual
`main.go` and `docker-compose.yml` — not assumed. Notable per-service specifics
future edits should preserve:
- **insight-svc** has exactly one listener, gRPC-only, no HTTP port at all
  (confirms the already-fixed fictional `INSIGHT_SVC_HTTP_PORT` from the
  `.env.example` section above stays gone) — its k8s probes are a bare
  `tcpSocket` check, not an invented HTTP path.
- **channel-svc** exposes only `/health`, not the `/healthz`+`/readyz` pair
  core-svc/collab-svc/search-svc all have — don't copy the two-path pattern
  onto it.
- **collab-svc**'s own env var names are `COLLAB_GRPC_ADDR`/`COLLAB_HTTP_ADDR`
  (not `COLLAB_SVC_GRPC_ADDR`/`COLLAB_SVC_HTTP_ADDR`, unlike every other
  service's naming convention) — docker-compose.yml never overrides them
  either, relying on the `:50053`/`:8083` code defaults instead; the new k8s
  manifest does the same rather than setting a var under the wrong name.

`deploy/k8s/configmap.yaml` also had six `*-addr` keys
(`user-svc-addr`, `catalog-svc-addr`, `search-svc-addr`, `order-svc-addr`,
`social-svc-addr`, `ml-svc-addr`) that no manifest ever actually read via
`configMapKeyRef` — dead configuration, three of them (`catalog-svc`,
`order-svc`, `social-svc`) for services that don't exist anywhere in
`services/`. Deleted rather than fixed, since nothing consumes them; every
real service gets its peer addresses from literal env values in its own
Deployment instead (see `bff-deployment.yaml`'s `CORE_SVC_ADDR` etc.).

`deploy/k8s/bff-deployment.yaml`, `search-svc-hpa.yaml` (now
`search-svc-deployment.yaml` in spirit, filename unchanged) and the deleted
`user-svc-deployment.yaml` all had the same `DATABASE_URL` mistake
`.env.example` had (see that section above) — `pkg/config` reads
`POSTGRES_DSN`. Fixed in both surviving files. `search-svc-hpa.yaml` also had
its gRPC/HTTP ports backwards (8082 labeled `"grpc"`, real gRPC port is
50052 per `SEARCH_SVC_GRPC_ADDR`'s default) and probed a port 9092 the
service never listens on (`/healthz`/`/readyz` are served on the HTTP port,
8082) — fixed to match `services/search-svc/cmd/search-svc/main.go`'s actual
`envOr` defaults.

## MoSJE tier 4 (finance linkage, impact dashboard, assisted mode, literacy)

A large batch (commits `cf6dd71`…`bf85c7f`) added F12–F15: finance-corporation
loan linkage, self-reported income vs. platform income, a ministry impact
dashboard, field-agent "assisted mode" (an agent acting for an artisan who
can't use the app themselves), and an 8-lesson digital-literacy track with a
verifiable certificate. A few things worth knowing before touching any of it:

- **The `mosje` package pattern.** All of it lives behind one bff subpackage,
  `services/bff/internal/bff/mosje/` (`routes.go` mounts every tier-4 REST
  route, `mosje.go` holds the handler struct, `onbehalf.go` is the assisted-
  mode allow-list — see below). It's kept separate from `handler/api.go`
  rather than merged in, so the whole feature area's routes, auth notes and
  allow-list sit in one place instead of interleaved with pre-existing
  routes. If you add a new tier-4 route, it goes in this package, not
  `api.go`.
- **The on-behalf allow-list is duplicated in two places and both must agree.**
  `mosje/onbehalf.go`'s `onBehalfAllowed` map is the real enforcement (the
  bff refuses any `X-On-Behalf-Of` call not on the list); `web/packages/api/
  src/acting.ts`'s `onBehalfAllowed` is the frontend's mirror, used only to
  decide whether to attach the header at all / show assisted-mode UI for a
  given call. Nothing checks the two stay in sync — if you add a route an
  agent should be able to call for an artisan, add it to both, or the
  frontend will either try a call the backend 403s, or silently not offer an
  action the backend would actually allow.
- **`services/bff/openapi.json` doesn't auto-track proto/route changes for
  this package.** Re-run `python3 scripts/gen_openapi_tier4.py` (see its own
  docstring) after changing a tier-4 proto or adding/changing a route in
  `mosje/routes.go`, then `pnpm --filter @kalakriti/api api:gen` to
  regenerate `schema.d.ts`. Same class of gap as the phone-change routes
  section above — nothing catches a route missing from the spec until
  someone hand-rolls a `fetch()` for it and gets the shape wrong.
- **`FINANCE_REF_SALT`** keys the HMAC over loan/beneficiary reference numbers
  (`services/core-svc/cmd/core-svc/main.go`, passed into
  `service.NewFinance`). Only `reference_last4` and `reference_hash` (the
  HMAC) are ever stored — the plaintext reference an artisan or agent submits
  to `POST /finance/links` is hashed immediately server-side and never
  logged, stored or returned. Must be ≥32 bytes in production (same shape as
  `JWT_SECRET`); `docker-compose.yml` defaults it to empty (`main.go` falls
  back to a dev value outside `APP_ENV=production`).
- **insight-svc gained a gRPC auth interceptor it didn't have before.** Before
  this batch, insight-svc ran `grpc.NewServer()` with no auth chain at all —
  same implicit-trust posture search-svc and collab-svc still have (see the
  "core-svc was rejecting every anonymous public-read RPC" section above).
  Adding the public `GET /verify/certificate/:short_code` route required a
  real interceptor so every *other* insight-svc RPC stays behind a token:
  `services/insight-svc/cmd/insight-svc/main.go` now builds an
  `auth.NewPublicMethods(...)` allow-list (currently just
  `insight.v1.InsightService/VerifyLiteracyCertificate`) and wires
  `auth.UnaryServerInterceptor`/`StreamServerInterceptor` with it. If you add
  another public insight-svc RPC, it needs adding to that list the same way
  core-svc's `PublicMethods()` works.
- **`BASE_URL` on insight-svc must be the public web origin, not the bff's own
  port.** It's printed into QR codes for income-statement `/v/` links and the
  new literacy-certificate `/verify/certificate/` links, both of which are
  pages nginx (the `web` service) serves — the bff alone doesn't serve either.
  `docker-compose.yml` sets insight-svc's `BASE_URL` to
  `${BASE_URL:-http://localhost}` (nginx origin), while bff's own `BASE_URL`
  (a different env var scope, same name) stays `http://localhost:8000`. Don't
  "fix" these to match each other — they're deliberately different origins
  for different purposes; the inline comment in `docker-compose.yml` explains
  it at the point of use.
- **Migration ordering:** `037_finance_link.sql`, `038_impact.sql`,
  `039_assisted.sql`, `040_literacy.sql`, `041_digital_ready_badge.sql` are
  correctly numbered and goose orders strictly by filename, so a fresh
  `make migrate-up` applies them in order with no issue. The one situation to
  watch for: `039_assisted.sql` was committed to git *before* `037`/`038` in
  an earlier pass of this batch (since fixed) — a database that somehow
  already applied a version-039-shaped migration under a different number
  during that window could refuse to re-apply 037/038 cleanly. Not a risk on
  a fresh clone; only relevant if you're migrating a long-lived dev database
  that tracked this repo through that window.
- **`cmd/seed-demo` now seeds MoSJE data too** (artisans with income
  baselines, offline sales and finance links across two impact-dashboard
  cohorts, so `/impact` doesn't just read "<5" everywhere) — see its own
  doc comment. Needs `POSTGRES_DSN` set (in addition to the existing
  `BFF_BASE_URL`/`JWT_SECRET`) to backdate registration past the 90-day
  `TOO_NEW` threshold and to bootstrap MINISTRY/FIELD_AGENT staff accounts
  through `services/core-svc/cmd/create-staff`; without it, the MoSJE
  artisans/sales/links still get created through the API, just with
  `registered_at` too recent for uplift to compute and no staff accounts to
  view them with.

## `make demo-up`/`make up` never rebuilt images — stale binaries masqueraded as live bugs

The single highest-value thing in this file: `demo-up` ran `$(COMPOSE) up -d`
with **no `--build`**, so Docker reused whatever service images happened to
exist locally. Images built before a fix keep running the old code forever,
and nothing says so — `docker compose ps` happily reports every container
"Up". Seen for real: images dated 17:16 were still serving while the fixes
for them had landed in commits at 17:42 and 23:16 the *same day*.

Symptom, which looks exactly like three separate live bugs: `make demo-up`
dies at `seed-demo` with
`POST /api/v1/artisans -> 500: {"error":"internal_error","message":"internal error"}`,
core-svc logs nothing at all, and the bff's access log claims `"status":200`
for the very request the client received as a 500. All three are the same
stale image — it predated the pgx `language_code[]` registration fix, the
idempotency double-write fix, *and* the masked-500 logging fix, so the error
was real, unlogged, and doubled on the wire all at once.

Tells that you are chasing a stale image rather than a code bug:
- `docker images --format '{{.Repository}} {{.CreatedAt}}' | grep kalakriti`
  shows a timestamp older than `git log -1 --format=%ad -- <the file you are
  reading>`. Check this **first**, before reading any handler.
- A masked 500 with no `unmapped error rendered as 500` line anywhere in the
  service's logs. `pkg/domain.WriteHTTPError` always logs that before masking,
  so its absence means the running binary does not contain that code.
- `Content-Length` exactly 2x a single JSON body (the pre-fix idempotency
  middleware forwarded live *and* replayed), or every access-log line saying
  `"status":200` regardless of the real status (pre-`AccessLogGin`).

Fixed by putting `--build` on `demo-up`'s compose invocation. It is a cache
no-op when nothing changed, so it costs nothing to keep and removes an entire
category of phantom debugging. `make up` deliberately still has no `--build`:
it only starts infra containers (postgres/redis/kafka/minio/jaeger/prometheus),
none of which are built from this repo. If you ever run `docker compose up -d`
by hand instead of through the Makefile, pass `--build` yourself.

## `cmd/seed-demo` could not succeed even against a correct stack

Three independent bugs, all of which had to be fixed before `make demo-up`
completed once:

- **It burned two OTP logins per artisan.** After `POST /artisans` it threw
  away the artisan-scoped token pair that endpoint returns and ran a *second*
  full OTP login for the same phone, on the stale belief that only a re-verify
  could produce a token carrying the new artisan id. `RegisterArtisan` has
  returned exactly that pair for this reason all along (see its handler, and
  `auth-flow.ts`'s `completeOtpVerification`). At 2 OTP requests per artisan
  against `/auth/otp/request`'s limiter, the default 3 artisans needed 6
  requests against a 5-per-10-minutes-per-IP budget — so `make demo-up` failed
  on the third artisan **every time, from a completely clean stack**. Now one
  login per artisan.
- **It read the wrong response keys.** `listingResp["id"]` and
  `orderResp["id"]`; the bff returns `listing_id` and `order_id`
  (`httpx.JSON(w, 201, map[string]string{"listing_id": ...})` in
  `handler/api.go`). The listing one was a hard `panic: interface conversion:
  interface {} is nil, not string`. Both now read the real keys and fail with
  a message naming the actual response instead of panicking.
- **The OTP limit made it un-re-runnable** even after the above. Fixed by
  making the limit configurable — `bff.Config.OTPRateLimit`/
  `OTPRateLimitWindow`, read from `OTP_RATE_LIMIT`/`OTP_RATE_LIMIT_WINDOW` in
  `cmd/bff/main.go`, defaulting to the previous hardcoded 5/10m when unset —
  and raising it to 100 for the local stack in `docker-compose.yml` only. The
  limit caps real SMS spend and phone enumeration; this stack's OTP sender is
  the logging stub (`AUTH_DEV_OTP_ENABLED=true`), so neither risk exists here.
  **A real deploy leaves both vars unset and gets 5/10m back** — do not put
  these in a production env file.

Verified end to end: two consecutive `make demo-up` runs, back to back with no
waiting, both reaching `done: 3 artisans, 3 published listings, 3 bulk orders`
and `✓ Demo environment ready!`, with `GET /api/v1/listings` returning real
PUBLISHED listings and zero 4xx/5xx in any service log. This also closes the
"no seed data for artisans/listings/orders" gap noted in the `make demo-up`
section above — `/api/v1/listings` no longer returns `{"listings":[]}`.

## Vite dev origins must be in `CORS_ALLOWED_ORIGINS` too, or login 403s

The NGINX origins (`http://localhost`, `:8081`, `:8082`) were already in
`docker-compose.yml`'s `CORS_ALLOWED_ORIGINS` default after that bug was found
once — but the `pnpm dev:*` origins were not, so the same failure came back the
moment anyone developed against the Vite dev server instead of the NGINX build.

Symptom: the artisan login screen at `http://localhost:5173/login` shows
"Something went wrong" and devtools shows
`POST /api/v1/auth/otp/request 403 (Forbidden)`. The bff logs
`csrf validation failure: untrusted origin origin=http://localhost:5173`.

Why it happens even though Vite's `/api` proxy is server-side: the browser hop
(browser → Vite) really is same-origin, but the proxy **forwards the browser's
original `Origin: http://localhost:5173` header** to the bff. `pkg/httpx`'s
`CSRFProtection` trusts only `cfg.AllowedOrigins` plus `SelfOrigin`
(`BASE_URL`'s origin, `:8000`), so it sees an untrusted origin and 403s. Note
`CSRFProtection` skips GET/HEAD/OPTIONS, so only mutating requests fail —
which is why the page itself and `GET /crafts` load fine and only the OTP POST
breaks, making it look like an auth bug rather than a CSRF one.

Fixed by adding `http://localhost:5173`/`:5174`/`:5175` (dev) and
`:4173`/`:4174`/`:4175` (preview) to the compose default. Those are the
`server.port`/`preview.port` values in
`web/apps/{artisan,buyer,admin}/vite.config.ts` — **keep the two in sync** if a
vite port ever changes. Setting `CORS_ALLOWED_ORIGINS` explicitly (as a real
deploy does) replaces the whole default list, so no localhost origin reaches
production.

Related: `AUTH_DEV_OTP_ENABLED=true` makes `000000` valid for **any** phone
number, so there is no "default demo phone" to special-case — any number logs
in with `000000` on the local stack. A number that has no artisan profile yet
simply lands in the registration flow after verifying.

## `demo-up` raced its own services on a fresh volume (fixed)

`demo-up`'s only wait was a `sleep 15` placed **before** `migrate-up`, which
guarded nothing: on a fresh volume (i.e. every `make demo-reset`, or a first
run) every service that opens a Postgres pool crash-loops from the moment
`up -d` starts them, because the schema does not exist yet —
`connecting to postgres: pinging postgres: registering pgvector types: vector
type not found in the database`. Docker's restart backoff grows to tens of
seconds, so by the time migrations finish the bff is still mid-backoff and
`seed-demo` dies with
`GET /crafts ...: dial tcp 127.0.0.1:8000: connect: connection refused`.
The sleep could never help: it elapses before the thing the services are
crash-looping on is fixed.

Fixed by adding, after `migrate-up`/`seed` and before `seed-demo`, an explicit
`$(COMPOSE) restart` of the six Go services (resets the backoff immediately
rather than waiting it out) followed by a poll of `$(BFF_BASE_URL)/healthz`
until it answers, with a 60s cap that fails loudly instead of falling through
into a seeder that cannot connect. If you add another step to `demo-up` that
talks to a service over the network, put it after that readiness gate.

Two unrelated things that look alarming in the logs right after a full Docker
Desktop restart and are **not** bugs: services log the `vector type not found`
error while Postgres is still coming up, and core-svc's outbox relay logs
`Unknown Topic Or Partition ... order.bulk.requested` if it publishes before
Kafka finishes creating topics. Both self-resolve on retry within a minute —
confirm by checking the timestamps are from the restart window and that
`kafka-topics --list` now shows the topic, rather than chasing either one.
