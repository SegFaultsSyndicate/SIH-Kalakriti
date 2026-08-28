#!/usr/bin/env bash
# scripts/chaos/run-suite.sh
# Run all chaos tests in sequence with monitoring

set -euo pipefail

CHAOS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/tmp/chaos-$(date +%Y%m%d-%H%M%S).log"

echo "=== Chaos Testing Suite ===" | tee "$LOG_FILE"
echo "Started: $(date)" | tee -a "$LOG_FILE"
echo "Log: $LOG_FILE" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"

# Array of tests to run
TESTS=(
    "redis-flush.sh"
    "service-crash.sh"
    "cpu-stress.sh"
    "kafka-partition.sh"
)

# Run each test
for test in "${TESTS[@]}"; do
    echo "========================================" | tee -a "$LOG_FILE"
    echo "Running: $test" | tee -a "$LOG_FILE"
    echo "========================================" | tee -a "$LOG_FILE"
    echo "" | tee -a "$LOG_FILE"

    if bash "$CHAOS_DIR/$test" 2>&1 | tee -a "$LOG_FILE"; then
        echo "✓ $test completed" | tee -a "$LOG_FILE"
    else
        echo "✗ $test failed" | tee -a "$LOG_FILE"
    fi

    echo "" | tee -a "$LOG_FILE"
    echo "Cooling down for 30s..." | tee -a "$LOG_FILE"
    sleep 30
    echo "" | tee -a "$LOG_FILE"
done

echo "========================================" | tee -a "$LOG_FILE"
echo "Chaos Testing Complete" | tee -a "$LOG_FILE"
echo "Completed: $(date)" | tee -a "$LOG_FILE"
echo "========================================" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"
echo "Review:" | tee -a "$LOG_FILE"
echo "  - Check service logs for errors" | tee -a "$LOG_FILE"
echo "  - Verify all services recovered" | tee -a "$LOG_FILE"
echo "  - Review metrics for anomalies" | tee -a "$LOG_FILE"
echo "  - Confirm no data loss" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"
echo "Full log: $LOG_FILE" | tee -a "$LOG_FILE"
