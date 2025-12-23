# 02 - Concurrency & Race Conditions

## Q1: Race Condition ในระบบจองตั๋วคืออะไร?

**A:** Classic concurrency problem:

```
Timeline    User A                     User B
────────────────────────────────────────────────────
T1          GET available → 100
T2                                     GET available → 100
T3          SET available → 99
T4                                     SET available → 99 ❌
────────────────────────────────────────────────────
Result: ทั้ง 2 คนจองได้ แต่มีที่นั่งลดลงแค่ 1 → OVERSELLING!
```

**ปัญหา:**
- Check (read) และ Update (write) เป็น 2 operations แยกกัน
- ระหว่าง check กับ update มี "window" ที่คนอื่นแทรกได้
- เรียกว่า **TOCTOU** (Time-of-Check to Time-of-Use)

**วิธีแก้ในโปรเจกต์นี้:** ใช้ **Redis Lua Scripts**

```lua
-- scripts/lua/reserve_seats.lua (Line 75-78)
-- === ATOMIC RESERVATION ===

-- Check AND Deduct in single operation
local available = redis.call("GET", zone_availability_key)
if available < quantity then
    return {0, "INSUFFICIENT_STOCK", "Not enough seats"}
end
redis.call("DECRBY", zone_availability_key, quantity)  -- Atomic!
```

---

## Q2: ทำไม Redis Lua ถึงเป็น Atomic?

**A:** 3 เหตุผลหลัก:

**1. Redis เป็น Single-Threaded**
```
Request Queue: [Lua Script A] [GET cmd] [Lua Script B] [SET cmd]
                     │
                     ▼
              Single Thread
              processes one
              at a time
```
- ไม่มี concurrent execution ใน Redis
- Lua script รันจนจบก่อน request อื่น

**2. Lua Script = Atomic Transaction**
```lua
-- ทุก command ใน script นี้รันต่อเนื่องกัน
-- ไม่มี command อื่นแทรกได้
local available = redis.call("GET", key)    -- Step 1
if available < qty then return error end    -- Step 2
redis.call("DECRBY", key, qty)              -- Step 3
redis.call("HSET", booking_key, ...)        -- Step 4
redis.call("EXPIRE", booking_key, ttl)      -- Step 5
```

**3. All-or-Nothing Execution**
- ถ้า script มี error → ไม่มี partial update
- เหมือน database transaction

---

## Q3: ทำไมไม่ใช้ PostgreSQL Transaction แทน?

**A:** เปรียบเทียบ:

| Criteria | PostgreSQL TX | Redis Lua |
|----------|---------------|-----------|
| **Latency** | 5-20ms | < 1ms |
| **Throughput** | ~2,000 TPS | ~100,000 TPS |
| **Lock contention** | High (row locks) | None |
| **Connection pool** | Limited (~100) | Large (500+) |
| **Storage** | Disk (durable) | Memory (fast) |

**PostgreSQL Transaction ทำงานยังไง:**
```sql
BEGIN;
SELECT available_seats FROM zones WHERE id = ? FOR UPDATE;  -- Lock row
-- ถ้ามี 10,000 users พร้อมกัน → 9,999 users รอ lock
UPDATE zones SET available_seats = available_seats - ? WHERE id = ?;
COMMIT;
```

**ปัญหาที่ 10,000 RPS:**
- Row lock contention สูงมาก
- Connection pool exhausted
- Latency spike (P99 > 1s)

**Redis Lua:**
```lua
-- No locks, no waiting, no contention
local available = redis.call("GET", key)
redis.call("DECRBY", key, qty)
-- Done in < 1ms
```

---

## Q4: Optimistic vs Pessimistic Locking?

**A:** โปรเจกต์นี้ใช้แนวคิด **Optimistic** ผ่าน Redis Lua:

**Pessimistic Locking (Traditional):**
```sql
-- ล็อคก่อน แล้วค่อยทำงาน
SELECT * FROM seats WHERE zone_id = ? FOR UPDATE;  -- Lock!
-- ทุกคนต้องรอจนกว่า lock จะถูกปล่อย
UPDATE seats SET available = available - 1;
COMMIT;  -- Release lock
```
- **ปัญหา**: Lock contention, deadlocks, poor scalability

**Optimistic Approach (Redis Lua):**
```lua
-- ไม่ lock, ทำเลย, check ไปด้วย
local available = redis.call("GET", zone_key)
if available < quantity then
    return {0, "INSUFFICIENT_STOCK"}  -- Fail fast!
end
redis.call("DECRBY", zone_key, quantity)  -- ทำได้เลย
```
- **ข้อดี**: No waiting, no locks, fail fast

**เมื่อไหร่ใช้แบบไหน:**

| Scenario | Recommended |
|----------|-------------|
| High contention (hot seats) | Optimistic + Retry |
| Low contention | Either works |
| Long transactions | Pessimistic |
| Read-heavy | Optimistic |
| Write-heavy + conflicts | Pessimistic |

---

## Q5: Go Concurrency ใช้ยังไงในโปรเจกต์?

**A:** ใช้ Goroutines + Channels:

**1. Goroutines สำหรับ Async Operations:**
```go
// backend-booking/internal/handler/booking_handler.go
func (h *BookingHandler) ReserveSeats(c *gin.Context) {
    // Sync: Reserve seats (must complete)
    result, err := h.service.ReserveSeats(ctx, req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // Async: Delete queue pass (don't wait)
    go func() {
        _ = h.queueService.DeleteQueuePass(ctx, userID, eventID)
    }()

    c.JSON(200, result)  // Return immediately
}
```

**2. Worker Goroutines:**
```go
// backend-booking/internal/worker/seat_release_worker.go
func (w *SeatReleaseWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

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

**3. Connection Pool ก็ใช้ Goroutines:**
```go
// pkg/database/postgres.go
// pgxpool internally uses goroutines for connection management
pool, _ := pgxpool.NewWithConfig(ctx, config)
// Each request gets a connection from pool
// Returns to pool when done
```

---

## Q6: Context ใช้ยังไงสำหรับ Cancellation?

**A:** Go Context pattern:

**1. Timeout Context:**
```go
// backend-booking/main.go:377-378
// Give outstanding requests 30 seconds to complete
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

**2. Propagate through layers:**
```go
// Handler → Service → Repository
func (h *BookingHandler) ReserveSeats(c *gin.Context) {
    ctx := c.Request.Context()  // Get request context
    result, err := h.service.ReserveSeats(ctx, req)
}

func (s *BookingService) ReserveSeats(ctx context.Context, req *dto.ReserveRequest) {
    // Check if cancelled before expensive operation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    return s.reservationRepo.Reserve(ctx, req)
}
```

**3. Graceful Shutdown:**
```go
// backend-booking/main.go:371-384
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit  // Block until signal

appLog.Info("Shutting down server...")

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := srv.Shutdown(ctx); err != nil {
    appLog.Fatal(fmt.Sprintf("Server forced to shutdown: %v", err))
}
```

**Context Best Practices:**
1. Always pass context as first parameter
2. Never store context in struct
3. Use context.WithTimeout for external calls
4. Check ctx.Done() before expensive operations
