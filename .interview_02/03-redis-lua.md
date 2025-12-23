# 03 - Redis Lua Scripts Deep Dive

## Q1: อธิบาย reserve_seats.lua แบบละเอียด

**A:** จาก `scripts/lua/reserve_seats.lua` (108 บรรทัด):

```lua
--[[
    Reserve Seats Lua Script
    ========================
    Atomically reserves seats for a booking.

    Key Structure:
    - KEYS[1]: zone:availability:{zone_id}           - Available seats count
    - KEYS[2]: user:reservations:{user_id}:{event_id} - User's total reserved
    - KEYS[3]: reservation:{booking_id}              - Reservation record (hash)

    Arguments:
    - ARGV[1]: quantity           - Number of seats to reserve
    - ARGV[2]: max_per_user       - Maximum seats allowed per user
    - ARGV[3]: user_id            - User ID
    - ARGV[4]: booking_id         - Booking ID
    - ARGV[5]: zone_id            - Zone ID
    - ARGV[6]: event_id           - Event ID
    - ARGV[7]: show_id            - Show ID
    - ARGV[8]: unit_price         - Price per seat
    - ARGV[9]: ttl_seconds        - Reservation TTL (default 600 = 10 min)
--]]

-- Parse inputs
local zone_availability_key = KEYS[1]
local user_reservations_key = KEYS[2]
local reservation_key = KEYS[3]

local quantity = tonumber(ARGV[1])
local max_per_user = tonumber(ARGV[2])
local ttl_seconds = tonumber(ARGV[9]) or 600
```

**Step-by-Step Flow:**

```
┌─────────────────────────────────────────────────────────────────┐
│ Step 1: Validate Quantity                                       │
├─────────────────────────────────────────────────────────────────┤
│ if quantity <= 0 then                                           │
│     return {0, "INVALID_QUANTITY", "..."}                       │
│ end                                                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Step 2: Check Zone Availability                                 │
├─────────────────────────────────────────────────────────────────┤
│ local available = redis.call("GET", zone_availability_key)      │
│ if not available then                                           │
│     return {0, "ZONE_NOT_FOUND", "..."}                         │
│ end                                                             │
│ if available < quantity then                                    │
│     return {0, "INSUFFICIENT_STOCK", "Available: " .. available}│
│ end                                                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Step 3: Check User Limit                                        │
├─────────────────────────────────────────────────────────────────┤
│ local user_reserved = redis.call("GET", user_reservations_key)  │
│ if (user_reserved + quantity) > max_per_user then               │
│     return {0, "USER_LIMIT_EXCEEDED", "..."}                    │
│ end                                                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Step 4: ATOMIC RESERVATION                                      │
├─────────────────────────────────────────────────────────────────┤
│ -- Deduct seats                                                 │
│ local remaining = redis.call("DECRBY", zone_key, quantity)      │
│                                                                 │
│ -- Track user reservations                                      │
│ redis.call("INCRBY", user_key, quantity)                        │
│ redis.call("EXPIRE", user_key, ttl_seconds + 60)                │
│                                                                 │
│ -- Create reservation record                                    │
│ redis.call("HSET", reservation_key,                             │
│     "booking_id", booking_id,                                   │
│     "user_id", user_id,                                         │
│     "quantity", quantity,                                       │
│     "status", "reserved",                                       │
│     "created_at", timestamp)                                    │
│                                                                 │
│ -- Set TTL for auto-cleanup                                     │
│ redis.call("EXPIRE", reservation_key, ttl_seconds)              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Step 5: Return Success                                          │
├─────────────────────────────────────────────────────────────────┤
│ return {1, remaining, new_user_reserved}                        │
└─────────────────────────────────────────────────────────────────┘
```

---

## Q2: EVALSHA vs EVAL ต่างกันยังไง?

**A:**

| Command | วิธีการ | Network | Performance |
|---------|---------|---------|-------------|
| EVAL | ส่ง script ทั้งตัวทุกครั้ง | High (script body) | Slow |
| EVALSHA | ส่ง SHA1 hash (40 chars) | Low | Fast |

