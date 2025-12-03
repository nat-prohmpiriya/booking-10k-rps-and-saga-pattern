# Testing Conventions

## Test File Location
- Same directory as source file
- Name: `{source}_test.go`

## Test Naming
```go
func TestBookingService_Reserve_Success(t *testing.T) {}
func TestBookingService_Reserve_InsufficientSeats(t *testing.T) {}
```

## Test Coverage Requirements
- Minimum: 70% for business logic (service layer)
- Critical paths (booking, payment): 90%

## Test Types

| Type | Location | Purpose |
|------|----------|---------|
| Unit | `*_test.go` | Test single function/method |
| Integration | `tests/integration/` | Test with real DB/Redis |
| Load | `tests/k6/` | Performance testing |
