# PostgreSQL Connection Pooling Configuration

**Last Updated:** 2026-09-15
**Library:** pgxpool

---

## Current Configuration

```go
// pkg/postgres/postgres.go — not pkg/database/pool.go, which doesn't exist
config, err := pgxpool.ParseConfig(dsn)
config.MaxConns = 20                          // real code default
config.MinConns = 2                           // real code default
config.MaxConnLifetime = 30 * time.Minute     // real code default
config.MaxConnIdleTime = 5 * time.Minute      // real code default
config.HealthCheckPeriod = 1 * time.Minute
```

`.env.example` ships override values of `25`/`5`/`1h`/`15m` for local dev —
those are template overrides, not "the current configuration." If you haven't
copied `.env.example` to `.env` and set `POSTGRES_MAX_CONNS`/`POSTGRES_MIN_CONNS`
explicitly, every service is running with the `20`/`2` code defaults above.

`pkg/postgres.New()`'s `AfterConnect` hook also registers pgvector's type codec
and, per `CLAUDE.md`, the `language_code` enum's base + array codecs — if you
add a future migration with another `sometype[]` column, add it to the
`registerEnumArrayTypes` call there or inserts into that column will fail with
an opaque "cannot find encode plan" error.

---

## Why These Numbers

### MaxConns = 20
7 services × 20 conns = 140 max total connections. PostgreSQL's own default
`max_connections` is 100 — **too low as-is**; the `docker-compose.yml` postgres
service should have `max_connections` raised if you run every service at full
pool size simultaneously. Increase per-service `POSTGRES_MAX_CONNS` for a
service under sustained load (watch `pg_stat_activity` for queued/waiting
connections); decrease it for a genuinely low-traffic service like insight-svc.

### MinConns = 2
Keeps a couple of connections warm to avoid cold-start latency
(connection establishment is roughly 10-20ms) without wasting idle backend
memory on services that see light, bursty traffic. Increase for a service with
a consistent baseline of concurrent requests (e.g. bff).

### MaxConnLifetime = 30 minutes
Forces periodic reconnection so a connection can't outlive a load-balancer
rotation, failover, or DNS change. Decrease if you're behind infrastructure
with a shorter session TTL.

### MaxConnIdleTime = 5 minutes
Idle connections still hold PostgreSQL backend memory. 5 minutes balances a
warm pool against wasted memory during quiet periods; decrease further in a
memory-constrained environment, increase if traffic is steady enough that
connections are rarely idle anyway.

---

## Environment Variables

Only these two are actually read (`pkg/config.Postgres`, `env:"..."` tags):

```bash
POSTGRES_MAX_CONNS=20
POSTGRES_MIN_CONNS=2
```

**Not real** — don't add these expecting them to do anything;
`POSTGRES_CONNECT_TIMEOUT` and `POSTGRES_STATEMENT_TIMEOUT` are not read
anywhere in `pkg/config` or `pkg/postgres`. `MaxConnLifetime`/`MaxConnIdleTime`
are currently hardcoded constants, not env-configurable — if you need them
tunable, that's a small addition to `pkg/config.Postgres` and
`pkg/postgres.New()`, not something you can set today.

---

## Tuning the PostgreSQL Server

`docker-compose.yml`'s `postgres` service doesn't currently set
`max_connections`/`shared_buffers`/etc. explicitly — it runs on the image's
defaults. To raise the connection ceiling:

```bash
docker compose exec postgres psql -U kalakriti -c "ALTER SYSTEM SET max_connections = 200;"
docker compose restart postgres
```

**pgBouncer is not part of this stack today** — the section below is a
proposal, not a description of anything running. There is no `pgbouncer`
service in `docker-compose.yml`.

```yaml
# Aspirational — not present in docker-compose.yml
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
    PGBOUNCER_DEFAULT_POOL_SIZE: 20
  ports:
    - "6432:6432"
```

If you add this, services would connect to `postgres:6432` instead of
`postgres:5432` via `POSTGRES_DSN`.

---

## Monitoring Connection Pool

```sql
-- Current connections per service (application_name is set per service)
SELECT application_name, count(*)
FROM pg_stat_activity
WHERE datname = 'kalakriti'
GROUP BY application_name;

-- Connection states
SELECT state, count(*) FROM pg_stat_activity WHERE datname = 'kalakriti' GROUP BY state;

-- Long-running queries holding connections
SELECT pid, usename, application_name, state, now() - query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active' AND now() - query_start > interval '5 seconds'
ORDER BY duration DESC;

-- Kill a stuck connection
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE pid = 12345;
```

If you add `pg_exporter`, `pg_stat_activity_count`, `pg_stat_database_numbackends`,
and `pg_settings_max_connections` become available to Prometheus/Grafana.

---

## Common Issues

### "sorry, too many clients already"
Total connections exceed PostgreSQL's `max_connections`. Fix: raise
`max_connections` server-side, lower a service's `POSTGRES_MAX_CONNS`, or add
pgBouncer (not currently deployed — see above).

### "connection pool exhausted"
A service needs more concurrent connections than its `MaxConns`. Check for
slow queries first, then consider raising `POSTGRES_MAX_CONNS` for that
service specifically.

### Connections not being reused
Usually a transaction that was never committed/rolled back:
```go
tx, err := pool.Begin(ctx)
if err != nil { return err }
defer tx.Rollback() // safe even after Commit()
// ... work ...
return tx.Commit()
```

---

## Load Testing

`tests/integration/loadtest_test.go` already exercises the BFF's HTTP layer
in-process — see `loadtest_results.md` at the repo root for the most recent
run's numbers. For a connection-pool-specific load test against the real
stack:

```bash
k6 run - <<'EOF'
import http from 'k6/http';
import { check, sleep } from 'k6';
export let options = {
  stages: [
    { duration: '1m', target: 50 },
    { duration: '3m', target: 50 },
    { duration: '1m', target: 100 },
    { duration: '5m', target: 100 },
    { duration: '1m', target: 0 },
  ],
};
export default function() {
  let res = http.get('http://localhost:8000/api/v1/listings');
  check(res, { 'status is 200': (r) => r.status === 200 });
  sleep(1);
}
EOF
```

Watch connections during the run:
```bash
watch -n 1 "docker compose exec postgres psql -U kalakriti -c \"SELECT count(*) FROM pg_stat_activity WHERE datname = 'kalakriti';\""
```

---

## Summary

**Real current defaults:** `MaxConns=20`, `MinConns=2`,
`MaxConnLifetime=30m`, `MaxConnIdleTime=5m` (`pkg/postgres/postgres.go`).
**Configurable via env:** only `POSTGRES_MAX_CONNS`/`POSTGRES_MIN_CONNS`.
**Not deployed:** pgBouncer — this stack talks to Postgres directly, one pool
per service.
