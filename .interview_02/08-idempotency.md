# 08 - Idempotency & Error Handling

## Q1: Idempotency Key คืออะไร? ทำไมต้องมี?

**A:** Key สำหรับป้องกัน duplicate operations:

**ปัญหา - Double Charge:**
```
User clicks "Pay" button
        │
        ▼
┌───────────────┐    Network timeout!
│  POST /pay    │───────────────────→ ?
└───────────────┘
        │
        │ User clicks again (retry)
        ▼
┌───────────────┐
│  POST /pay    │───────────────────→ Success!
└───────────────┘

But... first request also succeeded!
Result: User charged TWICE! 💸💸
```

**Solution - Idempotency Key:**
```
User clicks "Pay" button
        │
        ▼
┌───────────────────────────────────┐
│  POST /pay                        │
│  X-Idempotency-Key: pay_user123_1 │
└───────────────────────────────────┘
        │
        ▼
┌───────────────────────────────────┐
│  Server checks:                   │
│  - Key "pay_user123_1" exists?    │
│  - No → Process + Store result    │
│  - Yes → Return stored result     │
└───────────────────────────────────┘
```

---

## Q2: Idempotency Middleware ทำงานยังไง?

**A:** จาก `pkg/middleware/idempotency.go`:

```go
func IdempotencyMiddleware(cfg *IdempotencyConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Extract idempotency key from header
        idempotencyKey := c.GetHeader("X-Idempotency-Key")
        if idempotencyKey == "" {
            // No key = not idempotent, proceed normally
            c.Next()
            return
        }

        redisKey := "idemp:" + idempotencyKey

        // 2. Check if we've seen this key before
        status, _ := cfg.Redis.Get(ctx, redisKey).Result()

        switch status {
        case "PROCESSING":
            // Another request is processing, wait or return 202
            c.JSON(202, gin.H{"status": "processing"})
            c.Abort()
            return

        case "": // Key doesn't exist
            // Mark as processing
            cfg.Redis.Set(ctx, redisKey, "PROCESSING", 60*time.Second)

            // Process request
            c.Next()

            // Store response
            if c.Writer.Status() == 200 {
                response := getResponseBody(c)
                cfg.Redis.Set(ctx, redisKey, response, 24*time.Hour)
            } else {
                // Failed, allow retry
                cfg.Redis.Del(ctx, redisKey)
            }

        default:
            // Key exists with cached response
            c.JSON(200, status) // Return cached response
            c.Abort()
            return
        }
    }
}
```

**จาก codebase - `backend-booking/main.go:281-289`:**
```go
// Configure idempotency middleware for write operations
idempotencyConfig := middleware.DefaultIdempotencyConfig(redisClient.Client())
idempotencyConfig.SkipPaths = []string{"/health", "/ready", "/metrics"}

{
    // Write operations with idempotency
    bookings.POST("/reserve", middleware.IdempotencyMiddleware(idempotencyConfig), container.BookingHandler.ReserveSeats)
    bookings.POST("/:id/confirm", middleware.IdempotencyMiddleware(idempotencyConfig), container.BookingHandler.ConfirmBooking)
    bookings.POST("/:id/cancel", middleware.IdempotencyMiddleware(idempotencyConfig), container.BookingHandler.CancelBooking)
}
```

---

## Q3: Dual-TTL Strategy?

**A:** 2 ระดับ TTL สำหรับ idempotency:

```
┌─────────────────────────────────────────────────────────────┐
│                    DUAL-TTL STRATEGY                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  State 1: PROCESSING                                       │
│  TTL: 60 seconds                                           │
│                                                             │
│  Purpose: ถ้า worker crash ระหว่าง process                  │
│           → Key expire → Allow retry                       │
│                                                             │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  State 2: COMPLETED (with response)                        │
│  TTL: 24 hours                                             │
│                                                             │
│  Purpose: ป้องกัน duplicate ในระยะยาว                       │
│           → User retry ได้ผลลัพธ์เดิม                        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Scenarios:**

| Scenario | Key State | Behavior |
|----------|-----------|----------|
| First request | None → PROCESSING | Process normally |
| Concurrent retry | PROCESSING | Return 202 Accepted |
| Retry after success | COMPLETED | Return cached response |
| Retry after failure | None (deleted) | Process again |
| Worker crash | PROCESSING (60s TTL) | Key expires → Retry allowed |

---

## Q4: Domain Errors vs Technical Errors?

**A:** แยก 2 ประเภท:

**Domain Errors (Business logic):**
```go
// backend-booking/internal/domain/errors.go

