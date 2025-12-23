# 01 - Architecture & System Design

## Q1: อธิบาย Architecture ของระบบนี้หน่อย

**A:** ระบบใช้ **Microservices Architecture** แบ่งเป็น 5 services หลัก:

```
┌─────────────────┐
│   Frontend      │ (Next.js, TypeScript)
│   :3000         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   API Gateway   │ (Rate limiting, JWT validation, routing)
│   :8080         │
└────────┬────────┘
         │
    ┌────┴────┬─────────┬─────────┐
    ▼         ▼         ▼         ▼
┌────────┐┌────────┐┌────────┐┌────────┐
│ Auth   ││Ticket  ││Booking ││Payment │
│ :8081  ││ :8082  ││ :8083  ││ :8084  │
└────┬───┘└────┬───┘└────┬───┘└────┬───┘
     │         │         │         │
     ▼         ▼         ▼         ▼
  ┌──────────────────────────────────┐
  │         PostgreSQL (per-service) │
  └──────────────────────────────────┘
                  │
                  ▼
          ┌──────────────┐
          │    Redis     │ (Inventory + Lua Scripts)
          └──────────────┘
                  │
                  ▼
          ┌──────────────┐
          │   Redpanda   │ (Kafka-compatible)
          └──────────────┘
```

**Service Responsibilities:**

| Service | Port | หน้าที่ |
|---------|------|---------|
| API Gateway | 8080 | Rate limiting, JWT validation, routing |
| Auth Service | 8081 | User registration, login, JWT generation |
| Ticket Service | 8082 | Event/show/zone catalog (read-heavy, cached) |
| Booking Service | 8083 | Core booking logic, Redis Lua, Saga orchestration |
| Payment Service | 8084 | Stripe integration, payment processing |

**จาก codebase - `backend-booking/main.go:264-338`:**
```go
// API routes
v1 := router.Group("/api/v1")
{
    // Booking routes
    bookings := v1.Group("/bookings")
    bookings.POST("/reserve", container.BookingHandler.ReserveSeats)
    bookings.POST("/:id/confirm", container.BookingHandler.ConfirmBooking)

    // Queue routes - Virtual Queue for high-demand events
    queue := v1.Group("/queue")
    queue.POST("/join", container.QueueHandler.JoinQueue)
    queue.GET("/position/:event_id/stream", container.QueueHandler.StreamPosition)

    // Admin routes
    admin.POST("/sync-inventory", container.AdminHandler.SyncInventory)
}
```

---

## Q2: ทำไมใช้ Go แทน Node.js หรือ Java?

**A:** เลือก Go เพราะ:

| Criteria | Go | Node.js | Java |
|----------|-----|---------|------|
| Throughput | 50,000+ RPS | ~10,000 RPS | ~30,000 RPS |
| Latency (P99) | < 2ms | 10-50ms | 5-20ms |
| Memory | ~50MB | ~200MB | ~500MB |
| Startup time | < 100ms | 500ms-2s | 2-10s |
| Concurrency model | Goroutines (2KB) | Event loop (single-threaded) | Threads (1MB) |

**Key advantages:**
1. **Goroutines**: lightweight concurrency (2KB stack vs 1MB thread)
2. **No GIL**: ไม่มี Global Interpreter Lock เหมือน Python
3. **Compiled**: native binary ขนาดเล็ก ~10MB
4. **GC optimized**: Go 1.24 GC pause < 1ms

**จาก codebase - `backend-booking/main.go:30-32`:**
```go
func main() {
    // Optimize Go runtime for high concurrency
    runtime.GOMAXPROCS(runtime.NumCPU())  // ใช้ทุก CPU core
```

---

## Q3: Database-per-Service Pattern คืออะไร?

**A:** แต่ละ service มี database ของตัวเอง:

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ Auth Service│    │Ticket Svc   │    │Booking Svc  │    │Payment Svc  │
└──────┬──────┘    └──────┬──────┘    └──────┬──────┘    └──────┬──────┘
       │                  │                  │                  │
       ▼                  ▼                  ▼                  ▼
   ┌───────┐          ┌───────┐          ┌───────┐          ┌───────┐
   │auth_db│          │ticket │          │booking│          │payment│
   │       │          │_db    │          │_db    │          │_db    │
   └───────┘          └───────┘          └───────┘          └───────┘
