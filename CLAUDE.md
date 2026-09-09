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
