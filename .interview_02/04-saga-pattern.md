# 04 - Saga Pattern & Distributed Transactions

## Q1: Saga Pattern คืออะไร? ทำไมต้องใช้?

**A:** Pattern สำหรับ distributed transactions ข้าม microservices

**ปัญหา - Distributed Transaction:**
```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Booking   │    │   Payment   │    │Notification │
│   Service   │    │   Service   │    │   Service   │
└──────┬──────┘    └──────┬──────┘    └──────┬──────┘
       │                  │                  │
       ▼                  ▼                  ▼
   ┌───────┐          ┌───────┐          ┌───────┐
   │booking│          │payment│          │notif  │
   │_db    │          │_db    │          │_db    │
   └───────┘          └───────┘          └───────┘

ปัญหา: ถ้า Payment สำเร็จ แต่ Notification ล้มเหลว
       → จะ rollback Payment ยังไง?
       → ไม่มี single transaction ข้าม databases
```

**Saga Solution:**
```
แบ่ง transaction เป็น steps
แต่ละ step มี Execute + Compensate (undo)

Step 1: Reserve Seats
  Execute: Lock seats in Redis
  Compensate: Release seats back

Step 2: Charge Payment
  Execute: Stripe.charge()
  Compensate: Stripe.refund()

Step 3: Confirm Booking
  Execute: Mark as confirmed
  Compensate: Mark as cancelled

ถ้า Step 3 fail → Compensate Step 2 → Compensate Step 1
```

---

## Q2: Orchestration vs Choreography?

**A:** 2 รูปแบบของ Saga:

**Choreography (Event-driven):**
```
┌─────────┐  booking.reserved  ┌─────────┐  payment.success  ┌─────────┐
│ Booking │ ──────────────────→│ Payment │ ─────────────────→│  Notif  │
│ Service │                    │ Service │                   │ Service │
└─────────┘                    └─────────┘                   └─────────┘
     ↑                              │
     │     payment.failed           │
     └──────────────────────────────┘

ข้อดี: Decoupled, no single point of failure
ข้อเสีย: ยาก debug, ไม่เห็น flow ชัด, circular dependencies
```

**Orchestration (Central coordinator):**
```
                    ┌─────────────────┐
                    │      Saga       │
                    │  Orchestrator   │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
         ▼                   ▼                   ▼
    ┌─────────┐         ┌─────────┐         ┌─────────┐
    │ Booking │         │ Payment │         │  Notif  │
    │ Service │         │ Service │         │ Service │
    └─────────┘         └─────────┘         └─────────┘

ข้อดี: เห็น flow ชัด, debug ง่าย, centralized logic
ข้อเสีย: Single point of failure (mitigate with HA)
```

**โปรเจกต์นี้ใช้ Orchestration**

---

## Q3: Saga State Machine?

**A:** States และ Transitions:

```
                    ┌─────────┐
                    │ PENDING │
                    └────┬────┘
                         │ start()
                         ▼
                    ┌─────────┐
              ┌─────│ RUNNING │─────┐
              │     └────┬────┘     │
              │          │          │
         step fails   all steps   timeout
              │       succeed       │
              ▼          │          ▼
       ┌────────────┐    │    ┌──────────┐
       │COMPENSATING│    │    │ TIMEOUT  │
       └─────┬──────┘    │    └────┬─────┘
             │           │         │
     all compensated     │    try compensate
             │           │         │
             ▼           ▼         ▼
       ┌──────────┐ ┌──────────┐ ┌────────┐
       │COMPENSATED│ │COMPLETED │ │ FAILED │
       └──────────┘ └──────────┘ └────────┘
```

**จาก codebase - `backend-booking/internal/saga/booking_saga.go:38-58`:**
```go
const (
    // Saga names
    BookingSagaName     = "booking-saga"
    PostPaymentSagaName = "post-payment-saga"

    // Step names
    StepReserveSeats     = "reserve-seats"    // Fast path (Redis Lua)
    StepProcessPayment   = "process-payment"  // Stripe
    StepConfirmBooking   = "confirm-booking"  // PostgreSQL
    StepSendNotification = "send-notification" // Email/SMS

    // Compensation steps
    StepRefundPayment = "refund-payment"
    StepReleaseSeats  = "release-seats"
)
```

