# 05 - Zero Overselling Strategy

## Q1: Multi-Layer Defense Architecture คืออะไร?

**A:** 4 ชั้นป้องกัน overselling:

```
┌─────────────────────────────────────────────────────────────────┐
│                    MULTI-LAYER DEFENSE                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Layer 1: Redis Lua Scripts (Primary Guard)                    │
│  ─────────────────────────────────────────                     │
│  • Atomic check-and-deduct at memory speed                     │
│  • If Redis says "no seats" → absolutely enforced              │
│  • Every request validated BEFORE mutation                      │
│                                                                 │
│           │                                                     │
│           ▼                                                     │
│                                                                 │
│  Layer 2: Saga Compensation (Safety Net)                       │
│  ─────────────────────────────────────────                     │
│  • If payment fails → auto release seats                       │
│  • No dead stock (abandoned reservations)                      │
│  • Transactional guarantee across microservices                │
│                                                                 │
│           │                                                     │
│           ▼                                                     │
│                                                                 │
│  Layer 3: PostgreSQL CHECK Constraint (Hard Stop)              │
│  ─────────────────────────────────────────                     │
│  • Database rejects if seats < 0                               │
│  • Even if code has bugs, DB catches it                        │
│  • Last line of defense                                        │
│                                                                 │
│           │                                                     │
│           ▼                                                     │
│                                                                 │
│  Layer 4: Inventory Sync Worker (Watchdog)                     │
│  ─────────────────────────────────────────                     │
│  • Compare Redis vs PostgreSQL every 5 seconds                 │
│  • Alert if discrepancy exceeds threshold                      │
│  • Auto-reconciliation                                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Q2: Layer 1 - Redis Lua Guard?

**A:** Primary defense at memory speed:

```lua
-- scripts/lua/reserve_seats.lua (Line 52-62)

-- Get current available seats
local available = redis.call("GET", zone_availability_key)
if not available then
    return {0, "ZONE_NOT_FOUND", "Zone availability not initialized"}
end
available = tonumber(available)

-- Check seat availability BEFORE any mutation
if available < quantity then
    return {0, "INSUFFICIENT_STOCK",
        "Not enough seats available. Available: " .. available ..
        ", Requested: " .. quantity}
end

-- Only after validation: atomic deduction
local remaining = redis.call("DECRBY", zone_availability_key, quantity)
```

**ทำไมเป็น Primary Guard:**
- **Speed**: < 1ms response
- **Atomic**: Check + Deduct ใน operation เดียว
- **Absolute**: ถ้า Lua บอก "no" คือ "no" จริงๆ

**Failure Modes:**
| Scenario | Result | Protection |
|----------|--------|------------|
| Normal request | Check → Deduct → Success | ✅ |
| Concurrent requests | All check same atomic state | ✅ |
| Redis crash | Request fails, no deduction | ✅ |
| Network timeout | Retry with idempotency key | ✅ |

---

## Q3: Layer 2 - Saga Compensation?

**A:** Auto-rollback when steps fail:

```
Happy Path:
┌──────────┐    ┌──────────┐    ┌──────────┐
│ Reserve  │───→│ Payment  │───→│ Confirm  │
│   ✅     │    │   ✅     │    │   ✅     │
└──────────┘    └──────────┘    └──────────┘

Failure Path (Payment fails):
┌──────────┐    ┌──────────┐
│ Reserve  │───→│ Payment  │ ←── FAILED!
│   ✅     │    │   ❌     │
└──────────┘    └──────────┘
      ↑              │
      │   Compensate │
      └──────────────┘
      Release seats back

Failure Path (Confirm fails):
┌──────────┐    ┌──────────┐    ┌──────────┐
│ Reserve  │───→│ Payment  │───→│ Confirm  │ ←── FAILED!
│   ✅     │    │   ✅     │    │   ❌     │
└──────────┘    └──────────┘    └──────────┘
      ↑              ↑              │
      │   Compensate │   Compensate │
      └──────────────┴──────────────┘
    Release seats   Refund payment
```

**จาก codebase - `backend-booking/internal/saga/booking_saga.go:318-337`:**
```go
// Step 2: Process Payment - Compensate (Refund)
func (b *BookingSagaBuilder) processPaymentCompensate(ctx context.Context, data map[string]interface{}) error {
    sagaData := &BookingSagaData{}
    sagaData.FromMap(data)

    if sagaData.PaymentID == "" {
        // No payment was made, nothing to refund
        return nil
    }

    if err := b.config.PaymentService.RefundPayment(ctx, sagaData.PaymentID, "Booking saga compensation"); err != nil {
        return fmt.Errorf("failed to refund payment: %w", err)
    }

    return nil
}
```

**Key Properties:**
- **Automatic**: Orchestrator triggers compensation
- **Idempotent**: Safe to retry compensation
- **Ordered**: Reverse order execution

---

## Q4: Layer 3 - PostgreSQL CHECK Constraint?

**A:** Database-level hard stop:

```sql
-- scripts/migrations/ticket/000001_create_seat_zones.up.sql

