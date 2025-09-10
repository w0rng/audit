package audit

import (
	"slices"
	"sync"
	"time"
)

type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
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

type Logger struct {
	mu     sync.RWMutex
	events map[string][]Event
}

func New() *Logger {
	return &Logger{
		events: make(map[string][]Event),
	}
}

// HiddenValue используется для передачи скрытых полей
func HiddenValue() Value {
	return Value{Hidden: true}
}

// Value создает обычное значение
func PlainValue(v any) Value {
	return Value{Data: v}
}

// LogChange регистрирует новое событие
func (l *Logger) LogChange(key string, action Action, author, description string, payload map[string]Value) {
	l.mu.Lock()
	defer l.mu.Unlock()

	event := Event{
		Timestamp:   time.Now(),
		Action:      action,
		Author:      author,
		Description: description,
		Payload:     payload,
	}

	l.events[key] = append(l.events[key], event)
}

func (l *Logger) Create(key string, author, description string, payload map[string]Value) {
	l.LogChange(key, ActionCreate, author, description, payload)
}

func (l *Logger) Update(key string, author, description string, payload map[string]Value) {
	l.LogChange(key, ActionUpdate, author, description, payload)
}

func (l *Logger) Delete(key string, author, description string, payload map[string]Value) {
	l.LogChange(key, ActionDelete, author, description, payload)
}

// Events возвращает события по ключу и фильтрует по полям
func (l *Logger) Events(key string, fields ...string) []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()

	fieldSet := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		fieldSet[f] = struct{}{}
	}

	events := l.events[key]
	var filtered []Event

	for _, e := range events {
		for k := range e.Payload {
			if _, ok := fieldSet[k]; ok {
				payload := map[string]Value{}
				for k, v := range e.Payload {
					if slices.Contains(fields, k) {
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
				break
			}
		}
	}

	return filtered
}

func (l *Logger) Logs(key string) []Change {
	l.mu.RLock()
	defer l.mu.RUnlock()

	state := make(map[string]any)
	var result []Change

	for _, e := range l.events[key] {
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
				from = "***"
				to = "***"
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
