# Dirty Scenario Testing Documentation

This document describes edge case tests for the booking system that simulate real-world failure scenarios.

## Overview

Dirty scenarios are tests that simulate unexpected or edge-case behaviors that may occur in production:
- Client disconnections
- Network timeouts
- Race conditions
- Service failures
- Consumer crashes

## Test Files

| File | Purpose |
|------|---------|
| `dirty_scenarios.js` | k6 load test script for dirty scenarios |
| `setup_dirty_scenarios.js` | Redis seed data setup script |
| `dirty_scenarios_seed.json` | Generated test data (after running setup) |
| `tests/integration/dirty_scenarios_test.go` | Go integration tests |

## Scenarios

### 1. Client Disconnect After Reserve

**Scenario:** Client reserves seats but disconnects before payment

**Expected Behavior:**
- Seats are released after 10-minute TTL expires
- No orphaned reservations in the system
- Redis TTL automatically cleans up expired reservations

**Test Flow:**
1. Client sends reserve request → receives booking_id
2. Client disconnects (no confirm/release call)
3. Reservation exists with TTL
4. After TTL expires, Redis key is automatically deleted

**Verification:**
- `k6 run dirty_scenarios.js --env SCENARIO=client_disconnect`
- Metric: `abandoned_reservations` counter

**Production Note:** The inventory sync worker should detect expired reservations and return seats to PostgreSQL inventory.

---

### 2. Idempotency Key Retry

**Scenario:** Client retries request with same idempotency key

**Expected Behavior:**
- Same booking_id returned for same idempotency_key
- No double-booking or duplicate reservations
- Seat count remains consistent

**Test Flow:**
1. First request with idempotency_key="ABC" → booking_id="123"
2. Retry with same idempotency_key="ABC" → booking_id="123" (same)
3. Multiple retries → all return same booking_id

**Verification:**
- `k6 run dirty_scenarios.js --env SCENARIO=idempotency`
- Metric: `idempotency_hits` counter

**Implementation:**
```go
// booking_service.go:114-129
if req.IdempotencyKey != "" {
    existingBooking, err := s.bookingRepo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
    if err == nil && existingBooking != nil {
        return &dto.ReserveSeatsResponse{
            BookingID: existingBooking.ID,
            ...
        }, nil
    }
}
```

---

### 3. Last Seat Race (100 Concurrent Requests)

**Scenario:** 100 concurrent requests compete for the last 1 seat

**Expected Behavior:**
- Exactly 1 request succeeds
- 99 requests fail with `INSUFFICIENT_SEATS`
- Total seat count unchanged (no negative inventory)

**Test Flow:**
1. Initialize zone with 1 seat
2. Launch 100 concurrent reserve requests
3. Exactly 1 winner, 99 losers

**Verification:**
- `k6 run dirty_scenarios.js --env SCENARIO=last_seat`
- Metrics: `concurrency_winners` = 1, `concurrency_losers` = 99

**Implementation:**
Redis Lua script ensures atomicity:
```lua
-- reserve_seats.lua:59-61
if available < quantity then
    return {0, "INSUFFICIENT_STOCK", "Not enough seats available..."}
end
```

---

### 4. Payment Service Timeout

**Scenario:** Payment service times out during confirmation

**Expected Behavior:**
- Saga pattern compensates failed transactions
- Seats are released back to inventory
- No stuck reservations

**Test Flow:**
1. Reserve seats successfully
2. Attempt to confirm with very short timeout
3. Either: confirm succeeds OR timeout occurs
4. System state remains consistent

**Verification:**
- `k6 run dirty_scenarios.js --env SCENARIO=payment_timeout`

**Production Note:** The Saga orchestrator should handle payment failures and trigger compensation (release seats).

---

### 5. Kafka Consumer Crashes Mid-Processing

**Scenario:** Kafka consumer crashes while processing booking events

**Expected Behavior:**
- Messages are reprocessed after restart (at-least-once delivery)
- Idempotency prevents duplicate side effects
- No data corruption

**Test Flow:**
1. Consumer receives booking.created event
2. Consumer crashes before committing offset
3. Consumer restarts, reprocesses same event
4. Idempotency ensures no duplicate booking

**Verification:**
- `INTEGRATION_TEST=true go test ./tests/integration/... -run TestDirtyScenario_KafkaConsumerIdempotency -v`

**Implementation:**
- Booking service uses idempotency keys
- Inventory worker aggregates deltas before applying
- PostgreSQL unique constraints prevent duplicates

---

### 6. Redis Crashes During Reservation

**Scenario:** Redis becomes unavailable during reservation

**Expected Behavior:**
- Service returns 503 (Service Unavailable)
- Graceful degradation without data corruption
- Recovery when Redis is back online

**Test Flow:**
1. High load on booking service
2. Simulate Redis unavailability
3. Service returns 503 errors
4. No partial reservations

