#!/usr/bin/env bash
# scripts/chaos/cpu-stress.sh
# Stress CPU on target service to test performance degradation

set -euo pipefail

SERVICE="${SERVICE:-core-svc}"
DURATION="${DURATION:-60}"
CORES="${CORES:-2}"

echo "=== Chaos Test: CPU Stress ==="
echo "Target: $SERVICE"
echo "Cores: $CORES"
echo "Duration: ${DURATION}s"
echo ""

# Docker environment
if command -v docker &> /dev/null; then
    CONTAINER_ID=$(docker ps --filter "name=$SERVICE" --format "{{.ID}}" | head -n1)

    if [[ -z "$CONTAINER_ID" ]]; then
        echo "Error: Container for $SERVICE not found"
        exit 1
    fi

    echo "Stressing container $CONTAINER_ID..."

    # Install stress-ng if not present, then run
    docker exec "$CONTAINER_ID" sh -c "
        if ! command -v stress-ng &> /dev/null; then
            apt-get update -qq && apt-get install -y -qq stress-ng
        fi
        stress-ng --cpu $CORES --timeout ${DURATION}s
    " &

    STRESS_PID=$!

    echo "✓ CPU stress running (PID: $STRESS_PID)"
    echo "Waiting ${DURATION}s..."

    wait "$STRESS_PID"

    echo "✓ CPU stress completed"
    echo ""
    echo "Expected impact:"
    echo "  - Slower request processing"
    echo "  - Increased p95/p99 latency"
    echo "  - Load balancer may shift traffic to other replicas"
    echo "  - No errors (requests just slower)"
    exit 0
fi

echo "Error: Docker not found"
exit 1
