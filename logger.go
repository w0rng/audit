// Package audit provides a simple, thread-safe audit logging library for tracking
// entity changes and events in Go applications.
//
// The library supports:
//   - Creating, updating, and deleting entity audit logs
//   - Tracking field-level changes with before/after values
//   - Hiding sensitive data (e.g., passwords, tokens)
//   - Concurrent access with sync.RWMutex
//   - Filtering events by payload fields
//
// Basic usage:
//
//	logger := audit.New()
//	logger.Create("user:123", "admin", "User created", map[string]audit.Value{
//	    "email": audit.PlainValue("user@example.com"),
//	    "password": audit.HiddenValue(),
//	})
//
// See the examples directory for complete usage examples.
package audit

import (
	"time"
)

type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"

	HideText string = "***"
)

type Value struct {
	Data   any
	Hidden bool
}

type ChangeField struct {
	Field string
	From  any
	To    any
}

type Change struct {
	Fields      []ChangeField
	Description string
	Author      string
	Timestamp   time.Time
}

type Event struct {
	Timestamp   time.Time
	Action      Action
	Author      string
	Description string
	Payload     map[string]Value
}

// Logger provides thread-safe audit logging functionality.
type Logger struct {
	storage Storage
}

// Option is a function that configures a Logger.
type Option func(*Logger)

// WithStorage sets a custom storage implementation for the logger.
// If not specified, NewInMemoryStorage() is used by default.
func WithStorage(storage Storage) Option {
	return func(l *Logger) {
		l.storage = storage
	}
}

// New creates a new Logger with the given options.
// If no options are provided, it uses in-memory storage by default.
//
// Example:
//
//	logger := audit.New() // uses in-memory storage
//	logger := audit.New(audit.WithStorage(customStorage)) // uses custom storage
func New(opts ...Option) *Logger {
	l := &Logger{
		storage: NewInMemoryStorage(), // default storage
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// HiddenValue creates a Value with Hidden=true to mask sensitive data in logs.
// Hidden values are displayed as "***" to prevent exposure of passwords, tokens, etc.
func HiddenValue() Value {
	return Value{Hidden: true}
}

// PlainValue creates a Value with the given data visible in logs.
// Use this for non-sensitive data that can be safely logged.
func PlainValue(v any) Value {
	return Value{Data: v}
}

// LogChange records a new audit event for the given key with the specified action,
// author, description, and payload. This is the core logging method used by Create,
// Update, and Delete convenience methods.
func (l *Logger) LogChange(key string, action Action, author, description string, payload map[string]Value) {
	event := Event{
		Timestamp:   time.Now(),
		Action:      action,
		Author:      author,
		Description: description,
		Payload:     payload,
	}

	l.storage.Store(key, event)
}

func (l *Logger) Create(key, author, description string, payload map[string]Value) {
	l.LogChange(key, ActionCreate, author, description, payload)
}

func (l *Logger) Update(key, author, description string, payload map[string]Value) {
	l.LogChange(key, ActionUpdate, author, description, payload)
}

func (l *Logger) Delete(key, author, description string, payload map[string]Value) {
	l.LogChange(key, ActionDelete, author, description, payload)
}

// Events retrieves audit events for a key, optionally filtering by specific payload fields.
// If no fields are specified, all events for the key are returned.
// When fields are provided, only events containing at least one of those fields are returned,
// with their payloads filtered to include only the requested fields.
func (l *Logger) Events(key string, fields ...string) []Event {
	events := l.storage.Get(key)

	// If no fields specified, return all events
	if len(fields) == 0 {
		result := make([]Event, len(events))
		copy(result, events)
		return result
	}

	// Build field set for O(1) lookup
	fieldSet := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		fieldSet[f] = struct{}{}
	}

	var filtered []Event
	for _, e := range events {
		// Check if event has any of the requested fields
		hasField := false
		for k := range e.Payload {
			if _, ok := fieldSet[k]; ok {
				hasField = true
				break
			}
		}

		if !hasField {
			continue
		}

		// Build filtered payload using fieldSet (not slices.Contains)
		payload := make(map[string]Value)
		for k, v := range e.Payload {
			if _, ok := fieldSet[k]; ok {
				payload[k] = v
			}
		}

		filtered = append(filtered, Event{
			Timestamp:   e.Timestamp,
			Action:      e.Action,
			Author:      e.Author,
			Description: e.Description,
			Payload:     payload,
		})
	}

	return filtered
}

// Logs returns the complete change history for a key with field-level state transitions.
// It reconstructs the state over time, tracking before/after values for each field.
func (l *Logger) Logs(key string) []Change {
	events := l.storage.Get(key)
	state := make(map[string]any)
	result := make([]Change, 0, len(events))

	for _, e := range events {
		change := Change{
			Description: e.Description,
			Author:      e.Author,
			Timestamp:   e.Timestamp,
			Fields:      make([]ChangeField, 0, len(e.Payload)),
		}
		for field, val := range e.Payload {
			old := state[field]

			from, to := old, val.Data
			if val.Hidden {
				from = HideText
				to = HideText
			}

			if val.Hidden || old != val.Data {
				change.Fields = append(change.Fields, ChangeField{
					Field: field,
					From:  from,
					To:    to,
				})
				if !val.Hidden {
					state[field] = val.Data
				}
			}
		}
		result = append(result, change)
	}

	return result
}
