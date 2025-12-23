# 06 - Performance Optimization

## Q1: ระบบรองรับ 10,000 RPS ได้ยังไง?

**A:** หลายเทคนิครวมกัน:

```
┌─────────────────────────────────────────────────────────────────┐
│                 10,000 RPS ARCHITECTURE                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Go + Gin Framework                                          │
│     • Goroutines: 2KB stack vs 1MB thread                      │
│     • No GC pause spikes                                       │
│     • Native binary, fast startup                              │
│                                                                 │
│  2. Redis Lua Scripts                                           │
│     • Sub-millisecond operations                               │
│     • No network round-trips between commands                  │
│     • Single-threaded = no lock contention                     │
│                                                                 │
│  3. Connection Pooling                                          │
│     • PostgreSQL: 20 connections, reused                       │
│     • Redis: 500 connections pool                              │
│     • No connection overhead per request                       │
│                                                                 │
│  4. Async Processing                                            │
│     • Reserve → return immediately                             │
│     • Confirmation, notification → Kafka async                 │
│                                                                 │
│  5. Virtual Queue                                               │
│     • Control traffic, prevent burst                           │
│     • Release 500 users/second                                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Q2: Connection Pooling Configuration?

**A:** Optimized pool settings:

**PostgreSQL Pool - `backend-booking/main.go:78-93`:**
```go
dbCfg := &database.PostgresConfig{
    Host:            cfg.BookingDatabase.Host,
    Port:            cfg.BookingDatabase.Port,
    User:            cfg.BookingDatabase.User,
    Password:        cfg.BookingDatabase.Password,
    Database:        cfg.BookingDatabase.DBName,

    // Pool settings optimized for 10k RPS
    MaxConns:        20,               // Max connections in pool
    MinConns:        5,                // Keep 5 ready
    MaxConnLifetime: 30 * time.Minute, // Refresh connections
    MaxConnIdleTime: 5 * time.Minute,  // Close idle sooner
    ConnectTimeout:  5 * time.Second,  // Fast fail
    MaxRetries:      3,
    RetryInterval:   1 * time.Second,
}
```

**Redis Pool - `backend-booking/main.go:102-118`:**
```go
redisCfg := &pkgredis.Config{
    Host:          cfg.Redis.Host,
    Port:          cfg.Redis.Port,
    Password:      cfg.Redis.Password,

    // Large pool for 10k RPS
    PoolSize:      500,               // 500 connections
    MinIdleConns:  100,               // Keep 100 ready
    MaxRetries:    3,
    RetryInterval: 100 * time.Millisecond,
    DialTimeout:   5 * time.Second,
    ReadTimeout:   3 * time.Second,
    WriteTimeout:  3 * time.Second,
    PoolTimeout:   4 * time.Second,
}
```

**Why these numbers?**

| Setting | Value | Reason |
|---------|-------|--------|
| PostgreSQL MaxConns: 20 | Low | Virtual Queue controls traffic, Redis handles hot path |
| Redis PoolSize: 500 | High | Every request hits Redis for reservation |
| MinIdleConns | 5/100 | Pre-warmed connections, no cold start |
| MaxConnIdleTime: 5min | Short | Close stale connections faster |

---

## Q3: HTTP Server Tuning?

**A:** จาก `backend-booking/main.go:340-351`:

```go
srv := &http.Server{
    Addr:              addr,
    Handler:           router,
    ReadTimeout:       5 * time.Second,   // Max time to read request
    WriteTimeout:      0,                  // Disabled for SSE!
    IdleTimeout:       120 * time.Second, // Keep-alive timeout
    ReadHeaderTimeout: 2 * time.Second,   // Header read timeout
    MaxHeaderBytes:    1 << 20,           // 1MB max header size
}
```

**Why WriteTimeout = 0?**
```go
// SSE (Server-Sent Events) for queue position updates
router.GET("/queue/position/:event_id/stream", handler.StreamPosition)

// SSE requires long-lived connections
// If WriteTimeout > 0, connection gets killed after timeout
// Solution: Disable WriteTimeout for SSE endpoints
```

**Alternative: Per-route timeout:**
```go
// For non-SSE routes, use middleware timeout
router.Use(timeout.New(timeout.WithTimeout(10 * time.Second)))

// Skip timeout for SSE routes
router.GET("/stream", skipTimeoutMiddleware, handler.StreamPosition)
```

---

## Q4: Goroutine Optimization?

**A:** จาก `backend-booking/main.go:30-32`:

```go
func main() {
    // Optimize Go runtime for high concurrency
    runtime.GOMAXPROCS(runtime.NumCPU())
```

**Goroutine Best Practices ในโปรเจกต์:**

**1. Async non-critical operations:**
```go
// Don't wait for queue pass deletion
go func() {
    _ = h.queueService.DeleteQueuePass(ctx, userID, eventID)
}()

c.JSON(200, result)  // Return immediately
```

**2. Worker pools for background tasks:**
```go
// Seat release worker
func (w *SeatReleaseWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.processExpiredReservations(ctx)
        }
    }
}
```

**3. Context cancellation:**
```go
// Graceful shutdown with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := srv.Shutdown(ctx); err != nil {
    // Force shutdown after 30s
}
```

---

## Q5: Middleware Ordering for Performance?

**A:** Fast path first, expensive operations later:

```go
// backend-booking/main.go:231-244
gin.SetMode(gin.ReleaseMode)  // No debug overhead
gin.DisableConsoleColor()

