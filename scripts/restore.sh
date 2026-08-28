#!/usr/bin/env bash
# scripts/restore.sh
# Restore PostgreSQL database from backup

set -euo pipefail

# Configuration
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-kalakriti}"
POSTGRES_DB="${POSTGRES_DB:-kalakriti}"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/kalakriti}"
S3_BUCKET="${S3_BUCKET:-}"

# Parse arguments
BACKUP_FILE=""
FORCE=0

usage() {
    cat << EOF
Usage: $0 [OPTIONS] <backup-file>

Restore PostgreSQL database from a backup file.

OPTIONS:
    -f, --force          Skip confirmation prompt (DANGEROUS)
    -s, --from-s3        Download from S3 bucket first
    -l, --list           List available backups and exit
    -h, --help           Show this help message

EXAMPLES:
    # List available local backups
    $0 --list

    # Restore from local file
    $0 kalakriti-20260920-030000.sql.gz

    # Download from S3 and restore
    $0 --from-s3 kalakriti-20260920-030000.sql.gz

    # Restore without confirmation (for automation)
    $0 --force kalakriti-20260920-030000.sql.gz

WARNING: This will DROP the existing database and replace it with the backup.
         All current data will be lost. Make sure you have a recent backup first.
EOF
    exit 0
}

list_backups() {
    echo "=== Local backups in ${BACKUP_DIR} ==="
    if [ -d "${BACKUP_DIR}" ]; then
        find "${BACKUP_DIR}" -name "kalakriti-*.sql.gz" -printf "%T@ %Tc %p\n" | sort -rn | cut -d' ' -f2- | head -20
    else
        echo "No backup directory found"
    fi

    if [ -n "${S3_BUCKET}" ] && command -v aws &> /dev/null; then
        echo ""
        echo "=== S3 backups in s3://${S3_BUCKET} ==="
        aws s3 ls "s3://${S3_BUCKET}/" | grep "kalakriti-.*\.sql\.gz" | tail -20
    fi
    exit 0
}

# Parse command line arguments
FROM_S3=0
while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--force)
            FORCE=1
            shift
            ;;
        -s|--from-s3)
            FROM_S3=1
            shift
            ;;
        -l|--list)
            list_backups
            ;;
        -h|--help)
            usage
            ;;
        *)
            BACKUP_FILE="$1"
            shift
            ;;
    esac
done

# Validate backup file was provided
if [ -z "${BACKUP_FILE}" ]; then
    echo "ERROR: No backup file specified" >&2
    usage
fi

# Download from S3 if requested
if [ "${FROM_S3}" -eq 1 ]; then
    if [ -z "${S3_BUCKET}" ]; then
        echo "ERROR: S3_BUCKET not configured" >&2
        exit 1
    fi
    if ! command -v aws &> /dev/null; then
        echo "ERROR: AWS CLI not available" >&2
        exit 1
    fi

    mkdir -p "${BACKUP_DIR}"
    BACKUP_PATH="${BACKUP_DIR}/${BACKUP_FILE}"

    echo "Downloading s3://${S3_BUCKET}/${BACKUP_FILE}..."
    aws s3 cp "s3://${S3_BUCKET}/${BACKUP_FILE}" "${BACKUP_PATH}"
else
    # Use local file
    if [[ "${BACKUP_FILE}" == /* ]]; then
        BACKUP_PATH="${BACKUP_FILE}"
    else
        BACKUP_PATH="${BACKUP_DIR}/${BACKUP_FILE}"
    fi
fi

# Verify backup file exists
if [ ! -f "${BACKUP_PATH}" ]; then
    echo "ERROR: Backup file not found: ${BACKUP_PATH}" >&2
    exit 1
fi

BACKUP_SIZE=$(du -h "${BACKUP_PATH}" | cut -f1)
echo "Backup file: ${BACKUP_PATH} (${BACKUP_SIZE})"

# Confirmation prompt (unless --force)
if [ "${FORCE}" -eq 0 ]; then
    echo ""
    echo "WARNING: This will DROP the database '${POSTGRES_DB}' and restore from backup."
    echo "         All current data will be PERMANENTLY LOST."
    echo ""
    read -p "Are you sure you want to continue? Type 'yes' to proceed: " confirmation

    if [ "${confirmation}" != "yes" ]; then
        echo "Restore cancelled."
        exit 0
    fi
fi

echo ""
echo "Starting restore at $(date -Iseconds)..."

# Stop services that might be using the database
if command -v docker &> /dev/null && docker compose ps &> /dev/null; then
    echo "Stopping services..."
    docker compose stop bff core-svc search-svc collab-svc insight-svc channel-svc || true
fi

# Restore using docker compose or local pg_restore
if command -v docker &> /dev/null && docker compose ps postgres &> /dev/null; then
    echo "Using docker compose exec to restore..."

    # Drop and recreate database
    docker compose exec -T postgres psql -U "${POSTGRES_USER}" -c "DROP DATABASE IF EXISTS ${POSTGRES_DB};" postgres
    docker compose exec -T postgres psql -U "${POSTGRES_USER}" -c "CREATE DATABASE ${POSTGRES_DB};" postgres

    # Restore backup
    gunzip -c "${BACKUP_PATH}" | docker compose exec -T postgres pg_restore -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" --no-owner --no-acl
elif command -v pg_restore &> /dev/null; then
    echo "Using local pg_restore..."

    # Drop and recreate database
    PGPASSWORD="${POSTGRES_PASSWORD:-}" psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -c "DROP DATABASE IF EXISTS ${POSTGRES_DB};" postgres
    PGPASSWORD="${POSTGRES_PASSWORD:-}" psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -c "CREATE DATABASE ${POSTGRES_DB};" postgres

    # Restore backup
    gunzip -c "${BACKUP_PATH}" | PGPASSWORD="${POSTGRES_PASSWORD:-}" pg_restore -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" --no-owner --no-acl
else
    echo "ERROR: Neither docker compose nor pg_restore is available" >&2
    exit 1
fi

echo ""
echo "Restore completed at $(date -Iseconds)"
echo ""
echo "Next steps:"
echo "  1. Verify data: docker compose exec postgres psql -U ${POSTGRES_USER} -d ${POSTGRES_DB}"
echo "  2. Start services: make demo-up"
echo "  3. Run smoke tests"
