# FOLDER STRUCTURE
```bash
booking-rush-10k-rps/
├── 📂 apps/
│   ├── 📂 web-client/            # [Frontend] Next.js (Dashboard + Booking UI)
│   │
│   ├── 📂 api-gateway/           # [Backend] รวม API ทุกตัวไว้ที่นี่ (หรือใช้ Nginx)
│   │
│   ├── 📂 auth-service/          # [Service] จัดการ User & JWT
│   │   ├── Dockerfile
│   │   └── main.go
│   │
│   ├── 📂 ticket-service/        # [Service] ข้อมูล Event (เน้น Read)
│   │   ├── Dockerfile
│   │   └── main.go               # Cache Event data in Redis
│   │
│   ├── 📂 booking-service/       # [Service] **Core System** (เน้น Write/High Load)
│   │   ├── internal/redis_lua/   # สคริปต์ตัดสต็อกเทพๆ อยู่ที่นี่
│   │   ├── internal/kafka/       # Producer ส่ง event ไปสร้าง Order
│   │   └── main.go
│   │
│   ├── 📂 payment-service/       # [Service] ระบบจำลองการจ่ายเงิน
│   │   └── main.go               # Consume Kafka -> Mock Pay -> Update DB
│   │
│   └── 📂 notification-service/  # (Optional) ส่งเมล/แจ้งเตือนเมื่อจองสำเร็จ
│       └── main.go
│
├── 📂 infra/
│   ├── docker-compose.yml        # รัน 5 Service + DB พร้อมกัน
│   ├── prometheus.yml            # Config Monitoring
│   └── grafana/                  # Dashboard JSON (โชว์กราฟ 10k RPS)
│
├── 📂 load-tests/                # k6 scripts
├── 📂 pkg/                       # Shared Library (Logger, DTO, Middleware)
└── README.md
```