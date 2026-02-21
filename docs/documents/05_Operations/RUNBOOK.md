# Runbook — Yomira Incident Playbooks

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22

When an alert fires, find the incident name below and follow the steps **in order**.

---

## Table of Contents

1. [High API Error Rate (5xx)](#1-high-api-error-rate-5xx)
2. [High API Latency](#2-high-api-latency)
3. [PostgreSQL Down / Unreachable](#3-postgresql-down--unreachable)
4. [Redis Down / Unreachable](#4-redis-down--unreachable)
5. [DB Connection Pool Exhausted](#5-db-connection-pool-exhausted)
6. [Crawler Stuck Jobs](#6-crawler-stuck-jobs)
7. [SMTP / Email Delivery Failure](#7-smtp--email-delivery-failure)
8. [Object Storage Unreachable](#8-object-storage-unreachable)
9. [Disk Space Critical](#9-disk-space-critical)
10. [Memory / OOM](#10-memory--oom)
11. [Failed Migration](#11-failed-migration)
12. [Security Incident (Token Compromise)](#12-security-incident-token-compromise)

---

## 1. High API Error Rate (5xx)

**Alert:** `HighErrorRate` — 5xx > 1% of requests for 2 minutes  
**Severity:** Critical → PagerDuty

### Diagnosis

```bash
# 1. Check what endpoints are erroring
# In Grafana: HTTP Overview dashboard → Error Rate by Path panel

# 2. Check recent logs for ERROR level
# Loki: {job="yomira-api"} | json | level="ERROR" | last 10 min

# 3. Check if DB is the cause
# Prometheus: db_query_duration_seconds{p99} spike?
# Grafana: Database dashboard → Query Duration panel
```

### Steps

1. **Check `/health/ready`** → is the service up?
   ```bash
   curl https://api.yomira.app/health/ready
   ```

2. **Check DB connectivity** — if `"database": "error"` in health response → see [§3](#3-postgresql-down--unreachable)

3. **Check for a bad deploy** — did a deploy happen in the last 30 min?
   ```bash
   flyctl releases list          # check recent deploys
   ```
   If yes → [rollback](#rollback):
   ```bash
   flyctl deploy --image ghcr.io/your-org/yomira-api:{previous-sha}
   ```

4. **Check for a specific panic** — look for `"panic recovered"` in logs

5. **If origin unknown** — enable debug logging temporarily:
   ```bash
   flyctl secrets set LOG_LEVEL=debug
   flyctl deploy   # rolling restart with new log level
   # ... investigate ...
   flyctl secrets set LOG_LEVEL=info && flyctl deploy
   ```

6. **Escalate** if error rate persists > 10 min after investigation

---

## 2. High API Latency

**Alert:** `HighLatency` — p95 > 500ms for 5 minutes  
**Severity:** Warning → Slack

### Diagnosis

```bash
# 1. Identify the slow endpoints
# Grafana: API Overview → Top 10 Slowest Endpoints

# 2. Check DB query times
# Grafana: Database dashboard → Query Duration p99

# 3. Check Redis latency (affects rate limiting, caching)
docker compose exec redis redis-cli --latency-history -i 1

# 4. pg_stat_statements — find the slow query
psql "${DATABASE_URL}" -c "
SELECT query, calls, mean_exec_time, max_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC LIMIT 10;"
```

### Steps

1. **Slow DB queries?** → Run `EXPLAIN ANALYZE` on the offending query. Add missing index if needed (use `CONCURRENTLY`).

2. **DB connection pool wait high?** → See [§5](#5-db-connection-pool-exhausted)

3. **Redis slow?** → Check `INFO memory` in redis-cli. High memory usage degrades performance.

4. **External dependency?** (SMTP, S3, OAuth provider) → Check provider status page.

5. **Traffic spike?** → Enable maintenance mode temporarily:
   ```bash
   # PUT /admin/settings {"key": "site.maintenance_mode", "value": "true"}
   curl -X PUT https://api.yomira.app/api/v1/admin/settings \
     -H "Authorization: Bearer ${ADMIN_TOKEN}" \
     -d '{"key": "site.maintenance_mode", "value": "true"}'
   ```

---

## 3. PostgreSQL Down / Unreachable

**Alert:** `HealthCheckFailing` + `database: error` in health check  
**Severity:** Critical → PagerDuty

### Steps

1. **Verify DB is actually down:**
   ```bash
   psql "${DATABASE_URL}" -c "SELECT 1;"
   # or via fly proxy:
   flyctl postgres connect -a yomira-db
   ```

2. **Check DB container/instance health:**
   ```bash
   # If self-hosted Docker:
   docker compose logs postgres
   docker compose ps postgres

   # If Supabase/Railway: check their status page / dashboard
   ```

3. **If OOM killed:** Increase PostgreSQL shared memory or reduce `max_connections`
   ```bash
   docker compose exec postgres psql -U yomira -c "SHOW max_connections;"
   ```

4. **If disk full:** See [§9](#9-disk-space-critical)

5. **Restart DB** (last resort — only if no data loss risk):
   ```bash
   docker compose restart postgres
   # Wait and verify:
   docker compose exec postgres pg_isready -U yomira
   ```

6. **Enable maintenance mode** while DB is recovering (see §2 step 5)

7. **Check for dirty migration** after restart:
   ```bash
   migrate -database "${DATABASE_URL}" -path src/common/DML/migrations version
   # If dirty=true → see §11
   ```

---

## 4. Redis Down / Unreachable

**Alert:** `HealthCheckFailing` + `redis: error` in health check  
**Severity:** Critical → PagerDuty

### Impact when Redis is down

- ❌ Rate limiting disabled (all requests allowed through)
- ❌ Search cache misses (higher DB load)
- ❌ Session blacklist checks bypassed
- ✅ Core reads/writes still work

### Steps

1. **Verify:**
   ```bash
   redis-cli -u "${REDIS_URL}" ping   # should return PONG
   ```

2. **Check memory:**
   ```bash
   redis-cli -u "${REDIS_URL}" info memory | grep used_memory_human
   ```

3. **Restart Redis** (data is ephemeral — cache data OK to lose):
   ```bash
   docker compose restart redis
   # or Upstash: restart from dashboard
   ```

4. **If maxmemory policy triggered (OOM):**
   ```bash
   redis-cli -u "${REDIS_URL}" config set maxmemory-policy allkeys-lru
   # This allows Redis to evict old keys when memory is full
   ```

5. **Monitor rate limit abuse** while Redis is recovering — manually block abusive IPs at load balancer level if needed

---

## 5. DB Connection Pool Exhausted

**Alert:** `DBPoolExhausted` — pool wait p95 > 100ms  
**Severity:** Critical → Slack

### Symptoms

- `db_pool_wait_seconds` high in Grafana
- API latency spike (requests wait for DB connection)
- Errors: `"context deadline exceeded"` in logs

### Steps

1. **Check active connections:**
   ```sql
   SELECT count(*), state, wait_event_type, wait_event
   FROM pg_stat_activity
   GROUP BY state, wait_event_type, wait_event
   ORDER BY count DESC;
   ```

2. **Find long-running queries (> 30s):**
   ```sql
   SELECT pid, now() - query_start AS duration, state, query
   FROM pg_stat_activity
   WHERE query_start < NOW() - INTERVAL '30 seconds'
     AND state != 'idle'
   ORDER BY duration DESC;
   ```

3. **Kill stuck queries** (if identified as safe):
   ```sql
   SELECT pg_cancel_backend(pid);   -- soft cancel
   SELECT pg_terminate_backend(pid); -- hard kill (use if cancel fails)
   ```

4. **If crawler jobs are the cause** (bulk DB writes):
   ```bash
   # Pause the crawler
   curl -X POST https://api.yomira.app/api/v1/admin/crawler/pause \
     -H "Authorization: Bearer ${ADMIN_TOKEN}"
   ```

5. **Tune pool size** (increase max connections — requires restart):
   ```bash
   flyctl secrets set DB_POOL_MAX=30   # increase from default 20
   flyctl deploy
   ```

---

## 6. Crawler Stuck Jobs

**Alert:** Custom — crawler jobs in `running` state > 30 min  
**Severity:** Warning → Slack

### Diagnosis

```sql
-- Find stuck jobs
SELECT id, sourceid, status, scheduledat, startedat,
       NOW() - startedat AS running_for
FROM crawler.job
WHERE status = 'running'
  AND startedat < NOW() - INTERVAL '30 minutes'
ORDER BY startedat ASC;
```

### Steps

1. **Reset stuck jobs to `pending`:**
   ```bash
   curl -X POST https://api.yomira.app/api/v1/admin/crawler/jobs/reset-stuck \
     -H "Authorization: Bearer ${ADMIN_TOKEN}"
   # Or directly:
   ```
   ```sql
   UPDATE crawler.job
   SET status = 'pending', startedat = NULL
   WHERE status = 'running'
     AND startedat < NOW() - INTERVAL '30 minutes';
   ```

2. **Check recent crawler logs:**
   ```sql
   SELECT log.level, log.message, log.createdat
   FROM crawler.log
   JOIN crawler.job ON log.jobid = job.id
   WHERE job.status = 'running'
   ORDER BY log.createdat DESC
   LIMIT 50;
   ```

3. **If target site is blocking Yomira** (403/429 in logs):
   - Disable the source: `PATCH /admin/crawler/sources/:id` `{"isactive": false}`
   - Check if IP rotation or user-agent change is needed

4. **Restart the crawler worker** if Go routine appears stuck:
   ```bash
   flyctl restart            # rolling restart of all instances
   ```

---

## 7. SMTP / Email Delivery Failure

**Alert:** `emails_sent_total{status="error"}` spike  
**Severity:** Warning → Slack

### Diagnosis

```bash
# Check recent email errors in logs
# Loki: {job="yomira-api"} | json | msg="mail send failed" | last 30m

# Check SMTP provider status
# Resend: https://status.resend.com
# SendGrid: https://status.sendgrid.com
```

### Steps

1. **Check provider status page** first — may be a provider outage

2. **Test delivery manually:**
   ```bash
   curl -X POST https://api.yomira.app/api/v1/admin/mail/send \
     -H "Authorization: Bearer ${ADMIN_TOKEN}" \
     -d '{"to": ["test@example.com"], "subject": "Test", "bodytext": "Test"}'
   ```

3. **If API key expired/rotated:**
   ```bash
   flyctl secrets set RESEND_API_KEY="re_new_key..."
   flyctl deploy
   ```

4. **Fallback to SMTP:**
   ```bash
   flyctl secrets set MAIL_PROVIDER=smtp SMTP_HOST=smtp.fallback.com \
     SMTP_PORT=587 SMTP_USER=... SMTP_PASS=...
   flyctl deploy
   ```

5. **Non-critical degradation** — email failures don't affect API functionality. Log the issue, notify affected users if signup emails are lost.

---

## 8. Object Storage Unreachable

**Alert:** `HealthCheckFailing` + `storage: error`, or upload errors in logs  
**Severity:** Warning → Slack

### Impact

- ❌ New image uploads fail
- ❌ Presigned URL generation fails
- ✅ Reading existing images still works (served from CDN cache)
- ✅ All non-upload API endpoints work normally

### Steps

1. **Check CDN/R2 status:** https://www.cloudflarestatus.com

2. **Test connectivity:**
   ```bash
   curl -I "https://cdn.yomira.app/covers/test.webp"
   ```

3. **Check credentials haven't expired:**
   ```bash
   flyctl secrets list | grep S3
   ```

4. **Temporary workaround** — disable new uploads via feature flag:
   ```bash
   # PUT /admin/settings {"key": "features.uploads_enabled", "value": "false"}
   ```

---

## 9. Disk Space Critical

**Alert:** `DiskUsageHigh` — disk available < 20%  
**Severity:** Warning → Slack (critical at < 5%)

### Steps

1. **Check disk usage:**
   ```bash
   df -h
   du -sh /var/lib/postgresql/data   # DB data dir
   ```

2. **Find large tables:**
   ```sql
   SELECT schemaname, tablename,
          pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
   FROM pg_tables
   ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
   LIMIT 10;
   ```

3. **Drop old partitions** (analytics — check retention policy first):
   ```sql
   -- Keep last 12 months. Drop data > 12 months old.
   DROP TABLE IF EXISTS analytics.pageview_2025_01;
   ```

4. **Run VACUUM to reclaim dead tuples:**
   ```sql
   VACUUM FULL analytics.pageview;   -- reclaims disk (blocks table briefly)
   ```

5. **Expand disk volume** via cloud provider dashboard if above steps insufficient.

---

## 10. Memory / OOM

**Alert:** Pod restarted with OOM reason  
**Severity:** Critical → PagerDuty

### Steps

1. **Check memory usage:**
   ```bash
   flyctl status          # check restart count
   flyctl logs -a yomira  # check for OOM message
   ```

2. **Find memory leak** — common causes:
   - Goroutine leak: `GET /debug/pprof/goroutine` (if pprof enabled in staging)
   - DB row scanner not closed: `rows.Close()` missing → connection held
   - Large response buffering

3. **Increase memory as temporary fix:**
   ```toml
   # fly.toml
   [[vm]]
     memory = "1gb"   # increase
   ```

4. **Enable profiling** on staging to reproduce:
   ```go
   import _ "net/http/pprof"
   r.Mount("/debug", middleware.Profiler())  // staging only
   ```

---

## 11. Failed Migration

**Alert:** Deployment failed with `dirty=true` migration  
**Severity:** Critical → PagerDuty (blocks deploys)

### Steps

1. **Check dirty version:**
   ```bash
   migrate -database "${DATABASE_URL}" -path src/common/DML/migrations version
   # Output: 5 (dirty)
   ```

2. **Understand what failed** — read the migration file. Did it partially apply?

3. **Fix the state:**
   - If migration DID NOT partially apply: force version back to N-1
     ```bash
     migrate -database "${DATABASE_URL}" -path src/common/DML/migrations force 4
     ```
   - If migration DID partially apply: manually undo the partial change in psql, then force version

4. **Re-apply corrected migration:**
   ```bash
   migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up 1
   ```

5. **Verify:**
   ```bash
   migrate -database "${DATABASE_URL}" -path src/common/DML/migrations version
   # Should show version N, dirty=false
   ```

---

## 12. Security Incident (Token Compromise)

**Alert:** Unusual login patterns, mass token reuse attempts  
**Severity:** Critical → PagerDuty + Security team

### Immediate steps (first 15 minutes)

1. **Revoke ALL sessions for affected users:**
   ```bash
   curl -X POST https://api.yomira.app/api/v1/admin/users/{userId}/sessions/revoke-all \
     -H "Authorization: Bearer ${ADMIN_TOKEN}"
   ```
   ```sql
   -- Or direct DB if API is compromised:
   UPDATE users.session SET revokedat = NOW()
   WHERE userid = $1 AND revokedat IS NULL;
   ```

2. **Rotate JWT keys** if private key is suspected compromised:
   ```bash
   # Generate new key pair
   openssl genrsa -out jwt_private_new.pem 2048
   openssl rsa -in jwt_private_new.pem -pubout -out jwt_public_new.pem

   # Deploy new keys (all existing tokens immediately invalid)
   flyctl secrets set \
     JWT_PRIVATE_KEY="$(cat jwt_private_new.pem)" \
     JWT_PUBLIC_KEY="$(cat jwt_public_new.pem)"
   flyctl deploy
   ```

3. **Enable maintenance mode** if breach is ongoing (buys time to assess scope)

4. **Audit log review:**
   ```sql
   SELECT * FROM system.auditlog
   WHERE createdat > NOW() - INTERVAL '24 hours'
   ORDER BY createdat DESC
   LIMIT 100;
   ```

5. **Notify affected users** via admin email send

6. **Write post-mortem** within 48 hours
