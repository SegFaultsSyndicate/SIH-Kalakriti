#!/usr/bin/env bash
# scripts/setup-production.sh
# Complete setup script for production deployment

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== Kalakriti Production Setup ==="
echo ""

# ============================================================================
# Step 1: Environment Setup
# ============================================================================
echo "Step 1: Environment Setup"

if [[ ! -f "$PROJECT_ROOT/.env" ]]; then
    echo "Creating .env from .env.example..."
    cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env"
    echo "⚠️  Edit .env and set production values (database password, secrets, etc.)"
    echo ""
fi

# ============================================================================
# Step 2: Database Migrations
# ============================================================================
echo "Step 2: Database Migrations"

if ! command -v goose &> /dev/null; then
    echo "Installing goose..."
    go install github.com/pressly/goose/v3/cmd/goose@latest
fi

# Load DATABASE_URL from .env
if [[ -f "$PROJECT_ROOT/.env" ]]; then
    export $(grep -v '^#' "$PROJECT_ROOT/.env" | grep POSTGRES_DSN | xargs)
    DATABASE_URL="${POSTGRES_DSN}"
fi

if [[ -z "${DATABASE_URL:-}" ]]; then
    echo "⚠️  DATABASE_URL not set. Using default..."
    DATABASE_URL="postgres://kalakriti:kalakriti@localhost:5432/kalakriti?sslmode=disable"
fi

echo "Running migrations..."
cd "$PROJECT_ROOT/migrations"
goose postgres "$DATABASE_URL" up

echo "✓ Migrations complete"
echo ""

# ============================================================================
# Step 3: Build Services
# ============================================================================
echo "Step 3: Build Services"

SERVICES=(
    "bff"
    "core-svc"
    "collab-svc"
    "search-svc"
    "channel-svc"
    "insight-svc"
)

for svc in "${SERVICES[@]}"; do
    echo "Building $svc..."
    cd "$PROJECT_ROOT/services/$svc"
    go build -o "../../bin/$svc" "./cmd/$svc"
done

# Build webhook worker
echo "Building webhook-worker..."
cat > "$PROJECT_ROOT/cmd/webhook-worker/main.go" <<'EOF'
package main

import (
    "context"
    "database/sql"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"

    _ "github.com/lib/pq"
    "github.com/ZoroNewbie00/kalakriti/pkg/webhook"
)

func main() {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        slog.Error("DATABASE_URL not set")
        os.Exit(1)
    }

    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        slog.Error("failed to connect to database", "error", err)
        os.Exit(1)
    }
    defer db.Close()

    if err := db.Ping(); err != nil {
        slog.Error("database ping failed", "error", err)
        os.Exit(1)
    }

    manager := webhook.NewManager(db)
    worker := webhook.NewWorker(manager)

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    slog.Info("webhook worker started", "poll_interval", "5s")
    worker.Run(ctx, 5*time.Second)
}
EOF

cd "$PROJECT_ROOT"
go build -o "./bin/webhook-worker" "./cmd/webhook-worker"

echo "✓ All services built"
echo ""

# ============================================================================
# Step 4: Deploy Webhook Worker
# ============================================================================
echo "Step 4: Deploy Webhook Worker"

cat > "$PROJECT_ROOT/deployments/webhook-worker.service" <<EOF
[Unit]
Description=Kalakriti Webhook Worker
After=network.target postgresql.service

[Service]
Type=simple
User=kalakriti
WorkingDirectory=$PROJECT_ROOT
EnvironmentFile=$PROJECT_ROOT/.env
ExecStart=$PROJECT_ROOT/bin/webhook-worker
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

echo "Systemd service file created: deployments/webhook-worker.service"
echo ""
echo "To install:"
echo "  sudo cp deployments/webhook-worker.service /etc/systemd/system/"
echo "  sudo systemctl daemon-reload"
echo "  sudo systemctl enable webhook-worker"
echo "  sudo systemctl start webhook-worker"
echo ""

# ============================================================================
# Step 5: Add i18n Middleware to BFF
# ============================================================================
echo "Step 5: Add i18n Middleware to BFF"

BFF_MAIN="$PROJECT_ROOT/services/bff/cmd/bff/main.go"

if grep -q "i18n.Middleware" "$BFF_MAIN"; then
    echo "✓ i18n middleware already added to BFF"
