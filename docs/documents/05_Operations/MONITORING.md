# Monitoring & Observability — Yomira

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22

---

## Table of Contents

1. [Observability Stack](#1-observability-stack)
2. [Metrics (Prometheus)](#2-metrics-prometheus)
3. [Dashboards (Grafana)](#3-dashboards-grafana)
4. [Logging (Loki / slog)](#4-logging-loki--slog)
5. [Alerting Rules](#5-alerting-rules)
6. [SLO / SLI Definitions](#6-slo--sli-definitions)
7. [Tracing (OpenTelemetry)](#7-tracing-opentelemetry)

---

## 1. Observability Stack

| Concern | Tool | Where |
|---|---|---|
| **Metrics** | Prometheus | `/metrics` endpoint scraped every 15s |
| **Dashboards** | Grafana | `grafana.yomira.internal` |
| **Logs** | `slog` (JSON) → Loki (or stdout → log aggregator) | Grafana Loki or Datadog |
| **Traces** | OpenTelemetry → Tempo | Grafana Tempo (optional, v2.0) |
| **Uptime** | Grafana Cloud Synthetic Monitoring | External probe from multiple regions |
| **Alerts** | Prometheus Alertmanager → PagerDuty / Slack | `#alerts` Slack channel |

---

## 2. Metrics (Prometheus)

### Expose metrics endpoint

```go
// cmd/server/main.go
import "github.com/prometheus/client_golang/prometheus/promhttp"

r.Handle("/metrics", promhttp.Handler())   // Prometheus scrapes this
```

### Custom metrics

```go
// shared/metrics/metrics.go
var (
    HTTPRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request latency by method, path, and status",
            Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
        },
        []string{"method", "path", "status"},
    )

    HTTPRequestTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total HTTP requests by method, path, and status",
        },
        []string{"method", "path", "status"},
    )

    DBQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "db_query_duration_seconds",
            Help:    "PostgreSQL query latency by operation",
            Buckets: prometheus.DefBuckets,
        },
        []string{"operation", "table"},
    )

    DBPoolWaitDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name:    "db_pool_wait_seconds",
        Help:    "Time spent waiting for a DB connection from the pool",
        Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5},
    })

    ActiveSessions = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "websocket_connections_active",
        Help: "Number of active WebSocket connections",
    })

    CacheHitTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "redis_cache_hits_total",
            Help: "Redis cache hits and misses",
        },
        []string{"key_prefix", "result"},   // result: "hit" | "miss"
    )

    EmailsSentTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "emails_sent_total",
            Help: "Total emails sent by template",
        },
        []string{"template", "status"},    // status: "success" | "error"
    )

    BatchJobDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "batch_job_duration_seconds",
            Help:    "Duration of background batch jobs",
            Buckets: []float64{1, 5, 15, 30, 60, 120, 300},
        },
        []string{"job_name", "status"},    // status: "success" | "error"
    )
)

func init() {
    prometheus.MustRegister(
        HTTPRequestDuration, HTTPRequestTotal,
        DBQueryDuration, DBPoolWaitDuration,
        CacheHitTotal, EmailsSentTotal,
        BatchJobDuration,
    )
}
```

### Metric collection in middleware

```go
// middleware/metrics.go
func Metrics() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            ww := NewStatusRecorder(w)

            next.ServeHTTP(ww, r)

            duration := time.Since(start).Seconds()
            status := strconv.Itoa(ww.Status())
            path := NormalizePath(r)   // "/comics/01952fb0-..." → "/comics/:id"

            metrics.HTTPRequestDuration.
                WithLabelValues(r.Method, path, status).
                Observe(duration)
            metrics.HTTPRequestTotal.
                WithLabelValues(r.Method, path, status).
                Inc()
        })
    }
}

// NormalizePath replaces UUIDs and numbers with placeholders to reduce cardinality
func NormalizePath(r *http.Request) string {
    path := r.URL.Path
    path = uuidRegex.ReplaceAllString(path, ":id")
    path = numRegex.ReplaceAllString(path, ":num")
    return path
}
```

### Key Prometheus metrics to watch

| Metric | Alert threshold | Meaning |
|---|---|---|
| `http_request_duration_seconds{p95}` | > 500ms | API getting slow |
| `http_requests_total{status=~"5.."}` rate | > 1% of total | Error spike |
| `db_query_duration_seconds{p99}` | > 1s | DB query slow |
| `db_pool_wait_seconds{p95}` | > 100ms | DB connection pool exhausted |
| `redis_cache_hits_total{result="miss"}` rate | > 50% miss rate | Cache ineffective |
| `batch_job_duration_seconds` | > 5min for incremental jobs | Stuck batch job |

---

## 3. Dashboards (Grafana)

### Dashboard 1 — API Overview

**Panels:**
- Request rate (req/s) — line chart by status code group (2xx, 4xx, 5xx)
- Error rate (%) — gauge + threshold (green < 1%, yellow < 5%, red ≥ 5%)
- p50/p95/p99 latency — line chart over time
- Top 10 slowest endpoints — table
- Rate limited requests — counter

### Dashboard 2 — Database

**Panels:**
- Active connections / pool utilization
- Query duration p50/p95 by table
- Slow queries (> 500ms) — table with query text
- Table sizes over time (core.comic, social.comment, analytics.pageview)
- Replication lag (if read replica configured)

### Dashboard 3 — Background Jobs

**Panels:**
- Job run frequency — heatmap
- Job success/failure rate
- Job duration by name
- Stuck jobs (running > 30min) — alert panel

### Dashboard 4 — Business Metrics

**Panels:**
- New registrations per day
- Daily active users (unique userids in analytics)
- Comics uploaded per week
- Chapters uploaded per week
- Email delivery rate

### Grafana provisioning (`grafana/dashboards/api_overview.json`)

```bash
# Mount dashboards via config in docker-compose (monitoring stack)
volumes:
  - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
  - ./grafana/datasources:/etc/grafana/provisioning/datasources
```

---

## 4. Logging (Loki / slog)

### Structured log format (production)

```go
// JSON format in production — parseable by Loki
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})))
```

### Log fields standardized per request

```json
{
  "time": "2026-02-22T01:02:58Z",
  "level": "INFO",
  "msg": "request",
  "method": "GET",
  "path": "/api/v1/comics/:id",
  "status": 200,
  "latency_ms": 12,
  "request_id": "01952fa3-...",
  "ip": "1.2.3.4",
  "user_id": "01952fa3-...",
  "user_agent": "Mozilla/5.0..."
}
```

### Loki label strategy

```yaml
# Only use low-cardinality labels for Loki
labels:
  - job: "yomira-api"
  - env: "production"
  - pod: "{pod_name}"  # for multi-instance

# Query examples:
# {job="yomira-api", env="production"} | json | level="ERROR"
# {job="yomira-api"} | json | path="/api/v1/auth/login" | status >= 400
# {job="yomira-api"} | json | request_id="01952fa3-..."
```

### Log retention

| Level | Retention |
|---|---|
| ERROR | 90 days |
| WARN | 30 days |
| INFO | 14 days |
| DEBUG | 3 days (staging only) |

---

## 5. Alerting Rules

### Prometheus Alertmanager rules (`alerts/yomira.yml`)

```yaml
groups:
  - name: yomira_api
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: |
          rate(http_requests_total{status=~"5.."}[5m])
          / rate(http_requests_total[5m]) > 0.01
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High 5xx error rate ({{ $value | humanizePercentage }})"
          runbook: "https://docs.yomira.app/runbook#high-error-rate"

      # High latency
      - alert: HighLatency
        expr: |
          histogram_quantile(0.95, 
            rate(http_request_duration_seconds_bucket[5m])) > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "p95 latency above 500ms"

      # DB connection pool near exhaustion
      - alert: DBPoolExhausted
        expr: db_pool_wait_seconds{quantile="0.95"} > 0.1
        for: 3m
        labels:
          severity: critical
        annotations:
          summary: "DB connection pool wait > 100ms p95"

      # Health check failing
      - alert: HealthCheckFailing
        expr: probe_success{job="yomira-api-health"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "API health check failing"

  - name: yomira_database
    rules:
      # Replication lag
      - alert: ReplicationLag
        expr: pg_replication_lag_seconds > 30
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "PostgreSQL replication lag > 30s"

      # Disk usage
      - alert: DiskUsageHigh
        expr: |
          (node_filesystem_avail_bytes / node_filesystem_size_bytes) < 0.20
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Disk usage above 80%"

  - name: yomira_redis
    rules:
      - alert: RedisMemoryHigh
        expr: redis_memory_used_bytes / redis_maxmemory_bytes > 0.80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Redis memory usage above 80%"
```

### Notification channels

```yaml
# alertmanager.yml
route:
  receiver: slack-critical
  routes:
    - match:
        severity: critical
      receiver: pagerduty
    - match:
        severity: warning
      receiver: slack-warning

receivers:
  - name: slack-critical
    slack_configs:
      - channel: "#alerts-critical"
        send_resolved: true
  - name: slack-warning
    slack_configs:
      - channel: "#alerts-warning"
  - name: pagerduty
    pagerduty_configs:
      - routing_key: "${PAGERDUTY_KEY}"
```

---

## 6. SLO / SLI Definitions

### Service Level Objectives

| SLO | Target | Measurement window |
|---|---|---|
| API Availability | 99.9% (≈ 8.7h downtime/year) | 30-day rolling |
| API Latency (p95) | < 300ms | 5-minute window |
| Error Rate | < 0.1% of all requests | 1-hour window |
| Email Delivery | > 99% delivered within 5 min | Per-day |

### Error Budget

```
SLO: 99.9% availability over 30 days
Total requests in 30d: ~50M (estimate)
Max allowed errors: 50M × 0.001 = 50,000 errors
Error budget burn rate alert: if burn rate > 2× in last 1h (fast burn)
```

### SLI Queries (Prometheus)

```yaml
# Availability SLI — % of successful requests
availability:
  query: |
    rate(http_requests_total{status!~"5.."}[5m])
    / rate(http_requests_total[5m])

# Latency SLI — % of requests under threshold (300ms)
latency:
  query: |
    rate(http_request_duration_seconds_bucket{le="0.3"}[5m])
    / rate(http_request_duration_seconds_count[5m])
```

---

## 7. Tracing (OpenTelemetry)

> Planned for v2.0. Notes below are for documentation.

```go
// Instrument Go server with OpenTelemetry (otelhttp)
import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

r.Use(func(next http.Handler) http.Handler {
    return otelhttp.NewHandler(next, "yomira-api")
})

// Instrument DB queries (pgx otel trace)
import "github.com/exaring/otelpgx"

tracer := otelpgx.NewTracer()
pgxConfig.Tracer = tracer
```

**Trace context propagation:**
- Incoming: `W3C Trace-Context` header (`traceparent`)
- Outgoing (to SMTP, S3): same header forwarded
- Storage: Grafana Tempo (or Jaeger)