**EVAL Flow:**
```
Client                          Redis
   │                              │
   │──── EVAL "script..." ──────→│ Parse script
   │     (1000+ bytes)            │ Compile
   │                              │ Cache by SHA
   │                              │ Execute
   │←─────── Result ──────────────│
```

**EVALSHA Flow:**
```
Client                          Redis
   │                              │
   │ At startup:                  │
   │──── SCRIPT LOAD "script" ──→│ Returns SHA
   │←─────── SHA: abc123... ─────│
   │                              │
   │ Every request:               │
   │──── EVALSHA abc123... ─────→│ Lookup cached
   │     (40 bytes only!)         │ Execute
   │←─────── Result ──────────────│
```

**จาก codebase - `backend-booking/internal/repository/redis_reservation_repo.go`:**
```go
// Pre-load at startup
func (r *RedisReservationRepo) LoadScripts(ctx context.Context) error {
    script := loadLuaScript("scripts/lua/reserve_seats.lua")

    // SCRIPT LOAD returns SHA
    sha, err := r.client.ScriptLoad(ctx, script).Result()
    if err != nil {
        return err
    }
    r.reserveScriptSHA = sha  // Cache SHA for later use
    return nil
}

// Use EVALSHA in production
func (r *RedisReservationRepo) ReserveSeats(ctx context.Context, req *ReserveRequest) error {
    result, err := r.client.EvalSha(ctx,
        r.reserveScriptSHA,  // Use cached SHA
        []string{zoneKey, userKey, bookingKey},
        quantity, maxPerUser, userID, /* ... */
    ).Result()
}
```

**จาก codebase - `backend-booking/main.go:167-178`:**
```go
// Pre-load Lua scripts into Redis
if err := reservationRepo.LoadScripts(ctx); err != nil {
    appLog.Warn(fmt.Sprintf("Failed to pre-load reservation Lua scripts: %v", err))
} else {
    appLog.Info("Reservation Lua scripts pre-loaded into Redis")
}
```

---

## Q3: Redis Keys ออกแบบยังไง?

**A:** Key naming convention:

```
┌─────────────────────────────────────────────────────────────┐
│                    KEY PATTERNS                              │
├─────────────────────────────────────────────────────────────┤
│ zone:availability:{zone_id}                                  │
│   → Integer: available seat count                           │
│   → Example: zone:availability:z-001 = 500                  │
│                                                              │
│ user:reservations:{user_id}:{event_id}                       │
│   → Integer: user's total reserved for this event           │
│   → Example: user:reservations:u-123:e-456 = 4              │
│   → TTL: reservation_ttl + 60s                              │
│                                                              │
│ reservation:{booking_id}                                     │
│   → Hash: reservation details                                │
│   → Fields: booking_id, user_id, quantity, status, etc.     │
│   → TTL: reservation_ttl (10 min default)                   │
│                                                              │
│ queue:{event_id}                                             │
│   → Sorted Set: FIFO queue by timestamp                     │
│   → Members: user_ids, Scores: join_timestamp               │
└─────────────────────────────────────────────────────────────┘
```

**Key Design Best Practices:**

| Practice | Example | Why |
|----------|---------|-----|
| Use colons as separator | `zone:availability:z-001` | Hierarchical, readable |
| Include type in name | `user:reservations:...` | Self-documenting |
| Short but descriptive | `zone:avail:z-001` → `zone:availability:z-001` | Clarity > brevity |
| Use UUIDs for IDs | `reservation:550e8400-...` | Globally unique |

---

## Q4: TTL Strategy ใช้ยังไง?

**A:** Dual-TTL strategy:

