# Backup & Restore

**Last Updated:** 2026-08-28  
**Status:** Ready for production use

---

## Overview

Automated daily PostgreSQL backups with S3 upload, 7-day local retention, and documented restore procedure.

**What's backed up:**
- PostgreSQL database (all tables, indexes, data)
- Compressed with gzip (~10:1 ratio)
- Uploaded to S3 with infrequent-access storage class

**What's NOT backed up:**
- MinIO object storage (media files) — replicate S3 bucket separately
- Redis cache — ephemeral, rebuilt on restart
- Kafka topics — event sourcing should replay from database

---

## Quick Start

### Daily Automated Backup

```bash
# Set up environment variables
export POSTGRES_USER=kalakriti
export POSTGRES_DB=kalakriti
export BACKUP_DIR=/var/backups/kalakriti
export S3_BUCKET=kalakriti-backups
export RETENTION_DAYS=7

# Run backup manually
./scripts/backup.sh

# Or add to cron (3:00 AM daily)
0 3 * * * /path/to/kalakriti/scripts/backup.sh >> /var/log/kalakriti-backup.log 2>&1
```

### Restore from Backup

```bash
# List available backups
./scripts/restore.sh --list

# Restore from local file (interactive confirmation)
./scripts/restore.sh kalakriti-20260920-030000.sql.gz

# Restore from S3
./scripts/restore.sh --from-s3 kalakriti-20260920-030000.sql.gz

# Automated restore (skip confirmation — dangerous!)
./scripts/restore.sh --force kalakriti-20260920-030000.sql.gz
```

---

## Backup Script (`scripts/backup.sh`)

### What It Does

1. **Dumps PostgreSQL database** using `pg_dump -Fc` (custom format, compressed)
2. **Gzips the dump** for additional compression (~10:1 ratio)
3. **Uploads to S3** (if `S3_BUCKET` is set)
4. **Cleans up old backups** (deletes files older than `RETENTION_DAYS`)

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_HOST` | `localhost` | PostgreSQL host |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_USER` | `kalakriti` | Database user |
| `POSTGRES_DB` | `kalakriti` | Database name |
| `POSTGRES_PASSWORD` | _(none)_ | Database password (only needed if not using docker compose) |
| `BACKUP_DIR` | `/var/backups/kalakriti` | Local backup directory |
| `S3_BUCKET` | _(none)_ | S3 bucket name (e.g., `kalakriti-backups`) |
| `RETENTION_DAYS` | `7` | How long to keep local backups |

### Backup File Naming

```
kalakriti-YYYYMMDD-HHMMSS.sql.gz
```

Example: `kalakriti-20260920-030015.sql.gz` (Sept 20, 2026 at 03:00:15)

### Storage Calculation

**Example database sizes:**
- Development: ~50 MB → ~5 MB compressed
- Production (1000 artisans, 10k listings): ~500 MB → ~50 MB compressed

**S3 costs (us-east-1, Infrequent Access):**
- 50 MB × 30 backups/month = 1.5 GB storage = ~$0.02/month
- 500 MB × 30 backups/month = 15 GB storage = ~$0.19/month

### Exit Codes

- `0` — Success
- `1` — Backup file empty or not created
- `1` — Docker compose or pg_dump not available

---

## Restore Script (`scripts/restore.sh`)

### What It Does

1. **Downloads from S3** (if `--from-s3` flag is set)
2. **Stops services** to prevent connections during restore
3. **Drops existing database** (destructive!)
4. **Creates fresh database**
5. **Restores backup** using `pg_restore`

### Flags

| Flag | Description |
|------|-------------|
| `-l, --list` | List available backups (local + S3) |
| `-s, --from-s3` | Download backup from S3 before restoring |
| `-f, --force` | Skip confirmation prompt (for automation) |
| `-h, --help` | Show help message |

### Safety Features

- **Interactive confirmation** — requires typing `yes` to proceed (unless `--force`)
- **Stops services first** — prevents in-flight transactions during restore
- **Reports next steps** — reminds you to verify data and restart services

### Exit Codes

- `0` — Success
- `1` — Backup file not found, S3_BUCKET not set, or restore failed

---

## Production Setup

### 1. Create S3 Bucket

