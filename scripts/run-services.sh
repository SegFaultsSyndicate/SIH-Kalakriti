#!/usr/bin/env bash
set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

# Prepare clean env without CRLF
tr -d '\r' < "$REPO_ROOT/.env" > /home/zoro/.kalakriti.env
set -a
source /home/zoro/.kalakriti.env
set +a

mkdir -p "$REPO_ROOT/logs"

# Ensure infrastructure containers are running
docker start kalakriti-postgres-1 kalakriti-redis-1 kalakriti-minio-1 2>/dev/null || true

# Stop existing processes
pkill -f "services/core-svc/core-svc" 2>/dev/null || true
pkill -f "services/bff/bff" 2>/dev/null || true

# Start core-svc
echo "Starting core-svc..."
cd "$REPO_ROOT/services/core-svc"
setsid ./core-svc >> "$REPO_ROOT/logs/core-svc.log" 2>&1 </dev/null &
sleep 2

# Start bff on port 8000
echo "Starting bff on port 8000..."
cd "$REPO_ROOT/services/bff"
export ADDR=:8000
setsid ./bff >> "$REPO_ROOT/logs/bff.log" 2>&1 </dev/null &
sleep 2

echo "Testing BFF health..."
STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/api/v1/crafts || true)
if [ "$STATUS" = "200" ]; then
    echo "✓ BFF is healthy and running on http://localhost:8000 (status $STATUS)"
else
    echo "Warning: BFF returned status $STATUS"
fi
