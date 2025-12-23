# 11 - Observability & Monitoring

## Q1: Three Pillars of Observability?

**A:** Logs, Metrics, Traces:

```
┌─────────────────────────────────────────────────────────────────┐
│                    OBSERVABILITY STACK                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────────────┐   ┌───────────────┐   ┌───────────────┐     │
│  │     LOGS      │   │    METRICS    │   │    TRACES     │     │
│  │               │   │               │   │               │     │
│  │  "What        │   │  "How many    │   │  "What path   │     │
│  │   happened?"  │   │   and how     │   │   did this    │     │
│  │               │   │   fast?"      │   │   request     │     │
│  │               │   │               │   │   take?"      │     │
│  │  ┌─────────┐  │   │  ┌─────────┐  │   │  ┌─────────┐  │     │
│  │  │  Zap    │  │   │  │Prometheus│ │   │  │  Tempo  │  │     │
│  │  │  JSON   │  │   │  │ Metrics │  │   │  │ Traces  │  │     │
│  │  └────┬────┘  │   │  └────┬────┘  │   │  └────┬────┘  │     │
│  │       │       │   │       │       │   │       │       │     │
│  │       ▼       │   │       ▼       │   │       ▼       │     │
│  │  ┌─────────┐  │   │  ┌─────────┐  │   │  ┌─────────┐  │     │
│  │  │  Loki   │  │   │  │Prometheus│ │   │  │  Tempo  │  │     │
│  │  │ Storage │  │   │  │  TSDB   │  │   │  │ Storage │  │     │
│  │  └────┬────┘  │   │  └────┬────┘  │   │  └────┬────┘  │     │
│  │       │       │   │       │       │   │       │       │     │
│  │       └───────┴───┴───────┴───────┴───┴───────┘       │     │
│  │                           │                           │     │
│  │                    ┌──────┴──────┐                    │     │
│  │                    │   Grafana   │                    │     │
│  │                    │  Dashboard  │                    │     │
│  │                    └─────────────┘                    │     │
│  │                                                       │     │
│  └───────────────────────────────────────────────────────┘     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Q2: OpenTelemetry Integration?

**A:** จาก `backend-booking/main.go:59-73`:

```go
// Initialize OpenTelemetry
telemetryCfg := &telemetry.Config{
    Enabled:        cfg.OTel.Enabled,
    ServiceName:    "booking-service",
    ServiceVersion: cfg.App.Version,
    Environment:    cfg.App.Environment,
    CollectorAddr:  cfg.OTel.CollectorAddr,  // otel-collector:4317
    SampleRatio:    cfg.OTel.SampleRatio,    // 0.1 = 10% sampling
}
if _, err := telemetry.Init(ctx, telemetryCfg); err != nil {
    appLog.Warn(fmt.Sprintf("Failed to initialize telemetry: %v", err))
} else if telemetryCfg.Enabled {
    appLog.Info(fmt.Sprintf("Telemetry initialized (collector: %s)", telemetryCfg.CollectorAddr))
}
defer telemetry.Shutdown(ctx)
```

**Tracing Middleware - `backend-booking/main.go:241-244`:**
```go
// Add OpenTelemetry tracing middleware if enabled
if cfg.OTel.Enabled {
    router.Use(telemetry.TracingMiddleware("booking-service"))
    router.Use(telemetry.TraceHeaderMiddleware())
}
```

**Manual Span Creation:**
```go
// pkg/telemetry/tracing.go

func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
    return otel.Tracer("booking-service").Start(ctx, name)
}

// Usage in service layer
func (s *BookingService) Reserve(ctx context.Context, req *dto.ReserveRequest) (*dto.ReserveResponse, error) {
    ctx, span := telemetry.StartSpan(ctx, "BookingService.Reserve")
    defer span.End()

    // Add attributes
    span.SetAttributes(
        attribute.String("user_id", req.UserID),
        attribute.String("event_id", req.EventID),
        attribute.Int("quantity", req.Quantity),
    )

    // Record errors
    result, err := s.reservationRepo.ReserveSeats(ctx, req)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }

    span.SetStatus(codes.Ok, "")
    return result, nil
}
```

---

## Q3: Structured Logging?

**A:** JSON logs with Zap:

**จาก `backend-booking/main.go:41-52`:**
```go
// Initialize logger with OTLP export support
logCfg := &logger.Config{
    Level:        cfg.App.Environment,
    ServiceName:  "booking-service",
    Development:  cfg.IsDevelopment(),
    OTLPEnabled:  cfg.OTel.Enabled && cfg.OTel.LogExportEnabled,
    OTLPEndpoint: cfg.OTel.CollectorAddr,
    OTLPInsecure: true,
}
if err := logger.Init(logCfg); err != nil {
    log.Fatalf("Failed to initialize logger: %v", err)
}
defer logger.Sync()
```

**Log Output Format:**
```json
{
  "level": "info",
  "ts": "2024-01-15T10:30:45.123Z",
  "caller": "service/booking_service.go:45",
  "msg": "Booking reserved",
  "service": "booking-service",
  "trace_id": "abc123def456",
  "span_id": "789xyz",
  "user_id": "user-123",
  "booking_id": "booking-456",
  "quantity": 2,
  "duration_ms": 15
}
```

**Correlation with Traces:**
```go
// Log with trace context
func (s *BookingService) Reserve(ctx context.Context, req *dto.ReserveRequest) {
    span := trace.SpanFromContext(ctx)
    traceID := span.SpanContext().TraceID().String()

    logger.Info("Booking reserved",
        zap.String("trace_id", traceID),
        zap.String("booking_id", result.BookingID),
        zap.String("user_id", req.UserID),
        zap.Int("quantity", req.Quantity),
    )
}
```

---

## Q4: Prometheus Metrics?

**A:** Custom metrics:

```go
// pkg/telemetry/metrics.go

