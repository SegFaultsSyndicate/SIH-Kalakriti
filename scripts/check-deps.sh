#!/usr/bin/env bash
# scripts/check-deps.sh - Probes every infra dependency from the host.
# Referenced by `make check`; mirrors the "Verifying each dependency by hand"
# section of the root README exactly, just scripted with pass/fail output.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

COMPOSE=${COMPOSE:-docker compose}
POSTGRES_USER=${POSTGRES_USER:-kalakriti}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-kalakriti}
POSTGRES_DB=${POSTGRES_DB:-kalakriti}

FAILED=0

check() {
  local name="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    echo "  ✓ $name"
  else
    echo "  ✗ $name"
    FAILED=1
  fi
}

echo "Probing infra dependencies..."

check "postgres (pgvector)" bash -c \
  "$COMPOSE exec -T -e PGPASSWORD='$POSTGRES_PASSWORD' postgres \
    psql -U '$POSTGRES_USER' -d '$POSTGRES_DB' -c 'CREATE EXTENSION IF NOT EXISTS vector;' -c '\dx'"

check "redis" bash -c "$COMPOSE exec -T redis redis-cli ping | grep -q PONG"

check "kafka" bash -c \
  "$COMPOSE exec -T kafka kafka-topics --bootstrap-server localhost:9092 --list"

check "minio" bash -c "$COMPOSE exec -T minio mc ready local"

if [ "$FAILED" -ne 0 ]; then
  echo ""
  echo "One or more dependencies failed. Run '$COMPOSE ps' and '$COMPOSE logs <service>' to investigate."
  exit 1
fi

echo ""
echo "All dependencies healthy."
