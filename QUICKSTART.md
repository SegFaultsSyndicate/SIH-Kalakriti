# Kalakriti — Quickstart Guide

## What This Is

7 backend microservices (6 Go + 1 Python `ml-svc`) behind one REST gateway
(`bff`), plus 3 SvelteKit web frontends served by NGINX in front of it.

```
Browser → NGINX :80 → static buyer/artisan/admin builds
                    → /api/*, /listing/*, /v/*  proxied to  bff :8000
                                                                ↓ gRPC
                                    core-svc / search-svc / collab-svc /
                                    channel-svc / insight-svc / ml-svc
```

- Web apps at `http://localhost` (NGINX on port 80) — buyer at `/`, artisan
  PWA at `/artisan/`, admin at `/admin/`.
- REST API at `http://localhost:8000/api/v1` (bff), same routes also proxied
  through NGINX at `http://localhost/api/v1/*`.
- gRPC between backend services; the bff is the only HTTP surface.
- Single Postgres (pgvector) + Redis + Kafka (KRaft) + MinIO.
- OpenTelemetry tracing (Jaeger) + Prometheus metrics.

There is **one** docker-compose file, `docker-compose.yml` — it already
includes the NGINX web tier. There is no separate "production" compose file.

**API reference:** the bff self-serves its generated OpenAPI spec at
`GET /api/v1/openapi.json` — that and `docs/API.md` are the source of truth
for routes; this doc is just how to get the stack running.

---

## Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Go | 1.23+ | workspace mode (`go.work`) |
| Docker + Compose v2 | 24+ | `docker compose`, not `docker-compose` |
| Python | 3.11 | `ml-svc` only |
| uv | latest | `ml-svc`'s proto codegen |
| make | GNU 4+ | |

`buf`, `goose`, `sqlc`, `golangci-lint` are optional — the Makefile falls
back to a pinned `go run` if the binary isn't on `PATH`.

You do need `protoc-gen-go` and `protoc-gen-go-grpc` as local binaries,
because `make up`/`make demo-up` both run `make proto sqlc` first:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Then make sure `$(go env GOPATH)/bin` (usually `~/go/bin`) is on `PATH` —
**this is the single most common cause of a fresh-clone failure.** Symptom:
`buf generate` fails with `exec: "protoc-gen-go": executable file not found
in $PATH`, even though the plugin is installed, because the shell that ran
`make` doesn't have `~/go/bin` on `PATH`.

---

## Fastest path: full stack, one command

```sh
cp .env.example .env      # not auto-loaded by the services; see the file's own comments
make demo-up               # proto/sqlc codegen, infra, all services, migrate, seed craft ontology
```

First run takes a few minutes (Docker image builds); subsequent runs are
much faster (image cache). This starts **everything**, including the web
tier — check status with `docker compose ps`.

```
http://localhost/            buyer marketplace
http://localhost:8081        artisan PWA
http://localhost:8082        admin/ministry dashboard
http://localhost/api/v1/*    REST API (proxied to bff)
http://localhost:8000        REST API (direct to bff)
http://localhost:16686       Jaeger tracing UI
http://localhost:9090        Prometheus
http://localhost:9001        MinIO console (minioadmin/minioadmin)
```

**What `make demo-up` seeds:** the craft ontology only (`scripts/data/*.csv`
— ~14 crafts, 50 aliases), via `make seed`. There is currently no seed data
for artisans, listings, or orders against the real schema — that gap is
tracked in `web/FRONTEND.md`. `GET /api/v1/listings` returning
`{"listings":[]}` right after `demo-up` is expected, not a bug; craft
browsing (`GET /api/v1/crafts`) has real data.

**Reset:** `make demo-reset` (down + delete volumes + `demo-up` again).

---

## Step-by-step path: infra first, then services by hand

Useful when working on one Go service and iterating with `go run` instead of
rebuilding a container each time.

### 1. Infra only

```sh
make up          # proto/sqlc codegen, then infra containers, waits for healthy
make check       # probes each dependency from the host
```

Starts Postgres 18+pgvector (`:5432`), Redis (`:6379`), Kafka KRaft
(`:9092`), MinIO (`:9000`, console `:9001`), Jaeger, Prometheus — no
application services yet.

### 2. Migrate + seed

```sh
make migrate-up   # goose, all pending migrations
make seed         # craft ontology CSVs (idempotent)
```

### 3. Run one service directly

```sh
cd services/core-svc && go run ./cmd/core-svc
```

