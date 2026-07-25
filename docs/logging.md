# Logging Architecture Document — Hospital Backend (MVP)

## 1. Current State Assessment

Looking at the codebase:

- **Only `log.Fatalf`** is used in `cmd/main.go` — there is zero structured logging inside services, repositories, or controllers
- **No request correlation** — `context.Background()` is passed ad hoc (e.g. in `appointment.services.go` and `payments.services.go`) but never carries a request ID or logger
- **Fiber's `*fiber.Ctx`** is the HTTP context, but there is no middleware attaching tracing metadata to it
- **Services are constructor-injected** (`NewContainer` in `appinit/app.go`) — this means a logger can be cleanly injected into every service the same way `*gorm.DB` is

This is a clean starting point to introduce logging correctly from scratch.

---

## 2. Package Recommendation: `zap` by Uber

**Use [`go.uber.org/zap`](https://github.com/uber-go/zap).**

| Criteria          | `zap`     | `zerolog` | `logrus` | `slog` (stdlib) |
|-------------------|-----------|-----------|----------|-----------------|
| Speed             | Fastest   | Very fast | Slow     | Fast            |
| Structured fields | Yes       | Yes       | Yes      | Yes             |
| Log levels        | Yes       | Yes       | Yes      | Yes             |
| Ecosystem         | Excellent | Good      | Good     | Growing         |
| JSON output       | Native    | Native    | Native   | Native          |
| MVP-ready         | Yes       | Yes       | Yes      | Yes             |

`zap` wins because it has `zap.Field` typed fields (compile-time safety), `SugaredLogger` for quick dev use, built-in JSON output for production, and an enormous ecosystem. `zerolog` is a very close second if you want zero-allocation.

### Install

```bash
go get go.uber.org/zap
go get go.uber.org/zap/zapcore
go get gopkg.in/natefinish/lumberjack.v2
```

---

## 3. Logger Design

### 3.1 Logger Package: `pkg/logger/logger.go`

```go
package logger

import (
    "os"
    "time"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "gopkg.in/natefinish/lumberjack.v2"
)

var Log *zap.Logger

func Init(env string) {
    var core zapcore.Core

    encoderCfg := zapcore.EncoderConfig{
        TimeKey:        "ts",
        LevelKey:       "level",
        NameKey:        "logger",
        CallerKey:      "caller",
        MessageKey:     "msg",
        StacktraceKey:  "stacktrace",
        LineEnding:     zapcore.DefaultLineEnding,
        EncodeLevel:    zapcore.LowercaseLevelEncoder,
        EncodeTime:     zapcore.ISO8601TimeEncoder,
        EncodeDuration: zapcore.MillisDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }

    if env == "production" {
        // Daily rotating file writer
        fileWriter := zapcore.AddSync(&lumberjack.Logger{
            Filename:   "logs/app.log",
            MaxSize:    100,   // MB before rotation
            MaxBackups: 30,    // number of old files to keep
            MaxAge:     30,    // days
            Compress:   true,
        })

        // Also write to stdout for container log collection
        consoleWriter := zapcore.AddSync(os.Stdout)

        core = zapcore.NewTee(
            zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), fileWriter, zapcore.InfoLevel),
            zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), consoleWriter, zapcore.InfoLevel),
        )
    } else {
        // Development: pretty colored console output
        consoleWriter := zapcore.AddSync(os.Stdout)
        devEncoderCfg := zap.NewDevelopmentEncoderConfig()
        devEncoderCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
            enc.AppendString(t.Format("15:04:05.000"))
        }
        core = zapcore.NewCore(
            zapcore.NewConsoleEncoder(devEncoderCfg),
            consoleWriter,
            zapcore.DebugLevel,
        )
    }

    Log = zap.New(core,
        zap.AddCaller(),
        zap.AddStacktrace(zapcore.ErrorLevel), // auto stack trace on error+
    )
}

func Sync() {
    _ = Log.Sync()
}
```

### 3.2 Initialize in `cmd/main.go`

```go
func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("%v", err)
    }

    // Initialize structured logger first
    logger.Init(cfg.Env) // "production" or "development"
    defer logger.Sync()

    logger.Log.Info("starting hospital backend", zap.String("env", cfg.Env))
    // ... rest of startup
}
```

### 3.3 Log Levels — When to Use Each

```go
// DEBUG — internal computation details, useful only in development
// Never use in production hot paths
logger.Log.Debug("slot calculation done",
    zap.String("date", date),
    zap.Int("slot_count", len(slots)),
)

// INFO — expected business events, important milestones
logger.Log.Info("appointment created",
    zap.String("appointment_id", model.ID),
    zap.String("patient_id", model.PatientID),
    zap.String("doctor_id", model.DoctorID),
)

// WARN — unexpected but non-fatal; system can continue
logger.Log.Warn("payment attempt already processed, skipping",
    zap.String("payment_attempt_id", paymentAttempt.ID),
    zap.String("current_status", paymentAttempt.PaymentStatus),
)

// ERROR — operation failed, needs investigation, data may be lost
logger.Log.Error("failed to create webhook event",
    zap.Error(err),
    zap.String("provider", provider),
    zap.String("payment_attempt_id", paymentAttempt.ID),
)

// FATAL — application cannot continue (startup only, in main.go)
logger.Log.Fatal("database connection failed", zap.Error(err))
```

---

## 4. Context and Request Tracing — The Right Approach for Fiber

This is the most important design decision. Fiber uses `*fiber.Ctx` (fasthttp underneath), **not** Go's standard `context.Context`. There are two options:

### Why Not Standard `context.Context`?

Fiber's `*fiber.Ctx` does not embed standard `context.Context` in the same propagation path. You would need to call `c.UserContext()` and pass it manually — at which point you might as well pass the logger directly, which is more explicit and avoids hidden coupling.

### Why Not a Global Logger With Request ID Fields?

You cannot attach per-request fields to a global logger without creating a child logger anyway. A single global logger would mix log lines from concurrent requests with no way to correlate them.

### Recommended: Logger in Fiber Locals → Passed as Method Parameter

Attach a **request-scoped child logger** to `c.Locals()` in middleware, then pass it as a parameter through controller → service → repository.

```
HTTP Request
    → Fiber Middleware (generate request_id, create child logger, store in c.Locals)
    → Controller     (extract logger from c.Locals, pass to service method)
    → Service        (accepts logger as parameter, logs business events, passes to repo)
    → Repository     (accepts logger as parameter, logs query errors)
```

---

## 5. Request Logger Middleware

### File: `pkg/middleware/request_logger.go`

```go
package middleware

import (
    "hospital-backend/pkg/logger"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "go.uber.org/zap"
)

const LoggerKey    = "req_logger"
const RequestIDKey = "request_id"

func RequestLogger() fiber.Handler {
    return func(c *fiber.Ctx) error {
        requestID := uuid.NewString()
        start := time.Now()

        // Build a child logger with all request-level fields bound
        reqLogger := logger.Log.With(
            zap.String("request_id", requestID),
            zap.String("method", c.Method()),
            zap.String("path", c.Path()),
            zap.String("ip", c.IP()),
        )

        c.Locals(LoggerKey, reqLogger)
        c.Locals(RequestIDKey, requestID)

        // Return request ID to client for support correlation
        c.Set("X-Request-ID", requestID)

        reqLogger.Info("request started")

        err := c.Next()

        latency    := time.Since(start)
        statusCode := c.Response().StatusCode()

        if err != nil || statusCode >= 500 {
            reqLogger.Error("request failed",
                zap.Int("status", statusCode),
                zap.Duration("latency_ms", latency),
                zap.Error(err),
            )
        } else if statusCode >= 400 {
            reqLogger.Warn("client error",
                zap.Int("status", statusCode),
                zap.Duration("latency_ms", latency),
            )
        } else {
            reqLogger.Info("request completed",
                zap.Int("status", statusCode),
                zap.Duration("latency_ms", latency),
            )
        }

        return err
    }
}

// GetLogger extracts the request-scoped logger from Fiber context.
// Falls back to the global logger when called outside an HTTP request (e.g. background jobs).
func GetLogger(c *fiber.Ctx) *zap.Logger {
    if l, ok := c.Locals(LoggerKey).(*zap.Logger); ok {
        return l
    }
    return logger.Log
}
```

### Register in `pkg/middleware/middleware.go`

```go
func HandleMiddleware(app *fiber.App) {
    app.Use(RequestLogger()) // must be first
    // ... existing middleware
}
```

---

## 6. Logging Through the Stack — Concrete Examples

### 6.1 Controller Layer

```go
// internal/payments/controllers.go

func (p *PaymentController) RazorPayWebhook(c *fiber.Ctx) error {
    log := middleware.GetLogger(c)

    log.Info("webhook received", zap.String("provider", "razorpay"))

    // parse body, extract signature ...

    ok, err := p.WebhookService.ProcessWebhook(payload, signature, "razorpay", log)
    if err != nil {
        // Do NOT re-log the error here — the service already logged it.
        // Just return the response.
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "webhook processing failed",
        })
    }
    return c.JSON(fiber.Map{"ok": ok})
}
```

### 6.2 Service Layer

Pass the logger as a **method parameter**, not a struct field. This keeps it flexible — the same service can be called from HTTP handlers (with request-scoped logger) or background jobs (with global logger).

```go
// internal/payments/webhook_service.go

func (w *IWebhookService) ProcessWebhook(
    payload   []byte,
    signature string,
    provider  string,
    log       *zap.Logger,
) (bool, error) {

    // Pin a component tag to every log line from this service
    log = log.With(zap.String("component", "webhook_service"))

    gateway, err := w.PaymentFactory.GetProvider(provider)
    if err != nil {
        log.Error("provider not found",
            zap.String("provider", provider),
            zap.Error(err),
        )
        return false, err
    }

    isVerified, err := gateway.VerifySignature(payload, signature)
    if err != nil || !isVerified {
        log.Warn("webhook signature verification failed",
            zap.String("provider", provider),
            zap.Bool("verified", isVerified),
            zap.Error(err),
        )
        return false, err
    }

    log.Info("webhook signature verified", zap.String("provider", provider))

    dtoEvent, err := gateway.ParseWebhookEvent(payload)
    if err != nil {
        log.Error("failed to parse webhook event", zap.Error(err))
        return false, err
    }

    log.Info("webhook event parsed",
        zap.String("event_type", dtoEvent.EventType),
        zap.String("provider_link_id", dtoEvent.ProviderLinkID),
    )

    switch dtoEvent.EventType {
    case constants.PaymentLinkPaid:
        log.Info("processing payment_link.paid",
            zap.String("provider_link_id", dtoEvent.ProviderLinkID),
        )
        // ... business logic ...
        if err != nil {
            log.Error("failed to update payment attempt",
                zap.String("payment_attempt_id", paymentAttempt.ID),
                zap.Error(err),
            )
            begin.Rollback()
            return false, err
        }
        begin.Commit()
        log.Info("payment processed successfully",
            zap.String("payment_attempt_id", paymentAttempt.ID),
            zap.Int64("amount_paid", dtoEvent.AmountPaid),
        )
        return true, nil

    case constants.PaymentLinkCancelled:
        log.Warn("payment link cancelled",
            zap.String("provider_link_id", dtoEvent.ProviderLinkID),
        )
        // ...
    }

    return false, nil
}
```

### 6.3 Repository Layer

Log **only errors** at this layer. Business events belong in the service layer.

```go
// internal/payments/webhook_repository.go

func (r *WebhookRepository) CreateWebhookEvent(event WebhookEvents, log *zap.Logger) error {
    result := r.db.Create(&event)
    if result.Error != nil {
        log.Error("db: failed to create webhook event",
            zap.String("event_id", event.ID),
            zap.String("event_type", event.EventType),
            zap.Error(result.Error),
        )
        return result.Error
    }
    return nil
}
```

### 6.4 Background Jobs (no `*fiber.Ctx`)

For background goroutines (e.g. `NotificationContainer.Start`), use the global logger with a component tag:

```go
func (n *Notificationservice) Create(ctx context.Context, req dto.CreateRequest) {
    log := logger.Log.With(
        zap.String("component", "notification_service"),
        zap.String("notification_type", req.NotificationType),
    )

    log.Info("sending notification")
    // ...
    if err != nil {
        log.Error("failed to send notification", zap.Error(err))
    }
}
```

---

## 7. Standard Field Names (Team Convention)

Agree on these field names across the entire codebase so log queries are consistent.

| Field             | Type     | Example value             | Set in                   |
|-------------------|----------|---------------------------|--------------------------|
| `request_id`      | string   | `"a1b2-c3d4-..."`         | Request middleware        |
| `component`       | string   | `"webhook_service"`       | Each service             |
| `method`          | string   | `"POST"`                  | Request middleware        |
| `path`            | string   | `"/api/v1/payment/webhook"`| Request middleware       |
| `latency_ms`      | duration | `142ms`                   | Request middleware        |
| `status`          | int      | `200`                     | Request middleware        |
| `ip`              | string   | `"10.0.0.1"`              | Request middleware        |
| `patient_id`      | string   | `"uuid"`                  | Service layer            |
| `appointment_id`  | string   | `"uuid"`                  | Service layer            |
| `payment_id`      | string   | `"uuid"`                  | Service layer            |
| `organisation_id` | string   | `"uuid"`                  | Service layer            |
| `provider`        | string   | `"razorpay"`              | Payment service           |
| `event_type`      | string   | `"payment_link.paid"`     | Webhook service           |
| `error`           | error    | (zap.Error field)         | Any layer on failure     |

---

## 8. Log Storage Options for MVP

### Option A — Daily Rotating Files (Zero Cost, Start Here)

`lumberjack` handles this automatically (configured in `pkg/logger/logger.go`):

```
logs/
  app.log                   ← current file (written live)
  app-2026-07-02.log.gz     ← yesterday, compressed
  app-2026-07-01.log.gz
```

**Search from terminal:**
```bash
# All errors for a specific request
grep '"request_id":"abc-123"' logs/app.log | jq .

# All payment failures
grep '"component":"webhook_service"' logs/app.log | grep '"level":"error"' | jq .

# Slow requests over 500ms
grep '"latency_ms"' logs/app.log | jq 'select(.latency_ms > 500000000)'
```

**Verdict:** Zero infrastructure, works today. Fine for local development and week 1. Weak for demos — no UI to show stakeholders.

---

### Option B — Grafana Cloud Free Tier (Recommended until MVP demo)

Use **[Grafana Cloud](https://grafana.com/products/cloud/)** free forever tier until the MVP is shown. It gives a hosted Loki + Grafana Explore UI with no Docker Compose to run.

| Phase | What to use | Why |
|-------|-------------|-----|
| Local / week 1 | JSON to stdout + rotating files (Option A) | Zero cost while building |
| Until MVP demo | **Grafana Cloud free tier** | Hosted UI, no infra, ~50 GB logs/month |
| After MVP | Self-host Loki (Option C) or stay on Grafana Cloud | Same LogQL either way |

**Why Grafana Cloud for the demo:**
- Free forever tier is enough for early traffic
- Open Grafana → Explore → filter by `request_id`, `component`, or `level=error`
- Same stack as self-hosted Loki (LogQL), without running compose on a laptop
- Ship with Grafana Alloy, Promtail, or HTTP push of JSON lines from stdout

**What not to do for the demo:**
- Files + `jq` alone — fine for engineers, weak on stage
- Self-hosted Loki / SigNoz before the demo — free but ops overhead before product is shown
- Paid APM (Datadog, etc.) — unnecessary until you need more than logs

**Practical path:**
1. Keep `zap` → JSON stdout (as configured in `pkg/logger/logger.go`)
2. Create a free Grafana Cloud stack
3. Point a small agent (Alloy / Promtail) at the app logs or stdout
4. Demo live Explore queries: `request_id`, payment errors, slow requests

**Verdict:** Best free option when you need to *show* logs at MVP. Switch to self-hosted Loki later if cost or data residency requires it.

---

### Option C — Grafana Loki Self-Hosted (Best Free OSS Platform — Recommended for MVP+)

Grafana Loki is the logging equivalent of Prometheus. It does **not** index the full log body (unlike Elasticsearch), which makes it very cheap to run. It stores logs as compressed chunks.

**Stack:**
- **Loki** — stores and indexes logs
- **Promtail** — agent that tails your log files and ships them to Loki
- **Grafana** — web UI for log search and dashboards

**Why Loki over ELK Stack:**
- ELK (Elasticsearch + Logstash + Kibana) needs 4–8 GB RAM minimum — too heavy for MVP
- Loki runs on 512 MB RAM
- Grafana is completely free and open source

#### Docker Compose: `deploy/docker-compose.logging.yml`

```yaml
version: "3"
services:
  loki:
    image: grafana/loki:3.0.0
    ports:
      - "3100:3100"
    volumes:
      - loki_data:/loki

  promtail:
    image: grafana/promtail:3.0.0
    volumes:
      - /path/to/hospital-backend/logs:/var/log/hospital
      - ./promtail-config.yaml:/etc/promtail/config.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana

volumes:
  loki_data:
  grafana_data:
```

#### Promtail Config: `deploy/promtail-config.yaml`

```yaml
server:
  http_listen_port: 9080

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: hospital-backend
    static_configs:
      - targets:
          - localhost
        labels:
          job: hospital-backend
          env: production
          __path__: /var/log/hospital/*.log
    pipeline_stages:
      - json:
          expressions:
            level: level
            component: component
            request_id: request_id
      - labels:
          level:
          component:
```

#### Example LogQL Queries in Grafana

```logql
# All errors from payment webhook
{job="hospital-backend"} | json | level="error" | component="webhook_service"

# Requests slower than 1 second
{job="hospital-backend"} | json | latency_ms > 1000000000

# Full trace for one request ID
{job="hospital-backend"} | json | request_id="a1b2-c3d4-e5f6"

# Payment failures in last 1 hour
{job="hospital-backend"} | json | level="error" | component=~"webhook_service|payment_service"
```

---

### Option D — SigNoz (Full Observability, OSS)

[SigNoz](https://signoz.io/) provides logs + metrics + distributed tracing in one platform, all open source.

```bash
git clone -b main https://github.com/SigNoz/signoz.git
cd signoz/deploy/
docker-compose up -d
```

Heavier than Loki but gives you more out of the box. Good step after MVP stabilises.

---

## 9. What a Complete Log Line Looks Like

**Development (console, human readable):**
```
22:45:12.345  INFO  payments/webhook_service.go:45  webhook signature verified
  {"request_id": "a1b2-c3d4", "component": "webhook_service", "provider": "razorpay"}
```

**Production (JSON, shipped to Loki):**
```json
{
  "ts": "2026-07-03T22:45:12.345+0530",
  "level": "error",
  "caller": "payments/webhook_service.go:71",
  "msg": "failed to update payment attempt",
  "request_id": "a1b2-c3d4-e5f6",
  "component": "webhook_service",
  "payment_attempt_id": "xyz-abc-...",
  "event_type": "payment_link.paid",
  "error": "pq: deadlock detected",
  "stacktrace": "hospital-backend/internal/payments..."
}
```

With this single error log line you immediately know: **which request** (`request_id`), **which service** (`component`), **which entity** (`payment_attempt_id`), **what happened** (`msg`), **why** (`error`). No guessing, no SSH digging.

---

## 10. Team Rules

1. **Never log PII** — no patient names, mobile numbers, email addresses, or passwords in log fields. Log IDs only (`patient_id`, `appointment_id`).
2. **Log errors once** — at the layer where they originate (repository or service). Wrap them upward with `fmt.Errorf("createWebhook: %w", err)` but do not re-log at every layer.
3. **Use `zap.Error(err)`** not `zap.String("error", err.Error())` — the former captures the full stack trace automatically.
4. **Do not log inside tight loops** — e.g. do not log each medicine in a bulk update; log the summary count and any individual failures only.
5. **`logger.Log.Fatal` is only allowed in `main.go`** — services must return errors, never call Fatal.
6. **`DEBUG` level is development only** — never add Debug logs in paths that run on every request in production.

---

## 11. Implementation Roadmap

### Phase 1 — Foundation (Day 1–2)
- [x] Create `pkg/logger/logger.go`
- [x] Add `LOG_LEVEL` and `ENV` to config / `.env`
- [x] Initialize logger in `cmd/main.go` before anything else
- [x] Add `RequestLogger` middleware and register it in `HandleMiddleware`
- [x] Replace `log.Fatalf` in `main.go` with `logger.Log.Fatal`

### Phase 2 — High-Risk Flows (Day 3–5)
- [ ] Add logging to `webhook_service.go` — payment processing is the highest business risk
- [ ] Add logging to `payments.services.go` — payment creation + transaction boundaries
- [ ] Add logging to `invoice.services.go` and `billing/controllers.go`
- [ ] Log all `db.Begin()`, `tx.Rollback()`, `tx.Commit()` points

### Phase 3 — Complete Coverage (Week 2)
- [ ] Appointment creation and slot validation
- [ ] Prescription and medicine inventory updates
- [ ] Notification service (background job)
- [ ] Add `organisation_id` and `patient_id` fields wherever those entities are handled

### Phase 4 — Observability (MVP demo → Post-MVP)
- [ ] Create Grafana Cloud free stack and ship JSON logs (Alloy / Promtail)
- [ ] Demo Explore queries: request_id, payment errors, slow requests
- [ ] After MVP: optionally deploy self-hosted Loki + Promtail + Grafana via Docker Compose
- [ ] Create dashboards: error rate per endpoint, slow requests, payment failure rate
- [ ] Set up Grafana alerting (free) for `level=error` spikes and payment failures
- [ ] Evaluate SigNoz for full traces

---

## 12. Decision Summary

| Decision            | Choice                         | Reason                                               |
|---------------------|--------------------------------|------------------------------------------------------|
| Package             | `go.uber.org/zap`              | Fastest, typed fields, battle-tested                |
| File rotation       | `lumberjack`                   | Effortless daily rotation, zero infra               |
| Context propagation | Fiber `c.Locals()` → method param | Explicit, matches existing code patterns      |
| Log platform local  | Rotating files + stdout        | Zero infra while building, searchable with `jq`     |
| Log platform MVP    | Grafana Cloud free tier        | Free hosted UI to show logs at demo; no Docker ops  |
| Log platform MVP+   | Grafana Loki self-hosted       | Free OSS, lightweight, same LogQL as Cloud          |
| Long term           | SigNoz                         | Full observability — logs + metrics + traces        |
