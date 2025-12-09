# Load Testing Documentation

คู่มือการทดสอบ Load Testing สำหรับ Booking Rush System ด้วย k6

## สารบัญ

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Make Commands](#make-commands)
- [Test Scenarios](#test-scenarios)
- [การเตรียม Test Data](#การเตรียม-test-data)
- [การรัน Load Test](#การรัน-load-test)
- [Thresholds และ Metrics](#thresholds-และ-metrics)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### 1. ติดตั้ง k6

**macOS:**
```bash
brew install k6
```

**Linux (Debian/Ubuntu):**
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6
```

**Docker:**
```bash
docker pull grafana/k6
```

### 2. ติดตั้ง Docker

ต้องมี Docker สำหรับ seed data (ใช้ postgres และ redis images)

### 3. Start Booking Service

```bash
# Terminal 1: Start booking service
make run-booking
```

---

## Quick Start

```bash
# Step 1: Seed test data
make load-seed

# Step 2: Start booking service (terminal อื่น)
make run-booking

# Step 3: Run smoke test
make load-smoke

# หรือ run full test suite พร้อม dashboard
make load-full
```

---

## Make Commands

| Command | Description | Duration |
|---------|-------------|----------|
| `make load-seed` | Seed test data ลง PostgreSQL และ Redis | ~30s |
| `make load-smoke` | Smoke test (1 VU, 30s) - ตรวจสอบ basic functionality | 30s |
| `make load-quick` | Quick test (10 VUs, 30s) - ไม่มี dashboard | 30s |
| `make load-ramp` | Ramp-up test (0→1000 VUs) พร้อม dashboard | 9m |
| `make load-sustained` | Sustained load (5000 RPS) พร้อม dashboard | 5m |
| `make load-spike` | Spike test (1000→10000 RPS) พร้อม dashboard | 2.5m |
| `make load-10k` | **10k RPS stress test** พร้อม dashboard | 5m |
| `make load-full` | **Full test suite** ทุก scenario พร้อม dashboard | ~25m |
| `make load-clean` | ลบ test data ทั้งหมด | ~10s |

### Custom BASE_URL

```bash
# ถ้า service อยู่ที่ port อื่น
make load-smoke BASE_URL=http://localhost:9000

# ถ้า service อยู่บน remote server
make load-smoke BASE_URL=http://100.104.0.42:8083
```

---

## Test Scenarios

### 1. Smoke Test
- **Purpose:** ตรวจสอบว่า API ทำงานได้
- **VUs:** 1
- **Duration:** 30 วินาที
- **Target:** ตรวจสอบ basic functionality

### 2. Ramp-up Test
- **Purpose:** ค่อยๆ เพิ่ม load เพื่อหา breaking point
- **VUs:** 0 → 100 → 500 → 1000 → 500 → 0
- **Duration:** 9 นาที
- **Target:** หา capacity และ breaking point

### 3. Sustained Load Test
- **Purpose:** ทดสอบว่า system รองรับ load คงที่ได้นานเท่าไร
- **RPS:** 5,000 RPS
- **Duration:** 5 นาที
- **Target:** ตรวจสอบ stability ที่ load สูง

### 4. Spike Test
- **Purpose:** ทดสอบการรับมือกับ traffic spike ฉับพลัน
- **RPS:** 1,000 → 10,000 → 1,000
- **Duration:** 2.5 นาที
- **Target:** ตรวจสอบ behavior เมื่อ traffic spike

### 5. 10k RPS Stress Test
- **Purpose:** **Target performance test** - เป้าหมายหลักของ project
- **RPS:** 10,000 RPS
- **Duration:** 5 นาที
- **Target:** พิสูจน์ว่า system รองรับ 10k RPS ได้

---

## การเตรียม Test Data

### Seed Data

```bash
make load-seed
```

Script จะสร้าง:

| Data | จำนวน |
|------|-------|
| Tenant | 1 |
| Users | 10,000 |
| Events | 3 |
| Shows | 9 (3 per event) |
| Seat Zones | 45 (5 per show) |
| Seats per Zone | 20,000 |
| **Total Available Seats** | **900,000** |

### Manual Seed (ถ้า make ไม่ทำงาน)

```bash
# PostgreSQL
docker run --rm -i \
  -e PGPASSWORD=<password> \
  postgres:15-alpine \
  psql -h 100.104.0.42 -U postgres -d booking_rush < tests/load/seed_data.sql

# Redis
./tests/load/seed_redis.sh
```

### Clean Up

```bash
make load-clean
```

---

## การรัน Load Test

### 1. Basic Run (ไม่มี Dashboard)

```bash
k6 run --env BASE_URL=http://localhost:8083 tests/load/booking_reserve.js
```

### 2. With Web Dashboard (แนะนำ)

```bash
K6_WEB_DASHBOARD=true k6 run --env BASE_URL=http://localhost:8083 tests/load/booking_reserve.js
```

เปิด browser ไปที่ **http://localhost:5665** เพื่อดู real-time dashboard

### 3. Export Results

```bash
# JSON output
k6 run --out json=results.json tests/load/booking_reserve.js

# InfluxDB (สำหรับ Grafana)
k6 run --out influxdb=http://localhost:8086/k6 tests/load/booking_reserve.js
```

---

## Thresholds และ Metrics

### Performance Thresholds

| Metric | Threshold | Description |
|--------|-----------|-------------|
| `http_req_duration p(95)` | < 500ms | 95% ของ request ต้องเร็วกว่า 500ms |
| `http_req_duration p(99)` | < 1000ms | 99% ของ request ต้องเร็วกว่า 1s |
| `reserve_success_rate` | > 95% | Success rate ต้องมากกว่า 95% |
| `http_req_failed` | < 5% | Error rate ต้องน้อยกว่า 5% |

### Custom Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `reserve_success_rate` | Rate | อัตราการจอง success |
| `reserve_fail_rate` | Rate | อัตราการจอง fail |
| `reserve_duration` | Trend | เวลาที่ใช้ในการจอง |
| `insufficient_seats_errors` | Counter | จำนวน error เมื่อที่นั่งไม่พอ |
| `server_errors` | Counter | จำนวน 5xx errors |

### Expected Results for 10k RPS

```
✓ http_req_duration p(95) < 500ms
✓ http_req_duration p(99) < 1000ms
✓ reserve_success_rate > 95%
✓ http_req_failed < 5%
✓ http_reqs ≈ 10,000/s
```

---

## Troubleshooting

### "Connection refused" error

```
dial tcp 127.0.0.1:8083: connect: connection refused
```

**Solution:** Booking service ยังไม่ได้รัน
```bash
make run-booking
```

### "k6 not installed" error

```bash
# macOS
brew install k6

# หรือใช้ Docker
docker run --rm -i grafana/k6 run - < tests/load/booking_reserve.js
```

### High error rate (> 5%)

1. ตรวจสอบว่า seed data แล้ว: `make load-seed`
2. ตรวจสอบ Redis connection
3. ตรวจสอบ PostgreSQL connection
4. ดู logs ของ booking service

### Performance ไม่ถึง target

1. เพิ่ม resources ให้ service (CPU, Memory)
2. เพิ่ม database connection pool
3. เพิ่ม Redis connection pool
4. ตรวจสอบ network latency

### Insufficient VUs

สำหรับ 10k RPS อาจต้องเพิ่ม VUs:
```bash
# แก้ไขใน booking_reserve.js
maxVUs: 10000,  # เพิ่มจาก 5000
```

หรือรัน distributed:
```bash
# Terminal 1
k6 run --execution-segment 0:1/2 tests/load/booking_reserve.js

# Terminal 2
k6 run --execution-segment 1/2:1 tests/load/booking_reserve.js
```

---

## Files Structure

```
tests/load/
├── booking_reserve.js    # Main k6 test script
├── seed_data.json        # Test data for k6
├── seed_data.sql         # PostgreSQL seed script
├── seed_redis.sh         # Redis seed script
├── seed_all.sh           # Combined seed script (อ่าน .env)
├── generate_test_data.js # Generate full test data JSON
└── README.md             # This documentation
```

---

## References

- [k6 Documentation](https://k6.io/docs/)
- [k6 Web Dashboard](https://grafana.com/docs/k6/latest/results-output/web-dashboard/)
- [k6 Thresholds](https://k6.io/docs/using-k6/thresholds/)
- [k6 Scenarios](https://k6.io/docs/using-k6/scenarios/)