```bash
# Create bucket with versioning and lifecycle policy
aws s3 mb s3://kalakriti-backups --region us-east-1

# Enable versioning (recover from accidental deletions)
aws s3api put-bucket-versioning \
  --bucket kalakriti-backups \
  --versioning-configuration Status=Enabled

# Lifecycle policy: delete backups older than 90 days
cat > lifecycle.json << 'EOF'
{
  "Rules": [{
    "Id": "DeleteOldBackups",
    "Status": "Enabled",
    "Prefix": "kalakriti-",
    "Expiration": {"Days": 90}
  }]
}
EOF
aws s3api put-bucket-lifecycle-configuration \
  --bucket kalakriti-backups \
  --lifecycle-configuration file://lifecycle.json
```

### 2. Set Up IAM Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "s3:PutObject",
      "s3:GetObject",
      "s3:ListBucket"
    ],
    "Resource": [
      "arn:aws:s3:::kalakriti-backups",
      "arn:aws:s3:::kalakriti-backups/*"
    ]
  }]
}
```

### 3. Configure Cron Job

```bash
# Add to /etc/cron.d/kalakriti-backup
0 3 * * * kalakriti /opt/kalakriti/scripts/backup.sh >> /var/log/kalakriti/backup.log 2>&1
```

**Or using systemd timer:**

```ini
# /etc/systemd/system/kalakriti-backup.service
[Unit]
Description=Kalakriti Database Backup
After=network.target

[Service]
Type=oneshot
User=kalakriti
EnvironmentFile=/opt/kalakriti/.env.backup
ExecStart=/opt/kalakriti/scripts/backup.sh
StandardOutput=append:/var/log/kalakriti/backup.log
StandardError=append:/var/log/kalakriti/backup.log
```

```ini
# /etc/systemd/system/kalakriti-backup.timer
[Unit]
Description=Kalakriti Database Backup Timer
Requires=kalakriti-backup.service

[Timer]
OnCalendar=daily
OnCalendar=03:00
Persistent=true

[Install]
WantedBy=timers.target
```

Enable the timer:
```bash
systemctl enable kalakriti-backup.timer
systemctl start kalakriti-backup.timer
systemctl status kalakriti-backup.timer
```

### 4. Environment File (`/opt/kalakriti/.env.backup`)

```bash
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=kalakriti
POSTGRES_DB=kalakriti
BACKUP_DIR=/var/backups/kalakriti
S3_BUCKET=kalakriti-backups
RETENTION_DAYS=7
AWS_REGION=us-east-1
AWS_PROFILE=kalakriti-backups
```

---

## Monitoring

### Check Backup Success

```bash
# View recent backups
ls -lh /var/backups/kalakriti/

# Check S3
aws s3 ls s3://kalakriti-backups/ | tail -5

# Verify backup file integrity
gunzip -t /var/backups/kalakriti/kalakriti-20260920-030000.sql.gz
```

### Backup Alerts

Set up CloudWatch alarms for:
1. **Backup file size = 0** — backup failed or database is empty
2. **No backup in 25 hours** — cron job failed or server down
3. **S3 upload failed** — check AWS credentials

**Example CloudWatch alarm (using S3 PutObject metric):**
```bash
aws cloudwatch put-metric-alarm \
  --alarm-name kalakriti-backup-missing \
  --alarm-description "No backup uploaded in 25 hours" \
  --metric-name NumberOfObjects \
  --namespace AWS/S3 \
  --statistic Maximum \
  --period 86400 \
  --evaluation-periods 1 \
  --threshold 1 \
  --comparison-operator LessThanThreshold
```

---

## Restore Drills

**Practice restores monthly** — backups you can't restore are useless.

### Monthly Restore Drill (first Monday, 10am)

```bash
# 1. Create test environment (docker-compose.test.yml does not exist in the
#    repo -- run this against a disposable copy of docker-compose.yml, or a
#    second project via `docker compose -p kalakriti-drill up -d`, not a
#    dedicated test compose file)
docker compose -f docker-compose.yml -p kalakriti-drill up -d

