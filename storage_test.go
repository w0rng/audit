package audit_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/w0rng/audit"
	"github.com/w0rng/audit/internal/be"
)

func TestInMemoryStorage_Store(t *testing.T) {
	t.Parallel()
	storage := audit.NewInMemoryStorage()
	event := audit.Event{
		Timestamp:   time.Now(),
		Action:      audit.ActionCreate,
		Author:      "test",
		Description: "Test event",
		Payload: map[string]audit.Value{
			"field": audit.PlainValue("value"),
		},
	}

	storage.Store("key1", event)
	events := storage.Get("key1")

	be.Equal(t, len(events), 1)
	be.Equal(t, events[0].Author, "test")
}

func TestInMemoryStorage_Store_MultipleEvents(t *testing.T) {
	t.Parallel()
	storage := audit.NewInMemoryStorage()

	for i := range 5 {
		event := audit.Event{
			Timestamp:   time.Now(),
			Action:      audit.ActionCreate,
			Author:      fmt.Sprintf("user%d", i),
			Description: "Test",
			Payload:     map[string]audit.Value{},
		}
		storage.Store("key1", event)
	}

	events := storage.Get("key1")
	be.Equal(t, len(events), 5)
}

func TestInMemoryStorage_Get(t *testing.T) {
	t.Parallel()

	storage := audit.NewInMemoryStorage()
	event := audit.Event{
		Timestamp:   time.Now(),
		Action:      audit.ActionCreate,
		Author:      "test",
		Description: "Test",
		Payload:     map[string]audit.Value{},
	}

	storage.Store("key1", event)

	tests := []struct {
		name       string
		key        string
		wantEvents int
	}{
		{"existing key", "key1", 1},
		{"non-existent key", "key2", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			events := storage.Get(tt.key)
			be.Equal(t, len(events), tt.wantEvents)
		})
	}
}

func TestInMemoryStorage_Has(t *testing.T) {
	t.Parallel()

	storage := audit.NewInMemoryStorage()
	event := audit.Event{
		Timestamp:   time.Now(),
		Action:      audit.ActionCreate,
		Author:      "test",
		Description: "Test",
		Payload:     map[string]audit.Value{},
	}

	storage.Store("key1", event)

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"existing key", "key1", true},
		{"non-existent key", "key2", false},
		{"empty key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			be.Equal(t, storage.Has(tt.key), tt.want)
		})
	}
}

func TestInMemoryStorage_Clear(t *testing.T) {
	t.Parallel()

	storage := audit.NewInMemoryStorage()
	event := audit.Event{
		Timestamp:   time.Now(),
		Action:      audit.ActionCreate,
		Author:      "test",
		Description: "Test",
		Payload:     map[string]audit.Value{},
	}

	storage.Store("key1", event)
	storage.Store("key2", event)

	if !storage.Has("key1") {
		t.Fatal("key1 should exist before Clear")
	}

	storage.Clear("key1")

	be.True(t, !storage.Has("key1"))
	be.True(t, storage.Has("key2"))
}

func TestInMemoryStorage_Clear_NonExistent(t *testing.T) {
	t.Parallel()

	storage := audit.NewInMemoryStorage()
	// Should not panic when clearing non-existent key
	storage.Clear("nonexistent")
}

func TestInMemoryStorage_Concurrency(t *testing.T) {
	t.Parallel()

	storage := audit.NewInMemoryStorage()
	const goroutines = 100
	const eventsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			for j := range eventsPerGoroutine {
				event := audit.Event{
					Timestamp:   time.Now(),
					Action:      audit.ActionCreate,
					Author:      fmt.Sprintf("user%d", id),
					Description: "Concurrent test",
					Payload: map[string]audit.Value{
						"value": audit.PlainValue(j),
					},
				}
				storage.Store(fmt.Sprintf("key:%d", id), event)
			}
		}(i)
	}

	wg.Wait()

	// Verify all events were stored
	totalEvents := 0
	for i := range goroutines {
		events := storage.Get(fmt.Sprintf("key:%d", i))
		totalEvents += len(events)
		be.Equal(t, len(events), eventsPerGoroutine)
	}

	expected := goroutines * eventsPerGoroutine
	be.Equal(t, totalEvents, expected)
}

func TestInMemoryStorage_Concurrency_ReadWrite(t *testing.T) {
	t.Parallel()

	storage := audit.NewInMemoryStorage()
	const duration = 100 * time.Millisecond

	done := make(chan bool, 3)

	// Writer goroutine
	go func() {
		start := time.Now()
		counter := 0
		for time.Since(start) < duration {
			event := audit.Event{
				Timestamp:   time.Now(),
				Action:      audit.ActionCreate,
				Author:      "writer",
				Description: "Write",
				Payload: map[string]audit.Value{
					"count": audit.PlainValue(counter),
				},
			}
			storage.Store("shared-key", event)
			counter++
		}
		done <- true
	}()

	// Reader goroutine 1
	go func() {
		start := time.Now()
		for time.Since(start) < duration {
			_ = storage.Get("shared-key")
			_ = storage.Has("shared-key")
		}
		done <- true
	}()

	// Reader goroutine 2
	go func() {
		start := time.Now()
		for time.Since(start) < duration {
			_ = storage.Get("shared-key")
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	// Verify data integrity
	events := storage.Get("shared-key")
	be.True(t, len(events) > 0)
}

func TestStorageInterface(t *testing.T) {
	// Verify that InMemoryStorage implements Storage interface
	var _ audit.Storage = (*audit.InMemoryStorage)(nil)
}

// mockStorage is a simple mock implementation for testing.
type mockStorage struct {
	mu     sync.RWMutex
	stored map[string][]audit.Event
	calls  map[string]int
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		stored: make(map[string][]audit.Event),
		calls:  make(map[string]int),
	}
}

func (m *mockStorage) Store(key string, event audit.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stored[key] = append(m.stored[key], event)
	m.calls["Store"]++
}

func (m *mockStorage) Get(key string) []audit.Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.calls["Get"]++
	return m.stored[key]
}

func (m *mockStorage) Has(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.calls["Has"]++
	_, ok := m.stored[key]
	return ok
}

func (m *mockStorage) Clear(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["Clear"]++
	delete(m.stored, key)
}

func TestLogger_WithMockStorage(t *testing.T) {
	t.Parallel()

	mock := newMockStorage()
	logger := audit.New(audit.WithStorage(mock))

	logger.Create("test", "user", "Created", map[string]audit.Value{
		"field": audit.PlainValue("value"),
	})

	be.Equal(t, mock.calls["Store"], 1)

	events := logger.Events("test")
	be.Equal(t, len(events), 1)
	be.Equal(t, mock.calls["Get"], 1)
}
