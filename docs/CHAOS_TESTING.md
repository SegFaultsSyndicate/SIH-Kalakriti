# Chaos Testing

**Last Updated:** 2026-08-28  
**Status:** Production-ready

---

## Overview

Automated chaos testing scenarios to verify system resilience under failure conditions. Tests infrastructure failures, resource exhaustion, and service crashes.

**Test scenarios:**
1. **Redis cache flush** — Cache miss storm
2. **Service crash** — Pod/container termination
3. **CPU stress** — Resource exhaustion
4. **Kafka partition** — Message queue failure
5. **Disk fill** — Out of space handling
6. **Database latency** — Network degradation

**Run time:** ~10 minutes for full suite (60s per test + 30s cooldown)

---

## Quick Start

```bash
# Run full chaos testing suite
cd scripts/chaos
bash run-suite.sh

# Run individual test
bash redis-flush.sh

# Run with custom parameters
SERVICE=core-svc DURATION=120 bash service-crash.sh
```

**Prerequisites:**
- Docker or Kubernetes cluster running
- Services deployed and healthy
- Monitoring/metrics enabled (recommended)

---

## Test Scenarios

### 1. Redis Cache Flush

**What it tests:** Cache miss handling, database load under cold cache

**Script:** `redis-flush.sh`

**Expected impact:**
- All cache misses hit database
- API latency spike (2-3x slower)
- Database CPU/connections increase
- Cache warms up over 5-10 minutes

**Success criteria:**
- No errors (slower responses acceptable)
- Cache hit rate recovers to 80%+
- Database doesn't fall over

**Run:**
```bash
REDIS_HOST=localhost REDIS_PORT=6379 bash redis-flush.sh
```

---

### 2. Service Crash

**What it tests:** Service recovery, load balancer failover, graceful degradation

**Script:** `service-crash.sh`

**Expected impact:**
- Service restarts automatically (Docker restart policy or Kubernetes ReplicaSet)
- Dependent services may see errors during restart (~5-10s window)
- Circuit breakers prevent cascading failures

**Success criteria:**
- Service restarts within 10s
- No data loss (transactions rolled back)
- Dependent services degrade gracefully (not crash)

**Run:**
```bash
# Kill random service
bash service-crash.sh

# Kill specific service
SERVICE=core-svc bash service-crash.sh
```

---

### 3. CPU Stress

**What it tests:** Performance under resource contention

**Script:** `cpu-stress.sh`

**Expected impact:**
- Slower request processing
- Increased p95/p99 latency
- Load balancer may shift traffic to other replicas
- No errors (requests just slower)

**Success criteria:**
- Latency increases but requests still succeed
- Service doesn't crash/OOM
- Performance recovers after stress ends

**Run:**
```bash
# Stress 2 CPU cores for 60s
SERVICE=core-svc CORES=2 DURATION=60 bash cpu-stress.sh
```

---

### 4. Kafka Network Partition

**What it tests:** Message queue unavailability, outbox pattern resilience

**Script:** `kafka-partition.sh`

**Expected impact:**
- Producers buffer messages locally (outbox table)
- Consumers stop processing
- Services should NOT crash (graceful degradation)
- Messages flush after recovery

**Success criteria:**
- No message loss (outbox pattern prevents it)
- Services continue serving non-Kafka operations
- Messages delivered after Kafka recovers

**Run:**
```bash
KAFKA_CONTAINER=kafka DURATION=60 bash kafka-partition.sh
```

**Monitoring:**
```bash
# Check outbox queue depth
psql -c "SELECT COUNT(*) FROM outbox WHERE sent_at IS NULL;"

# Check Kafka consumer lag
docker exec kafka kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group kalakriti-group
```

---

### 5. Disk Fill

**What it tests:** Out-of-space error handling

**Script:** `disk-fill.sh`

**Expected impact:**
- Write operations may fail
- PostgreSQL may refuse new connections
- Services should log errors gracefully
- No data corruption

**Success criteria:**
- Services handle write failures (return 500, not crash)
- Database doesn't corrupt data
- System recovers when space freed

**Run:**
```bash
# Fill 1GB on postgres container
SERVICE=postgres SIZE=1G DURATION=60 bash disk-fill.sh
```

---

### 6. Database Latency

**What it tests:** Slow database performance, connection pool exhaustion

**Script:** `db-latency.sh`

**Expected impact:**
- Slower API responses
- Circuit breakers may open (after 5 failures)
- Connection pool may exhaust (max 10 connections per service)

**Success criteria:**
- Circuit breakers prevent cascading failures
- Connection pool doesn't deadlock
- Performance recovers when latency removed

**Run:**
```bash
# Add 100ms latency to database
sudo bash db-latency.sh

# Custom latency
sudo LATENCY=500ms DURATION=120 bash db-latency.sh
```

**Note:** Requires root (uses `tc` for traffic control)

---

## Running Full Suite

```bash
cd scripts/chaos
bash run-suite.sh
```

**Suite runs:**
1. Redis flush
2. Service crash (random service)
3. CPU stress (random service)
4. Kafka partition
5. 30s cooldown between tests
6. Logs to `/tmp/chaos-YYYYMMDD-HHMMSS.log`

**Total duration:** ~10 minutes

---

## Monitoring During Tests

### Metrics to Watch

**Request latency:**
```bash
# p95/p99 should spike during tests
curl http://localhost:9090/metrics | grep http_request_duration
```