---

## Q4: Post-Payment Saga ทำงานยังไง?

**A:** Saga ที่ trigger หลัง payment success:

**Flow:**
```
┌────────────┐     ┌─────────────┐     ┌───────────────┐
│   Stripe   │────→│   Payment   │────→│     Kafka     │
│  Webhook   │     │   Service   │     │ payment.success
└────────────┘     └─────────────┘     └───────┬───────┘
                                               │
                                               ▼
                                     ┌─────────────────┐
                                     │      Saga       │
                                     │  Orchestrator   │
                                     └────────┬────────┘
                                              │
                         ┌────────────────────┼────────────────────┐
                         │                    │                    │
                         ▼                    ▼                    ▼
                  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐
                  │Step 1:      │      │Step 2:      │      │Step 3:      │
                  │Confirm      │ ───→ │Send         │ ───→ │Publish      │
                  │Booking      │      │Notification │      │Event        │
                  └─────────────┘      └─────────────┘      └─────────────┘
```

**จาก codebase - `backend-booking/internal/saga/booking_saga.go:420-453`:**
```go
// PostPaymentSagaBuilder creates a post-payment saga definition
func (b *PostPaymentSagaBuilder) Build() *pkgsaga.Definition {
    def := pkgsaga.NewDefinition(PostPaymentSagaName, "Post-payment booking confirmation saga")
    def.WithTimeout(1 * time.Minute)

    // Step 1: Confirm Booking
    // - Update booking status to confirmed in PostgreSQL
    // - Remove TTL from Redis (make reservation permanent)
    // - Generate confirmation code
    def.AddStep(&pkgsaga.Step{
        Name:        StepConfirmBooking,
        Description: "Confirm booking after payment success",
        Execute:     nil, // Executed by saga_step_worker
        Compensate:  nil, // Compensation handled separately
        Timeout:     b.config.StepTimeout,
        Retries:     b.config.MaxRetries,
    })

    // Step 2: Send Notification (NON-CRITICAL)
    // - If fails: Retry → DLQ (NO refund, NO seat release)
    def.AddStep(&pkgsaga.Step{
        Name:        StepSendNotification,
        Description: "Send booking confirmation notification",
        Execute:     nil,
        Compensate:  nil, // NON-CRITICAL: No compensation
        Timeout:     b.config.StepTimeout,
        Retries:     5,   // More retries for non-critical step
    })

    return def
}
```

---

## Q5: Saga Compensation ทำงานยังไง?

**A:** Reverse order execution:

```
Forward Execution:
Step 1 (Reserve) ──→ Step 2 (Payment) ──→ Step 3 (Confirm) ──→ Step 4 (Notify)
                                               ↓
                                           FAILED!

Compensation (Reverse Order):
Step 3 (Unconfirm) ←── Step 2 (Refund) ←── Start here
     ↓                      ↓
 (no-op, confirm           Stripe.refund()
  didn't happen)

Step 1 (Release) ←── Continue
     ↓
 release_seats.lua
```

**จาก codebase - `backend-booking/internal/saga/booking_saga.go:276-290`:**
```go
// Step 1: Reserve Seats - Compensate (Release)
func (b *BookingSagaBuilder) reserveSeatsCompensate(ctx context.Context, data map[string]interface{}) error {
    sagaData := &BookingSagaData{}
    sagaData.FromMap(data)

    if b.config.ReservationService == nil {
        return fmt.Errorf("reservation service is not configured")
    }

    if err := b.config.ReservationService.ReleaseSeats(ctx, sagaData.BookingID, sagaData.UserID); err != nil {
        return fmt.Errorf("failed to release seats: %w", err)
    }

    return nil
}
```

**Compensation Rules:**
1. Execute in reverse order
2. Idempotent (safe to retry)
3. Best effort (may fail, needs monitoring)
4. Non-critical steps may not have compensation

---

## Q6: Saga Persistence ทำยังไง?

