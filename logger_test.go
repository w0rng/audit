package audit_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/w0rng/audit"
	"github.com/w0rng/audit/internal/be"
)

func TestLogger_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		author      string
		description string
		payload     map[string]audit.Value
		wantAction  audit.Action
	}{
		{
			name:        "create with plain values",
			key:         "user:1",
			author:      "admin",
			description: "User created",
			payload: map[string]audit.Value{
				"email": audit.PlainValue("user@example.com"),
				"role":  audit.PlainValue("admin"),
			},
			wantAction: audit.ActionCreate,
		},
		{
			name:        "create with hidden value",
			key:         "user:2",
			author:      "system",
			description: "User registered",
			payload: map[string]audit.Value{
				"email":    audit.PlainValue("test@example.com"),
				"password": audit.HiddenValue(),
			},
			wantAction: audit.ActionCreate,
		},
		{
			name:        "create with empty payload",
			key:         "user:3",
			author:      "admin",
			description: "User initialized",
			payload:     map[string]audit.Value{},
			wantAction:  audit.ActionCreate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := audit.New()
			logger.Create(tt.key, tt.author, tt.description, tt.payload)

			events := logger.Events(tt.key)
			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}

			event := events[0]
			be.Equal(t, event.Action, tt.wantAction)
			be.Equal(t, event.Author, tt.author)
			be.Equal(t, event.Description, tt.description)
			be.Equal(t, event.Payload, tt.payload)
		})
	}
}

func TestLogger_Update(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	logger.Create("order:1", "user", "Created", map[string]audit.Value{
		"status": audit.PlainValue("pending"),
	})
	logger.Update("order:1", "user", "Updated", map[string]audit.Value{
		"status": audit.PlainValue("approved"),
	})

	events := logger.Events("order:1")
	be.Equal(t, len(events), 2)
	be.Equal(t, events[1].Action, audit.ActionUpdate)
}

func TestLogger_Delete(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	logger.Create("item:1", "user", "Created", map[string]audit.Value{
		"name": audit.PlainValue("test"),
	})
	logger.Delete("item:1", "admin", "Deleted", map[string]audit.Value{})

	events := logger.Events("item:1")
	be.Equal(t, len(events), 2)
	be.Equal(t, events[1].Action, audit.ActionDelete)
}

func TestLogger_Events_NoFilter(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	logger.Create("entity:1", "user", "Created", map[string]audit.Value{
		"field1": audit.PlainValue("value1"),
		"field2": audit.PlainValue("value2"),
	})
	logger.Update("entity:1", "user", "Updated", map[string]audit.Value{
		"field1": audit.PlainValue("new-value"),
	})

	events := logger.Events("entity:1")
	be.Equal(t, len(events), 2)
}

func TestLogger_Events_Filtering(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	logger.Create("entity:1", "user", "Created", map[string]audit.Value{
		"status": audit.PlainValue("new"),
		"email":  audit.PlainValue("test@example.com"),
		"token":  audit.HiddenValue(),
	})
	logger.Update("entity:1", "user", "Updated status", map[string]audit.Value{
		"status": audit.PlainValue("active"),
	})
	logger.Update("entity:1", "user", "Updated email", map[string]audit.Value{
		"email": audit.PlainValue("new@example.com"),
	})

	tests := []struct {
		name         string
		fields       []string
		wantEvents   int
		checkPayload func(t *testing.T, events []audit.Event)
	}{
		{
			name:       "filter by status",
			fields:     []string{"status"},
			wantEvents: 2,
			checkPayload: func(t *testing.T, events []audit.Event) {
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
			checkPayload: func(t *testing.T, events []audit.Event) {
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
			checkPayload: func(t *testing.T, events []audit.Event) {
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
			t.Parallel()
			events := logger.Events("entity:1", tt.fields...)
			be.Equal(t, len(events), tt.wantEvents)
			if tt.checkPayload != nil {
				tt.checkPayload(t, events)
			}
		})
	}
}

func TestLogger_Events_NonExistentKey(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	events := logger.Events("nonexistent")
	be.Equal(t, len(events), 0)
}

func TestLogger_Logs(t *testing.T) {
	t.Parallel()
	logger := audit.New()

	logger.Create("item:1", "alice", "Created", map[string]audit.Value{
		"color": audit.PlainValue("red"),
		"size":  audit.PlainValue("large"),
	})

	logger.Update("item:1", "bob", "Updated color", map[string]audit.Value{
		"color": audit.PlainValue("blue"),
	})

	logger.Update("item:1", "alice", "Updated size", map[string]audit.Value{
		"size": audit.PlainValue("small"),
	})

	changes := logger.Logs("item:1")
	be.Equal(t, len(changes), 3)

	// Check first change (create)
	be.Equal(t, changes[0].Author, "alice")
	be.Equal(t, len(changes[0].Fields), 2)

	// Check second change (color update)
	be.Equal(t, changes[1].Author, "bob")
	be.Equal(t, len(changes[1].Fields), 1)

	colorChange := changes[1].Fields[0]
	be.Equal(t, colorChange.Field, "color")
	be.Equal(t, colorChange.From, "red")
	be.Equal(t, colorChange.To, "blue")
}

func TestLogger_HiddenValues(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	const hideField = "password"

	logger.Create("user:1", "admin", "User created", map[string]audit.Value{
		"email":   audit.PlainValue("user@example.com"),
		hideField: audit.HiddenValue(),
	})

	logger.Update("user:1", "admin", "Password updated", map[string]audit.Value{
		hideField: audit.HiddenValue(),
	})

	changes := logger.Logs("user:1")
	be.Equal(t, len(changes), 2)

	// Check first change - password should be masked
	passwordField := changes[0].Fields[1] // Assuming password is second field
	if passwordField.Field == hideField {
		be.Equal(t, passwordField.From.(string), audit.HideText)
		be.Equal(t, passwordField.To.(string), audit.HideText)
	}

	// Check second change - password update should show *** -> ***
	be.True(t, len(changes[1].Fields) >= 1)
	passwordUpdate := changes[1].Fields[0]
	be.Equal(t, passwordUpdate.Field, hideField)
	be.Equal(t, passwordUpdate.From.(string), audit.HideText)
	be.Equal(t, passwordUpdate.To.(string), audit.HideText)
}

func TestLogger_Concurrency(t *testing.T) {
	t.Parallel()
	logger := audit.New()
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
					map[string]audit.Value{
						"value": audit.PlainValue(j),
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
		be.Equal(t, len(events), eventsPerGoroutine)
	}

	expected := goroutines * eventsPerGoroutine
	be.Equal(t, totalEvents, expected)
}

func TestLogger_Concurrency_ReadWrite(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	const duration = 100 * time.Millisecond

	done := make(chan bool)

	// Writer goroutine
	go func() {
		start := time.Now()
		counter := 0
		for time.Since(start) < duration {
			logger.Create("shared-key", "writer", "Write", map[string]audit.Value{
				"count": audit.PlainValue(counter),
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
	be.True(t, len(events) > 0)
}

func TestPlainValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input any
	}{
		{"string", "test"},
		{"int", 42},
		{"float", 3.14},
		{"bool", true},
		{"nil", nil},
		{"map", map[string]string{"key": "value"}},
		{"slice", []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := audit.PlainValue(tt.input)
			be.True(t, !v.Hidden)
			be.Equal(t, v.Data, tt.input)
		})
	}
}

func TestHiddenValue(t *testing.T) {
	t.Parallel()
	v := audit.HiddenValue()
	be.True(t, v.Hidden)
	be.Equal(t, v.Data, nil)
}
