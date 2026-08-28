#!/usr/bin/env bash
# scripts/chaos/service-crash.sh
# Kill random service pod/container to test recovery

set -euo pipefail

NAMESPACE="${NAMESPACE:-default}"
SERVICE="${SERVICE:-random}"

echo "=== Chaos Test: Service Crash ==="

# List available services
SERVICES=(
    "core-svc"
    "collab-svc"
    "search-svc"
    "channel-svc"
    "insight-svc"
    "bff"
)

if [[ "$SERVICE" == "random" ]]; then
    SERVICE="${SERVICES[$RANDOM % ${#SERVICES[@]}]}"
fi

echo "Target: $SERVICE"
echo ""

# Docker Compose environment
if command -v docker-compose &> /dev/null; then
    echo "Using docker-compose..."
    docker-compose kill "$SERVICE"
    echo "✓ Service killed"
    echo ""
    echo "Watch recovery:"
    echo "  docker-compose ps"
    echo "  docker-compose logs -f $SERVICE"
    echo ""
    echo "Expected impact:"
    echo "  - Service restarts automatically (restart: unless-stopped)"
    echo "  - Dependent services may see errors during restart window (~5-10s)"
    echo "  - Circuit breakers should prevent cascading failures"
    exit 0
fi

# Kubernetes environment
if command -v kubectl &> /dev/null; then
    echo "Using kubectl..."
    POD=$(kubectl get pods -n "$NAMESPACE" -l app="$SERVICE" -o jsonpath='{.items[0].metadata.name}')

    if [[ -z "$POD" ]]; then
        echo "Error: No pod found for service $SERVICE"
        exit 1
    fi

    echo "Killing pod: $POD"
    kubectl delete pod -n "$NAMESPACE" "$POD"

    echo "✓ Pod deleted"
    echo ""
    echo "Watch recovery:"
    echo "  kubectl get pods -n $NAMESPACE -l app=$SERVICE -w"
    echo ""
    echo "Expected impact:"
    echo "  - Kubernetes recreates pod automatically"
    echo "  - Load balancer routes traffic to healthy replicas"
    echo "  - <1s downtime if multiple replicas, ~5s if single replica"
    exit 0
fi

echo "Error: Neither docker-compose nor kubectl found"
exit 1
