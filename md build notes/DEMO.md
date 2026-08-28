# Kalakriti Demo Runbook

Complete demonstration guide for Smart India Hackathon 2026.

**Date:** September 2026  
**Duration:** 8 minutes  
**Prerequisites:** Docker, Docker Compose, Go 1.23

---

## Pre-Demo Setup (Do this before judges arrive)

### 1. Pull Docker Images (5-10 minutes, requires internet)

```bash
docker-compose -f docker-compose.full.yml pull
```

**Expected output:** All images downloaded successfully.

### 2. Build Application Images (3-5 minutes)

```bash
make docker-build
```

**Expected output:**
```
Building Docker images...
...
✓ All images built
```

### 3. Start Demo Environment (2-3 minutes)

```bash
make demo-up
```

**Expected output:**
```
Starting full demo environment...
[+] Running 17/17
 ✔ Container postgres started
 ✔ Container redis started
 ✔ Container kafka started
 ✔ Container minio started
 ...
✓ Demo environment ready!

Services:
  BFF API:        http://localhost:8000
  Jaeger UI:      http://localhost:16686
  Prometheus:     http://localhost:9090
  MinIO Console:  http://localhost:9001 (minioadmin/minioadmin)
```

### 4. Verify Services Are Healthy

```bash
curl http://localhost:8000/health
curl http://localhost:8080/health
curl http://localhost:8081/health
```

**Expected output:** `OK` from each service.

### 5. Generate QR Code Sheet (for physical demo)

```bash
make tags
```

**Expected output:** `✓ QR sheet generated: qr-sheet.pdf`

**Print this PDF** for scanning demonstrations.

---

## 8-Minute Demo Script

### **Minute 1-2: Problem Statement & Architecture**

**Script:**
> "Kalakriti empowers Indian artisans with AI-powered cataloging, blockchain-style provenance tracking, and collective fulfillment for bulk orders. Built as a microservices platform on Go, PostgreSQL, Kafka, and gRPC."

**Show:** Architecture diagram (prepare slide showing all services).

**Browser:**
- Open Jaeger UI: `http://localhost:16686`
- Show distributed tracing across services

### **Minute 3-4: AI-Powered Cataloging**

**Demonstrate:** Artisan uploads product photo → ML pipeline auto-generates multilingual listing.

#### Terminal 1: Watch Pipeline Logs
```bash
docker-compose -f docker-compose.full.yml logs -f pipeline-orchestrator
```

#### Terminal 2: Simulate Photo Upload
```bash
# Create test image upload
curl -X POST http://localhost:8000/api/v1/catalog/upload \
  -H "Authorization: Bearer <artisan-token>" \
  -F "image=@test-assets/madhubani-fish.jpg" \
  -F "artisan_id=<artisan-uuid>"
```

**Expected output in Terminal 1:**
```
[INFO] Processing image <id>
[INFO] ML: craft detected = madhubani (confidence 0.94)
[INFO] ML: title generated (EN/HI)
[INFO] Listing drafted successfully
```

**Browser:** Refresh catalog endpoint to show new draft listing with AI-generated titles.

```bash
curl http://localhost:8000/api/v1/catalog/listings?status=draft | jq
```

**Expected JSON:** Listing with `title_en`, `title_hi`, `description_en`, `craft_type`, all AI-generated.

### **Minute 4-5: Multilingual Search**

**Demonstrate:** Search works in English and Hindi with vector similarity.

```bash
# English search
curl "http://localhost:8000/api/v1/search?q=traditional+fish+painting" | jq

# Hindi search
curl "http://localhost:8000/api/v1/search?q=मछली+चित्रकला" | jq
```

**Expected output:** Both return the Madhubani listing (same result, different query languages).

**Browser:** Open Prometheus: `http://localhost:9090`
- Query: `search_vector_distance_seconds`
- Show sub-100ms vector search latency

### **Minute 5-6: Bulk Order Allocation (Saga Pattern)**

