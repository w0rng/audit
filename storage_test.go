package audit

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewInMemoryStorage(t *testing.T) {
	storage := NewInMemoryStorage()
	if storage == nil {
		t.Fatal("NewInMemoryStorage() returned nil")
	}
	if storage.events == nil {
		t.Fatal("storage.events is nil")
	}
}

func TestInMemoryStorage_Store(t *testing.T) {
	storage := NewInMemoryStorage()
	event := Event{
		Timestamp:   time.Now(),
		Action:      ActionCreate,
		Author:      "test",
		Description: "Test event",
		Payload: map[string]Value{
			"field": PlainValue("value"),
		},
	}

	storage.Store("key1", event)
	events := storage.Get("key1")

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Author != "test" {
		t.Errorf("expected author 'test', got %s", events[0].Author)
	}
}

func TestInMemoryStorage_Store_MultipleEvents(t *testing.T) {
	storage := NewInMemoryStorage()

	for i := 0; i < 5; i++ {
		event := Event{
			Timestamp:   time.Now(),
			Action:      ActionCreate,
			Author:      fmt.Sprintf("user%d", i),
			Description: "Test",
			Payload:     map[string]Value{},
		}
		storage.Store("key1", event)
	}

	events := storage.Get("key1")
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
}

func TestInMemoryStorage_Get(t *testing.T) {
	storage := NewInMemoryStorage()
	event := Event{
		Timestamp:   time.Now(),
		Action:      ActionCreate,
		Author:      "test",
		Description: "Test",
		Payload:     map[string]Value{},
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
			events := storage.Get(tt.key)
			if events == nil {
				t.Fatal("Get() returned nil")
			}
			if len(events) != tt.wantEvents {
				t.Errorf("expected %d events, got %d", tt.wantEvents, len(events))
			}
		})
	}
}

func TestInMemoryStorage_Has(t *testing.T) {
	storage := NewInMemoryStorage()
	event := Event{
		Timestamp:   time.Now(),
		Action:      ActionCreate,
		Author:      "test",
		Description: "Test",
		Payload:     map[string]Value{},
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
			if got := storage.Has(tt.key); got != tt.want {
				t.Errorf("Has(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestInMemoryStorage_Clear(t *testing.T) {
	storage := NewInMemoryStorage()
	event := Event{
		Timestamp:   time.Now(),
		Action:      ActionCreate,
		Author:      "test",
		Description: "Test",
		Payload:     map[string]Value{},
	}

	storage.Store("key1", event)
	storage.Store("key2", event)

	if !storage.Has("key1") {
		t.Fatal("key1 should exist before Clear")
	}

	storage.Clear("key1")

	if storage.Has("key1") {
		t.Error("key1 should not exist after Clear")
	}
	if !storage.Has("key2") {
		t.Error("key2 should still exist after clearing key1")
	}
}

func TestInMemoryStorage_Clear_NonExistent(t *testing.T) {
	storage := NewInMemoryStorage()
	// Should not panic when clearing non-existent key
	storage.Clear("nonexistent")
}

func TestInMemoryStorage_Concurrency(t *testing.T) {
	storage := NewInMemoryStorage()
	const goroutines = 100
	const eventsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := Event{
					Timestamp:   time.Now(),
					Action:      ActionCreate,
					Author:      fmt.Sprintf("user%d", id),
					Description: "Concurrent test",
					Payload: map[string]Value{
						"value": PlainValue(j),
					},
				}
				storage.Store(fmt.Sprintf("key:%d", id), event)
			}
		}(i)
	}

	wg.Wait()

	// Verify all events were stored
	totalEvents := 0
	for i := 0; i < goroutines; i++ {
		events := storage.Get(fmt.Sprintf("key:%d", i))
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

func TestInMemoryStorage_Concurrency_ReadWrite(t *testing.T) {
	storage := NewInMemoryStorage()
	const duration = 100 * time.Millisecond

	done := make(chan bool, 3)

	// Writer goroutine
	go func() {
		start := time.Now()
		counter := 0
		for time.Since(start) < duration {
			event := Event{
				Timestamp:   time.Now(),
				Action:      ActionCreate,
				Author:      "writer",
				Description: "Write",
				Payload: map[string]Value{
					"count": PlainValue(counter),
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
	if len(events) == 0 {
		t.Error("expected some events to be stored")
	}
}

func TestStorageInterface(t *testing.T) {
	// Verify that InMemoryStorage implements Storage interface
	var _ Storage = (*InMemoryStorage)(nil)
}

// mockStorage is a simple mock implementation for testing
type mockStorage struct {
	mu     sync.RWMutex
	stored map[string][]Event
	calls  map[string]int
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		stored: make(map[string][]Event),
		calls:  make(map[string]int),
	}
}

func (m *mockStorage) Store(key string, event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stored[key] = append(m.stored[key], event)
	m.calls["Store"]++
}

func (m *mockStorage) Get(key string) []Event {
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
	mock := newMockStorage()
	logger := New(WithStorage(mock))

	logger.Create("test", "user", "Created", map[string]Value{
		"field": PlainValue("value"),
	})

	if mock.calls["Store"] != 1 {
		t.Errorf("expected 1 Store call, got %d", mock.calls["Store"])
	}

	events := logger.Events("test")
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if mock.calls["Get"] != 1 {
		t.Errorf("expected 1 Get call, got %d", mock.calls["Get"])
	}
}