CREATE TABLE seat_zones (
    id UUID PRIMARY KEY,
    show_id UUID NOT NULL REFERENCES shows(id),
    zone_name VARCHAR(100) NOT NULL,
    total_seats INT NOT NULL,
    available_seats INT NOT NULL,

    -- LAYER 3 DEFENSE: Hard stop at database level
    CONSTRAINT available_seats_not_negative
        CHECK (available_seats >= 0),

    CONSTRAINT available_not_exceed_total
        CHECK (available_seats <= total_seats)
);
```

**ถ้า code มี bug:**
```go
// Bug: ลืม check availability ก่อน deduct
err := db.Exec("UPDATE seat_zones SET available_seats = available_seats - 100 WHERE id = ?", zoneID)

// Database response:
// ERROR: new row for relation "seat_zones" violates check constraint
//        "available_seats_not_negative"
// DETAIL: Failing row contains (..., -50, ...).
```

**ทำไมสำคัญ:**
- Code มี bugs ได้
- Redis อาจ out of sync
- DB constraint ไม่มีทาง bypass (ยกเว้น DROP constraint)

---

## Q5: Layer 4 - Inventory Sync Worker?

**A:** Watchdog process สำหรับ reconciliation:

```go
// backend-booking/internal/service/zone_syncer.go

type InventorySyncer struct {
    redisClient    *redis.Client
    ticketService  TicketServiceClient
    alertService   AlertService
    syncInterval   time.Duration
    discrepancyThreshold int
}

func (s *InventorySyncer) Start(ctx context.Context) {
    ticker := time.NewTicker(s.syncInterval)  // Every 5 seconds

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.reconcile(ctx)
        }
    }
}

func (s *InventorySyncer) reconcile(ctx context.Context) {
    zones, _ := s.ticketService.GetAllZones(ctx)

    for _, zone := range zones {
        // Get Redis value
        redisValue, _ := s.redisClient.Get(ctx, "zone:availability:"+zone.ID).Int()

        // Get PostgreSQL value
        dbValue := zone.AvailableSeats

        // Check discrepancy
        diff := abs(redisValue - dbValue)
        if diff > s.discrepancyThreshold {
            s.alertService.Send(AlertCritical, fmt.Sprintf(
                "Inventory mismatch! Zone %s: Redis=%d, DB=%d, Diff=%d",
                zone.ID, redisValue, dbValue, diff,
            ))

            // Auto-reconciliation: Trust PostgreSQL as source of truth
            s.redisClient.Set(ctx, "zone:availability:"+zone.ID, dbValue, 0)
        }
    }
}
```

**Reconciliation Strategy:**

| Scenario | Action |
|----------|--------|
| Redis > DB | Trust DB, reduce Redis (เผื่อ Redis ไม่ได้ deduct) |
| Redis < DB | Trust DB, increase Redis (เผื่อ seats released แต่ Redis ไม่ update) |
| Small diff (< threshold) | Ignore, may be in-flight transactions |
| Large diff | Alert + Auto-fix |

**จาก codebase - Admin endpoint สำหรับ manual sync:**
```go
// backend-booking/main.go:318-326
admin := v1.Group("/admin")
{
    // Sync zone availability from PostgreSQL to Redis
    admin.POST("/sync-inventory", container.AdminHandler.SyncInventory)

    // Get inventory status (PostgreSQL vs Redis)
    admin.GET("/inventory-status", container.AdminHandler.GetInventoryStatus)
}
```

---

## Bonus: ถ้า Redis Crash?

**A:** Graceful degradation:

```
Normal Flow:
Request → Redis Lua → Success

Redis Down:
Request → Redis Lua → Connection Error
                ↓
       ┌────────────────────────────┐
       │ Option 1: Fast Fail       │
       │ Return 503, user retries  │
       └────────────────────────────┘

       ┌────────────────────────────┐
       │ Option 2: Fallback to DB  │
       │ Slower but works          │
       │ (Not implemented yet)     │
       └────────────────────────────┘

       ┌────────────────────────────┐
       │ Option 3: Queue Mode      │
       │ Accept to Kafka, process  │
       │ when Redis recovers       │
       └────────────────────────────┘

Redis Recovers:
1. Inventory Syncer runs
2. Sync PostgreSQL → Redis
3. Resume normal operations
```

**Current Implementation:**
โปรเจกต์นี้ใช้ Option 1 (Fast Fail) + Recovery via sync

```go
// backend-booking/main.go:117-124
redisClient, err = pkgredis.NewClient(ctx, redisCfg)
if err != nil {
    appLog.Fatal(fmt.Sprintf("Redis connection failed: %v", err))
    // Service won't start without Redis
    // This is intentional: Redis is critical for consistency
}
```

**Trade-off:**
- Availability ลดลงถ้า Redis ล่ม
- แต่ได้ Consistency 100% (no overselling)
- เหมาะสำหรับ ticket booking ที่ consistency สำคัญกว่า availability
