# Kalakriti — Quickstart Guide

**Last updated:** 2026-08-28  
**Status:** 100% backend implementation complete, ready for frontend integration

---

## What This Is

Kalakriti backend: 7 microservices (6 Go + 1 Python) for AI-powered artisan marketplace.
- REST API at http://localhost:8000 (BFF service)
- gRPC between services
- Single PostgreSQL database with vector search
- Event-driven (Kafka + transactional outbox)
- Observability: OpenTelemetry tracing, Prometheus metrics

**Frontend team:** BFF exposes REST API at `/api/v1/*` — see API section below.

---

## Prerequisites

Install these before starting:

| Tool | Version | Install |
|------|---------|---------|
| Docker Desktop | 24+ | https://docker.com |
| Go | 1.23+ | https://go.dev/dl |
| Python | 3.11+ | https://python.org (only needed if rebuilding ml-svc) |
| Make | Any | Built into Git Bash on Windows |
| Git | Any | https://git-scm.com |

**Windows users:** Run commands in Git Bash (comes with Git for Windows).

---

## Step 1: Clone & Setup

```bash
cd /c/projects/kalakritibatch13/kalakriti  # You're already here

# Copy environment template
cp .env.example .env

# .env has local dev defaults, no editing needed for first run
```

---

## Step 2: Start Infrastructure Only (Postgres, Redis, Kafka, MinIO)

```bash
make up          # Starts infra containers, waits for healthy (~30s)
make check       # Verifies each service is responding
```

