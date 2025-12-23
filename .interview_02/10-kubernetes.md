# 10 - Kubernetes & Deployment

## Q1: Health Checks (Liveness vs Readiness)?

**A:** 2 ประเภท probes:

```yaml
# infra/k8s/booking-service.yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  name: booking-service
spec:
  template:
    spec:
      containers:
      - name: booking-service
        image: booking-service:latest

        # Liveness Probe: "Is the container alive?"
        # Fail → Restart container
        livenessProbe:
          httpGet:
            path: /health
            port: 8083
          initialDelaySeconds: 10   # Wait before first check
          periodSeconds: 10         # Check every 10s
          failureThreshold: 3       # 3 failures → restart

        # Readiness Probe: "Can it receive traffic?"
        # Fail → Remove from Service endpoints
        readinessProbe:
          httpGet:
            path: /ready
            port: 8083
          initialDelaySeconds: 5    # Start checking sooner
          periodSeconds: 5          # Check more frequently
          failureThreshold: 3       # 3 failures → stop traffic
```

**Health Endpoints - `backend-booking/main.go:246-248`:**
```go
// Health check endpoints
router.GET("/health", container.HealthHandler.Health)
router.GET("/ready", container.HealthHandler.Ready)
```

**Handler Implementation:**
```go
// /health - Basic liveness (is process running?)
func (h *HealthHandler) Health(c *gin.Context) {
    c.JSON(200, gin.H{"status": "healthy"})
}

// /ready - Readiness (can handle requests?)
func (h *HealthHandler) Ready(c *gin.Context) {
    // Check database connection
    if err := h.db.Ping(c.Request.Context()); err != nil {
        c.JSON(503, gin.H{"status": "not ready", "error": "database"})
        return
    }

    // Check Redis connection
    if err := h.redis.Ping(c.Request.Context()).Err(); err != nil {
        c.JSON(503, gin.H{"status": "not ready", "error": "redis"})
        return
    }

    c.JSON(200, gin.H{"status": "ready"})
}
```

**Difference:**

| Probe | Question | Fail Action | Use Case |
|-------|----------|-------------|----------|
| Liveness | "Alive?" | Restart pod | Deadlock, infinite loop |
| Readiness | "Ready?" | Stop traffic | DB not connected yet |

---

## Q2: Horizontal Pod Autoscaler (HPA)?

**A:** Auto-scale based on metrics:

```yaml
# infra/k8s/booking-service-hpa.yaml

apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: booking-service-hpa
  namespace: booking-rush
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: booking-service

  minReplicas: 2      # Always at least 2 (HA)
  maxReplicas: 20     # Cost control

  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70   # Scale up at 70% CPU

  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80   # Scale up at 80% memory

  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300  # Wait 5 min before scale down
      policies:
      - type: Percent
        value: 10                       # Remove max 10% of pods
        periodSeconds: 60

    scaleUp:
      stabilizationWindowSeconds: 0    # Scale up immediately
      policies:
      - type: Percent
        value: 100                      # Can double pods
        periodSeconds: 15
      - type: Pods
        value: 4                        # Add max 4 pods
        periodSeconds: 15
```

**Scaling Events:**
```
Load increases:
  CPU > 70% for 15s → Add pods (up to 100% or +4)

Load decreases:
  CPU < 70% for 5 min → Remove pods (max 10% per minute)

Flash sale scenario:
  2 pods → 4 → 8 → 16 → 20 (max) in ~60 seconds
```

---

## Q3: Resource Limits & Requests?

**A:** Resource allocation:

```yaml
# infra/k8s/booking-service.yaml

spec:
  containers:
  - name: booking-service
    resources:
      # Requests: Guaranteed resources (for scheduling)
      requests:
        cpu: "200m"      # 0.2 CPU cores
        memory: "256Mi"  # 256 MB RAM

      # Limits: Maximum allowed (for throttling/OOM)
      limits:
        cpu: "1000m"     # 1 CPU core
        memory: "512Mi"  # 512 MB RAM
```

**Best Practices:**

| Setting | Recommendation | Reason |
|---------|----------------|--------|
| Request CPU | 50-70% of limit | Realistic baseline |
| Request Memory | 80-90% of limit | Memory is less elastic |
| Limit CPU | 2-5x request | Allow bursts |
| Limit Memory | 1.2-1.5x request | Prevent OOM kill |

**Go Service Characteristics:**
```yaml
# Go services are memory-efficient
resources:
  requests:
    cpu: "200m"      # Go is CPU-efficient
    memory: "128Mi"  # Small binary, low overhead
  limits:
    cpu: "500m"      # Allow burst for GC
    memory: "256Mi"  # Buffer for connections
```

