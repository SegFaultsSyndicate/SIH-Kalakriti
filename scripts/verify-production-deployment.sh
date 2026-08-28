#!/usr/bin/env bash
# scripts/verify-production-deployment.sh
# Verify all production features are deployed correctly

set -euo pipefail

DATABASE_URL="${DATABASE_URL:-${POSTGRES_DSN:-postgres://kalakriti:kalakriti@localhost:5432/kalakriti?sslmode=disable}}"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

passed=0
failed=0

check() {
    local name="$1"
    local command="$2"

    echo -n "Checking $name... "
    if eval "$command" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        ((passed++))
        return 0
    else
        echo -e "${RED}✗${NC}"
        ((failed++))
        return 1
    fi
}

echo "========================================="
echo "Production Deployment Verification"
echo "========================================="
echo ""

# Database connectivity
check "Database connection" "psql \"$DATABASE_URL\" -c 'SELECT 1' -t -A"

# Migrations
echo ""
echo "Migration Status:"
check "Migration 025 (audit_log)" "psql \"$DATABASE_URL\" -c \"SELECT to_regclass('audit_log')\" -t -A | grep -q 'audit_log'"
check "Migration 026 (fraud_flags)" "psql \"$DATABASE_URL\" -c \"SELECT to_regclass('fraud_flags')\" -t -A | grep -q 'fraud_flags'"
check "Migration 027 (webhooks)" "psql \"$DATABASE_URL\" -c \"SELECT to_regclass('webhook_subscriptions')\" -t -A | grep -q 'webhook_subscriptions'"

# Verify triggers exist
echo ""
echo "Database Triggers:"
check "Audit triggers" "psql \"$DATABASE_URL\" -c \"SELECT COUNT(*) FROM pg_trigger WHERE tgname LIKE 'trigger_audit_%'\" -t -A | grep -v '^0$'"
check "Fraud triggers" "psql \"$DATABASE_URL\" -c \"SELECT COUNT(*) FROM pg_trigger WHERE tgname LIKE 'trigger_fraud_%'\" -t -A | grep -v '^0$'"
check "Webhook triggers" "psql \"$DATABASE_URL\" -c \"SELECT COUNT(*) FROM pg_trigger WHERE tgname LIKE 'trigger_webhook_%'\" -t -A | grep -v '^0$'"

# Verify indexes
echo ""
echo "Database Indexes:"
check "Audit log indexes" "psql \"$DATABASE_URL\" -c \"SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'audit_log'\" -t -A | grep -v '^0$'"
check "Fraud flags indexes" "psql \"$DATABASE_URL\" -c \"SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'fraud_flags'\" -t -A | grep -v '^0$'"
check "Webhook indexes" "psql \"$DATABASE_URL\" -c \"SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'webhook_deliveries'\" -t -A | grep -v '^0$'"

# Verify immutability rules
echo ""
echo "Data Protection:"
check "Audit log immutability" "psql \"$DATABASE_URL\" -c \"SELECT COUNT(*) FROM pg_rules WHERE tablename = 'audit_log'\" -t -A | grep '^2$'"

# Check if services are built
echo ""
echo "Service Binaries:"
check "BFF binary" "test -f bin/bff || test -f services/bff/cmd/bff/bff"
check "Webhook worker binary" "test -f bin/webhook-worker || test -f cmd/webhook-worker/main.go"

# Check if webhook worker is running (systemd)
echo ""
echo "Running Services:"
if systemctl is-active --quiet webhook-worker 2>/dev/null; then
    echo -e "Webhook worker: ${GREEN}✓ Running${NC}"
    ((passed++))
else
    echo -e "Webhook worker: ${YELLOW}⚠ Not running (or not using systemd)${NC}"
fi

# Check documentation exists
echo ""
echo "Documentation:"
check "Webhooks documentation" "test -f docs/WEBHOOKS.md"
check "i18n documentation" "test -f docs/I18N.md"
check "Chaos testing documentation" "test -f docs/CHAOS_TESTING.md"
check "Audit documentation" "test -f docs/COMPLIANCE_AUDIT.md"
check "Fraud documentation" "test -f docs/FRAUD_DETECTION.md"

# Check chaos testing scripts
echo ""
echo "Chaos Testing Scripts:"
check "Redis flush script" "test -f scripts/chaos/redis-flush.sh"
check "Service crash script" "test -f scripts/chaos/service-crash.sh"
check "CPU stress script" "test -f scripts/chaos/cpu-stress.sh"
check "Kafka partition script" "test -f scripts/chaos/kafka-partition.sh"
check "Disk fill script" "test -f scripts/chaos/disk-fill.sh"
check "DB latency script" "test -f scripts/chaos/db-latency.sh"
check "Chaos suite runner" "test -f scripts/chaos/run-suite.sh"

# Check i18n implementation
echo ""
echo "i18n Implementation:"
check "i18n package" "test -f pkg/i18n/i18n.go"
check "i18n middleware" "test -f pkg/i18n/middleware.go"
check "BFF uses i18n" "grep -q 'i18n.Middleware' services/bff/internal/bff/server.go"

# Data verification
echo ""
echo "Database Content Check:"
audit_count=$(psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM audit_log" -t -A 2>/dev/null || echo "0")
fraud_count=$(psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM fraud_flags" -t -A 2>/dev/null || echo "0")
webhook_subs=$(psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM webhook_subscriptions" -t -A 2>/dev/null || echo "0")

echo "  Audit log entries: $audit_count"
echo "  Fraud flags: $fraud_count"
echo "  Webhook subscriptions: $webhook_subs"

# Summary
echo ""
echo "========================================="
echo "Verification Summary"
echo "========================================="
echo -e "${GREEN}Passed: $passed${NC}"
if [[ $failed -gt 0 ]]; then
    echo -e "${RED}Failed: $failed${NC}"
else
    echo -e "${GREEN}Failed: $failed${NC}"
fi
echo ""

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ All checks passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Run monitoring: bash scripts/monitor-production.sh"
    echo "  2. Test webhooks: Create a test subscription"
    echo "  3. Test i18n: curl -H 'Accept-Language: hi' http://localhost:8080/api/health"
    echo "  4. Run chaos tests (when ready): bash scripts/chaos/run-suite.sh"
    exit 0
else
    echo -e "${RED}✗ Some checks failed. Review output above.${NC}"
    exit 1
fi
