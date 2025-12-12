module github.com/prohmpiriya/booking-rush-10k-rps/tests/integration

go 1.24.0

toolchain go1.24.11

require github.com/prohmpiriya/booking-rush-10k-rps/pkg v0.0.0

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/redis/go-redis/extra/rediscmd/v9 v9.17.2 // indirect
	github.com/redis/go-redis/extra/redisotel/v9 v9.17.2 // indirect
	github.com/redis/go-redis/v9 v9.17.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
)

replace github.com/prohmpiriya/booking-rush-10k-rps/pkg => ../../pkg