---

## Q4: ConfigMap & Secrets?

**A:** Configuration management:

```yaml
# infra/k8s/configmap.yaml

apiVersion: v1
kind: ConfigMap
metadata:
  name: booking-config
  namespace: booking-rush
data:
  # Service URLs
  TICKET_SERVICE_URL: "http://ticket-service:8082"
  AUTH_SERVICE_URL: "http://auth-service:8081"

  # Performance tuning
  DB_MAX_CONNS: "20"
  DB_MIN_CONNS: "5"
  REDIS_POOL_SIZE: "500"
  REDIS_MIN_IDLE: "100"

  # Booking settings
  MAX_TICKETS_PER_USER: "10"
  RESERVATION_TTL_MINUTES: "10"
  REQUIRE_QUEUE_PASS: "true"

  # OpenTelemetry
  OTEL_ENABLED: "true"
  OTEL_COLLECTOR_ADDR: "otel-collector:4317"
  OTEL_SAMPLE_RATIO: "0.1"
```

```yaml
# infra/k8s/secrets.yaml

apiVersion: v1
kind: Secret
metadata:
  name: booking-secrets
  namespace: booking-rush
type: Opaque
stringData:
  # Database credentials
  DB_PASSWORD: "your-secure-password"

  # Redis password
  REDIS_PASSWORD: "redis-password"

  # JWT secret
  JWT_SECRET: "your-jwt-secret"

  # Stripe API key
  STRIPE_SECRET_KEY: "sk_live_xxx"
```

**Using in Deployment:**
```yaml
spec:
  containers:
  - name: booking-service
    envFrom:
    - configMapRef:
        name: booking-config
    - secretRef:
        name: booking-secrets
```

---

## Q5: Deployment Strategy?

**A:** Rolling Update with zero downtime:

```yaml
# infra/k8s/booking-service.yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  name: booking-service
spec:
  replicas: 3

  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1        # 1 extra pod during update
      maxUnavailable: 0  # Never reduce capacity

  template:
    spec:
      containers:
      - name: booking-service

        # Graceful shutdown
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sh", "-c", "sleep 10"]

      terminationGracePeriodSeconds: 30
```

**Rolling Update Process:**
```
Initial: [Pod1-v1] [Pod2-v1] [Pod3-v1]

Step 1: Create new pod
        [Pod1-v1] [Pod2-v1] [Pod3-v1] [Pod4-v2]
        Wait for Pod4-v2 to be Ready

Step 2: Terminate old pod
        [Pod1-v1] [Pod2-v1] [Pod4-v2]
        Pod3-v1 receives SIGTERM, drains connections

Step 3: Repeat
        [Pod1-v1] [Pod4-v2] [Pod5-v2]
        ...
        [Pod4-v2] [Pod5-v2] [Pod6-v2]

Final:  All pods updated, zero downtime
```

**Graceful Shutdown - `backend-booking/main.go:371-384`:**
```go
// Wait for interrupt signal for graceful shutdown
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

appLog.Info("Shutting down server...")

// Give outstanding requests 30 seconds to complete
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := srv.Shutdown(ctx); err != nil {
    appLog.Fatal(fmt.Sprintf("Server forced to shutdown: %v", err))
}
```

---

## Q6: Service Mesh Considerations?

**A:** Optional enhancements:

```
Without Service Mesh:
┌─────────┐         ┌─────────┐
│ Booking │ ──────→ │ Payment │
│ Service │  HTTP   │ Service │
└─────────┘         └─────────┘

With Istio Service Mesh:
┌─────────────────────────────────┐
│  Booking Pod                    │
│  ┌─────────┐  ┌──────────────┐  │
│  │ Booking │  │ Envoy Proxy  │  │──→ mTLS, retries,
│  │ Service │──│ (sidecar)    │  │    circuit breaker
│  └─────────┘  └──────────────┘  │
└─────────────────────────────────┘
```

**Service Mesh Features:**

| Feature | Without | With Istio |
|---------|---------|------------|
| mTLS | Manual cert management | Automatic |
| Circuit Breaker | In-app (go-breaker) | Declarative |
| Retries | In-app logic | Declarative |
| Traffic Shifting | Manual | Canary deployments |
| Observability | OpenTelemetry | Enhanced tracing |

**โปรเจกต์นี้:**
- ไม่ใช้ Service Mesh (complexity trade-off)
- ใช้ in-app circuit breakers
- ใช้ OpenTelemetry สำหรับ tracing
- อาจ add Istio later สำหรับ production at scale
