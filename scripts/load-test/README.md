# Load Testing

**Last Updated:** 2026-08-28  
**Tool:** k6 (https://k6.io)

---

## Overview

Load test scripts simulating realistic production traffic patterns against the Kalakriti API.

**Test scenarios:**
1. **browse-listings.js** — Read-heavy: browse and view listings (200 concurrent users)
2. **search.js** — CPU-intensive: vector search queries (50 concurrent users)
3. **auth-flow.js** — Auth flow: OTP request → verify → refresh (10 concurrent users)
4. **mixed-workload.js** — Realistic mix: 80% reads, 20% writes (150 concurrent users)

---

## Quick Start

### Install k6

```bash
# macOS
brew install k6

# Windows (Chocolatey)
choco install k6

# Linux
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Or download binary from https://k6.io/docs/getting-started/installation/
```

### Run Tests

```bash
# Start the demo stack first
make demo-up

# Run individual scenarios
k6 run scripts/load-test/browse-listings.js
k6 run scripts/load-test/search.js
k6 run scripts/load-test/auth-flow.js
k6 run scripts/load-test/mixed-workload.js

# Override base URL
BASE_URL=https://staging.kalakriti.in k6 run scripts/load-test/mixed-workload.js

# Run with custom duration
k6 run --duration 10m --vus 100 scripts/load-test/browse-listings.js
```

---

## Test Scenarios

### 1. Browse Listings (`browse-listings.js`)

**What it tests:** Read-heavy listing browsing

**Load profile:**
- 1m: Ramp to 50 users
- 3m: Stay at 50 users
- 1m: Ramp to 100 users
- 5m: Stay at 100 users
- 2m: Spike to 200 users
- 2m: Hold spike
- 1m: Ramp down

**Actions per user:**
- GET `/api/v1/listings?status=published&limit=20`
- GET `/api/v1/listings/{id}` (random from above)
- Sleep 3s between requests

**Success criteria:**
- p95 response time < 500ms
- p99 response time < 1000ms
- Error rate < 1%

**Expected RPS:** ~60-120 req/s at 100 users

---

### 2. Search (`search.js`)

**What it tests:** Vector search (CPU + pgvector intensive)

**Load profile:**
- 1m: Ramp to 30 users
- 3m: Stay at 30 users
- 1m: Ramp to 50 users
- 5m: Stay at 50 users
- 1m: Ramp down

**Actions per user:**
- GET `/api/v1/search?q=<query>&limit=10`
- GET `/api/v1/search/suggest?q=<prefix>` (autocomplete)
- Sleep 3s between requests

**Success criteria:**
- p95 response time < 1000ms
- p99 response time < 2000ms
- Error rate < 5%

**Expected RPS:** ~15-25 req/s at 50 users  
(Lower than browse because search is expensive)

---

### 3. Auth Flow (`auth-flow.js`)

**What it tests:** Complete authentication flow

**Load profile:**
- 30s: Ramp to 10 users
- 2m: Stay at 10 users
- 30s: Ramp down

**Actions per user:**
- POST `/api/v1/auth/otp/request` → request OTP
- POST `/api/v1/auth/otp/verify` → verify OTP (code=000000 in dev mode)
- POST `/api/v1/auth/refresh` → refresh token
- Sleep 4s total

**Success criteria:**
- p95 response time < 1000ms
- Error rate < 5%

**Expected RPS:** ~7-10 req/s at 10 users

**Note:** Requires `AUTH_DEV_OTP_ENABLED=true` in `.env`

---

### 4. Mixed Workload (`mixed-workload.js`)

**What it tests:** Realistic production traffic mix

**Load profile:**
- 2m: Ramp to 50 users
- 5m: Stay at 100 users
- 2m: Ramp to 150 users
- 3m: Stay at 150 users
- 2m: Ramp down

**Actions per user (weighted):**
- 40%: Browse listings
- 20%: Search
- 20%: View listing detail
- 10%: Get artisan profile (authenticated)
- 10%: Generate upload URL (write operation)

**Success criteria:**
- p95 response time < 1000ms
- p99 response time < 2000ms
- Error rate < 2%

**Expected RPS:** ~50-75 req/s at 150 users

---

## Monitoring During Tests

### Watch PostgreSQL Connections

```bash
# Terminal 1: Run load test
k6 run scripts/load-test/mixed-workload.js

# Terminal 2: Watch connections
watch -n 1 'docker compose exec postgres psql -U kalakriti -c "SELECT count(*) FROM pg_stat_activity WHERE datname='\''kalakriti'\'';"'
```

### Watch Redis Memory

```bash
watch -n 1 'docker compose exec redis redis-cli INFO memory | grep used_memory_human'
```

### Watch Service Logs

```bash
# All services
docker compose logs -f --tail=50

# Specific service
docker compose logs -f bff | grep "status=500"
```

---

## Interpreting Results

### k6 Output Explained

```
scenarios: (100.00%) 1 scenario, 100 max VUs, 15m30s max duration
default: 100 looping VUs for 15m0s

✓ browse status 200
✓ browse response time < 500ms

checks.........................: 98.50% ✓ 29550  ✗ 450
data_received..................: 45 MB  50 kB/s
data_sent......................: 3.2 MB 3.6 kB/s
http_req_blocked...............: avg=1.2ms   min=2µs    med=8µs    max=89ms   p(90)=15µs  p(95)=21ms
http_req_connecting............: avg=980µs   min=0s     med=0s     max=34ms   p(90)=0s    p(95)=12ms
http_req_duration..............: avg=245ms   min=12ms   med=189ms  max=1.2s   p(90)=456ms p(95)=678ms
http_req_failed................: 1.50%  ✓ 450    ✗ 29550
http_req_receiving.............: avg=890µs   min=21µs   med=456µs  max=45ms   p(90)=1.8ms p(95)=2.5ms
http_req_sending...............: avg=67µs    min=8µs    med=34µs   max=12ms   p(90)=123µs p(95)=178µs
http_req_tls_handshaking.......: avg=0s      min=0s     med=0s     max=0s     p(90)=0s    p(95)=0s
http_req_waiting...............: avg=244ms   min=12ms   med=188ms  max=1.2s   p(90)=455ms p(95)=677ms
http_reqs......................: 30000  333.33/s
iteration_duration.............: avg=3.24s   min=3.01s  med=3.19s  max=4.2s   p(90)=3.46s p(95)=3.68s
iterations.....................: 10000  111.11/s
vus............................: 100    min=100  max=100
vus_max........................: 100    min=100  max=100
```

**Key metrics:**
- `http_req_duration (p95)` — 95% of requests faster than this → **should be < 1s**
- `http_req_failed` — Error rate → **should be < 2%**
- `http_reqs` — Throughput (requests per second)
- `checks` — Percentage of assertions that passed → **should be > 95%**

### When Tests Fail

**Symptom:** p95 > 1s, p99 > 2s

**Likely causes:**
- Database connection pool exhausted → increase `POSTGRES_MAX_CONNS`
- Slow queries → add indexes (see `docs/DATABASE_INDEXES.md`)
- CPU bound → scale horizontally (add more service replicas)

**Symptom:** Error rate > 5%

**Likely causes:**
- Rate limiting kicking in → increase `RATE_LIMIT_RPS`
- Database deadlocks → check logs for `could not serialize access`
- Service crashes → check `docker compose logs`

**Symptom:** Tests run fine for 2min, then degrade

**Likely causes:**
- Memory leak → check `docker stats`
- Connection leaks → check `pg_stat_activity`
- Disk I/O saturation → check `iostat`

---

## Baselines (Targets)

Run these tests **before production launch** to establish baselines:

| Scenario | Users | Duration | Target p95 | Target p99 | Target Error Rate |
|----------|-------|----------|------------|------------|-------------------|
| Browse | 100 | 10m | <500ms | <1000ms | <1% |
| Search | 50 | 10m | <1000ms | <2000ms | <5% |
| Auth | 10 | 5m | <1000ms | <1500ms | <5% |
| Mixed | 150 | 15m | <1000ms | <2000ms | <2% |

**Record results:**
```bash
k6 run scripts/load-test/mixed-workload.js --out json=results-mixed-$(date +%Y%m%d).json
```

---

## Pre-Production Checklist

Before going live, run all scenarios and verify:

- [ ] All tests pass success criteria
- [ ] PostgreSQL connections stay below 80% of `max_connections`
- [ ] Redis memory usage stays below 1GB
- [ ] No 500 errors in logs
- [ ] CPU usage stays below 70%
- [ ] Disk I/O not saturated
- [ ] Response times stable over 15-minute test

---

## CI Integration

Add to GitHub Actions / GitLab CI:

```yaml
# .github/workflows/load-test.yml
name: Load Test

on:
  schedule:
    - cron: '0 2 * * 1'  # Every Monday at 2am
  workflow_dispatch:

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install k6
        run: |
          sudo gpg -k
          sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      - name: Start services
        run: make demo-up
      - name: Run load tests
        run: |
          k6 run --quiet scripts/load-test/browse-listings.js
          k6 run --quiet scripts/load-test/search.js
          k6 run --quiet scripts/load-test/mixed-workload.js
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: load-test-results
          path: results-*.json
```

---

## Summary

**Tests created:** 4 scenarios (browse, search, auth, mixed)  
**Target load:** 150-200 concurrent users  
**Success criteria:** p95 < 1s, error rate < 2%  
**Next step:** Run baseline tests and record results

**Before launch:**
```bash
make demo-up
k6 run scripts/load-test/mixed-workload.js --out json=baseline-$(date +%Y%m%d).json
```
