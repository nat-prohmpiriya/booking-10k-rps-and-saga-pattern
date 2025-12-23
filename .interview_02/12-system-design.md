# 12 - System Design Questions

## Q1: ถ้าต้อง design ระบบนี้ใหม่ตั้งแต่ต้น จะทำยังไง?

**A:** Systematic approach:

**Step 1: Clarify Requirements**
```
Functional:
- Users can browse events, shows, zones
- Users can reserve seats (temporary hold)
- Users can pay and confirm booking
- Users receive confirmation

Non-Functional:
- 10,000 RPS sustained
- P99 latency < 100ms
- Zero overselling (consistency > availability)
- Flash sale support (500k concurrent users)
```

**Step 2: Identify Critical Path**
```
User → [Join Queue] → [Reserve Seats] → [Payment] → [Confirm]
                             ↑
                    This is the bottleneck!
                    Must be atomic + fast
```

**Step 3: High-Level Design**
```
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│   Users  ──→  CDN  ──→  Load Balancer  ──→  API Gateway     │
│                                                │             │
│                              ┌─────────────────┼────────┐    │
│                              │                 │        │    │
│                              ▼                 ▼        ▼    │
│                         ┌────────┐       ┌────────┐ ┌──────┐ │
│                         │ Auth   │       │Booking │ │Ticket│ │
│                         │Service │       │Service │ │Svc   │ │
│                         └────────┘       └────────┘ └──────┘ │
│                              │                 │        │    │
│                              │            ┌────┴────┐   │    │
│                              │            ▼         ▼   │    │
│                              │       ┌───────┐ ┌──────┐ │    │
│                              │       │ Redis │ │Kafka │ │    │
│                              │       │ (Lua) │ │      │ │    │
│                              │       └───────┘ └──────┘ │    │
│                              │            │             │    │
│                              ▼            ▼             ▼    │
│                         ┌─────────────────────────────────┐  │
│                         │        PostgreSQL Cluster       │  │
│                         └─────────────────────────────────┘  │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Step 4: Key Decisions**
```
1. Atomic seat reservation → Redis Lua scripts
2. Distributed transactions → Saga pattern
3. Flash sale traffic → Virtual Queue
4. High throughput → Go + connection pooling
5. Observability → OpenTelemetry
```

---

## Q2: Trade-offs ในโปรเจกต์นี้?

**A:**

| Decision | Trade-off | Why This Choice |
|----------|-----------|-----------------|
| Redis Lua over DB TX | Fast but less durable | Speed critical, sync worker for durability |
| Saga over 2PC | Eventually consistent | No distributed lock, more resilient |
| Go over Java | Less libraries, smaller ecosystem | Raw performance for 10k RPS |
| Microservices | Complex ops | Independent scaling |
| Virtual Queue | Adds latency | Prevents thundering herd |
| EVALSHA | Need script preload | 10x faster than EVAL |

**CAP Theorem Position:**
```
        Consistency
           /\
          /  \
         /    \
        / This \
       / System \
      /──────────\
     /            \
Availability ──── Partition Tolerance

Choice: CP (Consistency + Partition Tolerance)
- Zero overselling is mandatory
- Prefer rejecting requests over inconsistent state
```

---

## Q3: ถ้าต้อง scale เกิน 10k RPS?

**A:** Progressive scaling strategy:

**Phase 1: Vertical Optimization (10k → 30k RPS)**
```
- Add more replicas (HPA)
- Increase connection pools
- Tune GC (GOGC=100 → GOGC=200)
- Use Redis Cluster
```

**Phase 2: Horizontal Scaling (30k → 100k RPS)**
```
- Shard by event_id
- Regional Redis clusters
- Read replicas for PostgreSQL
- CDN for static content
```

**Phase 3: Geo-Distribution (100k+ RPS)**
```
                    ┌──────────────┐
                    │  Global DNS  │
                    │  (Anycast)   │
                    └──────┬───────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
         ▼                 ▼                 ▼
    ┌─────────┐       ┌─────────┐       ┌─────────┐
    │ US-East │       │ EU-West │       │ AP-South│
    │ Region  │       │ Region  │       │ Region  │
    └─────────┘       └─────────┘       └─────────┘
         │                 │                 │
    ┌────┴────┐       ┌────┴────┐       ┌────┴────┐
    │Local    │       │Local    │       │Local    │
    │Redis    │       │Redis    │       │Redis    │
    └─────────┘       └─────────┘       └─────────┘
         │                 │                 │
         └─────────────────┴─────────────────┘
                           │
                    ┌──────┴───────┐
                    │ Global DB    │
                    │ (CockroachDB │
                    │  or Spanner) │
                    └──────────────┘
