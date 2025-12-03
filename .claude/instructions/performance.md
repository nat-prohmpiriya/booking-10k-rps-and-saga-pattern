# Performance Rules

## MUST DO
- Use connection pooling (pgx, redis)
- Use batch operations where possible
- Use Redis Lua scripts for atomic operations
- Add indexes for frequently queried columns
- Use pagination for list endpoints (max 100 items)
- Cache hot data in Redis (events, shows)
- Use prepared statements

## MUST NOT DO
- N+1 queries
- SELECT * (always specify columns)
- Unbounded queries (always LIMIT)
- Blocking operations in hot paths
- Synchronous external calls in booking flow
- Large payloads in Kafka messages

## Connection Pool Settings

### PostgreSQL
```go
MaxOpenConns: 100
MaxIdleConns: 10
ConnMaxLifetime: 1 * time.Hour
```

### Redis
```go
PoolSize: 100
MinIdleConns: 10
```
