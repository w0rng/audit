package main

import (
	"fmt"
	"strings"

	"github.com/w0rng/audit"
)

func main() {
	logger := audit.New()

	// Create an order with visible and hidden fields
	logger.Create(
		"order:12345",
		"john.doe",
		"Order created",
		map[string]audit.Value{
			"status":        audit.PlainValue("pending"),
			"total":         audit.PlainValue(99.99),
			"payment_token": audit.HiddenValue(), // Hidden for security
		},
	)

	// Update order status
	logger.Update(
		"order:12345",
		"jane.smith",
		"Order approved",
		map[string]audit.Value{
			"status": audit.PlainValue("approved"),
		},
	)

	// Update with additional fields
	logger.Update(
		"order:12345",
		"john.doe",
		"Shipping address updated",
		map[string]audit.Value{
			"address": audit.PlainValue(map[string]string{
				"street": "123 Main St",
				"city":   "Springfield",
			}),
		},
	)

	// Update status again
	logger.Update(
		"order:12345",
		"warehouse.system",
		"Order shipped",
		map[string]audit.Value{
			"status":          audit.PlainValue("shipped"),
			"tracking_number": audit.PlainValue("TRK123456789"),
		},
	)

	// Display complete change history
	fmt.Println("Complete Change History:")
	fmt.Println("========================")
	logs := logger.Logs("order:12345")
	for i, change := range logs {
		fmt.Printf("\n%d. %s (by %s at %s)\n",
			i+1,
			change.Description,
			change.Author,
			change.Timestamp.Format("2006-01-02 15:04:05"),
		)
		for _, field := range change.Fields {
			fmt.Printf("   %s: %v -> %v\n", field.Field, field.From, field.To)
		}
	}

	// Display filtered events (only status changes)
	fmt.Println("\n\nStatus Change Timeline:")
	fmt.Println("=======================")
	events := logger.Events("order:12345", "status")
	statuses := []string{}
	for _, event := range events {
		for _, value := range event.Payload {
			statuses = append(statuses, fmt.Sprintf("%v", value.Data))
		}
	}
	fmt.Printf("Status progression: %s\n", strings.Join(statuses, " -> "))

	// Display events with multiple field filters
	fmt.Println("\n\nStatus and Tracking Events:")
	fmt.Println("===========================")
	trackingEvents := logger.Events("order:12345", "status", "tracking_number")
	fmt.Printf("Found %d events with status or tracking_number fields\n", len(trackingEvents))
	for _, event := range trackingEvents {
		fmt.Printf("- %s by %s\n", event.Description, event.Author)
		for field, value := range event.Payload {
			fmt.Printf("  %s: %v\n", field, value.Data)
		}
	}
}
