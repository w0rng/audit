package audit

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	logger := New()
	if logger == nil {
		t.Fatal("New() returned nil")
	}
	if logger.storage == nil {
		t.Fatal("Logger storage is nil")
	}
}

func TestNewWithStorage(t *testing.T) {
	storage := NewInMemoryStorage()
	logger := NewWithStorage(storage)
	if logger == nil {
		t.Fatal("NewWithStorage() returned nil")
	}
	if logger.storage != storage {
		t.Error("Logger storage does not match provided storage")
	}
}

func TestLogger_Create(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		author      string
		description string
		payload     map[string]Value
		wantAction  Action
	}{
		{
			name:        "create with plain values",
			key:         "user:1",
			author:      "admin",
			description: "User created",
			payload: map[string]Value{
				"email": PlainValue("user@example.com"),
				"role":  PlainValue("admin"),
			},
			wantAction: ActionCreate,
		},
		{
			name:        "create with hidden value",
			key:         "user:2",
			author:      "system",
			description: "User registered",
			payload: map[string]Value{
				"email":    PlainValue("test@example.com"),
				"password": HiddenValue(),
			},
			wantAction: ActionCreate,
		},
		{
			name:        "create with empty payload",
			key:         "user:3",
			author:      "admin",
			description: "User initialized",
			payload:     map[string]Value{},
			wantAction:  ActionCreate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New()
			logger.Create(tt.key, tt.author, tt.description, tt.payload)

			events := logger.Events(tt.key)
			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}

			event := events[0]
			if event.Action != tt.wantAction {
				t.Errorf("expected action %v, got %v", tt.wantAction, event.Action)
			}
			if event.Author != tt.author {
				t.Errorf("expected author %q, got %q", tt.author, event.Author)
			}
			if event.Description != tt.description {
				t.Errorf("expected description %q, got %q", tt.description, event.Description)
			}
			if len(event.Payload) != len(tt.payload) {
				t.Errorf("expected payload length %d, got %d", len(tt.payload), len(event.Payload))
			}
		})
	}
}

func TestLogger_Update(t *testing.T) {
	logger := New()
	logger.Create("order:1", "user", "Created", map[string]Value{
		"status": PlainValue("pending"),
	})
	logger.Update("order:1", "user", "Updated", map[string]Value{
		"status": PlainValue("approved"),
	})

	events := logger.Events("order:1")
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[1].Action != ActionUpdate {
		t.Errorf("expected action %v, got %v", ActionUpdate, events[1].Action)
	}
}

func TestLogger_Delete(t *testing.T) {
	logger := New()
	logger.Create("item:1", "user", "Created", map[string]Value{
		"name": PlainValue("test"),
	})
	logger.Delete("item:1", "admin", "Deleted", map[string]Value{})

	events := logger.Events("item:1")
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[1].Action != ActionDelete {
		t.Errorf("expected action %v, got %v", ActionDelete, events[1].Action)
	}
}

func TestLogger_Events_NoFilter(t *testing.T) {
	logger := New()
	logger.Create("entity:1", "user", "Created", map[string]Value{
		"field1": PlainValue("value1"),
		"field2": PlainValue("value2"),
	})
	logger.Update("entity:1", "user", "Updated", map[string]Value{
		"field1": PlainValue("new-value"),
	})

	events := logger.Events("entity:1")
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestLogger_Events_Filtering(t *testing.T) {
	logger := New()
	logger.Create("entity:1", "user", "Created", map[string]Value{
		"status": PlainValue("new"),
		"email":  PlainValue("test@example.com"),
		"token":  HiddenValue(),
	})
	logger.Update("entity:1", "user", "Updated status", map[string]Value{
		"status": PlainValue("active"),
	})
	logger.Update("entity:1", "user", "Updated email", map[string]Value{
		"email": PlainValue("new@example.com"),
	})

	tests := []struct {
		name         string
		fields       []string
		wantEvents   int
		checkPayload func(t *testing.T, events []Event)
	}{
		{
			name:       "filter by status",
			fields:     []string{"status"},
			wantEvents: 2,
			checkPayload: func(t *testing.T, events []Event) {
				for i, e := range events {
					if _, ok := e.Payload["status"]; !ok {
						t.Errorf("event %d missing status field", i)
					}
					if _, ok := e.Payload["email"]; ok {
						t.Errorf("event %d should not have email field", i)
					}
					if len(e.Payload) != 1 {
						t.Errorf("event %d payload should have 1 field, got %d", i, len(e.Payload))
					}
				}
			},
		},
		{
			name:       "filter by email",
			fields:     []string{"email"},
			wantEvents: 2,
			checkPayload: func(t *testing.T, events []Event) {
				for _, e := range events {
					if _, ok := e.Payload["email"]; !ok {
						t.Error("event missing email field")
					}
				}
			},
		},
		{
			name:       "filter by token",
			fields:     []string{"token"},
			wantEvents: 1,
			checkPayload: func(t *testing.T, events []Event) {
				if _, ok := events[0].Payload["token"]; !ok {
					t.Error("event missing token field")
				}
			},
		},
		{
			name:       "filter by multiple fields",
			fields:     []string{"status", "email"},
			wantEvents: 3,
		},
		{
			name:       "filter by non-existent field",
			fields:     []string{"nonexistent"},
			wantEvents: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := logger.Events("entity:1", tt.fields...)
			if len(events) != tt.wantEvents {
				t.Errorf("expected %d events, got %d", tt.wantEvents, len(events))
			}
			if tt.checkPayload != nil {
				tt.checkPayload(t, events)
			}
		})
	}
}

