# Interview Q&A - Booking Rush 10k RPS

> คำถาม-คำตอบสำหรับการสัมภาษณ์งาน จากการวิเคราะห์ codebase จริง

## Project Overview

**Booking Rush** - High-Concurrency Ticket Booking System
- **Target**: 10,000 RPS + Zero Overselling
- **Tech Stack**: Go + Gin, Redis Lua, PostgreSQL, Redpanda (Kafka), Kubernetes
- **Architecture**: Microservices + Clean Architecture + Saga Pattern

## Files Structure

| File | Topic | Questions |
|------|-------|-----------|
| [01-architecture.md](./01-architecture.md) | Architecture & System Design | 8 |
| [02-concurrency.md](./02-concurrency.md) | Concurrency & Race Conditions | 6 |
| [03-redis-lua.md](./03-redis-lua.md) | Redis Lua Scripts Deep Dive | 8 |
| [04-saga-pattern.md](./04-saga-pattern.md) | Saga Pattern & Distributed TX | 7 |
| [05-zero-overselling.md](./05-zero-overselling.md) | Zero Overselling Strategy | 5 |
| [06-performance.md](./06-performance.md) | Performance Optimization | 8 |
| [07-virtual-queue.md](./07-virtual-queue.md) | Virtual Queue System | 6 |
| [08-idempotency.md](./08-idempotency.md) | Idempotency & Error Handling | 6 |
| [09-testing.md](./09-testing.md) | Testing Strategy | 5 |
| [10-kubernetes.md](./10-kubernetes.md) | Kubernetes & Deployment | 6 |
| [11-observability.md](./11-observability.md) | Observability & Monitoring | 5 |
| [12-system-design.md](./12-system-design.md) | System Design Questions | 6 |

## Key Talking Points

1. **10,000 RPS**: Go + Redis Lua + Connection pooling + Async processing
2. **Zero Overselling**: Multi-layer defense (Lua → Saga → DB constraint → Reconciliation)
3. **Distributed Transactions**: Saga pattern with orchestration
4. **Flash Sales**: Virtual queue with FIFO ordering
5. **Idempotency**: Safe retries with dual-TTL strategy
6. **Observability**: OpenTelemetry (Traces + Metrics + Logs)
7. **Clean Architecture**: Handler → Service → Repository → Domain
8. **Testing**: Unit → Integration → Load (k6)

## How to Use

1. อ่านแต่ละหัวข้อตาม topic ที่สนใจ
2. ทุกคำตอบอ้างอิงจาก code จริงใน codebase
3. มี code snippets จากไฟล์จริงประกอบ
4. เหมาะสำหรับเตรียมสัมภาษณ์ระดับ Senior/Staff Engineer

---

*Generated from codebase analysis*
