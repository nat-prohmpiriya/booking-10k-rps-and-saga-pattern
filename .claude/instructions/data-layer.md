# Data Layer Conventions

## Database (PostgreSQL)

### Migration Naming
```
{sequence}_{service}_{action}_{table}.{up|down}.sql
```
Examples:
- `000001_auth_create_users.up.sql`
- `000002_booking_create_bookings.up.sql`
- `000003_ticket_create_events.up.sql`

### Table Naming
- Plural: `users`, `bookings`, `events`
- Join tables: `{table1}_{table2}` alphabetically

### Column Naming
- `snake_case`
- Primary key: `id` (UUID)
- Foreign keys: `{table_singular}_id`
- Timestamps: `created_at`, `updated_at`, `deleted_at`
- Soft delete: use `deleted_at` (nullable)

### Query Patterns
```go
// GOOD: Use pgx with context
row := db.QueryRow(ctx, "SELECT id, name FROM users WHERE id = $1", id)

// BAD: String concatenation (SQL injection risk)
query := "SELECT * FROM users WHERE id = '" + id + "'"
```

---

## Redis

### Key Naming
```
{service}:{entity}:{identifier}:{field}
```
Examples:
- `booking:show:uuid:available_seats`
- `booking:reservation:uuid:data`
- `auth:user:uuid:refresh_token`
- `ratelimit:ip:127.0.0.1:tokens`

### TTL Standards
| Key Type | TTL |
|----------|-----|
| Reservation | 10 minutes |
| Cache (events) | 5 minutes |
| Rate limit tokens | 1 minute |
| Refresh token | 7 days |
| Queue pass | 5 minutes |

### Lua Script Rules
- All seat operations MUST use Lua scripts (atomicity)
- Scripts location: `apps/booking-service/scripts/lua/`
- Always check and decrement in single script
- Return structured responses: `{success, message, data}`

---

## Kafka

### Topic Naming
```
{domain}.{event}
```
Examples:
- `booking.created`
- `booking.confirmed`
- `booking.expired`
- `payment.success`
- `payment.failed`

### Message Structure
```go
type KafkaMessage struct {
    ID        string    `json:"id"`         // Idempotency key
    Type      string    `json:"type"`       // Event type
    Timestamp time.Time `json:"timestamp"`
    Data      any       `json:"data"`
    Metadata  Metadata  `json:"metadata"`
}

type Metadata struct {
    TraceID   string `json:"trace_id"`
    UserID    string `json:"user_id"`
    TenantID  string `json:"tenant_id"`
}
```

### Consumer Rules
- Always commit after successful processing
- Implement idempotency (check processed IDs)
- Use dead letter queue for failed messages
- Max retries: 3 with exponential backoff
