#!/usr/bin/env bash
# scripts/chaos/db-latency.sh
# Inject database latency using tc (traffic control)

set -euo pipefail

POSTGRES_IP="${POSTGRES_IP:-127.0.0.1}"
LATENCY="${LATENCY:-100ms}"
DURATION="${DURATION:-60}"

echo "=== Chaos Test: Database Latency ==="
echo "Target: $POSTGRES_IP"
echo "Latency: $LATENCY"
echo "Duration: ${DURATION}s"
echo ""

# Check if running with sudo
if [[ $EUID -ne 0 ]]; then
   echo "Error: This script must be run as root (use sudo)"
   exit 1
fi

# Add latency to PostgreSQL traffic
tc qdisc add dev lo root netem delay "$LATENCY"

echo "✓ Latency injected"
echo "Waiting ${DURATION}s..."

sleep "$DURATION"

# Remove latency
tc qdisc del dev lo root

echo "✓ Latency removed"
echo ""
echo "Expected impact:"
echo "  - Slower API responses"
echo "  - Circuit breakers may open"
echo "  - Connection pool may exhaust"
echo ""
echo "Check metrics for request duration spikes and error rate increases"