```
┌────────────────────────────────────────────────────────────┐
│              RESERVATION LIFECYCLE                          │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  T=0                T=10min              T=10min+60s        │
│   │                    │                      │             │
│   ▼                    ▼                      ▼             │
│ ┌────┐              ┌────┐                 ┌────┐           │
│ │User│ reserves     │Auto│ expire         │User│ limit     │
│ │    │ seats        │    │ reservation    │    │ reset     │
│ └────┘              └────┘                 └────┘           │
│                                                             │
│ Keys created:                                               │
│ - reservation:{id} → TTL: 10 min                           │
│ - user:reservations:{u}:{e} → TTL: 10 min + 60s            │
│                                                             │
│ Why +60s for user limit?                                    │
│ → Prevents user from immediately re-reserving              │
│ → Grace period for cleanup                                  │
└────────────────────────────────────────────────────────────┘
```

**จาก codebase - `scripts/lua/reserve_seats.lua:83-84`:**
```lua
-- 3. Set expiry on user reservation key (same as booking TTL + buffer)
redis.call("EXPIRE", user_reservations_key, ttl_seconds + 60)
```

**TTL Values:**
| Key | TTL | Purpose |
|-----|-----|---------|
| `reservation:{id}` | 10 min | Auto-release if not paid |
| `user:reservations:{u}:{e}` | 10 min + 60s | Prevent immediate re-reserve |
| `queue:{event_id}` | No TTL | Permanent during event |
| `idemp:{key}` PROCESSING | 60s | Allow retry if worker dies |
| `idemp:{key}` COMPLETED | 24h | Prevent long-term duplicates |

---

## Q5: release_seats.lua ทำอะไร?

**A:** Compensation script สำหรับ rollback:

```lua
--[[
    Release Seats Lua Script
    ========================
    Returns reserved seats back to inventory.

    Called when:
    1. Reservation TTL expires (automatic)
    2. User cancels booking
    3. Payment fails (saga compensation)
--]]

-- Get reservation details first
local reservation = redis.call("HGETALL", reservation_key)
if not reservation then
    return {0, "RESERVATION_NOT_FOUND"}
end

local quantity = tonumber(reservation.quantity)
local user_id = reservation.user_id
local event_id = reservation.event_id

-- 1. Return seats to zone availability
redis.call("INCRBY", zone_availability_key, quantity)

-- 2. Decrement user's reservation count
redis.call("DECRBY", user_reservations_key, quantity)

-- 3. Delete reservation record
redis.call("DEL", reservation_key)

return {1, "SUCCESS"}
```

**When is it called?**

```
┌────────────────────────────────────────────────┐
│ Automatic (TTL expires)                        │
│                                                │
│   reservation:{id} expires                     │
│         ↓                                      │
│   Seat Release Worker detects                  │
│         ↓                                      │
│   Calls release_seats.lua                      │
│         ↓                                      │
│   Seats returned to inventory                  │
└────────────────────────────────────────────────┘

┌────────────────────────────────────────────────┐
│ Manual (User cancels)                          │
│                                                │
│   POST /bookings/:id/cancel                    │
│         ↓                                      │
│   BookingService.Cancel()                      │
│         ↓                                      │
│   Calls release_seats.lua                      │
│         ↓                                      │
│   Booking status → cancelled                   │
└────────────────────────────────────────────────┘

┌────────────────────────────────────────────────┐
│ Saga Compensation (Payment fails)              │
│                                                │
│   Saga step "process-payment" fails            │
│         ↓                                      │
│   Saga orchestrator triggers compensation      │
│         ↓                                      │
│   Calls release_seats.lua                      │
│         ↓                                      │
│   Saga state → COMPENSATED                     │
└────────────────────────────────────────────────┘
```

---

## Q6: confirm_booking.lua ทำอะไร?

**A:** ทำให้ reservation เป็น permanent:

```lua
--[[
    Confirm Booking Lua Script
    ==========================
    Marks a reservation as confirmed after payment success.

    Actions:
    1. Remove TTL from reservation (make permanent)
    2. Update reservation status to "confirmed"
    3. Generate confirmation code
--]]

-- Verify reservation exists
local reservation = redis.call("HGETALL", reservation_key)
if not reservation or reservation.status ~= "reserved" then
    return {0, "INVALID_STATE"}
end

-- 1. Update status
redis.call("HSET", reservation_key, "status", "confirmed")
redis.call("HSET", reservation_key, "confirmed_at", timestamp)

-- 2. Remove TTL (make permanent)
redis.call("PERSIST", reservation_key)

-- 3. Remove from user reservation tracking (they've paid)
redis.call("DEL", user_reservations_key)

return {1, "CONFIRMED", confirmation_code}
```

**Lifecycle:**

```
reserved (TTL=10min) ──payment success──→ confirmed (no TTL)
         │
         │ TTL expires or
         │ payment fails
         ▼
     released/cancelled
```

---

## Q7: Lua Script Error Handling?

**A:** Error codes และ handling:

**Error Response Format:**
```lua
-- Success: {1, data...}
return {1, remaining_seats, user_reserved}

-- Error: {0, error_code, error_message}
return {0, "INSUFFICIENT_STOCK", "Not enough seats available. Available: 0, Requested: 2"}
```

**Error Codes:**
| Code | Meaning | HTTP Status |
|------|---------|-------------|
| `INVALID_QUANTITY` | quantity <= 0 | 400 |
| `ZONE_NOT_FOUND` | zone key doesn't exist | 404 |
| `INSUFFICIENT_STOCK` | not enough seats | 409 |
| `USER_LIMIT_EXCEEDED` | user reached max | 400 |
| `RESERVATION_NOT_FOUND` | booking doesn't exist | 404 |
| `INVALID_STATE` | wrong status for operation | 409 |

**Go Handling:**
```go
// backend-booking/internal/repository/redis_reservation_repo.go
func (r *RedisReservationRepo) ReserveSeats(ctx context.Context, req *ReserveRequest) (*ReserveResult, error) {
    result, err := r.client.EvalSha(ctx, r.reserveScriptSHA, keys, args...).Result()
    if err != nil {
        return nil, fmt.Errorf("redis error: %w", err)
    }

    // Parse Lua response
    res := result.([]interface{})
    success := res[0].(int64)

    if success == 0 {
        errorCode := res[1].(string)
        errorMsg := res[2].(string)

        switch errorCode {
        case "INSUFFICIENT_STOCK":
            return nil, domain.ErrInsufficientStock
        case "USER_LIMIT_EXCEEDED":
            return nil, domain.ErrUserLimitExceeded
        case "ZONE_NOT_FOUND":
            return nil, domain.ErrZoneNotFound
        default:
            return nil, fmt.Errorf("%s: %s", errorCode, errorMsg)
        }
    }

    return &ReserveResult{
        RemainingSeats: res[1].(int64),
        UserReserved:   res[2].(int64),
    }, nil
}
```

---

## Q8: Redis Cluster Support?

**A:** Lua scripts ต้องระวังเรื่อง key distribution:

**ปัญหา:**
```
Redis Cluster: 3 nodes
- Node 1: hash slots 0-5460
- Node 2: hash slots 5461-10922
- Node 3: hash slots 10923-16383

Lua script ใช้หลาย keys:
- zone:availability:z-001  → hash slot 1234 → Node 1
- user:reservations:u-123  → hash slot 8765 → Node 2
- reservation:b-456        → hash slot 12000 → Node 3

ERROR: All keys must be in same slot!
```

**วิธีแก้ - Hash Tags:**
```lua
-- Force all keys to same hash slot using {tag}
KEYS[1] = "zone:availability:{event_123}:z-001"
KEYS[2] = "user:reservations:{event_123}:u-123"
KEYS[3] = "reservation:{event_123}:b-456"

-- Redis only hashes content inside {}
-- All keys hash to same slot based on "event_123"
```

**โปรเจกต์นี้:**
- ใช้ single Redis instance (ไม่ใช่ cluster)
- ถ้า scale ต้องใช้ hash tags หรือ Redis sharding by zone_id