**What just started:**
- PostgreSQL 18 + pgvector on port 5432
- Redis 7 on port 6379
- Kafka (KRaft mode) on port 9092
- MinIO S3 on port 9000 (console: http://localhost:9001)
- Jaeger tracing UI on http://localhost:16686

**Troubleshooting:**
- "port already in use" → something else using 5432/6379/9092/9000
  - Stop other services or edit docker-compose.yml ports
- "Docker daemon not running" → start Docker Desktop

---

## Step 3: Apply Migrations + Seed Data

```bash
make migrate-up  # Runs all 13 SQL migrations
make seed        # Loads craft ontology (20 crafts, 50+ aliases)
```

**What this creates:**
- 13 tables: users, artisans, listings, media, orders, payments, etc.
- Indexes including HNSW vector index for search
- Seed data: craft types (Madhubani, Warli, Kalamkari, etc.)

---

## Step 4: Full Demo (All Services)

```bash
make demo-up
```

**This does everything:**
1. Starts infrastructure (Postgres, Redis, Kafka, MinIO)
2. Builds Docker images for all 6 services
3. Starts all services in containers
4. Runs migrations
5. Seeds 20 artisans, 60 listings, 3 bulk orders

**Wait time:** ~5 minutes first run (Docker builds), ~2 minutes after.

**Services started:**
- `core-svc` → Identity, catalog, media, provenance (gRPC :50051)
- `search-svc` → Vector search (gRPC :50052)
- `collab-svc` → Bulk orders, allocations (gRPC :50053)
- `bff` → REST API (:8000) — **this is what frontend calls**
- `ml-svc` → ML inference in MOCK_MODE (:50055)
- `insight-svc` → Income statements (gRPC :50056)
- `channel-svc` → Notifications, follows, exports (gRPC :9096)

**Check if running:**
```bash
docker compose -f docker-compose.full.yml ps
```

All services should show "healthy" status.

---

## Step 5: Test the API

### BFF Health Check
```bash
curl http://localhost:8000/healthz
# Should return: {"status":"ok"}
```

### Register an Artisan
```bash
# 1. Request OTP (mock mode, always sends 000000)
curl -X POST http://localhost:8000/api/v1/auth/otp \
  -H "Content-Type: application/json" \
  -d '{"phone_e164": "+919876543210"}'

# Response: {"challenge_id": "some-uuid"}

# 2. Verify OTP
curl -X POST http://localhost:8000/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "challenge_id": "some-uuid",
    "phone_e164": "+919876543210",
    "code": "000000"
  }'

# Response: {"access_token": "eyJ...", "user_id": "uuid"}
# Save this token for next requests
export TOKEN="eyJ..."

# 3. Create artisan profile
curl -X POST http://localhost:8000/api/v1/artisans \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Sunita Devi",
    "craft": "madhubani",
    "village": "Jitwarpur",
    "district": "Madhubani",
    "state": "Bihar"
  }'

# Response: {"id": "artisan-uuid", "full_name": "Sunita Devi", ...}
```

### Upload a Photo (3-step process)
```bash
# 1. Request presigned upload URL
curl -X POST http://localhost:8000/api/v1/media/upload-url \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "artisan_id": "artisan-uuid",
    "content_type": "image/jpeg",
    "size_bytes": 524288
  }'

# Response: {"media_id": "media-uuid", "upload_url": "http://localhost:9000/..."}

# 2. Upload file directly to MinIO (replace with real image)
curl -X PUT "http://localhost:9000/..." \
  -H "Content-Type: image/jpeg" \
  --data-binary "@test-photo.jpg"

# 3. Confirm upload (triggers ML pipeline)
curl -X POST http://localhost:8000/api/v1/media/media-uuid/confirm \
  -H "Authorization: Bearer $TOKEN"

# ML pipeline runs in background (~5s in mock mode)
# Creates draft listing with AI-generated title/description
```

### Check Listings
```bash
# Wait 5 seconds, then:
curl http://localhost:8000/api/v1/listings?status=draft \
  -H "Authorization: Bearer $TOKEN"

# Should see the auto-generated listing from your uploaded photo
```

### Search Listings
```bash
curl "http://localhost:8000/api/v1/search?q=madhubani+fish" \
  -H "Authorization: Bearer $TOKEN"

# Hybrid search: BM25 + vector similarity (works in English/Hindi)
```

---

## API Overview for Frontend

**Base URL:** `http://localhost:8000/api/v1`

**Auth:** All endpoints except `/auth/*` require header:
```
Authorization: Bearer <token>
```

### Core Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/auth/otp` | Request OTP for phone |
| POST | `/auth/verify` | Verify OTP, get token |
| POST | `/artisans` | Create artisan profile |
| GET | `/artisans/{id}` | Get artisan details |
| PATCH | `/artisans/{id}` | Update profile |
| POST | `/media/upload-url` | Get presigned upload URL |
| POST | `/media/{id}/confirm` | Confirm upload (triggers ML) |
| GET | `/listings` | List/filter listings |
| GET | `/listings/{id}` | Get listing details |
| POST | `/listings/{id}/publish` | Publish draft listing |
| GET | `/search` | Search listings (query, craft, price range) |
| POST | `/orders` | Create bulk order |
| GET | `/orders/{id}` | Get order status |
| POST | `/statements` | Generate income statement PDF |
| GET | `/statements/{code}/verify` | Verify statement signature |

**30 endpoints total** — see `services/bff/internal/bff/handler/*.go` for complete list.

---

## Testing End-to-End

### Journey Test (Manual)
1. Register artisan (POST `/artisans`)
2. Upload photo (3-step: request URL → PUT to MinIO → confirm)
3. Wait 5s for ML pipeline
4. Check listings (GET `/listings?status=draft`)
5. Publish listing (POST `/listings/{id}/publish`)
6. Search for it (GET `/search?q=...`)

**Expected:** Listing appears with AI-generated title/description.

### Automated Integration Tests
```bash
cd tests/integration
go test -v ./...

# Tests 4 journeys:
# 1. Artisan registration → photo → ML → listing
# 2. Multilingual search (English/Hindi)
# 3. Bulk order allocation across 3 artisans
# 4. Income statement generation + verification
```

---

## Observability

### Jaeger (Distributed Tracing)
http://localhost:16686

Search for service `bff` to see traces across all services. Every request gets a trace ID propagated through Kafka.

### Prometheus (Metrics)
http://localhost:9090

Query `http_requests_total`, `grpc_server_handled_total`, etc.

### Logs
```bash
# All services
docker compose -f docker-compose.full.yml logs -f

# One service
docker compose -f docker-compose.full.yml logs -f bff

# Infrastructure only
docker compose logs -f postgres redis kafka
```

---

## Stopping Everything

```bash
# Stop services, keep data
docker compose -f docker-compose.full.yml down

# Stop infrastructure
make down

# Nuclear reset (deletes all data)
make demo-reset
```

---

## Git Push Strategy

### Before Pushing

1. **Add `.env` to `.gitignore`** (if not already there):
```bash
echo ".env" >> .gitignore
echo "bin/" >> .gitignore
echo "*.log" >> .gitignore
```

2. **Create `.env.example`** (template with no secrets):
```bash
cat > .env.example << 'EOF'
# Infrastructure
POSTGRES_USER=kalakriti
POSTGRES_PASSWORD=kalakriti
POSTGRES_DB=kalakriti
POSTGRES_PORT=5432
POSTGRES_DSN=postgres://kalakriti:kalakriti@localhost:5432/kalakriti?sslmode=disable

# Services
ENV=development
AUTH_DEV_OTP_ENABLED=true
JWT_SECRET=dev-secret-change-in-production
MEDIA_PENDING_TTL=1h

# MinIO
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
MINIO_ENDPOINT=localhost:9000
MINIO_BUCKET=kalakriti-media

# Kafka
KAFKA_BOOTSTRAP_SERVERS=localhost:9092

# Redis
REDIS_ADDR=localhost:6379

# Observability
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
PROMETHEUS_PORT=9090
EOF
```

3. **Test clean checkout** (optional but recommended):
```bash
# In a temp directory
git clone <your-repo-url> kalakriti-test
cd kalakriti-test
cp .env.example .env
make demo-up

# If this works, your repo is good to share
```

### Push Commands

```bash
# Initial push
git init  # If not already a repo
git add .
git commit -m "feat: complete backend implementation

- 7 microservices (core, search, collab, bff, ml, insight, channel)
- 13 database migrations
- Docker compose for local dev
- Integration tests
- Observability (tracing, metrics)

Co-Authored-By: Claude <noreply@anthropic.com>"

git remote add origin https://github.com/yourorg/kalakriti.git
git branch -M main
git push -u origin main
```

### For Your Team

Share this in your repo README or Slack:

```markdown
## For Frontend Team

**Backend API:** http://localhost:8000/api/v1

**Setup (5 minutes):**
1. Install Docker Desktop + Go 1.23
2. `git clone <repo>` && `cd kalakriti`
3. `cp .env.example .env`
4. `make demo-up` (wait 5 min first time)
5. API ready at http://localhost:8000

**API Docs:** See QUICKSTART.md Section "API Overview"

**Test Credentials:**
- Phone: any `+91` number
- OTP: always `000000` in dev mode
```

---

## Troubleshooting

### "Services won't start"
```bash
# Check logs
docker compose -f docker-compose.full.yml logs bff core-svc

# Common fixes:
make demo-reset  # Nuclear option, deletes everything and rebuilds
```

### "ML service slow"
ML-svc runs in MOCK_MODE by default (no model weights, returns fake data in <2s).
Real mode needs 4GB model files. Mock mode is enough for frontend dev.

### "Database connection failed"
```bash
# Check Postgres is up
docker compose ps postgres

# Test connection
docker compose exec postgres psql -U kalakriti -d kalakriti -c 'SELECT version();'
```

### "Port conflicts"
Edit `docker-compose.yml` and `docker-compose.full.yml`:
```yaml
ports:
  - "8000:8000"  # Change left number: "8080:8000"
```

---

## What's Next

**For backend:**
- ✅ All 18 batches complete
- ✅ Integration tests passing
- ⏳ Manual testing (you're doing this now)
- ⏳ Demo practice for judges

**For frontend:**
- Connect to http://localhost:8000/api/v1
- All endpoints return JSON
- Auth: POST /auth/otp → POST /auth/verify → Bearer token
- Real S3 presigned URLs for media upload (no backend proxy)

**For deployment (later):**
- K8s manifests in `deploy/k8s/`
- Prometheus config in `deploy/prometheus.yml`
- Multi-stage Dockerfiles already optimized (<30MB per service)

---

## Need Help?

**Error messages:** Check `docker compose logs -f <service-name>`  
**API questions:** Read `services/bff/internal/bff/handler/*.go`  
**Database schema:** See `migrations/*.sql`  
**Questions:** Ask your backend dev (that's you!)

---

**Timeline:** 23 days until demo (Sept 20, 2026)  
**Status:** Backend 100% ready for frontend integration  
**Next:** `make demo-up` and test one full journey
