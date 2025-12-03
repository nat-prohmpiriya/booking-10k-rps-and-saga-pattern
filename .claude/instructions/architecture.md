# Architecture

## Service Layout
```
apps/
├── api-gateway/      # Entry point, routing, rate limiting, JWT validation
├── auth-service/     # User registration, login, token management
├── ticket-service/   # Event, show, zone management
├── booking-service/  # Core booking (Redis Lua + Kafka producer)
└── payment-service/  # Payment processing (Kafka consumer)
```

## Clean Architecture (per service)
```
internal/
├── domain/           # Entities, interfaces (zero dependencies)
├── service/          # Business logic (depends on domain only)
├── repository/       # Data access (implements domain interfaces)
├── handler/          # HTTP handlers (depends on service)
└── worker/           # Background workers (if needed)
```

**Dependency Rule:** domain ← service ← repository/handler (never reverse)

## Data Flow for Booking
```
Client → API Gateway → Booking Service → Redis (Lua Script)
                                      → Kafka (async)
                                      → Payment Service (consumer)
                                      → PostgreSQL (eventual)
```
