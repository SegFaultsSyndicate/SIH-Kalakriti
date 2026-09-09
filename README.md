# Kalakriti

AI cataloging, provenance and collective-fulfilment platform for Indian artisans.
Event-driven Go microservices, one Python ML service, single Postgres with pgvector.

## Layout

```
web/                   three SvelteKit apps (buyer, artisan PWA, admin) + design system
Dockerfile.web         multi-stage Docker build for all 3 apps + Alpine NGINX server
proto/                 protobuf contracts (source of truth for service APIs)
pkg/                   shared Go packages + generated pb code (pkg/pb)
services/core-svc/     identity, artisan profiles, catalog, provenance
services/search-svc/   pgvector similarity search and discovery
services/collab-svc/   collective fulfilment, pooled orders, allocation
services/channel-svc/  outbound channel sync and notifications
services/bff/          the only REST/JSON surface
services/ml-svc/       Python 3.11 gRPC server wrapping the models
migrations/            goose migrations (single database, schema per service)
deploy/
  nginx/               production NGINX reverse proxy & SPA fallback configuration
  k8s/                 Kubernetes manifests (deployments, ingress, configmap)
scripts/               dev helpers, Postgres init SQL, ontology seed CSVs
```

Go code is one workspace (`go.work`); each service is its own module so it can be
built and deployed independently.

## Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Go | 1.23+ | workspace mode |
| Docker + Compose v2 | 24+ | `docker compose`, not `docker-compose` |
| Python | 3.11 | `ml-svc` only |
| uv | latest | runs `ml-svc`'s proto codegen; `make up` needs it on `PATH` |
| make | GNU 4+ | |
| psql | 18 | optional, for manual poking |

`buf`, `goose`, `sqlc` and `golangci-lint` are optional: the Makefile falls back to
a pinned `go run` if the binary is not on `PATH`.

`make up` runs `make proto` first, which invokes `buf generate` — this needs the
`protoc-gen-go` and `protoc-gen-go-grpc` plugins as local binaries on `PATH` (buf
does not fetch these itself, unlike `buf`/`sqlc`/`goose` above). Install them once:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Make sure `$(go env GOPATH)/bin` is on `PATH` afterwards, or `buf generate` will
fail with `exec: "protoc-gen-go": executable file not found in $PATH`.

## Getting started

```sh
cp .env.example .env
make up          # brings all infra up and waits for healthy
make check       # probes every dependency from the host
```

`make up` uses `docker compose up -d --wait`, so it only returns 0 once every
container reports healthy. `minio-init` runs once and exits 0 after creating
the `kalakriti` bucket (`S3_BUCKET`).

## Verifying each dependency by hand

```sh
# Postgres + pgvector
docker compose exec -e PGPASSWORD=kalakriti postgres \
  psql -U kalakriti -d kalakriti -c 'CREATE EXTENSION IF NOT EXISTS vector;' -c '\dx'

# Redis
docker compose exec redis redis-cli ping                    # PONG

# Kafka (KRaft, single broker)
docker compose exec kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list

# MinIO
docker compose exec minio mc ready local
open http://localhost:9001                                  # console, minioadmin/minioadmin
```

## Everyday targets

```
make up | down | logs | ps | reset     infrastructure lifecycle
make check                             probe all dependencies
make proto                             buf generate -> pkg/pb + services/ml-svc/pb
make migrate-up | migrate-down         goose against POSTGRES_DSN
make seed                              load the craft ontology CSVs (idempotent)
make sqlc                              regenerate query code
make build | test | lint | tidy        Go across every module
make test-integration                  container-backed tests (needs Docker)
```

## Running core-svc

`make proto && make sqlc` must both have run at least once: core-svc compiles
against generated protobuf types in `pkg/pb/` and generated query code in
`services/core-svc/internal/core/repo/db/`, neither of which is committed.

```
make up && make migrate-up && make seed
cd services/core-svc && go run ./cmd/core-svc
```

It serves gRPC on `:50051` and `/healthz` + `/readyz` on `:8081`, and runs the
outbox relay in-process. With `AUTH_DEV_OTP_ENABLED=true` the login handshake
accepts the well-known code `000000`, so a demo needs no SMS provider; startup
refuses that flag when `ENV=production`.

```
grpcurl -plaintext -d '{"phone_e164":"+919876543210"}' \
  localhost:50051 identity.v1.IdentityService/RequestOtp
grpcurl -plaintext -d '{"challenge_id":"<id>","phone_e164":"+919876543210","code":"000000"}' \
  localhost:50051 identity.v1.IdentityService/VerifyOtp
```

Generated Python lands in `services/ml-svc/pb/` and its modules import each other
absolutely, so run `ml-svc` with `PYTHONPATH=services/ml-svc/pb`.

