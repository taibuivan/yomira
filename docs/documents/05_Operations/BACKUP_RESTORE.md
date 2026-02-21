# Backup & Restore Guide — Yomira

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22

---

## Table of Contents

1. [Backup Strategy Overview](#1-backup-strategy-overview)
2. [PostgreSQL Backups](#2-postgresql-backups)
3. [Object Storage Backups](#3-object-storage-backups)
4. [Redis](#4-redis)
5. [Automated Backup Script](#5-automated-backup-script)
6. [Restore Procedures](#6-restore-procedures)
7. [Backup Verification](#7-backup-verification)
8. [Retention Policy](#8-retention-policy)

---

## 1. Backup Strategy Overview

```
Asset             │ Method                │ Frequency │ Retention │ RTO*    │ RPO**
──────────────────┼───────────────────────┼───────────┼───────────┼─────────┼──────
PostgreSQL DB     │ pg_dump (custom fmt)  │ Daily     │ 30 days   │ < 1h    │ 24h
                  │ WAL archiving (PITR)  │ Continuous│ 7 days    │ < 15min │ minutes
Object Storage    │ R2 → R2 cross-region  │ Daily     │ 90 days   │ < 1h    │ 24h
Redis             │ Not backed up         │ —         │ —         │ < 1min  │ N/A
```

*RTO — Recovery Time Objective (how long to restore)  
**RPO — Recovery Point Objective (max data loss)

---

## 2. PostgreSQL Backups

### Manual backup (local dev)

```bash
# Full backup — custom format (compressed, supports parallel restore)
pg_dump \
    --format=custom \
    --compress=9 \
    --file=backup_$(date +%Y%m%d_%H%M%S).dump \
    "${DATABASE_URL}"

# Schema only (useful for audit/review)
pg_dump --schema-only --file=schema.sql "${DATABASE_URL}"

# Single schema backup
pg_dump --schema=core --format=custom --file=core_$(date +%Y%m%d).dump "${DATABASE_URL}"
```

### Production backup (to S3/R2)

```bash
#!/bin/bash
# scripts/backup_postgres.sh
set -euo pipefail

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="yomira_db_${DATE}.dump"
BUCKET="yomira-backups"

echo "Starting PostgreSQL backup: ${BACKUP_FILE}"

# Dump to file
pg_dump \
    --format=custom \
    --compress=9 \
    --file="/tmp/${BACKUP_FILE}" \
    "${DATABASE_URL}"

# Upload to object storage
aws s3 cp \
    "/tmp/${BACKUP_FILE}" \
    "s3://${BUCKET}/postgres/${BACKUP_FILE}" \
    --storage-class STANDARD_IA  # cheaper for infrequent access

# Clean up local file
rm "/tmp/${BACKUP_FILE}"

echo "Backup complete: s3://${BUCKET}/postgres/${BACKUP_FILE}"
```

### WAL archiving (Point-in-Time Recovery)

For production, configure PostgreSQL WAL archiving to enable PITR (restore to any point in the last 7 days):

```
# postgresql.conf (Supabase/RDS manages this automatically)
wal_level = replica
archive_mode = on
archive_command = 'aws s3 cp %p s3://yomira-backups/wal/%f'
archive_timeout = 300   # archive WAL every 5 min even if no changes
```

---

## 3. Object Storage Backups

### R2 → R2 cross-region replication

Cloudflare R2 does not yet support native replication. Use rclone for scheduled sync:

```bash
#!/bin/bash
# scripts/backup_r2.sh
# Requires rclone configured with R2 credentials

rclone sync \
    r2:yomira-media \
    r2-backup:yomira-media-backup \
    --transfers=16 \
    --checkers=16 \
    --log-level=INFO \
    --log-file=/var/log/rclone_backup.log

echo "R2 backup complete: $(date)"
```

```ini
# ~/.config/rclone/rclone.conf
[r2]
type = s3
provider = Cloudflare
access_key_id = YOUR_KEY
secret_access_key = YOUR_SECRET
endpoint = https://ACCOUNT_ID.r2.cloudflarestorage.com

[r2-backup]
type = s3
provider = Cloudflare
access_key_id = BACKUP_KEY
secret_access_key = BACKUP_SECRET
endpoint = https://BACKUP_ACCOUNT_ID.r2.cloudflarestorage.com
```

---

## 4. Redis

Redis data is **ephemeral by design** in Yomira. Everything in Redis can be reconstructed:

| Key type | What happens on Redis loss | Recovery |
|---|---|---|
| Rate limit counters | Resets — brief window with no rate limiting | Automatic (refills as requests come in) |
| Search cache | Cache miss — DB is queried | Automatic (repopulates on next request) |
| Session blacklist | No impact — refresh tokens still valid in DB | Automatic |

**No backup needed for Redis.** If Redis goes down, restart it — it will be empty and refill automatically.

If Redis persistence is wanted for audit purposes:
```bash
# Enable AOF persistence in redis.conf
appendonly yes
appendfsync everysec
```

---

## 5. Automated Backup Script

```bash
#!/bin/bash
# scripts/daily_backup.sh
# Run via cron: 0 2 * * * /scripts/daily_backup.sh >> /var/log/backup.log 2>&1

set -euo pipefail

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BUCKET="yomira-backups"
NOTIFY_WEBHOOK="${SLACK_WEBHOOK:-}"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }
notify() {
    if [ -n "${NOTIFY_WEBHOOK}" ]; then
        curl -s -X POST "${NOTIFY_WEBHOOK}" \
            -H 'Content-type: application/json' \
            -d "{\"text\": \"$1\"}"
    fi
}

# --- PostgreSQL ---
log "Starting PostgreSQL backup..."
BACKUP_FILE="postgres/yomira_${TIMESTAMP}.dump"

pg_dump \
    --format=custom \
    --compress=9 \
    --file="/tmp/yomira_pg.dump" \
    "${DATABASE_URL}"

aws s3 cp "/tmp/yomira_pg.dump" "s3://${BUCKET}/${BACKUP_FILE}" \
    --storage-class STANDARD_IA --quiet

rm /tmp/yomira_pg.dump
PG_SIZE=$(aws s3 ls "s3://${BUCKET}/${BACKUP_FILE}" | awk '{print $3}')
log "PostgreSQL backup complete: ${BACKUP_FILE} (${PG_SIZE} bytes)"

# --- R2 Object Storage ---
log "Starting R2 sync..."
rclone sync r2:yomira-media r2-backup:yomira-media-backup \
    --transfers=16 --quiet
log "R2 sync complete"

# --- Cleanup old backups ---
log "Cleaning up backups older than 30 days..."
aws s3 ls "s3://${BUCKET}/postgres/" \
    | awk '{print $4}' \
    | while read -r file; do
        file_date=$(echo "${file}" | grep -oE '[0-9]{8}')
        if [ "$(date -d "${file_date}" +%s 2>/dev/null || date -j -f '%Y%m%d' "${file_date}" +%s)" \
            -lt "$(date -d '30 days ago' +%s 2>/dev/null || date -v-30d +%s)" ]; then
            aws s3 rm "s3://${BUCKET}/postgres/${file}" --quiet
            log "Deleted old backup: ${file}"
        fi
    done

notify "✅ Yomira daily backup complete: $(date '+%Y-%m-%d')"
log "All backups complete"
```

---

## 6. Restore Procedures

### Full PostgreSQL restore

```bash
# 1. Download backup from S3
aws s3 cp "s3://yomira-backups/postgres/yomira_20260222_020000.dump" \
    /tmp/restore.dump

# 2. Create empty target database (or drop and recreate)
psql "${DATABASE_URL}" -c "DROP DATABASE yomira;"
psql "${DATABASE_URL}" -c "CREATE DATABASE yomira;"

# 3. Restore
pg_restore \
    --dbname "${DATABASE_URL}" \
    --jobs 4 \               # parallel restore (4 workers)
    --verbose \
    /tmp/restore.dump

# 4. Verify
psql "${DATABASE_URL}" -c "SELECT COUNT(*) FROM core.comic;"
psql "${DATABASE_URL}" -c "SELECT version, dirty FROM schema_migrations;"

# 5. Cleanup
rm /tmp/restore.dump

echo "Restore complete"
```

### Point-in-Time Recovery (WAL)

```bash
# Restore to a specific point in time (requires WAL archiving enabled)

# 1. Stop the PostgreSQL server

# 2. Restore base backup
pg_restore --format=custom --dbname "${DATABASE_URL}" /tmp/base_backup.dump

# 3. Create recovery.conf (PostgreSQL 12+: use postgresql.conf)
cat >> postgresql.conf <<EOF
restore_command = 'aws s3 cp s3://yomira-backups/wal/%f %p'
recovery_target_time = '2026-02-22 01:00:00+07'
recovery_target_action = 'promote'
EOF

# 4. Create recovery signal file
touch $PGDATA/recovery.signal

# 5. Start PostgreSQL — it will apply WAL until the target timestamp
pg_ctl start

# 6. Verify data is at expected state
psql "${DATABASE_URL}" -c "SELECT MAX(createdat) FROM users.account;"
```

### Restore single table from backup

```bash
# Restore core.comic table only (without touching other tables)
pg_restore \
    --dbname "${DATABASE_URL}" \
    --table=comic \
    --schema=core \
    --data-only \
    /tmp/restore.dump
```

### Restore object storage

```bash
# Sync backup bucket back to primary
rclone sync \
    r2-backup:yomira-media-backup \
    r2:yomira-media \
    --transfers=16 \
    --progress

echo "Object storage restore complete"
```

---

## 7. Backup Verification

Never trust unverified backups. Run verification weekly:

```bash
#!/bin/bash
# scripts/verify_backup.sh

# Download latest backup
LATEST=$(aws s3 ls "s3://yomira-backups/postgres/" \
    | sort | tail -1 | awk '{print $4}')
aws s3 cp "s3://yomira-backups/postgres/${LATEST}" /tmp/verify.dump

# Restore to a test database
createdb yomira_backup_test
pg_restore --dbname "postgres://user:pass@localhost/yomira_backup_test" \
    --jobs 2 /tmp/verify.dump

# Verify critical tables exist and have data
psql "postgres://user:pass@localhost/yomira_backup_test" <<EOF
SELECT 'users' AS schema, COUNT(*) AS rows FROM users.account
UNION ALL
SELECT 'core', COUNT(*) FROM core.comic
UNION ALL
SELECT 'library', COUNT(*) FROM library.entry;
EOF

# Cleanup
dropdb yomira_backup_test
rm /tmp/verify.dump

echo "Backup verification complete: ${LATEST}"
```

---

## 8. Retention Policy

| Backup type | Retention | Location | Storage class |
|---|---|---|---|
| Daily PostgreSQL dump | 30 days | `s3://yomira-backups/postgres/` | STANDARD_IA |
| WAL segments | 7 days | `s3://yomira-backups/wal/` | STANDARD |
| Monthly snapshot (1st of month) | 12 months | `s3://yomira-backups/monthly/` | GLACIER |
| Annual snapshot (Jan 1st) | Indefinite | `s3://yomira-backups/annual/` | GLACIER |
| Object storage (R2) | 90 days rolling | `r2-backup:yomira-media-backup` | Standard |

### Automated lifecycle rules

```json
// S3/R2 lifecycle rule (set via console or AWS CLI)
{
  "Rules": [
    {
      "ID": "expire-daily-backups",
      "Filter": { "Prefix": "postgres/" },
      "Status": "Enabled",
      "Expiration": { "Days": 30 }
    },
    {
      "ID": "expire-wal",
      "Filter": { "Prefix": "wal/" },
      "Status": "Enabled",
      "Expiration": { "Days": 7 }
    },
    {
      "ID": "glacier-monthly",
      "Filter": { "Prefix": "monthly/" },
      "Status": "Enabled",
      "Transitions": [{ "Days": 30, "StorageClass": "GLACIER" }]
    }
  ]
}
```
