#!/usr/bin/env bash
# scripts/chaos/kafka-partition.sh
# Simulate Kafka network partition

set -euo pipefail

KAFKA_CONTAINER="${KAFKA_CONTAINER:-kafka}"
DURATION="${DURATION:-60}"

echo "=== Chaos Test: Kafka Network Partition ==="
echo "Target: $KAFKA_CONTAINER"
echo "Duration: ${DURATION}s"
echo ""

# Block Kafka traffic using iptables
echo "Blocking Kafka traffic..."
docker exec "$KAFKA_CONTAINER" sh -c "
    iptables -A INPUT -p tcp --dport 9092 -j DROP
    iptables -A OUTPUT -p tcp --sport 9092 -j DROP
"

echo "✓ Network partition active"
echo "Waiting ${DURATION}s..."

sleep "$DURATION"

# Restore Kafka traffic
echo "Restoring network..."
docker exec "$KAFKA_CONTAINER" sh -c "
    iptables -D INPUT -p tcp --dport 9092 -j DROP
    iptables -D OUTPUT -p tcp --sport 9092 -j DROP
"

echo "✓ Network restored"
echo ""
echo "Expected impact:"
echo "  - Producers buffer messages locally"
echo "  - Consumers stop processing"
echo "  - Services should NOT crash (graceful degradation)"
echo "  - Messages flush after recovery"
echo ""
echo "Check for:"
echo "  - Producer send errors in logs"
echo "  - Consumer lag spike"
echo "  - Outbox pattern prevents message loss"