var (
    // Counter: Total booking attempts
    BookingAttemptsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "booking",
            Name:      "attempts_total",
            Help:      "Total number of booking attempts",
        },
        []string{"status", "zone_id"},  // Labels
    )

    // Histogram: Booking request duration
    BookingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: "booking",
            Name:      "request_duration_seconds",
            Help:      "Booking request duration in seconds",
            Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1},
        },
        []string{"status"},
    )

    // Gauge: Current inventory levels
    InventoryGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "booking",
            Name:      "inventory_available",
            Help:      "Current available seats by zone",
        },
        []string{"zone_id", "event_id"},
    )
)

func init() {
    prometheus.MustRegister(BookingAttemptsTotal)
    prometheus.MustRegister(BookingDuration)
    prometheus.MustRegister(InventoryGauge)
}
```

**Usage:**
```go
func (s *BookingService) Reserve(ctx context.Context, req *dto.ReserveRequest) {
    timer := prometheus.NewTimer(BookingDuration.WithLabelValues("pending"))
    defer timer.ObserveDuration()

    result, err := s.reservationRepo.ReserveSeats(ctx, req)

    if err != nil {
        BookingAttemptsTotal.WithLabelValues("failed", req.ZoneID).Inc()
        return nil, err
    }

    BookingAttemptsTotal.WithLabelValues("success", req.ZoneID).Inc()
    InventoryGauge.WithLabelValues(req.ZoneID, req.EventID).Set(float64(result.Remaining))

    return result, nil
}
```

**Metrics Endpoint - `backend-booking/main.go:250-262`:**
```go
// Metrics endpoint for monitoring
router.GET("/metrics", func(c *gin.Context) {
    stats := db.Stats()
    c.JSON(http.StatusOK, gin.H{
        "db_pool": gin.H{
            "total_conns":        stats.TotalConns(),
            "acquired_conns":     stats.AcquiredConns(),
            "idle_conns":         stats.IdleConns(),
            "max_conns":          stats.MaxConns(),
        },
    })
})
```

---

## Q5: Grafana Dashboards?

**A:** Key dashboards:

**1. Service Overview:**
```
┌─────────────────────────────────────────────────────────────┐
│                    BOOKING SERVICE                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   10,234     │  │    15ms      │  │   0.02%      │       │
│  │     RPS      │  │  P99 Latency │  │  Error Rate  │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
│                                                             │
│  [========================================] Request Rate    │
│  [============================          ] Success Rate     │
│  [==                                    ] Error Rate       │
│                                                             │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                                                        │ │
│  │     Request Latency (P50, P95, P99)                   │ │
│  │                                                        │ │
│  │  100ms ┤                                              │ │
│  │   50ms ┤     ╭──╮                                     │ │
│  │   25ms ┤  ╭──╯  ╰──╮                                  │ │
│  │   10ms ┤──╯        ╰──────────────────────────────    │ │
│  │        └──────────────────────────────────────────    │ │
│  │         10:00    10:15    10:30    10:45    11:00     │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**2. Resource Utilization:**
```
┌─────────────────────────────────────────────────────────────┐
│                    RESOURCE METRICS                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  DB Connection Pool:                                        │
│  [==================] 18/20 connections in use              │
│                                                             │
│  Redis Pool:                                                │
│  [====              ] 120/500 connections in use            │
│                                                             │
│  Memory Usage:                                              │
│  [========          ] 180MB / 512MB (35%)                   │
│                                                             │
│  CPU Usage:                                                 │
│  [============      ] 60% average                           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**3. Business Metrics:**
```
┌─────────────────────────────────────────────────────────────┐
│                    BUSINESS METRICS                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Bookings Today:     12,456                                 │
│  Revenue Today:      $1,234,567                             │
│  Avg Ticket Price:   $99.12                                 │
│                                                             │
│  Top Events:                                                │
│  1. Taylor Swift Concert     - 5,234 bookings              │
│  2. NBA Finals Game 7        - 3,456 bookings              │
│  3. Hamilton Musical         - 2,345 bookings              │
│                                                             │
│  Booking Status Distribution:                               │
│  [====================] Confirmed: 85%                      │
│  [===                 ] Reserved:  12%                      │
│  [=                   ] Cancelled:  3%                      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Grafana Queries (PromQL):**
```promql
# Request rate
sum(rate(http_requests_total{service="booking-service"}[5m]))

# P99 latency
histogram_quantile(0.99,
  sum(rate(http_request_duration_seconds_bucket{service="booking-service"}[5m]))
  by (le)
)

# Error rate
sum(rate(http_requests_total{service="booking-service", status=~"5.."}[5m]))
/
sum(rate(http_requests_total{service="booking-service"}[5m]))

# Inventory levels
booking_inventory_available{zone_id=~".*"}
```