Requires `make proto sqlc` to have run at least once (generates
`pkg/pb/` and `services/core-svc/internal/core/repo/db/`, neither committed
to git). core-svc serves gRPC on `:50051` and `/healthz`+`/readyz` HTTP on
`:8081`. It reads config from real process env vars (not `.env` directly —
`export` them or use your shell's dotenv support) — see `.env.example` for
the exact names `pkg/config` expects; several of the *_ADDR/*_PORT names
there only matter if another service is dialing this one, not for its own
bind address (each service's own `cmd/*/main.go` has its own flag names,
usually `<SVC>_GRPC_ADDR`/`<SVC>_HTTP_ADDR`).

With `AUTH_DEV_OTP_ENABLED=true` the login handshake accepts `000000` as the
OTP with no real SMS provider; startup refuses this flag when
`APP_ENV=production`.

```sh
grpcurl -plaintext -d '{"phone_e164":"+919876543210"}' \
  localhost:50051 identity.v1.IdentityService/RequestOtp
```

(Note: this raw gRPC contract takes `phone_e164`/`challenge_id` — the bff's
REST layer simplifies this to a plain `phone`/`otp` pair with no challenge
ID; see `docs/API.md`.)

---

## Testing the API once something is running

```sh
curl http://localhost:8000/healthz

curl -X POST http://localhost:8000/api/v1/auth/otp/request \
  -H 'Content-Type: application/json' -d '{"phone":"+919876543210"}'

curl -X POST http://localhost:8000/api/v1/auth/otp/verify \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+919876543210","otp":"000000"}'
# {"access_token":"...","refresh_token":"..."}

export TOKEN="<access_token>"

# public reads need no token
curl http://localhost:8000/api/v1/crafts
curl "http://localhost:8000/api/v1/listings?page_size=10"

curl "http://localhost:8000/api/v1/search?q=madhubani" \
  -H "Authorization: Bearer $TOKEN"
```

Full route list, auth requirements, error format, idempotency and rate
limiting: `docs/API.md`.

### Automated integration tests

```sh
cd tests/integration && go test -v ./...
```

Four journeys: artisan registration → discovery, bulk order allocation +
reallocation, provenance seal + verify, income statement generation. Needs a
running stack (`make demo-up` or the step-by-step path above) — these hit
real containers, not mocks.

---

## Frontend development

Two ways to run the three SvelteKit apps (`web/apps/{buyer,artisan,admin}`)
— full detail in `web/FRONTEND.md`.

**Hot reload, against a running backend:**
```sh
cd web && pnpm install
pnpm dev:artisan     # :5173
pnpm dev:buyer       # :5174
pnpm dev:admin       # :5175
```
Each app's Vite dev server proxies `/api` to `http://localhost:8000` (the
bff) — so the backend (`make demo-up`, or at least `bff` + its dependencies)
needs to be running first. Every screen still boots without it (offline-first
design), but nothing beyond local/cached state resolves.

To see the offline/airplane-mode demo mock fallbacks instead of real errors
when a call fails, opt in with `VITE_USE_MOCKS=1 pnpm dev:admin` (default off
— see `web/README.md`).

**Full container stack** (NGINX serving built apps, no hot reload):
already covered by `make demo-up` above — apps are at `http://localhost`,
`/artisan/`, `/admin/`.

---

## Troubleshooting

**`exec: "protoc-gen-go": executable file not found in $PATH`** — see
Prerequisites above; `~/go/bin` isn't on `PATH` in the shell running `make`.

**Kafka container exits immediately, log says `Cluster ID ... does not
appear to be a valid UUID`** — `CLUSTER_ID` in whichever compose file you're
using must be a base64url-encoded 16-byte UUID, not an arbitrary string.
Regenerate with:
```sh
python3 -c "import uuid, base64; print(base64.urlsafe_b64encode(uuid.uuid4().bytes).rstrip(b'=').decode())"
```

**A service crash-loops with `registering pgvector types: vector type not
found`** — migrations haven't run. `make up` only starts infra; run
`make migrate-up` (or use `make demo-up`, which does this for you).

**`no space left on device` mid-build** — Docker's build cache/data root
lives on `/`; see the "Disk space" section of `CLAUDE.md` for the safe
cleanup order.

**Port already in use (5432/6379/9092/9000/8000/80)** — something else is
bound to it; stop that process or edit the port mapping in
`docker-compose.yml`.

**Frontend dev server: `http proxy error ... ECONNREFUSED` on every
`/api/*` call** — the bff isn't reachable at the port the app's
`vite.config.ts` proxies to (`server.proxy['/api'].target`, should be
`http://localhost:8000`). Vite doesn't hot-reload its own config file —
restart `pnpm dev:*` after checking/fixing it.

**`docker compose ps` shows a service unhealthy/restarting** —
`docker compose logs -f <service>`. Common causes: `JWT_SECRET` under 32
bytes (crashes bff/core-svc at startup), a `POSTGRES_DSN`/`S3_*`/`KAFKA_*`
env var typo (see `.env.example`'s comments for the exact names
`pkg/config` requires), or the `minio-init` one-shot bucket-creation step
not having completed yet.

More accumulated gotchas (docker-compose specifics, migration history, the
Go-workspace-vs-Docker-build `go.sum` trap, sqlc's nullable-param pattern):
`CLAUDE.md` at the repo root.
