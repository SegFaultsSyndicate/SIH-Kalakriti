#!/usr/bin/env bash
# scripts/chaos/redis-flush.sh
# Clear Redis cache to test cache miss handling

set -euo pipefail

REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"

echo "=== Chaos Test: Redis Cache Flush ==="
echo "Target: $REDIS_HOST:$REDIS_PORT"
echo ""

# Flush all Redis keys
redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" FLUSHALL

echo "✓ Redis flushed"
echo ""
echo "Expected impact:"
echo "  - All cache misses hit database"
echo "  - API latency spike (~2-3x slower)"
echo "  - Database CPU/connections increase"
echo "  - Cache warms up over ~5-10 minutes"
echo ""
echo "Monitor:"
echo "  - API p95/p99 latency"
echo "  - Database connection count"
echo "  - Redis hit rate (should recover to ~80%+)"