func TestLogger_Events_NonExistentKey(t *testing.T) {
	logger := New()
	events := logger.Events("nonexistent")
	if events == nil {
		t.Error("Events() should not return nil")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestLogger_Logs(t *testing.T) {
	logger := New()

	logger.Create("item:1", "alice", "Created", map[string]Value{
		"color": PlainValue("red"),
		"size":  PlainValue("large"),
	})

	logger.Update("item:1", "bob", "Updated color", map[string]Value{
		"color": PlainValue("blue"),
	})

	logger.Update("item:1", "alice", "Updated size", map[string]Value{
		"size": PlainValue("small"),
	})

	changes := logger.Logs("item:1")
	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(changes))
	}

	// Check first change (create)
	if changes[0].Author != "alice" {
		t.Errorf("expected author alice, got %s", changes[0].Author)
	}
	if len(changes[0].Fields) != 2 {
		t.Fatalf("expected 2 field changes, got %d", len(changes[0].Fields))
	}

	// Check second change (color update)
	if changes[1].Author != "bob" {
		t.Errorf("expected author bob, got %s", changes[1].Author)
	}
	if len(changes[1].Fields) != 1 {
		t.Fatalf("expected 1 field change, got %d", len(changes[1].Fields))
	}
	colorChange := changes[1].Fields[0]
	if colorChange.Field != "color" {
		t.Errorf("expected field 'color', got %s", colorChange.Field)
	}
	if colorChange.From != "red" {
		t.Errorf("expected from 'red', got %v", colorChange.From)
	}
	if colorChange.To != "blue" {
		t.Errorf("expected to 'blue', got %v", colorChange.To)
	}
}

func TestLogger_HiddenValues(t *testing.T) {
	logger := New()

	logger.Create("user:1", "admin", "User created", map[string]Value{
		"email":    PlainValue("user@example.com"),
		"password": HiddenValue(),
	})

	logger.Update("user:1", "admin", "Password updated", map[string]Value{
		"password": HiddenValue(),
	})

	changes := logger.Logs("user:1")
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}

	// Check first change - password should be masked
	passwordField := changes[0].Fields[1] // Assuming password is second field
	if passwordField.Field == "password" {
		if passwordField.From != "***" {
			t.Errorf("expected hidden from value '***', got %v", passwordField.From)
		}
		if passwordField.To != "***" {
			t.Errorf("expected hidden to value '***', got %v", passwordField.To)
		}
	}

	// Check second change - password update should show *** -> ***
	if len(changes[1].Fields) < 1 {
		t.Fatal("expected at least 1 field change in second change")
	}
	passwordUpdate := changes[1].Fields[0]
	if passwordUpdate.Field != "password" {
		t.Errorf("expected field 'password', got %s", passwordUpdate.Field)
	}
	if passwordUpdate.From != "***" {
		t.Errorf("expected hidden from value '***', got %v", passwordUpdate.From)
	}
	if passwordUpdate.To != "***" {
		t.Errorf("expected hidden to value '***', got %v", passwordUpdate.To)
	}
}

func TestLogger_Concurrency(t *testing.T) {
	logger := New()
	const goroutines = 100
	const eventsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				logger.Create(
					fmt.Sprintf("key:%d", id),
					fmt.Sprintf("user:%d", id),
					"Concurrent create",
					map[string]Value{
						"value": PlainValue(j),
					},
				)
			}
		}(i)
	}

	wg.Wait()

	// Verify all events were recorded
	totalEvents := 0
	for i := 0; i < goroutines; i++ {
		events := logger.Events(fmt.Sprintf("key:%d", i))
		totalEvents += len(events)
		if len(events) != eventsPerGoroutine {
			t.Errorf("key:%d expected %d events, got %d", i, eventsPerGoroutine, len(events))
		}
	}

	expected := goroutines * eventsPerGoroutine
	if totalEvents != expected {
		t.Errorf("expected %d total events, got %d", expected, totalEvents)
	}
}

func TestLogger_Concurrency_ReadWrite(t *testing.T) {
	logger := New()
	const duration = 100 * time.Millisecond

	done := make(chan bool)

	// Writer goroutine
	go func() {
		start := time.Now()
		counter := 0
		for time.Since(start) < duration {
			logger.Create("shared-key", "writer", "Write", map[string]Value{
				"count": PlainValue(counter),
			})
			counter++
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		start := time.Now()
		for time.Since(start) < duration {
			_ = logger.Events("shared-key")
			_ = logger.Logs("shared-key")
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify data integrity
	events := logger.Events("shared-key")
	if len(events) == 0 {
		t.Error("expected some events to be recorded")
	}
}

func TestPlainValue(t *testing.T) {
	tests := []struct {
		name       string
		input      any
		comparable bool
	}{
		{"string", "test", true},
		{"int", 42, true},
		{"float", 3.14, true},
		{"bool", true, true},
		{"nil", nil, true},
		{"map", map[string]string{"key": "value"}, false},
		{"slice", []int{1, 2, 3}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := PlainValue(tt.input)
			if v.Hidden {
				t.Error("PlainValue should not be hidden")
			}
			if tt.comparable && v.Data != tt.input {
				t.Errorf("expected data %v, got %v", tt.input, v.Data)
			}
			// For non-comparable types, just check that Data is not nil
			if !tt.comparable && tt.input != nil && v.Data == nil {
				t.Error("expected non-nil data for non-comparable type")
			}
		})
	}
}

func TestHiddenValue(t *testing.T) {
	v := HiddenValue()
	if !v.Hidden {
		t.Error("HiddenValue should be hidden")
	}
	if v.Data != nil {
		t.Errorf("expected nil data, got %v", v.Data)
	}
}
