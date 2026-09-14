#!/usr/bin/env bash
# scripts/run-backend.sh - Starts all Kalakriti backend microservices
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

echo "🚀 Starting Kalakriti Backend Microservices..."

PIDS=()

cleanup() {
  echo ""
  echo "🛑 Stopping all backend services..."
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
  echo "✓ All services stopped."
}

trap cleanup SIGINT SIGTERM EXIT

./bin/core-svc &
PID=$!
PIDS+=($PID)
echo "  ✓ core-svc running (PID $PID)"

./bin/search-svc &
PID=$!
PIDS+=($PID)
echo "  ✓ search-svc running (PID $PID)"

./bin/collab-svc &
PID=$!
PIDS+=($PID)
echo "  ✓ collab-svc running (PID $PID)"

./bin/channel-svc &
PID=$!
PIDS+=($PID)
echo "  ✓ channel-svc running (PID $PID)"

./bin/insight-svc &
PID=$!
PIDS+=($PID)
echo "  ✓ insight-svc running (PID $PID)"

sleep 1

./bin/bff &
PID=$!
PIDS+=($PID)
echo "  ✓ bff running (PID $PID) on http://localhost:8000"

echo ""
echo "🎉 All backend services are active!"
echo "   BFF REST API: http://localhost:8000"
echo "   Press Ctrl+C to stop all services."
echo ""

wait
