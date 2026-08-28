# PostgreSQL Connection Pooling Configuration

**Last Updated:** 2026-08-28  
**Library:** pgxpool (PostgreSQL driver for Go)

---

## Current Configuration

Each service uses pgxpool with these defaults:

```go
// pkg/database/pool.go
config, err := pgxpool.ParseConfig(dsn)
config.MaxConns = 25              // Maximum connections per service
config.MinConns = 5               // Minimum idle connections
config.MaxConnLifetime = 1 * time.Hour
config.MaxConnIdleTime = 15 * time.Minute
config.HealthCheckPeriod = 1 * time.Minute
```

---

## Why These Numbers?

### MaxConns = 25
**Reasoning:**
- 6 services × 25 conns = 150 total max connections
- PostgreSQL default `max_connections` = 100 (too low!)
- Recommended: Set PostgreSQL `max_connections` = 200
- Leaves 50 connections for:
  - Admin tools (psql, migrations)
  - Monitoring (Prometheus exporters)
  - Future services

**When to increase:**
- Service under heavy load (connection pool exhausted errors)
- Many slow queries (connections held longer)
- Check `pg_stat_activity` — if you see waiting connections

**When to decrease:**
- Low traffic service (e.g., insight-svc might only need 10)
- Too many idle connections wasting memory

### MinConns = 5
**Reasoning:**
- Keeps 5 connections warm (no cold-start latency)
- Connection establishment takes ~10-20ms
- Under light load, services don't fight for connections

**When to increase:**
- Consistent baseline traffic (e.g., BFF always has 10+ concurrent requests)

**When to decrease:**
- Service rarely used (e.g., admin tools) — set to 2

### MaxConnLifetime = 1 hour
**Reasoning:**
- Prevents stale connections (load balancer rotations, failover, etc.)
- Forces periodic reconnection to pick up DNS/IP changes
- Long enough to avoid thrashing

**When to decrease:**
- Behind a load balancer with shorter session TTL
- Frequent database maintenance windows

### MaxConnIdleTime = 15 minutes
**Reasoning:**
- Idle connections consume PostgreSQL backend memory (~10MB each)
- 15min balances warm pool vs resource waste
- After 15min idle, connection closes and MinConns maintained

**When to decrease:**
- Memory-constrained environments
- Spiky traffic patterns (burst → idle → burst)

**When to increase:**
- Steady traffic, rarely idle
- Connection establishment overhead matters

---

## Environment Variables

Set in `.env` (already in `.env.example`):

```bash
# Per-service overrides (optional)
POSTGRES_MAX_CONNS=25
POSTGRES_MIN_CONNS=5
POSTGRES_MAX_CONN_LIFETIME=1h
POSTGRES_MAX_CONN_IDLE_TIME=15m

# Connection timeout
POSTGRES_CONNECT_TIMEOUT=5s

# Query timeout (prevents long-running queries from holding connections)
POSTGRES_STATEMENT_TIMEOUT=30s
```

---

## Tuning PostgreSQL Server

Edit `postgresql.conf` or set in `docker-compose.yml`:

```yaml
postgres:
  environment:
    # Connection limits
    POSTGRES_MAX_CONNECTIONS: "200"
    
    # Memory for connections
    POSTGRES_SHARED_BUFFERS: "256MB"  # 25% of RAM
    POSTGRES_WORK_MEM: "4MB"          # Per-connection sort/hash
    POSTGRES_MAINTENANCE_WORK_MEM: "64MB"
    
    # Connection pooling at server level (optional, but recommended)
    # pgBouncer can sit between services and PostgreSQL
    # PGBOUNCER_POOL_MODE: "transaction"  # Most efficient
```

**Recommended pgBouncer setup:**
```yaml
pgbouncer:
  image: pgbouncer/pgbouncer:latest
  environment:
    DATABASES_HOST: postgres
    DATABASES_PORT: 5432
    DATABASES_USER: kalakriti
    DATABASES_PASSWORD: kalakriti
    DATABASES_DBNAME: kalakriti
    PGBOUNCER_POOL_MODE: transaction
    PGBOUNCER_MAX_CLIENT_CONN: 1000
    PGBOUNCER_DEFAULT_POOL_SIZE: 25
  ports:
    - "6432:6432"
```

Then services connect to pgBouncer (port 6432) instead of PostgreSQL directly.

---

## Monitoring Connection Pool

### In Application Logs
Each service logs pool stats every minute:
```
level=info service=core-svc pool_acquire_count=1234 pool_total_conns=25 pool_idle_conns=18
```

### PostgreSQL Queries

**Current connections per service:**
```sql
SELECT application_name, count(*) 
FROM pg_stat_activity 
WHERE datname = 'kalakriti' 
GROUP BY application_name;
```

