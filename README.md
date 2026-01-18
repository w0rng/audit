# audit

A lightweight, thread-safe audit logging library for Go that tracks entity changes and events with field-level precision.

[![Go Reference](https://pkg.go.dev/badge/github.com/w0rng/audit.svg)](https://pkg.go.dev/github.com/w0rng/audit)
[![Go Report Card](https://goreportcard.com/badge/github.com/w0rng/audit)](https://goreportcard.com/report/github.com/w0rng/audit)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Simple API** - Create, update, and delete entity audit logs with minimal code
- **Field-level tracking** - Track individual field changes with before/after values
- **Sensitive data masking** - Hide passwords, tokens, and other sensitive information
- **Thread-safe** - Safe for concurrent use across goroutines
- **Flexible storage** - Pluggable storage interface with in-memory default
- **Zero dependencies** - Uses only Go standard library
- **State reconstruction** - Automatically tracks state transitions over time

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/w0rng/audit"
)

func main() {
    logger := audit.New()

    // Log entity creation
    logger.Create(
        "order:12345",
        "john.doe",
        "Order created",
        map[string]audit.Value{
            "status":        audit.PlainValue("pending"),
            "payment_token": audit.HiddenValue(), // Masks sensitive data
        },
    )

    // Log entity update
    logger.Update(
        "order:12345",
        "jane.smith",
        "Order approved",
        map[string]audit.Value{
            "status": audit.PlainValue("approved"),
        },
    )

    // Get change history with field-level tracking
    changes := logger.Logs("order:12345")
    for _, change := range changes {
        fmt.Printf("%s by %s\n", change.Description, change.Author)
        for _, field := range change.Fields {
            fmt.Printf("  %s: %v → %v\n", field.Field, field.From, field.To)
        }
    }
}
```

## Usage

### Creating a Logger

```go
// Default in-memory storage
logger := audit.New()

// Custom storage implementation
customStorage := NewMyStorage()
logger := audit.NewWithStorage(customStorage)
```

### Logging Events

```go
// Create event
logger.Create(key, author, description, payload)

// Update event
logger.Update(key, author, description, payload)

// Delete event
logger.Delete(key, author, description, payload)

// Generic log change (used internally by Create/Update/Delete)
logger.LogChange(key, audit.ActionCreate, author, description, payload)
```

**Parameters:**
- `key` - Unique identifier for the entity (e.g., "user:123", "order:456")
- `author` - Who made the change (user ID, system name, etc.)
- `description` - Human-readable description of what changed
- `payload` - Map of field names to values

### Retrieving Events

```go
// Get all events for a key
events := logger.Events("order:123")

// Get events filtered by specific fields
statusEvents := logger.Events("order:123", "status", "priority")

// Get change history with field-level state tracking
changes := logger.Logs("order:123")
```

### Hiding Sensitive Data

```go
logger.Create("user:1", "admin", "User created", map[string]audit.Value{
    "email":    audit.PlainValue("user@example.com"),
    "password": audit.HiddenValue(), // Shows as "***" in logs
})
```

Hidden values are masked in the change history to prevent exposure of sensitive information like passwords, API keys, and tokens.

### Field-Level Change Tracking

The `Logs()` method reconstructs the complete state history by tracking changes to individual fields:

```go
logger.Create("item:1", "alice", "Created", map[string]audit.Value{
    "color": audit.PlainValue("red"),
})

logger.Update("item:1", "bob", "Updated", map[string]audit.Value{
    "color": audit.PlainValue("blue"),
})

changes := logger.Logs("item:1")
// changes[0].Fields[0] = {Field: "color", From: nil, To: "red"}
// changes[1].Fields[0] = {Field: "color", From: "red", To: "blue"}
```

## Custom Storage

Implement the `Storage` interface to use custom backends like Redis, PostgreSQL, or file systems:

```go
type Storage interface {
    Store(key string, event Event)
    Get(key string) []Event
    Has(key string) bool
    Clear(key string)
}
```

### Example: JSON File Storage

```go
type JSONFileStorage struct {
    mu       sync.RWMutex
    filepath string
    events   map[string][]audit.Event
}

func (s *JSONFileStorage) Store(key string, event audit.Event) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.events[key] = append(s.events[key], event)
    // Save to file...
}

// Implement Get, Has, Clear...

// Use it
storage := NewJSONFileStorage("audit.json")
logger := audit.NewWithStorage(storage)
```

See [examples/custom_storage](./examples/custom_storage) for a complete implementation.

## API Reference

### Types

#### Action
```go
type Action string

const (
    ActionCreate Action = "create"
    ActionUpdate Action = "update"
    ActionDelete Action = "delete"
)
```

#### Value
```go
type Value struct {
    Data   any
    Hidden bool  // If true, displayed as "***"
}
```

#### Event
```go
type Event struct {
    Timestamp   time.Time
    Action      Action
    Author      string
    Description string
    Payload     map[string]Value
}
```

#### Change
```go
type Change struct {
    Fields      []ChangeField
    Description string
    Author      string
    Timestamp   time.Time
}
```

#### ChangeField
```go
type ChangeField struct {
    Field string
    From  any
    To    any
}
```

### Functions

#### New
```go
func New() *Logger
```
Creates a new Logger with in-memory storage.

#### NewWithStorage
```go
func NewWithStorage(storage Storage) *Logger
```
Creates a Logger with a custom storage implementation.

#### PlainValue
```go
func PlainValue(v any) Value
```
Creates a Value that is visible in logs.

#### HiddenValue
```go
func HiddenValue() Value
```
Creates a Value that is masked as "***" in logs.

### Logger Methods

#### Create
```go
func (l *Logger) Create(key string, author, description string, payload map[string]Value)
```
Logs an entity creation event.

#### Update
```go
func (l *Logger) Update(key string, author, description string, payload map[string]Value)
```
Logs an entity update event.

#### Delete
```go
func (l *Logger) Delete(key string, author, description string, payload map[string]Value)
```
Logs an entity deletion event.

#### Events
```go
func (l *Logger) Events(key string, fields ...string) []Event
```
Retrieves events for a key. If fields are specified, returns only events containing those fields, with payloads filtered to the requested fields.

#### Logs
```go
func (l *Logger) Logs(key string) []Change
```
Returns the complete change history with field-level state transitions reconstructed from events.

## Examples

### Basic Usage
See [examples/basic](./examples/basic) for a complete example demonstrating:
- Creating entities with visible and hidden fields
- Updating entity state through a workflow
- Retrieving complete change history
- Filtering events by specific fields

Run the example:
```bash
go run examples/basic/main.go
```

### Custom Storage
See [examples/custom_storage](./examples/custom_storage) for an example showing:
- Custom JSON file-based storage implementation
- Persisting audit logs to disk
- Loading events from storage on startup

Run the example:
```bash
go run examples/custom_storage/main.go
```

## Use Cases

- **Compliance & Auditing** - Track all changes to sensitive data for regulatory requirements
- **Debugging** - Understand how entities evolved over time
- **User Activity Tracking** - Monitor who made what changes and when
- **State Reconstruction** - Rebuild entity state at any point in time
- **Change Notifications** - Trigger actions based on specific field changes
- **Rollback Support** - Understand previous states for undo functionality

## Thread Safety

All Logger operations are thread-safe. The default `InMemoryStorage` uses `sync.RWMutex` for concurrent access. Custom storage implementations must also be thread-safe.

## Performance

The library is optimized for:
- Fast writes with minimal allocations
- Efficient field-based event filtering (O(1) field lookup)
- Concurrent reads and writes without blocking

Run benchmarks:
```bash
go test -bench=. -benchmem
```

## Testing

The library has comprehensive test coverage including:
- Unit tests for all public APIs
- Concurrency tests with race detection
- Table-driven tests for edge cases
- Benchmark tests for performance

Run tests:
```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Check coverage
go test -cover ./...

# Run examples
go test -run Example
```