router := gin.New()

// Minimal middleware stack
router.Use(gin.Recovery())                    // 1. Panic recovery (essential)

if cfg.OTel.Enabled {
    router.Use(telemetry.TracingMiddleware()) // 2. Tracing (if enabled)
    router.Use(telemetry.TraceHeaderMiddleware())
}

// Note: No logging middleware in hot path!
// Logging happens in service layer with structured logger
```

**Middleware Performance Impact:**

| Middleware | Latency Added | When to Use |
|------------|---------------|-------------|
| Recovery | < 0.1ms | Always (essential) |
| Tracing | 0.1-0.5ms | Production with sampling |
| Request logging | 0.5-2ms | Development only |
| Rate limiting | 0.1-0.5ms | At API Gateway |
| Auth validation | 0.5-1ms | At API Gateway |

---

## Q6: pprof Profiling?

**A:** Built-in profiling server:

**จาก `backend-booking/main.go:7-8, 353-360`:**
```go
import (
    _ "net/http/pprof"  // Import pprof for profiling
)

// Start pprof server on separate port for profiling
go func() {
    pprofAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port+1000)
    appLog.Info(fmt.Sprintf("pprof server listening on %s", pprofAddr))
    if err := http.ListenAndServe(pprofAddr, nil); err != nil {
        appLog.Error(fmt.Sprintf("pprof server error: %v", err))
    }
}()
```

**Usage:**
```bash
# CPU profiling (30 seconds)
go tool pprof http://localhost:9083/debug/pprof/profile?seconds=30

# Memory profiling
go tool pprof http://localhost:9083/debug/pprof/heap

# Goroutine dump
go tool pprof http://localhost:9083/debug/pprof/goroutine

# Block profiling (for lock contention)
go tool pprof http://localhost:9083/debug/pprof/block
```

**Common issues found via pprof:**
- Goroutine leaks
- Memory allocation hotspots
- Lock contention
- Slow JSON serialization

---

## Q7: Database Query Optimization?

**A:** Indexing และ query patterns:

**Indexes:**
```sql
-- scripts/migrations/booking/000001_create_bookings.up.sql

CREATE INDEX idx_bookings_user_id ON bookings(user_id);
CREATE INDEX idx_bookings_event_id ON bookings(event_id);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_bookings_idempotency_key ON bookings(idempotency_key);

-- Partial index for expired reservations cleanup
CREATE INDEX idx_bookings_pending_expired
    ON bookings(reservation_expires_at)
    WHERE status = 'reserved';
```

**Query Patterns:**
```go
// ✅ Good: Use index
query := `SELECT * FROM bookings
          WHERE user_id = $1 AND status = 'confirmed'
          ORDER BY created_at DESC LIMIT 10`

// ❌ Bad: Full table scan
query := `SELECT * FROM bookings
          WHERE LOWER(confirmation_code) = LOWER($1)`
```

**Batch Operations:**
```go
// ✅ Batch insert instead of loop
query := `INSERT INTO bookings (id, user_id, ...) VALUES
          ($1, $2, ...),
          ($3, $4, ...),
          ($5, $6, ...)`

// ❌ N+1 problem
for _, booking := range bookings {
    db.Exec("INSERT INTO bookings ...")
}
```

---

## Q8: Caching Strategy?

**A:** Multi-level caching:

```
┌─────────────────────────────────────────────────────────────────┐
│                    CACHING LAYERS                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Layer 1: Redis (Inventory)                                    │
│  ─────────────────────────────                                 │
│  • zone:availability:{id} = available seats                    │
│  • Sub-millisecond reads                                       │
│  • Write-through on booking                                    │
│                                                                 │
│  Layer 2: Redis (Session/Queue)                                │
│  ─────────────────────────────                                 │
│  • queue:{event_id} = ZSET of users                            │
│  • reservation:{booking_id} = booking details                  │
│                                                                 │
│  Layer 3: CDN (Static content)                                 │
│  ─────────────────────────────                                 │
│  • Event images, assets                                        │
│  • HTML pages (SSG)                                            │
│                                                                 │
│  NOT Cached (Consistency critical):                            │
│  ─────────────────────────────────                             │
│  • Booking status (always from DB)                             │
│  • Payment status (always from Stripe)                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Cache Invalidation:**
```go
// When booking confirmed: update both Redis and PostgreSQL
func (s *BookingService) Confirm(ctx context.Context, bookingID string) error {
    // 1. Update PostgreSQL
    err := s.bookingRepo.UpdateStatus(ctx, bookingID, "confirmed")

    // 2. Update Redis (remove TTL, make permanent)
    err = s.redisClient.Persist(ctx, "reservation:"+bookingID)

    // 3. Sync inventory (PostgreSQL is source of truth)
    s.inventorySyncer.Sync(ctx, booking.ZoneID)

    return nil
}
```

**Trade-offs:**

| Approach | Consistency | Performance | Complexity |
|----------|-------------|-------------|------------|
| Write-through | Strong | Good | Medium |
| Write-behind | Eventual | Best | High |
| Cache-aside | Eventual | Good | Low |

โปรเจกต์นี้ใช้ **Write-through** สำหรับ inventory เพราะ consistency สำคัญ