**A:** Store state ใน PostgreSQL:

```sql
-- pkg/saga/postgres_instance_store.go
CREATE TABLE saga_instances (
    id UUID PRIMARY KEY,
    saga_type VARCHAR(100) NOT NULL,
    state VARCHAR(50) NOT NULL,
    current_step INT NOT NULL DEFAULT 0,
    payload JSONB,
    completed_steps TEXT[],
    failed_step VARCHAR(100),
    error TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_saga_instances_state ON saga_instances(state);
CREATE INDEX idx_saga_instances_created_at ON saga_instances(created_at);
```

**จาก codebase - `pkg/saga/store.go`:**
```go
// SagaInstance represents a running saga
type SagaInstance struct {
    ID             string                 `json:"id"`
    SagaType       string                 `json:"saga_type"`
    State          SagaState              `json:"state"`
    CurrentStep    int                    `json:"current_step"`
    Payload        map[string]interface{} `json:"payload"`
    CompletedSteps []string               `json:"completed_steps"`
    FailedStep     string                 `json:"failed_step,omitempty"`
    Error          string                 `json:"error,omitempty"`
    CreatedAt      time.Time              `json:"created_at"`
    UpdatedAt      time.Time              `json:"updated_at"`
}
```

**Why PostgreSQL (not Redis)?**
- Durability: ต้องการ persistence หลัง crash
- Recovery: Resume saga หลัง orchestrator restart
- Audit: Query saga history

---

## Q7: Saga Worker Architecture?

**A:** แยก workers ตาม responsibility:

```
┌─────────────────────────────────────────────────────────────┐
│                    SAGA ARCHITECTURE                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────┐                                       │
│  │ Saga Orchestrator│  Consumes: saga.*.event               │
│  │ (cmd/saga-       │  Produces: saga.*.command             │
│  │  orchestrator)   │  Manages: state transitions           │
│  └────────┬─────────┘                                       │
│           │                                                 │
│           │ Kafka                                           │
│           ▼                                                 │
│  ┌──────────────────────────────────────────────────┐       │
│  │                   Workers                         │       │
│  ├──────────────────────────────────────────────────┤       │
│  │                                                   │       │
│  │  ┌────────────────┐    ┌────────────────────┐    │       │
│  │  │ Saga Step      │    │ Saga Payment       │    │       │
│  │  │ Worker         │    │ Worker             │    │       │
│  │  │                │    │                    │    │       │
│  │  │ Handles:       │    │ Handles:           │    │       │
│  │  │ - reserve-seats│    │ - process-payment  │    │       │
│  │  │ - confirm-book │    │ - refund-payment   │    │       │
│  │  │ - release-seats│    │                    │    │       │
│  │  └────────────────┘    └────────────────────┘    │       │
│  │                                                   │       │
│  └──────────────────────────────────────────────────┘       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**จาก codebase - `backend-booking/internal/saga/booking_saga.go:1-27`:**
```go
// Package saga provides the booking saga implementation using the Saga pattern.
//
// # Architecture Overview
//
// This implementation uses an EVENT-DRIVEN approach with Kafka:
//
//  1. Saga Orchestrator (cmd/saga-orchestrator):
//     - Consumes events from workers
//     - Manages saga state transitions
//     - Sends commands to workers via Kafka
//
//  2. Step Workers (separate processes):
//     - saga_step_worker: handles reserve-seats, confirm-booking, release-seats
//     - saga-payment-worker: handles process-payment, refund-payment
//
//  3. Flow:
//     API creates saga instance → sends first command to Kafka →
//     Worker executes → sends event → Orchestrator advances saga →
//     sends next command → repeat until complete or compensate
```

**Communication Flow:**
```
Orchestrator ──command──→ Kafka ──→ Worker
                                      │
Worker ────────event───→ Kafka ──→ Orchestrator
```

| Topic | Direction | Example |
|-------|-----------|---------|
| `saga.booking.*.command` | Orchestrator → Worker | "confirm-booking" |
| `saga.booking.*.event` | Worker → Orchestrator | "booking-confirmed" |