**Verification:**
- `k6 run dirty_scenarios.js --env SCENARIO=degradation`
- Metric: `service_unavailable_503` counter

---

### 7. Network Timeout Mid-Request

**Scenario:** Network timeout occurs during reservation request

**Expected Behavior:**
- No duplicate reservations
- Client can safely retry with same idempotency key

**Test Flow:**
1. Send request with short timeout
2. Request may timeout before response
3. Retry with same idempotency key
4. No duplicate bookings created

**Verification:**
- `k6 run dirty_scenarios.js --env SCENARIO=network_timeout`
- Metric: `network_timeouts` counter

---

### 8. Double Release Attempt

**Scenario:** Client attempts to release the same booking twice

**Expected Behavior:**
- First release succeeds
- Second release fails with `RESERVATION_NOT_FOUND`
- No double credit (seats not added twice)

**Verification:**
- `INTEGRATION_TEST=true go test ./tests/integration/... -run TestDirtyScenario_DoubleRelease -v`

---

### 9. Confirm After Expiration

**Scenario:** Client tries to confirm an expired reservation

**Expected Behavior:**
- Confirmation fails with `RESERVATION_NOT_FOUND`
- Client must create new reservation

**Verification:**
- `INTEGRATION_TEST=true go test ./tests/integration/... -run TestDirtyScenario_ConfirmAfterExpiration -v`

---

## Running Tests

### k6 Load Tests

```bash
# Setup test data in Redis
cd tests/load
REDIS_HOST=localhost REDIS_PORT=6379 node setup_dirty_scenarios.js

# Run all scenarios
k6 run dirty_scenarios.js

# Run specific scenario
k6 run dirty_scenarios.js --env SCENARIO=last_seat_race
k6 run dirty_scenarios.js --env SCENARIO=idempotency
k6 run dirty_scenarios.js --env SCENARIO=client_disconnect
k6 run dirty_scenarios.js --env SCENARIO=payment_timeout
k6 run dirty_scenarios.js --env SCENARIO=degradation
k6 run dirty_scenarios.js --env SCENARIO=network_timeout

# Reset test data
REDIS_HOST=localhost node setup_dirty_scenarios.js reset
```

### Go Integration Tests

```bash
# Run all dirty scenario tests
INTEGRATION_TEST=true \
TEST_REDIS_HOST=localhost \
TEST_REDIS_PASSWORD=<password> \
go test ./tests/integration/... -v -run "TestDirtyScenario"

# Run specific test
INTEGRATION_TEST=true TEST_REDIS_HOST=localhost \
go test ./tests/integration/... -v -run TestDirtyScenario_ConcurrentLastSeatRace
```

## Metrics

| Metric | Description |
|--------|-------------|
| `scenario_success_rate` | Overall success rate of scenario tests |
| `abandoned_reservations` | Reservations created but not confirmed |
| `idempotency_hits` | Successful idempotent request handling |
| `insufficient_seats_errors` | INSUFFICIENT_SEATS errors (expected in race tests) |
| `concurrency_winners` | Successful concurrent bookings |
| `concurrency_losers` | Failed concurrent bookings |
| `service_unavailable_503` | 503 errors during degradation |
| `network_timeouts` | Network timeout occurrences |
| `reservation_duration` | P50/P95/P99 of reserve operations |
| `confirm_duration` | P50/P95/P99 of confirm operations |

## Expected Results

| Scenario | Expected Outcome |
|----------|------------------|
| Client Disconnect | Seats released after TTL (10 min) |
| Idempotency Retry | Same booking_id returned, no duplicates |
| Last Seat Race | 1 winner, 99 losers, no negative inventory |
| Payment Timeout | Consistent state (confirmed or reserved) |
| Kafka Crash | Events reprocessed, idempotency prevents duplicates |
| Redis Crash | 503 errors, graceful degradation |
| Network Timeout | No duplicate reservations with same idempotency key |

## Architecture Protection Layers

1. **Redis Lua Scripts**: Atomic operations prevent race conditions
2. **Idempotency Keys**: Prevent duplicate bookings at service layer
3. **TTL Expiration**: Automatic cleanup of abandoned reservations
4. **PostgreSQL Constraints**: Final safeguard against data corruption
5. **Saga Pattern**: Compensation for distributed transaction failures
6. **Kafka At-Least-Once**: Ensures no lost events

## Troubleshooting

### Orphaned Reservations
- Check `reservation:*` keys in Redis with no corresponding PostgreSQL booking
- Run inventory sync worker to reconcile

### Negative Inventory
- Should NEVER happen with Lua scripts
- If seen, check for bypassing Redis (direct DB updates)

### Duplicate Bookings
- Check idempotency key handling in booking service
- Verify PostgreSQL unique constraints

### Missing Events
- Check Kafka consumer lag
- Verify consumer group offset commits