```

---

## Q4: CQRS จะช่วยได้ไหม?

**A:** Yes, สำหรับ advanced scaling:

**Current: Single Model**
```
[Booking Service] ─── Read/Write ───→ [PostgreSQL]
```

**CQRS: Separate Models**
```
                      ┌────────────────┐
      Write ─────────→│  Write Model   │
                      │  (PostgreSQL)  │
                      └───────┬────────┘
                              │
                         Events│
                              │
                              ▼
                      ┌────────────────┐
                      │  Event Store   │
                      │    (Kafka)     │
                      └───────┬────────┘
                              │
                         Project│
                              │
                              ▼
                      ┌────────────────┐
      Read ──────────→│  Read Model    │
                      │ (Elasticsearch │
                      │   or Redis)    │
                      └────────────────┘
```

**Benefits:**
- Separate scaling for read vs write
- Optimized read models (denormalized)
- Event sourcing for audit trail

**Trade-offs:**
- Eventual consistency (read may lag)
- More infrastructure
- Complex debugging

**When to adopt:**
- Read/Write ratio > 10:1
- Complex query requirements
- Audit/compliance requirements

---

## Q5: Event Sourcing vs Current?

**A:**

**Current (State-based):**
```sql
-- Current state only
UPDATE bookings SET status = 'confirmed' WHERE id = ?
-- Previous state is lost
```

**Event Sourcing:**
```json
// Event store
{"type": "BookingCreated", "booking_id": "123", "ts": "..."}
{"type": "SeatsReserved", "booking_id": "123", "quantity": 2, "ts": "..."}
{"type": "PaymentReceived", "booking_id": "123", "amount": 200, "ts": "..."}
{"type": "BookingConfirmed", "booking_id": "123", "ts": "..."}

// Current state = Replay all events
```

**Comparison:**

| Aspect | State-based | Event Sourcing |
|--------|-------------|----------------|
| Storage | Current state | All events |
| History | Lost | Complete |
| Replay | Not possible | Full rebuild |
| Complexity | Lower | Higher |
| Debugging | Current only | Time travel |
| Performance | Fast reads | Need projections |

**When to use Event Sourcing:**
- Audit requirements (finance, healthcare)
- Complex state transitions
- Need to answer "what happened at time T?"
- Analytics on historical data

---

## Q6: Lessons Learned?

**A:** Key insights จากโปรเจกต์นี้:

**1. Start Simple, Optimize Later**
```
Initial: PostgreSQL transactions
Problem: Lock contention at 2k RPS
Solution: Redis Lua scripts

Lesson: Don't over-engineer upfront, but design for change
```

**2. Consistency is Expensive**
```
Strong consistency = Lower throughput
But for ticket booking, consistency is mandatory
Solution: Multi-layer defense (accept the cost)
```

**3. Async is Your Friend**
```
Sync: Reserve → Pay → Confirm → Notify (slow, coupled)
Async: Reserve → Pay → (Kafka) → Confirm, Notify (fast, decoupled)

Fast path: Only what user needs to wait for
Async: Everything else
```

**4. Observability from Day 1**
```
Without observability:
  "Something is slow" → ??? (days to debug)

With OpenTelemetry:
  "Reserve is slow" → Trace shows Redis P99 spike → Fix in hours
```

**5. Test at Scale**
```
Unit tests: Logic works
Integration tests: Components work together
Load tests: System works at 10k RPS

All three are essential!
```

**6. Failure Handling is Critical**
```
What if Redis crashes?
What if payment times out?
What if Kafka is unavailable?

Answer these BEFORE production, not after
```

---

## Summary: Interview Talking Points

เมื่อถูกถามเกี่ยวกับโปรเจกต์นี้:

1. **Start with the problem**: "ระบบ booking ที่ต้องรองรับ 10k RPS และห้าม overselling"

2. **Explain key decisions**:
   - Redis Lua for atomic operations
   - Saga for distributed transactions
   - Virtual Queue for flash sales
   - Multi-layer defense for consistency

3. **Show depth**: อธิบาย trade-offs และทำไมถึงเลือกแบบนี้

4. **Acknowledge limitations**: "ถ้าต้อง scale เกิน 100k RPS จะต้อง..."

5. **Demonstrate learning**: "สิ่งที่เรียนรู้คือ..."

6. **Connect to business value**: "Zero overselling = ไม่มี refund, ไม่มี angry customers"