**Error rate:**
```bash
# Should stay <1% (except during service crash)
curl http://localhost:9090/metrics | grep http_requests_total
```

**Circuit breaker state:**
```bash
# Check for open circuit breakers
grep "circuit.*open" logs/*.log
```

**Database connections:**
```bash
# Watch connection pool usage
psql -c "SELECT count(*) FROM pg_stat_activity WHERE datname='kalakriti';"
```

**Kafka consumer lag:**
```bash
docker exec kafka kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe --group kalakriti-group
```

---

## Expected Results

### Pass Criteria

| Test | Metric | Pass Threshold |
|------|--------|----------------|
| Redis flush | Error rate | <1% |
| Redis flush | Latency spike | <5x baseline |
| Service crash | Recovery time | <10s |
| Service crash | Data loss | 0 messages |
| CPU stress | Error rate | <1% |
| CPU stress | Latency spike | <10x baseline |
| Kafka partition | Message loss | 0 messages |
| Kafka partition | Service crashes | 0 crashes |
| Disk fill | Data corruption | None |
| DB latency | Circuit breaker opens | Yes (expected) |

### Failure Indicators

**Service crashes unexpectedly:**
- Check for nil pointer dereferences
- Check for missing error handling
- Review graceful shutdown logic

**Data loss:**
- Check outbox pattern implementation
- Verify database transactions committed
- Review Kafka producer acks

**Cascading failures:**
- Circuit breakers not working
- Missing timeouts
- Synchronous calls without fallback

---

## Automated Chaos Testing

### CI/CD Integration

```yaml
# .github/workflows/chaos-test.yml
name: Chaos Testing

on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2am

jobs:
  chaos:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Start services
        run: docker-compose up -d
      - name: Wait for healthy
        run: sleep 30
      - name: Run chaos suite
        run: bash scripts/chaos/run-suite.sh
      - name: Check service health
        run: docker-compose ps
      - name: Upload logs
        uses: actions/upload-artifact@v3
        with:
          name: chaos-logs
          path: /tmp/chaos-*.log
```

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: chaos-testing
spec:
  schedule: "0 2 * * *"  # Daily at 2am
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: chaos
            image: kalakriti/chaos:latest
            command:
            - /bin/bash
            - -c
            - |
              cd /chaos
              bash run-suite.sh
              kubectl get pods -A  # Check all pods healthy
          restartPolicy: OnFailure
```

---

## Advanced Scenarios

### Combined Failures

Test multiple failures at once (more realistic):

```bash
# Redis flush + CPU stress
bash redis-flush.sh &
SERVICE=core-svc bash cpu-stress.sh &
wait
```

### Rolling Chaos

Kill services one by one:

```bash
for svc in core-svc collab-svc search-svc; do
    SERVICE=$svc bash service-crash.sh
    sleep 30
done
```

### Load Test + Chaos

Combine with k6 load testing:

```bash
# Terminal 1: Run load test
k6 run scripts/load-test/mixed-workload.js

# Terminal 2: Inject chaos
bash scripts/chaos/service-crash.sh
```

---

## Troubleshooting

### Script Fails with "Container not found"

**Cause:** Service name mismatch

**Fix:**
```bash
# List running containers
docker ps

# Use exact container name
SERVICE=kalakriti-core-svc-1 bash service-crash.sh
```

### "Permission denied" for db-latency.sh

**Cause:** Needs root for `tc` (traffic control)

**Fix:**
```bash
sudo bash db-latency.sh
```

### Kafka partition script fails

**Cause:** iptables not available in container

**Fix:** Use alternative method (docker network disconnect):
```bash
docker network disconnect kalakriti_default kafka
sleep 60
docker network connect kalakriti_default kafka
```

### Services don't recover after test

**Cause:** Restart policy not set

**Fix:**
```yaml
# docker-compose.yml
services:
  core-svc:
    restart: unless-stopped
```

---

## Chaos Engineering Best Practices

### Start Small

1. Run tests in dev/staging first
2. Run during business hours (when team available)
3. Start with single-service failures
4. Gradually increase complexity

### Monitor Everything

Before chaos test:
- Set up metrics dashboard
- Configure alerting
- Tail service logs
- Prepare rollback plan

### Document Findings

After each test:
- What failed?
- What surprised us?
- What needs fixing?
- What worked well?

### Regular Cadence

- Weekly: Single service crash (dev)
- Monthly: Full chaos suite (staging)
- Quarterly: Game day (production, with customer notice)

---

## Integration with Existing Tests

### Run After Load Tests

```bash
# Run load test to establish baseline
k6 run scripts/load-test/mixed-workload.js

# Run chaos tests
bash scripts/chaos/run-suite.sh

# Re-run load test to verify recovery
k6 run scripts/load-test/mixed-workload.js
```

### Pre-Production Checklist

Before deploying to production:
- [ ] All chaos tests pass
- [ ] Services recover automatically
- [ ] No data loss observed
- [ ] Circuit breakers work as expected
- [ ] Monitoring alerts fired appropriately

---

## Summary

**7 chaos scenarios** covering infrastructure failures, resource exhaustion, and service crashes  
**~10 minutes** for full suite  
**Pass criteria:** <1% error rate, <10s recovery time, 0 data loss  
**Run frequency:** Daily in staging, weekly in production

**Next steps:**
1. Run `bash scripts/chaos/run-suite.sh` in staging
2. Review logs for unexpected failures
3. Fix identified issues
4. Add to CI/CD pipeline for automated testing
5. Schedule monthly chaos game days
