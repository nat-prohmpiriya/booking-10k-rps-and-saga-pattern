# Security & Observability

## Authentication
- JWT with RS256
- Access token: 15 min expiry
- Refresh token: 7 days expiry
- Store refresh tokens in Redis (revocable)

## Input Validation
- Validate ALL user inputs
- Use `validator` tags on request structs
- Sanitize before database operations

## Rate Limiting (Token Bucket + Burst)

| Endpoint | Rate | Burst |
|----------|------|-------|
| `/auth/login` | 5/min | 3 |
| `/auth/register` | 3/min | 2 |
| `/bookings/reserve` | 20/min | 10 |
| General | 100/min | 20 |

---

## Observability

### OpenTelemetry Spans
Every handler MUST create a span:
```go
ctx, span := tracer.Start(ctx, "BookingHandler.Reserve")
defer span.End()
```

### Required Span Attributes
- `user.id`
- `tenant.id`
- `booking.id` (when applicable)
- `show.id` (when applicable)

### Logging
- Use structured logging (zap)
- Always include: `request_id`, `user_id`, `trace_id`
- Log levels: DEBUG, INFO, WARN, ERROR
- Never log sensitive data (passwords, tokens, PII)
