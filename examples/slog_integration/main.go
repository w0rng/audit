package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/w0rng/audit"
	auditslog "github.com/w0rng/audit/slog"
)

func main() {
	fmt.Println("Slog Integration Example")
	fmt.Println("========================")
	fmt.Println()

	// Create audit logger
	auditLogger := audit.New()

	// Create slog handler that sends matching logs to audit
	handler := auditslog.NewHandler(auditLogger, auditslog.HandlerOptions{
		// Also send to stdout as JSON
		Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),

		// Extract entity key from "entity" attribute
		KeyExtractor: auditslog.AttrExtractor("entity"),

		// Only audit Info level and above
		ShouldAudit: func(record slog.Record) bool {
			return record.Level >= slog.LevelInfo
		},
	})

	// Set as default slog logger
	logger := slog.New(handler)
	slog.SetDefault(logger)

	fmt.Println("Logging events through slog:")
	fmt.Println()

	// Log user creation (will be audited)
	slog.Info("User account created",
		"entity", "user:123",
		"action", "create",
		"author", "admin",
		"email", "alice@example.com",
		"role", "editor",
	)

	// Log user update (will be audited)
	slog.Info("User role updated",
		"entity", "user:123",
		"action", "update",
		"author", "admin",
		"role", "admin",
	)

	// Debug log (will NOT be audited due to ShouldAudit filter)
	slog.Debug("Debug message",
		"entity", "user:123",
		"debug_info", "some details",
	)

	// Regular log without entity (will NOT be audited - no entity key)
	slog.Info("Application started",
		"version", "1.0.0",
	)

	// Order creation
	slog.Info("Order created",
		"entity", "order:456",
		"action", "create",
		"user", "alice", // can use "user" instead of "author"
		"total", 99.99,
		"status", "pending",
	)

	// Order update
	slog.Warn("Order payment failed",
		"entity", "order:456",
		"action", "update",
		"author", "payment-system",
		"status", "failed",
		"error", "insufficient funds",
	)

	fmt.Println()
	fmt.Println("======================")
	fmt.Println()
	fmt.Println("Audit Events for user:123:")
	fmt.Println("---------------------------")

	// Retrieve audit events
	userChanges := auditLogger.Logs("user:123")
	for i, change := range userChanges {
		fmt.Printf("%d. %s (by %s)\n", i+1, change.Description, change.Author)
		for _, field := range change.Fields {
			fmt.Printf("   %s: %v → %v\n", field.Field, field.From, field.To)
		}
	}

	fmt.Println()
	fmt.Println("Audit Events for order:456:")
	fmt.Println("----------------------------")

	orderChanges := auditLogger.Logs("order:456")
	for i, change := range orderChanges {
		fmt.Printf("%d. %s (by %s)\n", i+1, change.Description, change.Author)
		for _, field := range change.Fields {
			fmt.Printf("   %s: %v → %v\n", field.Field, field.From, field.To)
		}
	}

	fmt.Println()
	fmt.Println("======================")
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("- %d events for user:123\n", len(userChanges))
	fmt.Printf("- %d events for order:456\n", len(orderChanges))
	fmt.Println()
	fmt.Println("Note: Debug logs and logs without 'entity' attribute were not audited")
}