**Demonstrate:** 500-unit order automatically allocated across 3 artisans with capacity constraints.

```bash
# Place bulk order
curl -X POST http://localhost:8000/api/v1/orders/bulk \
  -H "Content-Type: application/json" \
  -d '{
    "buyer_id": "<buyer-uuid>",
    "craft": "pottery",
    "quantity": 500,
    "unit_price_paise": 50000
  }' | jq
```

**Expected output:**
```json
{
  "order_id": "<uuid>",
  "status": "allocating",
  "total_paise": 25000000
}
```

#### Watch Saga Progress
```bash
docker-compose -f docker-compose.full.yml logs -f order-svc
```

**Expected logs:**
```
[INFO] Saga started: order allocation
[INFO] Allocated 200 units to artisan A
[INFO] Allocated 200 units to artisan B
[INFO] Allocated 100 units to artisan C
[INFO] Saga completed: all allocated
```

#### Verify Allocations
```bash
curl "http://localhost:8000/api/v1/orders/<order-id>/allocations" | jq
```

**Expected output:** 3 allocations totaling exactly 500 units.

**Browser - Prometheus:**
- Query: `saga_state_count{saga_type="order_allocation"}`
- Show saga state transitions in real-time

### **Minute 6-7: Provenance Verification**

**Demonstrate:** QR code verification of handcrafted provenance with process video.

**Physical demo:** Scan printed QR code with phone camera.

**OR Terminal demo:**
```bash
# Simulate QR scan
curl "http://localhost:8000/api/v1/provenance/verify?qr=PROV_abc12345" | jq
```

**Expected output:**
```json
{
  "valid": true,
  "listing_id": "<uuid>",
  "artisan": {
    "name": "Lakshmi Devi",
    "craft": "madhubani",
    "state": "bihar"
  },
  "process_videos": ["https://storage/video1.mp4"],
  "materials": ["natural dyes", "cotton canvas"],
  "location": {"lat": 25.5941, "lng": 85.1376},
  "sealed_at": "2026-08-15T10:30:00Z"
}
```

**Browser:** Play process video URL to show artisan at work (have sample video ready).

### **Minute 7-8: Income Statement for Bank Loans**

**Demonstrate:** Verifiable income statement with QR code for bank verification.

```bash
# Generate income statement
curl -X POST http://localhost:8000/api/v1/artisan/<artisan-id>/income-statement \
  -H "Content-Type: application/json" \
  -d '{"year": 2026, "month": 8}' | jq
```

**Expected output:**
```json
{
  "pdf_url": "https://storage/statements/<uuid>.pdf",
  "qr_code": "INCOME_xyz67890",
  "total_paise": 345000,
  "order_count": 12
}
```

#### Bank Verification (Show QR scan)
```bash
curl "http://localhost:8000/api/v1/income/verify?qr=INCOME_xyz67890" | jq
```

**Expected output:**
```json
{
  "valid": true,
  "artisan_id": "<uuid>",
  "year": 2026,
  "month": 8,
  "total_paise": 345000,
  "verified_at": "2026-08-28T10:30:00Z"
}
```

**Key message:** "Banks can scan this QR to instantly verify artisan income without manual paperwork."

### **Wrap-up (30 seconds)**

**Show observability stack:**
- **Jaeger:** Full request trace across 5+ microservices
- **Prometheus:** RED metrics (Rate, Errors, Duration) for all endpoints

**Final statement:**
> "Production-ready microservices architecture, sub-100ms search, AI-powered cataloging in 2 seconds, and verifiable provenance—all working offline after initial setup."

---

## Troubleshooting

### Issue 1: Services not starting

**Symptom:** `docker-compose up` hangs or services exit immediately.

**Check:**
```bash
docker-compose -f docker-compose.full.yml ps
```

**Fix:**
```bash
# Check logs for specific service
docker-compose -f docker-compose.full.yml logs postgres
docker-compose -f docker-compose.full.yml logs kafka

# Common fix: reset volumes
make demo-reset
```

