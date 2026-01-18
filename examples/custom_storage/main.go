package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/w0rng/audit"
)

// JSONFileStorage is a custom storage implementation that persists events to a JSON file.
// This is a simple example - production implementations should handle errors and
// implement proper file locking.
type JSONFileStorage struct {
	mu       sync.RWMutex
	filepath string
	events   map[string][]audit.Event
}

// NewJSONFileStorage creates a new JSON file-based storage.
func NewJSONFileStorage(filepath string) *JSONFileStorage {
	storage := &JSONFileStorage{
		filepath: filepath,
		events:   make(map[string][]audit.Event),
	}
	storage.load()
	return storage
}

// Store appends an event and persists to file.
func (s *JSONFileStorage) Store(key string, event audit.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[key] = append(s.events[key], event)
	s.save()
}

// Get retrieves all events for a key.
func (s *JSONFileStorage) Get(key string) []audit.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := s.events[key]
	if events == nil {
		return []audit.Event{}
	}
	return events
}

// Has checks if events exist for a key.
func (s *JSONFileStorage) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.events[key]
	return ok
}

// Clear removes all events for a key and persists.
func (s *JSONFileStorage) Clear(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.events, key)
	s.save()
}

// load reads events from the JSON file.
func (s *JSONFileStorage) load() {
	data, err := os.ReadFile(s.filepath)
	if err != nil {
		// File doesn't exist yet, start with empty events
		return
	}
	json.Unmarshal(data, &s.events)
}

// save writes events to the JSON file.
func (s *JSONFileStorage) save() {
	data, err := json.MarshalIndent(s.events, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling events: %v\n", err)
		return
	}
	if err := os.WriteFile(s.filepath, data, 0644); err != nil {
		fmt.Printf("Error writing file: %v\n", err)
	}
}

func main() {
	fmt.Println("Custom Storage Example")
	fmt.Println("======================")
	fmt.Println()

	// Create logger with custom JSON file storage using options pattern
	filepath := "audit_events.json"
	storage := NewJSONFileStorage(filepath)
	logger := audit.New(audit.WithStorage(storage))

	fmt.Printf("Using JSON file storage: %s\n\n", filepath)

	// Create some audit events
	logger.Create("user:1", "admin", "User account created", map[string]audit.Value{
		"email": audit.PlainValue("alice@example.com"),
		"role":  audit.PlainValue("editor"),
	})

	logger.Update("user:1", "admin", "Role updated", map[string]audit.Value{
		"role": audit.PlainValue("admin"),
	})

	logger.Create("user:2", "admin", "User account created", map[string]audit.Value{
		"email": audit.PlainValue("bob@example.com"),
		"role":  audit.PlainValue("viewer"),
	})

	// Display events
	fmt.Println("Events for user:1:")
	events := logger.Events("user:1")
	for i, event := range events {
		fmt.Printf("  %d. %s by %s\n", i+1, event.Description, event.Author)
	}

	fmt.Println("\nEvents for user:2:")
	events = logger.Events("user:2")
	for i, event := range events {
		fmt.Printf("  %d. %s by %s\n", i+1, event.Description, event.Author)
	}

	// Display change history
	fmt.Println("\nChange history for user:1:")
	changes := logger.Logs("user:1")
	for _, change := range changes {
		fmt.Printf("- %s\n", change.Description)
		for _, field := range change.Fields {
			fmt.Printf("  %s: %v → %v\n", field.Field, field.From, field.To)
		}
	}

	fmt.Printf("\nEvents have been persisted to %s\n", filepath)
	fmt.Println("Run this example again to see events loaded from file!")

	// Show file size
	if info, err := os.Stat(filepath); err == nil {
		fmt.Printf("File size: %d bytes\n", info.Size())
	}
}
