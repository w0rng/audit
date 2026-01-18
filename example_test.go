package audit_test

import (
	"fmt"

	"github.com/w0rng/audit"
)

func ExampleLogger_Create() {
	logger := audit.New()

	logger.Create(
		"user:123",
		"admin",
		"User account created",
		map[string]audit.Value{
			"email":    audit.PlainValue("user@example.com"),
			"password": audit.HiddenValue(),
		},
	)

	events := logger.Events("user:123")
	fmt.Printf("Created %d event(s)\n", len(events))
	// Output: Created 1 event(s)
}

func ExampleLogger_Update() {
	logger := audit.New()

	logger.Create("order:1", "user", "Order created", map[string]audit.Value{
		"status": audit.PlainValue("pending"),
	})

	logger.Update("order:1", "admin", "Order approved", map[string]audit.Value{
		"status": audit.PlainValue("approved"),
	})

	events := logger.Events("order:1")
	fmt.Printf("Total events: %d\n", len(events))
	// Output: Total events: 2
}

func ExampleLogger_Delete() {
	logger := audit.New()

	logger.Create("item:1", "user", "Item created", map[string]audit.Value{
		"name": audit.PlainValue("Widget"),
	})

	logger.Delete("item:1", "admin", "Item deleted", map[string]audit.Value{})

	events := logger.Events("item:1")
	fmt.Printf("Total events: %d\n", len(events))
	fmt.Printf("Last action: %s\n", events[len(events)-1].Action)
	// Output:
	// Total events: 2
	// Last action: delete
}

func ExampleLogger_Events() {
	logger := audit.New()

	logger.Create("order:1", "user", "Order created", map[string]audit.Value{
		"status": audit.PlainValue("pending"),
		"total":  audit.PlainValue(100.50),
	})

	logger.Update("order:1", "user", "Status updated", map[string]audit.Value{
		"status": audit.PlainValue("paid"),
	})

	// Get only status-related events
	events := logger.Events("order:1", "status")
	fmt.Printf("Found %d status events\n", len(events))
	// Output: Found 2 status events
}

func ExampleLogger_Events_noFilter() {
	logger := audit.New()

	logger.Create("item:1", "user", "Created", map[string]audit.Value{
		"name": audit.PlainValue("Widget"),
	})
	logger.Update("item:1", "user", "Updated", map[string]audit.Value{
		"price": audit.PlainValue(29.99),
	})

	// Get all events (no filter)
	events := logger.Events("item:1")
	fmt.Printf("Total events: %d\n", len(events))
	// Output: Total events: 2
}

func ExampleLogger_Logs() {
	logger := audit.New()

	logger.Create("item:1", "alice", "Item created", map[string]audit.Value{
		"color": audit.PlainValue("red"),
	})

	logger.Update("item:1", "bob", "Color changed", map[string]audit.Value{
		"color": audit.PlainValue("blue"),
	})

	changes := logger.Logs("item:1")
	for _, change := range changes {
		fmt.Printf("%s by %s\n", change.Description, change.Author)
		for _, field := range change.Fields {
			fmt.Printf("  %s: %v -> %v\n", field.Field, field.From, field.To)
		}
	}
	// Output:
	// Item created by alice
	//   color: <nil> -> red
	// Color changed by bob
	//   color: red -> blue
}

func ExampleHiddenValue() {
	logger := audit.New()

	logger.Create("user:1", "admin", "User created", map[string]audit.Value{
		"email":    audit.PlainValue("user@example.com"),
		"password": audit.HiddenValue(),
	})

	changes := logger.Logs("user:1")
	for _, field := range changes[0].Fields {
		if field.Field == "password" {
			fmt.Printf("%s: %v -> %v\n", field.Field, field.From, field.To)
		}
	}
	// Output: password: *** -> ***
}

func ExamplePlainValue() {
	logger := audit.New()

	logger.Create("config:1", "admin", "Config updated", map[string]audit.Value{
		"timeout": audit.PlainValue(30),
		"enabled": audit.PlainValue(true),
		"name":    audit.PlainValue("production"),
	})

	events := logger.Events("config:1")
	fmt.Printf("Logged %d event(s)\n", len(events))
	// Output: Logged 1 event(s)
}

func ExampleNewWithStorage() {
	// Create a custom storage implementation
	storage := audit.NewInMemoryStorage()

	// Create logger with custom storage using options pattern
	logger := audit.New(audit.WithStorage(storage))

	logger.Create("test:1", "user", "Test event", map[string]audit.Value{
		"field": audit.PlainValue("value"),
	})

	events := logger.Events("test:1")
	fmt.Printf("Events: %d\n", len(events))
	// Output: Events: 1
}

func ExampleWithStorage() {
	// Create a custom storage implementation
	customStorage := audit.NewInMemoryStorage()

	// Use WithStorage option to configure logger
	logger := audit.New(audit.WithStorage(customStorage))

	logger.Create("order:1", "user", "Order created", map[string]audit.Value{
		"status": audit.PlainValue("pending"),
	})

	events := logger.Events("order:1")
	fmt.Printf("Logged %d event(s) with custom storage\n", len(events))
	// Output: Logged 1 event(s) with custom storage
}

func ExampleLogger_Events_multipleFields() {
	logger := audit.New()

	logger.Create("user:1", "admin", "User created", map[string]audit.Value{
		"email":  audit.PlainValue("user@example.com"),
		"role":   audit.PlainValue("editor"),
		"status": audit.PlainValue("active"),
	})

	logger.Update("user:1", "admin", "Role updated", map[string]audit.Value{
		"role": audit.PlainValue("admin"),
	})

	logger.Update("user:1", "system", "Status changed", map[string]audit.Value{
		"status": audit.PlainValue("inactive"),
	})

	// Get events that have either role or status fields
	events := logger.Events("user:1", "role", "status")
	fmt.Printf("Events with role or status: %d\n", len(events))
	// Output: Events with role or status: 3
}

func Example() {
	// Create a new audit logger
	logger := audit.New()

	// Log entity creation
	logger.Create("order:12345", "john.doe", "Order created", map[string]audit.Value{
		"status": audit.PlainValue("pending"),
		"total":  audit.PlainValue(99.99),
	})

	// Log entity update
	logger.Update("order:12345", "jane.smith", "Order approved", map[string]audit.Value{
		"status": audit.PlainValue("approved"),
	})

	// Get change history
	changes := logger.Logs("order:12345")
	fmt.Printf("Total changes: %d\n", len(changes))
	fmt.Printf("Last change: %s\n", changes[len(changes)-1].Description)
	// Output:
	// Total changes: 2
	// Last change: Order approved
}