### Issue 2: Database connection refused

**Symptom:** `connection refused` or `database does not exist`.

**Check:**
```bash
docker-compose -f docker-compose.full.yml exec postgres pg_isready -U kalakriti
```

**Fix:**
```bash
# Wait for postgres to be fully ready
sleep 10

# Re-run migrations
DATABASE_URL="postgres://kalakriti:kalakriti@localhost:5432/kalakriti?sslmode=disable" make migrate
```

### Issue 3: Kafka connection timeout

**Symptom:** Services log `kafka: connection timeout`.

**Check:**
```bash
docker-compose -f docker-compose.full.yml exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092
```

**Fix:**
```bash
# Restart Kafka
docker-compose -f docker-compose.full.yml restart kafka

# Wait 15 seconds for full startup
sleep 15
```

### Issue 4: MinIO buckets not created

**Symptom:** `NoSuchBucket` errors in catalog-svc logs.

**Fix:**
```bash
# Access MinIO console: http://localhost:9001
# Login: minioadmin / minioadmin
# Create buckets: images, videos, documents

# OR via CLI:
docker-compose -f docker-compose.full.yml exec minio \
  mc alias set local http://localhost:9000 minioadmin minioadmin
docker-compose -f docker-compose.full.yml exec minio \
  mc mb local/images local/videos local/documents
```

### Issue 5: Seed data not inserted

**Symptom:** Search returns empty results, no artisans in database.

**Check:**
```bash
docker-compose -f docker-compose.full.yml exec postgres \
  psql -U kalakriti -c "SELECT COUNT(*) FROM users WHERE role='artisan';"
```

**Expected:** 20 artisans.

**Fix:**
```bash
# Re-run seed
DATABASE_URL="postgres://kalakriti:kalakriti@localhost:5432/kalakriti?sslmode=disable" make seed

# If still failing, check seed script output
go run scripts/seed/main.go
```

---

## Health Check Commands

Run these before starting the demo to ensure everything is working:

```bash
# Infrastructure
curl http://localhost:5432  # Should connect (empty response OK)
redis-cli -h localhost ping  # Should return PONG
curl http://localhost:9000/minio/health/live  # Should return 200

# Application services
curl http://localhost:8080/health  # user-svc
curl http://localhost:8081/health  # catalog-svc
curl http://localhost:8082/health  # search-svc
curl http://localhost:8083/health  # order-svc
curl http://localhost:8084/health  # social-svc
curl http://localhost:8000/health  # bff

# Readiness (checks dependencies)
curl http://localhost:8080/ready | jq
```

**All should return** `OK` or `{"postgres":true,"redis":true}`.

---

## Quick Reset

If anything goes wrong during the demo:

```bash
make demo-reset
```

This drops all data, recreates containers, runs migrations, and reseeds. Takes **2-3 minutes**.

---

## Offline Demo Notes

After running `make demo-up` once with internet:

1. All Docker images are cached locally
2. All dependencies are in containers
3. **The entire demo works with network disconnected**

To verify offline capability:
```bash
# Disconnect network, then:
docker-compose -f docker-compose.full.yml down
make demo-up
# Should work without internet
```

---

## Performance Benchmarks

Expected metrics (run on laptop with 16GB RAM):

- **Startup time:** 90 seconds (cold start with migrations)
- **Search latency:** <100ms (p99)
- **ML pipeline:** 2-4 seconds per image
- **Bulk allocation saga:** <1 second for 500 units
- **Docker image sizes:** 20-28 MB per Go service

---

## Contact & Support

For demo day support, have these ready:

- This runbook (printed)
- QR code sheet (printed)
- Sample product images in `test-assets/`
- Backup laptop with pre-pulled images
- Mobile phone for QR scanning demos

**Last system check before judges:** Run full demo script once at T-30 minutes.
