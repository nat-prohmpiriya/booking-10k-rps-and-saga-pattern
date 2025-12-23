# 09 - Testing Strategy

## Q1: Testing Pyramid ในโปรเจกต์นี้?

**A:** 3 ระดับ:

```
                    ┌───────────┐
                    │   Load    │  ← k6, 10k RPS validation
                    │   Tests   │
                    └─────┬─────┘
                          │
                ┌─────────┴─────────┐
                │   Integration     │  ← Real Redis/PostgreSQL
                │      Tests        │
                └─────────┬─────────┘
                          │
          ┌───────────────┴───────────────┐
          │         Unit Tests            │  ← Mock dependencies
          │  (Service, Repository, Lua)   │
          └───────────────────────────────┘
```

| Level | Count | Speed | Coverage |
|-------|-------|-------|----------|
| Unit | Many | Fast (ms) | Business logic |
| Integration | Some | Medium (s) | Component interaction |
| Load | Few | Slow (min) | Performance validation |

---

## Q2: Unit Testing Patterns?

**A:** Table-driven tests with mocks:

```go
// backend-booking/internal/service/booking_service_test.go

func TestBookingService_Reserve(t *testing.T) {
    tests := []struct {
        name          string
        request       *dto.ReserveRequest
        mockSetup     func(*MockBookingRepo, *MockReservationRepo)
        expectedError error
        expectedID    string
    }{
        {
            name: "successful reservation",
            request: &dto.ReserveRequest{
                UserID:    "user-123",
                EventID:   "event-456",
                ZoneID:    "zone-789",
                Quantity:  2,
            },
            mockSetup: func(bookingRepo *MockBookingRepo, reservationRepo *MockReservationRepo) {
                reservationRepo.On("ReserveSeats", mock.Anything, mock.Anything).
                    Return(&ReserveResult{Success: true, Remaining: 98}, nil)
                bookingRepo.On("Create", mock.Anything, mock.Anything).
                    Return("booking-001", nil)
            },
            expectedError: nil,
            expectedID:    "booking-001",
        },
        {
            name: "insufficient stock",
            request: &dto.ReserveRequest{
                UserID:    "user-123",
                EventID:   "event-456",
                ZoneID:    "zone-789",
                Quantity:  100,
            },
            mockSetup: func(bookingRepo *MockBookingRepo, reservationRepo *MockReservationRepo) {
                reservationRepo.On("ReserveSeats", mock.Anything, mock.Anything).
                    Return(nil, domain.ErrInsufficientStock)
            },
            expectedError: domain.ErrInsufficientStock,
            expectedID:    "",
        },
        {
            name: "user limit exceeded",
            request: &dto.ReserveRequest{
                UserID:    "user-123",
                EventID:   "event-456",
                ZoneID:    "zone-789",
                Quantity:  5,
            },
            mockSetup: func(bookingRepo *MockBookingRepo, reservationRepo *MockReservationRepo) {
                reservationRepo.On("ReserveSeats", mock.Anything, mock.Anything).
                    Return(nil, domain.ErrUserLimitExceeded)
            },
            expectedError: domain.ErrUserLimitExceeded,
            expectedID:    "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mocks
            bookingRepo := new(MockBookingRepo)
            reservationRepo := new(MockReservationRepo)
            tt.mockSetup(bookingRepo, reservationRepo)

            // Create service
            service := NewBookingService(bookingRepo, reservationRepo, nil, nil)

            // Execute
            result, err := service.Reserve(context.Background(), tt.request)

            // Assert
            if tt.expectedError != nil {
                assert.ErrorIs(t, err, tt.expectedError)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expectedID, result.BookingID)
            }

            // Verify mock expectations
            bookingRepo.AssertExpectations(t)
            reservationRepo.AssertExpectations(t)
        })
    }
}
```

**Run unit tests:**
```bash
make test-unit
# or
go test ./backend-booking/internal/... -v -short
```

---