```

**ข้อดี:**
- **Autonomy**: แก้ไข schema ได้อิสระ ไม่กระทบ service อื่น
- **Scalability**: scale database แยกกันได้
- **Technology diversity**: อาจใช้ MongoDB สำหรับ analytics ได้

**ข้อเสีย:**
- **Distributed transactions**: ซับซ้อน → แก้ด้วย Saga Pattern
- **Data consistency**: ใช้ Eventual Consistency
- **Query across services**: ต้อง API call ระหว่าง services

**จาก codebase - `backend-booking/main.go:77-99`:**
```go
// Uses BookingDatabase config (Microservice - each service has its own database)
dbCfg := &database.PostgresConfig{
    Host:     cfg.BookingDatabase.Host,
    Port:     cfg.BookingDatabase.Port,
    User:     cfg.BookingDatabase.User,
    Password: cfg.BookingDatabase.Password,
    Database: cfg.BookingDatabase.DBName,  // booking_db
    // ...
}
```

---

## Q4: Clean Architecture ใน Go service เป็นยังไง?

**A:** แบ่งเป็น 4 layers:

```
┌─────────────────────────────────────────────────┐
│                   Handler                        │  ← HTTP layer
│   - Parse request, validate input               │
│   - Call service, format response               │
├─────────────────────────────────────────────────┤
│                   Service                        │  ← Business logic
│   - Orchestrate operations                      │
│   - Apply business rules                        │
├─────────────────────────────────────────────────┤
│                  Repository                      │  ← Data access
│   - PostgreSQL queries                          │
│   - Redis operations                            │
├─────────────────────────────────────────────────┤
│                   Domain                         │  ← Entities
│   - Business entities                           │
│   - Domain errors                               │
└─────────────────────────────────────────────────┘
```

**Directory Structure:**
```
backend-booking/
├── main.go                    # Entry point
├── internal/
│   ├── di/
│   │   └── container.go      # Dependency injection
│   ├── handler/
│   │   └── booking_handler.go # HTTP handlers
│   ├── service/
│   │   └── booking_service.go # Business logic
│   ├── repository/
│   │   ├── postgres_booking_repo.go
│   │   └── redis_reservation_repo.go
│   ├── domain/
│   │   ├── booking.go        # Entities
│   │   └── errors.go         # Domain errors
│   └── dto/
│       └── booking_dto.go    # Request/Response DTOs
```

**จาก codebase - `backend-booking/internal/di/container.go`:**
```go
type Container struct {
    // Repositories
    BookingRepo     repository.BookingRepository
    ReservationRepo repository.ReservationRepository

    // Services
    BookingService *service.BookingService
    QueueService   *service.QueueService

    // Handlers
    BookingHandler *handler.BookingHandler
    QueueHandler   *handler.QueueHandler
}
```

---

## Q5: Dependency Injection ทำยังไงใน Go?

**A:** ใช้ Constructor Injection ผ่าน Container pattern:

**จาก codebase - `backend-booking/internal/di/container.go`:**
```go
// Container holds all dependencies
type Container struct {
    BookingHandler *handler.BookingHandler
    QueueHandler   *handler.QueueHandler
    AdminHandler   *handler.AdminHandler
    SagaHandler    *handler.SagaHandler
    HealthHandler  *handler.HealthHandler
}

func NewContainer(cfg *ContainerConfig) *Container {
    // 1. Create repositories
    bookingRepo := cfg.BookingRepo
    reservationRepo := cfg.ReservationRepo

    // 2. Create services (inject repositories)
    bookingService := service.NewBookingService(
        bookingRepo,
        reservationRepo,
        cfg.EventPublisher,
        cfg.ServiceConfig,
    )

    // 3. Create handlers (inject services)
    bookingHandler := handler.NewBookingHandler(
        bookingService,
        cfg.BookingHandlerConfig,
    )

    return &Container{
        BookingHandler: bookingHandler,
        // ...
    }
}
```

**ทำไมไม่ใช้ DI framework (Wire, Dig)?**
- Go philosophy: explicit is better than implicit
- Compile-time safety vs runtime reflection
- Easy to understand and debug

---

## Q6: API Gateway ทำอะไรบ้าง?

**A:** 4 หน้าที่หลัก:

```
Client Request
      │
      ▼