`make reset` destroys the named volumes. `make reset && make up` is the supported
way back to a clean state; it re-runs `scripts/postgres-init/` and recreates the
bucket.

## Frontend & NGINX Web Gateway

The web layer contains three independent SvelteKit applications in `web/apps/`:
- `apps/buyer`: Public responsive artisan marketplace
- `apps/artisan`: Offline-first mobile PWA for artisans
- `apps/admin`: Desktop portal for cluster officers and administrators

In production and full docker-compose mode, all three apps are built and served by NGINX (`Dockerfile.web` and `deploy/nginx/nginx.conf`) exposed on **Port 80**:
- Buyer: `http://localhost/` (or `kalakriti.in`)
- Artisan: `http://localhost/artisan/` (or `artisan.kalakriti.in`)
- Admin: `http://localhost/admin/` (or `admin.kalakriti.in`)
- API Gateway Proxy: `http://localhost/api/v1/*` (proxies to Go BFF at `:8000` with SSE support)
- Direct BFF API: `http://localhost:8000/api/v1/*`

Kubernetes deployment manifests are in `deploy/k8s/` (`bff-deployment.yaml`, `web-deployment.yaml`, `ingress.yaml`).

## Conventions

- All timestamps `TIMESTAMPTZ`, stored UTC. All money in paise as `BIGINT`.
- All IDs are UUIDv7 so they sort by creation time.
- gRPC between services; REST/JSON only at the BFF edge.
- Every Kafka-producing write goes through the transactional outbox.
- Every mutating endpoint accepts an idempotency key.

## Roadmap

Batch scope past #2 is provisional and will be adjusted as the build lands.

- [x] 1 — Repo scaffold, docker-compose stack, Makefile, buf, README
- [x] 2 — Proto contracts for every service, `make proto` green
- [x] 3 — Schema, goose migrations, sqlc wiring, ontology seeder
- [x] 4 — Shared `pkg/`: config, logging, errors, UUIDv7, pgxpool, redis, kafka,
      outbox, idempotency, storage, httpx, money — batch 7's outbox/Kafka
      plumbing shipped early as `pkg/outbox` + `pkg/kafka`; item 7 below is now
      just wiring services to it
- [x] 5 — core-svc: identity, auth, JWT, artisan profiles, clusters and SHGs.
      Adds `proto/identity/v1`, migration 013, and `pkg/auth`; core-svc now runs
      the outbox relay, so item 7's wiring is done for this service too
- [x] 6 — core-svc: catalog listings, media upload to MinIO
- [x] 7 — Wire the remaining services to the outbox and Kafka consumers
- [x] 8 — ml-svc: embeddings, auto-tagging, description generation
- [x] 9 — core-svc: AI cataloging pipeline consuming ml-svc
- [x] 10 — Provenance: craft lineage, verification events, hash chain
- [x] 11 — search-svc: pgvector HNSW, hybrid lexical + semantic search
- [x] 12 — collab-svc: artisan collectives, capacity, membership
- [x] 13 — collab-svc: pooled orders, allocation and settlement in paise
- [x] 14 — channel-svc: outbound marketplace sync, notifications
- [x] 15 — bff: REST edge, auth middleware, aggregation endpoints
- [x] 16 — PDF provenance certificates (gofpdf) and QR verification
- [x] 17 — Service Dockerfiles, compose app tier, seed data
- [x] 18 — Kubernetes manifests, observability, demo script

## Media uploads

Bytes never pass through a Go service. The client asks for a ticket, PUTs
straight to MinIO, and then tells the service to verify it:

```sh
# 1. ask for somewhere to put the bytes (grpcurl against a local stack)
grpcurl -plaintext -H "authorization: Bearer $TOKEN" \
  -d '{"artisan_id":"'$ARTISAN'","content_type":"image/jpeg","size_bytes":'$(stat -c%s photo.jpg)'}' \
  localhost:50051 catalog.v1.MediaService/RequestUpload

# 2. PUT the file at the URL that came back; the Content-Type must match
curl -X PUT --upload-file photo.jpg -H 'Content-Type: image/jpeg' "$UPLOAD_URL"

# 3. confirm; the service stats the object, flips the row to UPLOADED and
#    emits media.uploaded exactly once, however many times you call it
grpcurl -plaintext -H "authorization: Bearer $TOKEN" \
  -d '{"media_id":"'$MEDIA_ID'"}' \
  localhost:50051 catalog.v1.MediaService/ConfirmUpload
```

A ticket that is never used leaves a PENDING row; the reaper deletes it, and any
orphaned object, after `MEDIA_PENDING_TTL`.