var (
    // Validation errors
    ErrInvalidQuantity    = errors.New("quantity must be positive")
    ErrInvalidZone        = errors.New("zone not found")

    // Business rule violations
    ErrInsufficientStock  = errors.New("not enough seats available")
    ErrUserLimitExceeded  = errors.New("user booking limit exceeded")
    ErrBookingExpired     = errors.New("booking reservation expired")

    // State errors
    ErrBookingNotFound    = errors.New("booking not found")
    ErrInvalidBookingState = errors.New("invalid booking state for this operation")
)
```

**Technical Errors:**
```go
// Internal errors - don't expose to user
var (
    errDatabaseConnection = errors.New("database connection failed")
    errRedisTimeout       = errors.New("redis operation timeout")
    errKafkaPublishFailed = errors.New("failed to publish event")
)
```

**HTTP Status Mapping:**

| Error Type | HTTP Status | User Message |
|------------|-------------|--------------|
| ErrInsufficientStock | 409 Conflict | "Sold out" |
| ErrUserLimitExceeded | 400 Bad Request | "Limit exceeded" |
| ErrBookingNotFound | 404 Not Found | "Booking not found" |
| ErrBookingExpired | 410 Gone | "Reservation expired" |
| Database error | 500 Internal | "Something went wrong" |
| Redis error | 503 Service Unavailable | "Please try again" |

**Handler Implementation:**
```go
func (h *BookingHandler) ReserveSeats(c *gin.Context) {
    result, err := h.service.Reserve(ctx, req)

    if err != nil {
        switch {
        case errors.Is(err, domain.ErrInsufficientStock):
            c.JSON(409, gin.H{
                "error":   "SOLD_OUT",
                "message": "Not enough seats available",
            })
        case errors.Is(err, domain.ErrUserLimitExceeded):
            c.JSON(400, gin.H{
                "error":   "LIMIT_EXCEEDED",
                "message": err.Error(),
            })
        case errors.Is(err, domain.ErrZoneNotFound):
            c.JSON(404, gin.H{
                "error":   "ZONE_NOT_FOUND",
                "message": "Zone not found",
            })
        default:
            // Log technical error internally
            logger.Error("Unexpected error", zap.Error(err))
            c.JSON(500, gin.H{
                "error":   "INTERNAL_ERROR",
                "message": "Something went wrong",
            })
        }
        return
    }

    c.JSON(200, result)
}
```

---

## Q5: Error Response Format?

**A:** Standardized error format:

```go
// pkg/response/error.go

type ErrorResponse struct {
    Error   string            `json:"error"`            // Error code
    Message string            `json:"message"`          // Human-readable message
    Details map[string]string `json:"details,omitempty"` // Additional context
    TraceID string            `json:"trace_id,omitempty"` // For debugging
}
```

**Examples:**

```json
// 409 Conflict - Sold out
{
    "error": "INSUFFICIENT_STOCK",
    "message": "Not enough seats available. Available: 5, Requested: 10",
    "details": {
        "available": "5",
        "requested": "10",
        "zone_id": "zone-123"
    },
    "trace_id": "abc123def456"
}

// 400 Bad Request - Validation error
{
    "error": "VALIDATION_ERROR",
    "message": "Invalid request",
    "details": {
        "quantity": "must be positive",
        "zone_id": "required"
    }
}

// 500 Internal Error - Technical issue
{
    "error": "INTERNAL_ERROR",
    "message": "Something went wrong. Please try again.",
    "trace_id": "xyz789abc123"
}
```

---

## Q6: Retry Strategy?

**A:** Client-side retry with exponential backoff:

**จาก `.docs/CLIENT_RETRY_GUIDELINES.md`:**

```javascript
// Recommended client-side retry logic

async function reserveWithRetry(request, maxRetries = 3) {
    const idempotencyKey = generateIdempotencyKey(request);

    for (let attempt = 0; attempt < maxRetries; attempt++) {
        try {
            const response = await fetch('/api/v1/bookings/reserve', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Idempotency-Key': idempotencyKey,
                },
                body: JSON.stringify(request),
            });

            if (response.status === 202) {
                // Request is processing, wait and retry
                await sleep(1000);
                continue;
            }

            if (response.ok) {
                return await response.json();
            }

            // Don't retry on client errors (4xx)
            if (response.status >= 400 && response.status < 500) {
                throw new ClientError(await response.json());
            }

            // Retry on server errors (5xx)
            throw new ServerError(response.status);

        } catch (error) {
            if (attempt === maxRetries - 1) throw error;

            // Exponential backoff: 1s, 2s, 4s
            const delay = Math.pow(2, attempt) * 1000;
            await sleep(delay);
        }
    }
}

function generateIdempotencyKey(request) {
    // Combine user ID, action, and request hash
    return `${userId}_reserve_${hash(request)}_${Date.now()}`;
}
```

**Retry Decision Matrix:**

| Status Code | Retry? | Reason |
|-------------|--------|--------|
| 200 OK | No | Success |
| 202 Accepted | Yes | Still processing |
| 400 Bad Request | No | Client error |
| 401 Unauthorized | No | Need re-auth |
| 403 Forbidden | No | Not allowed |
| 404 Not Found | No | Resource gone |
| 409 Conflict | Maybe | Check error code |
| 429 Too Many | Yes | Wait and retry |
| 500 Internal | Yes | Server issue |
| 502 Bad Gateway | Yes | Upstream issue |
| 503 Unavailable | Yes | Temporary |
| 504 Timeout | Yes | Slow response |