# 2. Restore latest backup
./scripts/restore.sh --force $(ls -t /var/backups/kalakriti/*.sql.gz | head -1)

# 3. Verify data
docker compose exec postgres psql -U kalakriti -d kalakriti -c "SELECT COUNT(*) FROM artisan;"
docker compose exec postgres psql -U kalakriti -d kalakriti -c "SELECT COUNT(*) FROM listing;"

# 4. Smoke test API (bff's GET /listings responds {"listings": [...]}, not {"items": [...]})
curl http://localhost:8000/api/v1/listings | jq '.listings | length'

# 5. Document results
echo "$(date -Iseconds): Restore drill successful" >> /var/log/kalakriti/restore-drills.log

# 6. Clean up test environment
docker compose -f docker-compose.test.yml down -v
```

Add to calendar: **First Monday of every month, 10:00 AM**

---

## Disaster Recovery Scenarios

### Scenario 1: Database Corruption

**Symptoms:** Queries failing, data inconsistent, PostgreSQL won't start

**Recovery:**
1. Stop all services: `docker compose stop`
2. List backups: `./scripts/restore.sh --list`
3. Restore most recent good backup: `./scripts/restore.sh kalakriti-YYYYMMDD-HHMMSS.sql.gz`
4. Start services: `make demo-up`
5. Verify: Run smoke tests

**Downtime:** ~5-10 minutes (depends on backup size)

### Scenario 2: Accidental Data Deletion

**Symptoms:** User accidentally deleted 100 listings, needs rollback

**Recovery:**
1. **Don't panic** — S3 versioning keeps deleted backups
2. Restore to staging environment first: `./scripts/restore.sh --from-s3 kalakriti-YYYYMMDD-HHMMSS.sql.gz`
3. Export missing data: `pg_dump -t listing -U kalakriti kalakriti | gzip > missing-listings.sql.gz`
4. Import to production: `gunzip -c missing-listings.sql.gz | psql -U kalakriti kalakriti`

**Downtime:** 0 (staging restore)

### Scenario 3: Complete Server Loss

**Symptoms:** Server died, hardware failure, need to rebuild on new instance

**Recovery:**
1. Provision new server, install dependencies
2. Clone repo: `git clone https://github.com/ZoroNewbie00/kalakriti.git`
3. Start infrastructure: `docker compose up -d postgres redis kafka`
4. Restore from S3: `./scripts/restore.sh --from-s3 --force kalakriti-YYYYMMDD-HHMMSS.sql.gz`
5. Start services: `make demo-up`
6. Update DNS to point to new server

**Downtime:** ~30-60 minutes (new server setup + DNS propagation)

---

## Backup Testing Checklist

Before production launch, verify:

- [ ] Backup script runs without errors
- [ ] Backup file is created and non-zero size
- [ ] S3 upload succeeds
- [ ] Old backups are cleaned up (check after 8 days)
- [ ] Restore script can restore from local file
- [ ] Restore script can download from S3 and restore
- [ ] Restored database matches original (row counts, spot checks)
- [ ] Services start successfully after restore
- [ ] API smoke tests pass after restore

---

## Troubleshooting

### Backup file is empty

**Cause:** pg_dump failed (permission denied, connection refused, out of disk space)

**Fix:**
```bash
# Check PostgreSQL is running
docker compose ps postgres

# Check disk space
df -h /var/backups

# Test pg_dump manually
docker compose exec postgres pg_dump -U kalakriti kalakriti | head -20
```

### S3 upload fails with "AccessDenied"

**Cause:** AWS credentials not configured or IAM policy missing

**Fix:**
```bash
# Verify AWS credentials
aws sts get-caller-identity

# Test S3 write access
echo "test" | aws s3 cp - s3://kalakriti-backups/test.txt
aws s3 rm s3://kalakriti-backups/test.txt
```

### Restore fails with "database is being accessed by other users"

**Cause:** Services still connected to database

**Fix:**
```bash
# Force disconnect all users
docker compose exec postgres psql -U kalakriti -c \
  "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='kalakriti';" postgres

# Then retry restore
./scripts/restore.sh kalakriti-20260920-030000.sql.gz
```

### Restore succeeds but data is old

**Cause:** Restored wrong backup file, or backup was from before the data was added

**Fix:**
```bash
# List all backups with timestamps
ls -lh /var/backups/kalakriti/

# Restore from a more recent backup
./scripts/restore.sh kalakriti-20260921-030000.sql.gz
```

---

## Summary

**Backups run:** Daily at 3:00 AM  
**Retention:** 7 days local, 90 days S3  
**Storage cost:** ~$0.02-0.20/month  
**Restore time:** 5-10 minutes  
**Practice drills:** First Monday of every month

**Before production launch:**
1. Create S3 bucket with lifecycle policy
2. Set up IAM permissions
3. Configure cron job or systemd timer
4. Run first backup manually and verify S3 upload
5. Schedule first restore drill

**After launch:**
1. Monitor backup logs daily for first week
2. Set up CloudWatch alarms
3. Document first real restore (when it happens)
4. Update retention policy based on compliance requirements
