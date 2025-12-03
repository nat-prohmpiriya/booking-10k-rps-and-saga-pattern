# Booking Rush 10k RPS

High-performance ticket booking microservices system designed to handle **10,000 RPS** without overselling.

## Key Goals
- 10,000 RPS on `/bookings/reserve` endpoint
- Zero overselling (atomic seat reservation)
- Server P99 latency < 50ms
- E2E P99 latency < 200ms

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.23 |
| Framework | Gin |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Message Broker | Kafka 3.6 |
| Observability | OpenTelemetry |

**Libraries:** `pgx/v5`, `go-redis/v9`, `segmentio/kafka-go`, `viper`, `zap`, `validator/v10`

## Quick Commands
```bash
make dev          # Start infrastructure
make test         # Run tests
make lint         # Run linter
make migrate-up   # Run migrations
```

## Documentation
- [Specification](.docs/01-spec.md)
- [Development Roadmap](.docs/02-task.md)
- [Known Issues](.docs/03-issue.md)

## Detailed Instructions
See `.claude/instructions/` for detailed conventions:
- `architecture.md` - Service layout & data flow
- `code-conventions.md` - Naming & error handling
- `data-layer.md` - Database, Redis, Kafka patterns
- `api-conventions.md` - REST API standards
- `performance.md` - Performance rules
- `testing.md` - Testing conventions
- `security.md` - Security & observability
