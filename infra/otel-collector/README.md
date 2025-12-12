# OTEL Collector Setup (Coolify)

## Architecture

```
Go Services (k3s)
       │
       │ OTLP gRPC (:4317)
       ▼
┌─────────────────────────────────────────┐
│  OTEL Collector (Coolify)               │
│  ┌───────────────────────────────────┐  │
│  │ Receivers: otlp (4317, 4318)      │  │
│  │ Processors: batch, memory_limiter │  │
│  │ Exporters:                        │  │
│  │   - Traces  → Tempo (:3200)       │  │
│  │   - Metrics → Prometheus (:8889)  │  │
│  │   - Logs    → Loki (:3100)        │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

## Deploy in Coolify

### Step 1: Create New Service

1. ไปที่ Coolify Dashboard
2. เลือก Project → Add New Resource → Docker Image
3. ใส่ข้อมูล:
   - **Name**: `otel-collector`
   - **Image**: `otel/opentelemetry-collector-contrib:0.96.0`

### Step 2: Configure Ports

| Port | Protocol | Description |
|------|----------|-------------|
| 4317 | TCP | OTLP gRPC (from Go services) |
| 4318 | TCP | OTLP HTTP |
| 8889 | TCP | Prometheus metrics |
| 13133 | TCP | Health check |

### Step 3: Mount Config File

1. Copy content of `otel-collector-config.yaml`
2. ใน Coolify → Configuration → Volumes/Mounts
3. Add file mount:
   - **Path in container**: `/etc/otelcol-contrib/config.yaml`
   - **Content**: paste config

### Step 4: Set Command (ถ้าจำเป็น)

```
--config=/etc/otelcol-contrib/config.yaml
```

### Step 5: Network Configuration

ต้องให้ OTEL Collector อยู่ใน network เดียวกับ:
- Tempo
- Loki
- Prometheus

ใน Coolify อาจต้อง:
- ใช้ Docker network ชื่อเดียวกัน
- หรือใช้ IP address แทน hostname

### Step 6: Update Config Endpoints

แก้ไข endpoints ใน config ให้ตรงกับ Coolify:

```yaml
exporters:
  otlp/tempo:
    # ใช้ container name หรือ IP
    endpoint: <tempo-container-name>:3200

  loki:
    endpoint: http://<loki-container-name>:3100/loki/api/v1/push
```

## Verify Installation

### Health Check
```bash
curl http://<coolify-server>:13133/health
```

### Test OTLP
```bash
# ส่ง test trace
curl -X POST http://<coolify-server>:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{}'
```

## Update Go Services

หลังจาก deploy แล้ว update `.env.prod`:

```env
OTEL_COLLECTOR_ADDR=<coolify-server-ip>:4317
```

## Prometheus Scrape Config

เพิ่มใน Prometheus config:

```yaml
scrape_configs:
  - job_name: 'otel-collector'
    static_configs:
      - targets: ['otel-collector:8889']
```
