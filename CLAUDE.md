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

## Idempotency header mismatch between frontend and bff (found, not yet fixed)

`services/bff/internal/bff/middleware/idempotency.go` reads
**`X-Idempotency-Key`**. `web/packages/api/src/transport.ts` sends
**`Idempotency-Key`** (no `X-` prefix). Every mutating BFF route wrapped in
`withIdempotency` (artisan/listing/order/statement/cluster/SHG creation,
moderation actions) therefore never actually sees the frontend's key — a
retried write from the offline outbox is indistinguishable from a new one
server-side. This is live and unfixed as of this writing; pick a side
(add `X-` client-side, or accept the bare header name server-side) before
relying on retry-safety anywhere the outbox drains a queued write twice.

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

`web/packages/api/src/transport.ts` now sends `X-Idempotency-Key`, matching
`services/bff/internal/bff/middleware/idempotency.go`. (Previously documented
here as sending the bare `Idempotency-Key` — that entry is superseded by this
one; the fix has shipped.)