**Connection states:**
```sql
SELECT state, count(*) 
FROM pg_stat_activity 
WHERE datname = 'kalakriti' 
GROUP BY state;
```

**Long-running queries holding connections:**
```sql
SELECT pid, usename, application_name, state, 
       now() - query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active' 
  AND now() - query_start > interval '5 seconds'
ORDER BY duration DESC;
```

**Kill a stuck connection:**
```sql
SELECT pg_terminate_backend(pid) 
FROM pg_stat_activity 
WHERE pid = 12345;
```

### Prometheus Metrics

If you enable `pg_exporter`:
- `pg_stat_activity_count` — active connections
- `pg_stat_database_numbackends` — connections per database
- `pg_settings_max_connections` — server limit

**Alert when nearing limit:**
```yaml
# prometheus alerts
- alert: PostgreSQLConnectionsHigh
  expr: pg_stat_database_numbackends / pg_settings_max_connections > 0.8
  for: 5m
  annotations:
    summary: "PostgreSQL connections at 80% capacity"
```

---

## Common Issues

### "sorry, too many clients already"
**Cause:** Total connections exceed PostgreSQL `max_connections`.

**Fix:**
1. Increase `max_connections` in PostgreSQL config
2. Or decrease `MaxConns` in services
3. Or add pgBouncer connection pooler

**Quick fix:**
```bash
# In postgres container
docker compose exec postgres psql -U kalakriti -c "ALTER SYSTEM SET max_connections = 200;"
docker compose restart postgres
```

### "connection pool exhausted"
**Cause:** Service needs more than `MaxConns` concurrent connections.

**Fix:**
1. Check for slow queries — optimize them first
2. Increase `MaxConns` for that service
3. Check for connection leaks (connections not returned to pool)

**Debug:**
```go
// Add to service code
stats := pool.Stat()
log.Printf("pool: acquired=%d idle=%d max=%d", 
    stats.AcquiredConns(), stats.IdleConns(), stats.MaxConns())
```

### Connections not being reused
**Cause:** Transactions not committed/rolled back.

**Fix:**
Ensure every query using `pool.Begin()` has matching `Commit()` or `Rollback()`:
```go
tx, err := pool.Begin(ctx)
if err != nil { return err }
defer tx.Rollback()  // Safe to call even after Commit()

// ... do work ...

return tx.Commit()
```

### Memory usage growing over time
**Cause:** Too many idle connections or connection leaks.

**Fix:**
1. Decrease `MaxConnIdleTime` to close idle connections faster
2. Check for leaked connections (never returned to pool)
3. Monitor `pg_stat_activity` for orphaned connections

---

## Load Testing

Test connection pool under load:

```bash
# Install k6
brew install k6  # or download from k6.io

# Load test script
cat > load-test.js << 'EOF'
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '1m', target: 50 },   // Ramp to 50 users
    { duration: '3m', target: 50 },   // Stay at 50
    { duration: '1m', target: 100 },  // Ramp to 100
    { duration: '5m', target: 100 },  // Stay at 100
    { duration: '1m', target: 0 },    // Ramp down
  ],
};

export default function() {
  let res = http.get('http://localhost:8000/api/v1/listings?status=published');
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  sleep(1);
}
EOF

# Run load test
k6 run load-test.js

# Watch PostgreSQL connections during test
watch -n 1 'docker compose exec postgres psql -U kalakriti -c "SELECT count(*) FROM pg_stat_activity WHERE datname = '\''kalakriti'\'';"'
```

**What to watch:**
- Connection count should plateau at `MaxConns × active services`
- No "too many clients" errors
- Response times stay under 500ms
- No connection pool exhausted errors

---

## Per-Service Recommendations

Based on expected load:

| Service | MaxConns | MinConns | Reasoning |
|---------|----------|----------|-----------|
| **bff** | 50 | 10 | Highest traffic (all REST requests) |
| **core-svc** | 30 | 8 | High (catalog, media, identity) |
| **search-svc** | 25 | 5 | Medium (vector queries are expensive) |
| **collab-svc** | 20 | 5 | Medium (bulk orders) |
| **insight-svc** | 10 | 2 | Low (statement generation) |
| **channel-svc** | 15 | 3 | Low-medium (notifications) |

**Total max:** 50+30+25+20+10+15 = 150 connections  
**PostgreSQL limit:** 200 (50 headroom)

---

## Summary

**Current settings:** 25 max / 5 min per service (good defaults)  
**PostgreSQL limit:** Must be ≥200 for 6 services  
**Next step:** Monitor under load, tune per-service  
**Future:** Add pgBouncer for 1000+ concurrent users

**Apply settings:**
1. Copy `.env.example` values to `.env`
2. Restart services: `make demo-reset`
3. Monitor: `docker compose exec postgres psql -U kalakriti -c "SELECT * FROM pg_stat_activity;"`
