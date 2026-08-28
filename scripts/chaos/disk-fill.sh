#!/usr/bin/env bash
# scripts/chaos/disk-fill.sh
# Fill disk to test out-of-space handling

set -euo pipefail

SERVICE="${SERVICE:-postgres}"
SIZE="${SIZE:-1G}"
DURATION="${DURATION:-60}"

echo "=== Chaos Test: Disk Fill ==="
echo "Target: $SERVICE"
echo "Size: $SIZE"
echo "Duration: ${DURATION}s"
echo ""

# Docker environment
if command -v docker &> /dev/null; then
    CONTAINER_ID=$(docker ps --filter "name=$SERVICE" --format "{{.ID}}" | head -n1)

    if [[ -z "$CONTAINER_ID" ]]; then
        echo "Error: Container for $SERVICE not found"
        exit 1
    fi

    echo "Filling disk in container $CONTAINER_ID..."

    # Create large file
    docker exec "$CONTAINER_ID" sh -c "fallocate -l $SIZE /tmp/chaos-fill.dat"

    echo "✓ Disk filled"
    echo "Waiting ${DURATION}s..."

    sleep "$DURATION"

    # Clean up
    docker exec "$CONTAINER_ID" rm -f /tmp/chaos-fill.dat

    echo "✓ Disk space restored"
    echo ""
    echo "Expected impact:"
    echo "  - Write operations may fail"
    echo "  - PostgreSQL may refuse new connections"
    echo "  - Services should log errors gracefully"
    echo "  - No data corruption"
    exit 0
fi

echo "Error: Docker not found"
exit 1