else
    echo "⚠️  Manual step required: Add i18n.Middleware to BFF router"
    echo ""
    echo "Edit: $BFF_MAIN"
    echo ""
    echo "Add import:"
    echo '  "github.com/ZoroNewbie00/kalakriti/pkg/i18n"'
    echo ""
    echo "Add middleware (before route handlers):"
    echo "  r.Use(i18n.Middleware)"
fi
echo ""

# ============================================================================
# Step 6: Monitoring Setup
# ============================================================================
echo "Step 6: Monitoring Setup"

cat > "$PROJECT_ROOT/scripts/monitor-production.sh" <<'MONITOR_EOF'
#!/usr/bin/env bash
# scripts/monitor-production.sh
# Production monitoring queries

set -euo pipefail

DATABASE_URL="${DATABASE_URL:-postgres://kalakriti:kalakriti@localhost:5432/kalakriti?sslmode=disable}"

echo "=== Production Monitoring Dashboard ==="
echo ""

# Webhook delivery success rate
echo "Webhook Delivery (last 24h):"
psql "$DATABASE_URL" -c "
SELECT
    COUNT(*) FILTER (WHERE status = 'succeeded') AS succeeded,
    COUNT(*) FILTER (WHERE status = 'failed') AS failed,
    COUNT(*) FILTER (WHERE status = 'pending') AS pending,
    ROUND(100.0 * COUNT(*) FILTER (WHERE status = 'succeeded') / NULLIF(COUNT(*), 0), 2) AS success_rate_pct
FROM webhook_deliveries
WHERE created_at > NOW() - INTERVAL '24 hours';
"
echo ""

# Fraud flags
echo "Fraud Flags:"
psql "$DATABASE_URL" -c "
SELECT
    COUNT(*) FILTER (WHERE status IN ('pending', 'reviewing')) AS pending,
    COUNT(*) FILTER (WHERE status IN ('pending', 'reviewing') AND severity IN ('high', 'critical')) AS high_severity,
    COUNT(*) FILTER (WHERE status = 'resolved_fraud') AS confirmed_fraud
FROM fraud_flags;
"
echo ""

# Audit log growth
echo "Audit Log (last 24h):"
psql "$DATABASE_URL" -c "
SELECT
    action,
    COUNT(*) AS count
FROM audit_log
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY action
ORDER BY count DESC
LIMIT 10;
"
echo ""

# Database health
echo "Database Connections:"
psql "$DATABASE_URL" -c "
SELECT
    COUNT(*) AS total_connections,
    COUNT(*) FILTER (WHERE state = 'active') AS active,
    COUNT(*) FILTER (WHERE state = 'idle') AS idle
FROM pg_stat_activity
WHERE datname = 'kalakriti';
"
echo ""

# Webhook worker health
echo "Webhook Worker Status:"
if systemctl is-active --quiet webhook-worker 2>/dev/null; then
    echo "✓ Running"
    systemctl status webhook-worker --no-pager | grep -E "(Active|Memory|CPU)"
else
    echo "✗ Not running (or not using systemd)"
fi
echo ""

echo "=== End Monitoring Dashboard ==="
MONITOR_EOF

chmod +x "$PROJECT_ROOT/scripts/monitor-production.sh"

echo "Created monitoring script: scripts/monitor-production.sh"
echo ""
echo "Run with: bash scripts/monitor-production.sh"
echo ""

# ============================================================================
# Summary
# ============================================================================
echo "========================================="
echo "✓ Production Setup Complete"
echo "========================================="
echo ""
echo "What was done:"
echo "  1. ✓ Database migrations (025, 026, 027)"
echo "  2. ✓ Webhook worker built"
echo "  3. ✓ Systemd service file created"
echo "  4. ⚠️  i18n middleware (manual step)"
echo "  5. ✓ Monitoring script created"
echo ""
echo "Next manual steps:"
echo "  1. Install webhook worker service (see Step 4 output above)"
echo "  2. Add i18n.Middleware to BFF (see Step 5 output above)"
echo "  3. Run monitoring: bash scripts/monitor-production.sh"
echo "  4. Run chaos tests (when ready): bash scripts/chaos/run-suite.sh"
echo ""
echo "Monitoring:"
echo "  - Webhook deliveries: SELECT * FROM webhook_deliveries ORDER BY created_at DESC LIMIT 10;"
echo "  - Fraud flags: SELECT * FROM fraud_flags WHERE status = 'pending';"
echo "  - Audit log: SELECT COUNT(*) FROM audit_log WHERE timestamp > NOW() - INTERVAL '1 day';"
echo ""
