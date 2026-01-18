package audit

import "sync"

// Storage defines the interface for storing and retrieving audit events.
// Implementations must be safe for concurrent access.
type Storage interface {
	// Store appends an event to the storage for the given key
	Store(key string, event Event)

	// Get retrieves all events for a given key.
	// Returns an empty slice if key doesn't exist.
	Get(key string) []Event

	// Has checks if any events exist for a given key
	Has(key string) bool

	// Clear removes all events for a given key
	Clear(key string)
}

// InMemoryStorage provides a thread-safe in-memory storage implementation
// backed by a map. This is the default storage used by New().
type InMemoryStorage struct {
	mu     sync.RWMutex
	events map[string][]Event
}

// NewInMemoryStorage creates a new in-memory storage instance.
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		events: make(map[string][]Event),
	}
}

// Store appends an event to the storage for the given key.
func (s *InMemoryStorage) Store(key string, event Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[key] = append(s.events[key], event)
}

// Get retrieves all events for a given key.
// Returns an empty slice if the key doesn't exist.
func (s *InMemoryStorage) Get(key string) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := s.events[key]
	if events == nil {
		return []Event{}
	}
	return events
}

// Has checks if any events exist for a given key.
func (s *InMemoryStorage) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.events[key]
	return ok
}

// Clear removes all events for a given key.
func (s *InMemoryStorage) Clear(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.events, key)
}