┌─────────────────────────────────┐
│         API Gateway             │
│  ┌───────────────────────────┐  │
│  │ 1. Rate Limiting          │  │  ← ป้องกัน DDoS, fair usage
│  │    - Per-user limits      │  │
│  │    - Per-endpoint limits  │  │
│  ├───────────────────────────┤  │
│  │ 2. JWT Validation         │  │  ← Verify token signature
│  │    - Extract claims       │  │
│  │    - Set X-User-ID header │  │
│  ├───────────────────────────┤  │
│  │ 3. Request Routing        │  │  ← Route to correct service
│  │    /bookings → :8083      │  │
│  │    /payments → :8084      │  │
│  ├───────────────────────────┤  │
│  │ 4. CORS, Security Headers │  │  ← Security policies
│  └───────────────────────────┘  │
└─────────────────────────────────┘
      │
      ▼
  Microservice
```

**Rate Limiting per endpoint:**
```go
var rateLimits = map[string]RateLimit{
    "/api/v1/bookings/reserve": {
        RequestsPerSecond: 100,  // Per user
        BurstSize:         10,
    },
    "/api/v1/auth/login": {
        RequestsPerSecond: 5,    // Prevent brute force
        BurstSize:         3,
    },
}
```

---

## Q7: Service-to-Service Communication เป็นแบบไหน?

**A:** ใช้ 2 แบบ:

**1. Synchronous (HTTP REST)**
- ใช้เมื่อต้องการ response ทันที
- เช่น Booking Service → Ticket Service (get zone info)

```go
// backend-booking/internal/service/zone_syncer.go
func (s *ZoneSyncer) SyncZone(ctx context.Context, zoneID string) error {
    url := fmt.Sprintf("%s/api/v1/zones/%s", s.ticketServiceURL, zoneID)
    resp, err := s.httpClient.Get(url)
    // ...
}
```

**2. Asynchronous (Kafka/Redpanda)**
- ใช้เมื่อไม่ต้องการ response ทันที
- เช่น Booking confirmed → Send notification

```go
// backend-booking/internal/service/event_publisher.go
func (p *KafkaEventPublisher) PublishBookingConfirmed(ctx context.Context, booking *domain.Booking) error {
    event := &BookingConfirmedEvent{
        BookingID: booking.ID,
        UserID:    booking.UserID,
        // ...
    }
    return p.producer.Produce(ctx, "booking.confirmed", event)
}
```

**Trade-offs:**

| Approach | Pros | Cons |
|----------|------|------|
| Synchronous | Simple, immediate response | Coupling, cascading failures |
| Asynchronous | Decoupled, resilient | Eventual consistency, debugging harder |

---

## Q8: ทำไมใช้ Redpanda แทน Kafka?

**A:** Redpanda = Kafka-compatible แต่ดีกว่าใน:

| Feature | Kafka | Redpanda |
|---------|-------|----------|
| Written in | Java/Scala | C++ |
| JVM dependency | Yes | No |
| Memory usage | 4GB+ | 500MB |
| Latency (P99) | 10-50ms | < 10ms |
| Zookeeper | Required (or KRaft) | Built-in Raft |
| Setup complexity | High | Low |

**Kafka-compatible:**
- ใช้ Kafka client libraries เดิมได้
- เปลี่ยน broker address อย่างเดียว

**จาก codebase - `backend-booking/main.go:127-141`:**
```go
// Initialize Kafka event publisher (works with Redpanda)
eventPubCfg := &service.EventPublisherConfig{
    Brokers:     cfg.Kafka.Brokers,  // redpanda:9092
    Topic:       "booking-events",
    ServiceName: "booking-service",
    ClientID:    cfg.Kafka.ClientID,
}
eventPublisher, err = service.NewKafkaEventPublisher(ctx, eventPubCfg)
```

**When to use real Kafka:**
- Enterprise features (tiered storage, exactly-once)
- Very high throughput (millions of msgs/sec)
- Existing Kafka expertise in team