## Q3: Integration Testing?

**A:** Test กับ real dependencies:

```go
// backend-booking/internal/repository/redis_reservation_repo_test.go

func TestRedisReservationRepo_ReserveSeats_Integration(t *testing.T) {
    if os.Getenv("INTEGRATION_TEST") == "" {
        t.Skip("Skipping integration test")
    }

    // Connect to real Redis
    redisClient := redis.NewClient(&redis.Options{
        Addr:     os.Getenv("TEST_REDIS_HOST") + ":6379",
        Password: os.Getenv("TEST_REDIS_PASSWORD"),
    })
    defer redisClient.Close()

    repo := NewRedisReservationRepository(redisClient)

    // Load Lua scripts
    err := repo.LoadScripts(context.Background())
    require.NoError(t, err)

    // Setup test data
    zoneID := "test-zone-" + uuid.New().String()
    redisClient.Set(context.Background(), "zone:availability:"+zoneID, 100, 0)

    t.Run("successful reservation", func(t *testing.T) {
        req := &ReserveRequest{
            BookingID:  uuid.New().String(),
            UserID:     "user-123",
            EventID:    "event-456",
            ZoneID:     zoneID,
            Quantity:   2,
            MaxPerUser: 10,
        }

        result, err := repo.ReserveSeats(context.Background(), req)

        assert.NoError(t, err)
        assert.True(t, result.Success)
        assert.Equal(t, int64(98), result.Remaining)
    })

    t.Run("concurrent reservations no overselling", func(t *testing.T) {
        // Reset inventory
        redisClient.Set(context.Background(), "zone:availability:"+zoneID, 10, 0)

        var wg sync.WaitGroup
        successCount := atomic.Int32{}
        failCount := atomic.Int32{}

        // 20 concurrent requests for 1 seat each (only 10 available)
        for i := 0; i < 20; i++ {
            wg.Add(1)
            go func(userNum int) {
                defer wg.Done()
                req := &ReserveRequest{
                    BookingID:  uuid.New().String(),
                    UserID:     fmt.Sprintf("user-%d", userNum),
                    EventID:    "event-456",
                    ZoneID:     zoneID,
                    Quantity:   1,
                    MaxPerUser: 10,
                }

                result, err := repo.ReserveSeats(context.Background(), req)
                if err == nil && result.Success {
                    successCount.Add(1)
                } else {
                    failCount.Add(1)
                }
            }(i)
        }

        wg.Wait()

        // Exactly 10 should succeed (no overselling)
        assert.Equal(t, int32(10), successCount.Load())
        assert.Equal(t, int32(10), failCount.Load())
    })

    // Cleanup
    redisClient.Del(context.Background(), "zone:availability:"+zoneID)
}
```

**Run integration tests:**
```bash
INTEGRATION_TEST=true \
TEST_REDIS_HOST=localhost \
TEST_REDIS_PASSWORD= \
go test ./backend-booking/internal/repository/... -v
```

---

## Q4: Load Testing with k6?

**A:** จาก `tests/load/`:

**Test Scenarios:**
```javascript
// tests/load/scenarios/booking-reserve.js

export let options = {
    scenarios: {
        // Smoke test - basic functionality
        smoke: {
            executor: 'constant-vus',
            vus: 1,
            duration: '30s',
        },

        // Ramp-up - find breaking point
        ramp_up: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                { duration: '2m', target: 500 },
                { duration: '3m', target: 1000 },
                { duration: '2m', target: 2000 },
                { duration: '2m', target: 0 },
            ],
        },

        // Sustained load - stability test
        sustained: {
            executor: 'constant-arrival-rate',
            rate: 5000,         // 5000 requests per second
            duration: '5m',
            preAllocatedVUs: 500,
            maxVUs: 1000,
        },

        // Target 10k RPS
        target_10k: {
            executor: 'constant-arrival-rate',
            rate: 10000,        // 10,000 requests per second
            duration: '5m',
            preAllocatedVUs: 1000,
            maxVUs: 2000,
        },
    },

    thresholds: {
        http_req_duration: ['p(95)<100', 'p(99)<200'],  // P95 < 100ms
        http_req_failed: ['rate<0.001'],                // Error rate < 0.1%
        'http_req_duration{status:200}': ['p(99)<100'], // Success < 100ms
    },
};

export default function() {
    const eventID = __ENV.EVENT_ID || 'test-event';
    const zoneID = __ENV.ZONE_ID || 'test-zone';

    const payload = JSON.stringify({
        event_id: eventID,
        zone_id: zoneID,
        quantity: 1,
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'X-User-ID': `user-${__VU}-${__ITER}`,
            'X-Idempotency-Key': `idemp-${__VU}-${__ITER}-${Date.now()}`,
        },
    };

    const response = http.post(
        `${__ENV.BASE_URL}/api/v1/bookings/reserve`,
        payload,
        params
    );

    check(response, {
        'status is 200 or 409': (r) => r.status === 200 || r.status === 409,
        'response time < 100ms': (r) => r.timings.duration < 100,
    });
}
```

**Run load tests:**
```bash
cd tests/load

# Smoke test
k6 run --env BASE_URL=http://localhost:8080 scenarios/booking-reserve.js

# Full 10k test
k6 run \
  --env BASE_URL=http://localhost:8080 \
  --env EVENT_ID=real-event-id \
  --env ZONE_ID=real-zone-id \
  scenarios/booking-reserve.js
```

---

## Q5: Test Results Analysis?

**A:** Metrics ที่ต้องดู:

**k6 Output:**
```
          /\      |‾‾| /‾‾/   /‾‾/
     /\  /  \     |  |/  /   /  /
    /  \/    \    |     (   /   ‾‾\
   /          \   |  |\  \ |  (‾)  |
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: scenarios/booking-reserve.js
     output: -

  scenarios: (100.00%) 1 scenario, 2000 max VUs, 5m30s max duration
           * target_10k: 10000.00 iters/s for 5m0s

     ✓ status is 200 or 409
     ✓ response time < 100ms

     checks.........................: 100.00% ✓ 3000000    ✗ 0
     data_received..................: 1.2 GB  4.0 MB/s
     data_sent......................: 892 MB  3.0 MB/s
     http_req_blocked...............: avg=12µs   min=1µs    p(95)=25µs
     http_req_connecting............: avg=8µs    min=0s     p(95)=18µs
   ✓ http_req_duration..............: avg=15ms   min=2ms    p(95)=45ms   p(99)=78ms
       { status:200 }...............: avg=12ms   min=2ms    p(95)=35ms   p(99)=65ms
       { status:409 }...............: avg=18ms   min=3ms    p(95)=55ms   p(99)=95ms
   ✓ http_req_failed................: 0.05%   ✓ 1500       ✗ 2998500
     http_reqs......................: 3000000 10000/s
     iteration_duration.............: avg=25ms   min=5ms    p(95)=60ms
     iterations.....................: 3000000 10000/s
     vus............................: 1000    min=0        max=2000
     vus_max........................: 2000    min=2000     max=2000
```

**Key Metrics:**

| Metric | Target | Result | Status |
|--------|--------|--------|--------|
| RPS | 10,000 | 10,000 | ✅ |
| P95 Latency | < 100ms | 45ms | ✅ |
| P99 Latency | < 200ms | 78ms | ✅ |
| Error Rate | < 0.1% | 0.05% | ✅ |
| Success Rate | > 99.9% | 99.95% | ✅ |

**Bottleneck Identification:**
```
# If P99 > target:
1. Check database connection pool usage
2. Check Redis latency (redis-cli --latency)
3. Profile with pprof
4. Check GC pauses (GODEBUG=gctrace=1)

# If error rate high:
1. Check error logs for patterns
2. Check rate limiting (429 errors)
3. Check circuit breaker status
```
