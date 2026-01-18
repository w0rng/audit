# audit

A lightweight, thread-safe audit logging library for Go that tracks entity changes and events with field-level precision.

[![Go Reference](https://pkg.go.dev/badge/github.com/w0rng/audit.svg)](https://pkg.go.dev/github.com/w0rng/audit)
[![Go Report Card](https://goreportcard.com/badge/github.com/w0rng/audit)](https://goreportcard.com/report/github.com/w0rng/audit)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- Simple API for entity audit logging (create, update, delete)
- Field-level change tracking with before/after values
- Sensitive data masking for passwords and tokens
- Thread-safe concurrent operations
- Pluggable storage interface (in-memory default)
- Slog integration for automatic audit from standard logs
- Zero dependencies in core package

## Installation

```bash
go get github.com/w0rng/audit
```

## Quick Start

```go
logger := audit.New()

// Log entity creation
logger.Create("order:123", "john.doe", "Order created", map[string]audit.Value{
    "status": audit.PlainValue("pending"),
    "token":  audit.HiddenValue(), // Masked as "***"
})

// Log update
logger.Update("order:123", "admin", "Order approved", map[string]audit.Value{
    "status": audit.PlainValue("approved"),
})

// Get change history
changes := logger.Logs("order:123")
for _, change := range changes {
    fmt.Printf("%s by %s\n", change.Description, change.Author)
    for _, field := range change.Fields {
        fmt.Printf("  %s: %v -> %v\n", field.Field, field.From, field.To)
    }
}
```

## Usage

### Creating a Logger

```go
// Default in-memory storage
logger := audit.New()

// With custom storage
logger := audit.New(audit.WithStorage(customStorage))
```

### Logging Events

```go
// Create, update, delete
logger.Create(key, author, description, payload)
logger.Update(key, author, description, payload)
logger.Delete(key, author, description, payload)

// Payload with hidden fields
payload := map[string]audit.Value{
    "email":    audit.PlainValue("user@example.com"),
    "password": audit.HiddenValue(), // Shows as "***"
}
```

### Retrieving Events

```go
// All events for entity
events := logger.Events("order:123")

// Filter by fields
statusEvents := logger.Events("order:123", "status", "total")

// Change history with state reconstruction
changes := logger.Logs("order:123")
```

## Custom Storage

Implement the `Storage` interface for custom backends (Redis, PostgreSQL, etc.):

```go
type Storage interface {
    Store(key string, event Event)
    Get(key string) []Event
    Has(key string) bool
    Clear(key string)
}

// Use it
storage := NewMyStorage()
logger := audit.New(audit.WithStorage(storage))
```

See [examples/custom_storage](./examples/custom_storage) for JSON file storage implementation.

## Slog Integration

Automatically create audit logs from standard `slog` logs:

```go
import auditslog "github.com/w0rng/audit/slog"

handler := auditslog.NewHandler(auditLogger, auditslog.HandlerOptions{
    KeyExtractor: auditslog.AttrExtractor("entity"),
    ShouldAudit: func(r slog.Record) bool {
        return r.Level >= slog.LevelInfo
    },
})

logger := slog.New(handler)
logger.Info("User created",
    "entity", "user:123",
    "action", "create",
    "email", "user@example.com",
)
```

See [examples/slog_integration](./examples/slog_integration) for complete example.

## Examples

Run examples to see the library in action:

```bash
go run examples/basic/main.go              # Basic usage
go run examples/custom_storage/main.go     # Custom storage
go run examples/slog_integration/main.go   # Slog integration
```

## Testing

```bash
go test ./...              # Run all tests
go test -race ./...        # With race detector
go test -cover ./...       # Check coverage
go test -bench=. -benchmem # Run benchmarks
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
