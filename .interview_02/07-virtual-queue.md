# 07 - Virtual Queue System

## Q1: Virtual Queue คืออะไร? ทำไมต้องมี?

**A:** ระบบจัดคิวสำหรับ Flash Sale:

**ปัญหา - Thundering Herd:**
```
เวลา 10:00:00 - ตั๋วเปิดขาย

     500,000 users กด refresh พร้อมกัน
                    │
                    ▼
           ┌───────────────┐
           │  API Gateway  │ ← 500,000 RPS
           └───────────────┘
                    │
                    ▼
           ┌───────────────┐
           │    Server     │ ← 💥 CRASH
           └───────────────┘
```

**Virtual Queue Solution:**
```
     500,000 users กด "Join Queue"
                    │
                    ▼
           ┌───────────────┐
           │  Join Queue   │ ← O(1) operation
           └───────────────┘
                    │
                    ▼
           ┌───────────────────────────────┐
           │  Redis Sorted Set (FIFO)      │
           │  ┌─────────────────────────┐  │
           │  │ user_1 | 1702800000.001 │  │
           │  │ user_2 | 1702800000.002 │  │
           │  │ user_3 | 1702800000.003 │  │
           │  │ ...                     │  │
           │  └─────────────────────────┘  │
           └───────────────────────────────┘
                    │
                    ▼
           ┌───────────────┐
           │Queue Release  │ ← 500 users/second
           │   Worker      │
           └───────────────┘
                    │
                    ▼
           ┌───────────────┐
           │  User gets    │
           │  queue_pass   │ → Can book now
           └───────────────┘
```

---

## Q2: Queue Data Structure ใช้อะไร?

**A:** Redis Sorted Set (ZSET):

```
ZADD queue:{event_id} {timestamp} {user_id}

Example:
ZADD queue:evt-123 1702800000.001 "user-001"
ZADD queue:evt-123 1702800000.002 "user-002"
ZADD queue:evt-123 1702800000.003 "user-003"

Sorted Set (auto-sorted by score):
┌──────────────┬─────────────────┐
│   Member     │     Score       │
├──────────────┼─────────────────┤
│   user-001   │ 1702800000.001  │ ← First in queue
│   user-002   │ 1702800000.002  │
│   user-003   │ 1702800000.003  │
│   ...        │ ...             │
└──────────────┴─────────────────┘
```

**ทำไมใช้ Sorted Set?**

| Operation | Command | Complexity |
|-----------|---------|------------|
| Join queue | ZADD | O(log N) |
| Get position | ZRANK | O(log N) |
| Get queue size | ZCARD | O(1) |
| Remove from queue | ZREM | O(log N) |
| Get first N users | ZRANGE 0 N | O(log N + M) |

---

## Q3: join_queue.lua ทำอะไร?

**A:** Atomic queue join operation:

```lua
--[[
    Join Queue Lua Script
    =====================
    Adds user to virtual queue for an event.

    Keys:
    - KEYS[1]: queue:{event_id}

    Arguments:
    - ARGV[1]: user_id
    - ARGV[2]: max_queue_size (0 = unlimited)
    - ARGV[3]: current_timestamp
--]]

local queue_key = KEYS[1]
local user_id = ARGV[1]
local max_size = tonumber(ARGV[2]) or 0
local timestamp = tonumber(ARGV[3])

-- Check if user already in queue
local existing_score = redis.call("ZSCORE", queue_key, user_id)
if existing_score then
    -- User already in queue, return current position
    local position = redis.call("ZRANK", queue_key, user_id)
    return {1, "ALREADY_IN_QUEUE", position + 1}
end

-- Check queue size limit
if max_size > 0 then
    local current_size = redis.call("ZCARD", queue_key)
    if current_size >= max_size then
        return {0, "QUEUE_FULL", current_size}
    end
end

-- Add to queue with timestamp as score (FIFO ordering)
redis.call("ZADD", queue_key, timestamp, user_id)

-- Get position (0-indexed, convert to 1-indexed for UX)
local position = redis.call("ZRANK", queue_key, user_id)

-- Calculate estimated wait time
local queue_size = redis.call("ZCARD", queue_key)

return {1, "JOINED", position + 1, queue_size}
```

**จาก codebase - `backend-booking/main.go:299-316`:**
```go
// Queue routes - Virtual Queue for high-demand events
queue := v1.Group("/queue")
queue.Use(userIDMiddleware())
{
    // Join queue (requires authentication)
    queue.POST("/join", container.QueueHandler.JoinQueue)

    // Get current position in queue (polling - legacy)
    queue.GET("/position/:event_id", container.QueueHandler.GetPosition)

    // Stream position updates via SSE (reduces polling overhead by 50x)
    queue.GET("/position/:event_id/stream", container.QueueHandler.StreamPosition)

    // Leave queue
    queue.DELETE("/leave", container.QueueHandler.LeaveQueue)
}
```

---

## Q4: Queue Pass JWT ทำงานยังไง?

**A:** Temporary token สำหรับ booking:

**Flow:**
```
1. User joins queue
   └─→ Gets position #45,231

2. Queue Release Worker releases 500 users/second
   └─→ User #45,231 reached!

3. Generate Queue Pass JWT
   ┌────────────────────────────────────┐
   │ {                                   │
   │   "user_id": "user-123",           │
   │   "event_id": "evt-456",           │
   │   "type": "queue_pass",            │
   │   "exp": 1702800300 (5 min from now)│
   │ }                                   │
   └────────────────────────────────────┘

4. User makes booking request
   POST /bookings/reserve
   X-Queue-Pass: eyJhbGciOiJIUzI1NiIs...

5. Booking handler validates queue pass
   └─→ Valid + not expired → Allow booking
   └─→ Invalid/expired → 403 Forbidden
```

**จาก codebase - `backend-booking/internal/service/queue_service.go`:**
```go
func (s *QueueService) GenerateQueuePass(userID, eventID string) (string, error) {
    claims := jwt.MapClaims{
        "user_id":  userID,
        "event_id": eventID,
        "type":     "queue_pass",
        "exp":      time.Now().Add(5 * time.Minute).Unix(),
        "iat":      time.Now().Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(s.jwtSecret))
}

func (s *QueueService) ValidateQueuePass(tokenString string, eventID string) (*QueuePassClaims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return []byte(s.jwtSecret), nil
    })

    if err != nil || !token.Valid {
        return nil, ErrInvalidQueuePass
    }

    claims := token.Claims.(jwt.MapClaims)
    if claims["event_id"] != eventID {
        return nil, ErrQueuePassEventMismatch
    }

    return &QueuePassClaims{
        UserID:  claims["user_id"].(string),
        EventID: claims["event_id"].(string),
    }, nil
}
```

---

## Q5: SSE (Server-Sent Events) สำหรับ Queue Position?

**A:** Real-time position updates without polling:

**Polling (Old way):**
```
Client                          Server
   │                              │
   │── GET /queue/position ─────→│
   │←──── Position: 45,231 ──────│
   │                              │
   │    (wait 1 second)           │
   │                              │
   │── GET /queue/position ─────→│
   │←──── Position: 44,998 ──────│
   │                              │
   │    (repeat 45,000 times!)    │

Problem: 45,000 HTTP requests per user!
```

**SSE (New way):**
```
Client                          Server
   │                              │
   │── GET /queue/position/stream→│
   │    (connection stays open)   │
   │                              │
   │←── data: {"position": 45231}─│
   │←── data: {"position": 44998}─│
   │←── data: {"position": 44765}─│
   │←── ...                       │
   │←── data: {"status": "ready"} │
   │                              │
   │    (1 connection total!)     │

Benefit: 50x less overhead!
```

**จาก codebase - `backend-booking/main.go:307-309`:**
```go
// Stream position updates via SSE (reduces polling overhead by 50x)
queue.GET("/position/:event_id/stream", container.QueueHandler.StreamPosition)
```

**SSE Handler Implementation:**
```go
func (h *QueueHandler) StreamPosition(c *gin.Context) {
    eventID := c.Param("event_id")
    userID := c.GetString("user_id")

    // Set SSE headers
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")

    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-c.Request.Context().Done():
            return
        case <-ticker.C:
            position, err := h.queueService.GetPosition(c, userID, eventID)
            if err != nil {
                c.SSEvent("error", err.Error())
                return
            }

            if position == 0 {
                // User is out of queue (released or left)
                c.SSEvent("message", map[string]interface{}{
                    "status": "ready",
                    "message": "You can now book!",
                })
                return
            }

            c.SSEvent("message", map[string]interface{}{
                "position": position,
                "estimated_wait": position * 3, // 3 seconds per user
            })
            c.Writer.Flush()
        }
    }
}
```

---

## Q6: Queue Release Worker?

**A:** Background worker ปล่อย users จาก queue:

```go
// backend-booking/internal/worker/queue_release_worker.go

type QueueReleaseWorker struct {
    queueService    *service.QueueService
    releaseRate     int           // Users per second
    batchSize       int           // Users per batch
    releaseInterval time.Duration // Time between batches
}

func (w *QueueReleaseWorker) Start(ctx context.Context) {
    // Release 500 users per second = 50 users every 100ms
    ticker := time.NewTicker(w.releaseInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.releaseBatch(ctx)
        }
    }
}

func (w *QueueReleaseWorker) releaseBatch(ctx context.Context) {
    // Get events with active queues
    events := w.queueService.GetActiveQueueEvents(ctx)

    for _, eventID := range events {
        // Get first N users from queue
        users, _ := w.queueService.PopFromQueue(ctx, eventID, w.batchSize)

        for _, userID := range users {
            // Generate queue pass
            queuePass, _ := w.queueService.GenerateQueuePass(userID, eventID)

            // Notify user (via SSE or push notification)
            w.notifyUser(ctx, userID, queuePass)
        }
    }
}
```

**Release Configuration:**

| Setting | Value | Calculation |
|---------|-------|-------------|
| Release rate | 500/sec | Based on server capacity |
| Batch size | 50 | 500 / (1000ms / 100ms) |
| Interval | 100ms | 1 second / 10 batches |
| Queue pass TTL | 5 min | Time to complete booking |

**Flow Control:**
```
Available capacity: 10,000 RPS
Reserved for bookings: 5,000 RPS
Queue release rate: 500 users/second
Each user ~10 requests → 5,000 RPS

500 users/sec × 10 requests = 5,000 RPS (fits within capacity)
```
