#!/usr/bin/env bash
# scripts/backup.sh
# Daily PostgreSQL backup with S3 upload

set -euo pipefail

# Configuration from environment or defaults
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-kalakriti}"
POSTGRES_DB="${POSTGRES_DB:-kalakriti}"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/kalakriti}"
S3_BUCKET="${S3_BUCKET:-}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

# Generate timestamped filename
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="kalakriti-${TIMESTAMP}.sql.gz"
BACKUP_PATH="${BACKUP_DIR}/${BACKUP_FILE}"

echo "Starting backup at $(date -Iseconds)"
echo "Database: ${POSTGRES_USER}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}"
echo "Output: ${BACKUP_PATH}"

# Create backup directory if it doesn't exist
mkdir -p "${BACKUP_DIR}"

# Run pg_dump through docker compose (or directly if pg_dump is available)
if command -v docker &> /dev/null && docker compose ps postgres &> /dev/null; then
    echo "Using docker compose exec to dump database..."
    docker compose exec -T postgres pg_dump -U "${POSTGRES_USER}" -Fc "${POSTGRES_DB}" | gzip > "${BACKUP_PATH}"
elif command -v pg_dump &> /dev/null; then
    echo "Using local pg_dump..."
    PGPASSWORD="${POSTGRES_PASSWORD:-}" pg_dump -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -Fc "${POSTGRES_DB}" | gzip > "${BACKUP_PATH}"
else
    echo "ERROR: Neither docker compose nor pg_dump is available" >&2
    exit 1
fi

# Verify backup file was created and is not empty
if [ ! -s "${BACKUP_PATH}" ]; then
    echo "ERROR: Backup file is empty or was not created" >&2
    exit 1
fi

BACKUP_SIZE=$(du -h "${BACKUP_PATH}" | cut -f1)
echo "Backup completed: ${BACKUP_SIZE}"

# Upload to S3 if configured
if [ -n "${S3_BUCKET}" ]; then
    echo "Uploading to s3://${S3_BUCKET}/${BACKUP_FILE}..."
    if command -v aws &> /dev/null; then
        aws s3 cp "${BACKUP_PATH}" "s3://${S3_BUCKET}/${BACKUP_FILE}" --storage-class STANDARD_IA
        echo "Upload successful"
    else
        echo "WARNING: AWS CLI not available, skipping S3 upload" >&2
    fi
fi

# Clean up old local backups
if [ "${RETENTION_DAYS}" -gt 0 ]; then
    echo "Cleaning up backups older than ${RETENTION_DAYS} days..."
    find "${BACKUP_DIR}" -name "kalakriti-*.sql.gz" -mtime +${RETENTION_DAYS} -delete
    echo "Cleanup complete"
fi

echo "Backup finished at $(date -Iseconds)"
echo "Latest backup: ${BACKUP_PATH}"
