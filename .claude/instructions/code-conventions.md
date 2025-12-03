# Code Conventions

## File Naming
- Go files: `snake_case.go`
- Test files: `snake_case_test.go`
- Handlers: `{resource}_handler.go`
- Services: `{resource}_service.go`
- Repositories: `{resource}_repository.go`

## Package Naming
- All lowercase, no underscores
- Short and descriptive: `handler`, `service`, `repository`

## Function Naming
- Exported: `PascalCase`
- Unexported: `camelCase`
- Constructors: `New{Type}` (e.g., `NewBookingService`)
- HTTP handlers: `{Action}{Resource}` (e.g., `CreateBooking`, `GetBookingByID`)

## Struct Tags
```go
type CreateBookingRequest struct {
    ShowID    string `json:"show_id" validate:"required,uuid"`
    ZoneID    string `json:"zone_id" validate:"required,uuid"`
    Quantity  int    `json:"quantity" validate:"required,min=1,max=6"`
}
```

## Error Handling
- Always wrap errors with context: `fmt.Errorf("failed to reserve seats: %w", err)`
- Use custom error types from `pkg/errors`
- Never ignore errors silently

## Do's and Don'ts

### DO
- Write tests for all business logic
- Use context for cancellation and timeouts
- Add request ID to all logs
- Use transactions for multi-table writes
- Document public functions
- Handle graceful shutdown

### DON'T
- Commit `.env` files
- Use `panic` in production code
- Ignore errors
- Use global variables
- Hardcode configuration values
- Skip input validation
- Use `time.Sleep` for synchronization
