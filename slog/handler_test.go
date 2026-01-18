package slog

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/w0rng/audit"
)

func TestNewHandler(t *testing.T) {
	logger := audit.New()
	opts := HandlerOptions{
		KeyExtractor: AttrExtractor("entity"),
	}

	handler := NewHandler(logger, opts)
	if handler == nil {
		t.Fatal("NewHandler returned nil")
	}
	if handler.logger != logger {
		t.Error("Handler logger not set correctly")
	}
}

func TestNewHandler_PanicsWithoutKeyExtractor(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewHandler should panic when KeyExtractor is nil")
		}
	}()

	logger := audit.New()
	NewHandler(logger, HandlerOptions{})
}

func TestHandler_Handle_Basic(t *testing.T) {
	logger := audit.New()
	handler := NewHandler(logger, HandlerOptions{
		KeyExtractor: AttrExtractor("entity"),
	})

	ctx := context.Background()
	record := slog.Record{
		Message: "User created",
	}
	record.AddAttrs(
		slog.String("entity", "user:123"),
		slog.String("email", "test@example.com"),
	)

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	// Check that event was logged
	events := logger.Events("user:123")
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.Description != "User created" {
		t.Errorf("expected description 'User created', got %q", event.Description)
	}
	if event.Author != "system" {
		t.Errorf("expected author 'system', got %q", event.Author)
	}
	if event.Action != audit.ActionCreate {
		t.Errorf("expected action Create, got %v", event.Action)
	}
}

func TestHandler_Handle_WithAction(t *testing.T) {
	logger := audit.New()
	handler := NewHandler(logger, HandlerOptions{
		KeyExtractor: AttrExtractor("entity"),
	})

	ctx := context.Background()
	record := slog.Record{
		Message: "User updated",
	}
	record.AddAttrs(
		slog.String("entity", "user:123"),
		slog.String("action", "update"),
		slog.String("status", "active"),
	)

	handler.Handle(ctx, record)

	events := logger.Events("user:123")
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Action != audit.ActionUpdate {
		t.Errorf("expected action Update, got %v", events[0].Action)
	}
}

func TestHandler_Handle_WithAuthor(t *testing.T) {
	logger := audit.New()
	handler := NewHandler(logger, HandlerOptions{
		KeyExtractor: AttrExtractor("entity"),
	})

	ctx := context.Background()
	record := slog.Record{
		Message: "User created",
	}
	record.AddAttrs(
		slog.String("entity", "user:123"),
		slog.String("author", "admin"),
	)

	handler.Handle(ctx, record)

	events := logger.Events("user:123")
	if events[0].Author != "admin" {
		t.Errorf("expected author 'admin', got %q", events[0].Author)
	}
}

func TestHandler_Handle_ShouldAudit(t *testing.T) {
	logger := audit.New()
	handler := NewHandler(logger, HandlerOptions{
		KeyExtractor: AttrExtractor("entity"),
		ShouldAudit: func(record slog.Record) bool {
			// Only audit Info level and above
			return record.Level >= slog.LevelInfo
		},
	})

	ctx := context.Background()

	// Debug level - should not be audited
	debugRecord := slog.Record{
		Message: "Debug message",
		Level:   slog.LevelDebug,
	}
	debugRecord.AddAttrs(slog.String("entity", "user:123"))
	handler.Handle(ctx, debugRecord)

	// Info level - should be audited
	infoRecord := slog.Record{
		Message: "Info message",
		Level:   slog.LevelInfo,
	}
	infoRecord.AddAttrs(slog.String("entity", "user:456"))
	handler.Handle(ctx, infoRecord)

	// Check results
	debugEvents := logger.Events("user:123")
	if len(debugEvents) != 0 {
		t.Error("Debug record should not be audited")
	}

	infoEvents := logger.Events("user:456")
	if len(infoEvents) != 1 {
		t.Errorf("Info record should be audited, got %d events", len(infoEvents))
	}
}

func TestHandler_Handle_NoEntityKey(t *testing.T) {
	logger := audit.New()
	handler := NewHandler(logger, HandlerOptions{
		KeyExtractor: AttrExtractor("entity"),
	})

	ctx := context.Background()
	record := slog.Record{
		Message: "Regular log without entity",
	}
	record.AddAttrs(slog.String("other", "value"))

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Errorf("Handle should not error when entity key not found: %v", err)
	}

	// Should not create any audit events
	events := logger.Events("nonexistent")
	if len(events) != 0 {
		t.Error("Should not create audit events without entity key")
	}
}

func TestHandler_WithAttrs(t *testing.T) {
	logger := audit.New()
	baseHandler := NewHandler(logger, HandlerOptions{
		KeyExtractor: AttrExtractor("entity"),
	})

	// Add attributes
	handlerWithAttrs := baseHandler.WithAttrs([]slog.Attr{
		slog.String("entity", "user:123"),
	})

	ctx := context.Background()
	record := slog.Record{
		Message: "User updated",
	}
	record.AddAttrs(slog.String("status", "active"))

	handlerWithAttrs.Handle(ctx, record)

	// Check that entity from handler attrs was used
	events := logger.Events("user:123")
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestHandler_WithGroup(t *testing.T) {
	logger := audit.New()
	handler := NewHandler(logger, HandlerOptions{
		KeyExtractor: AttrExtractor("entity"),
	})

	groupHandler := handler.WithGroup("mygroup")
	if groupHandler == nil {
		t.Fatal("WithGroup returned nil")
	}

	// Empty group should return same handler
	sameHandler := handler.WithGroup("")
	if sameHandler != handler {
		t.Error("WithGroup with empty name should return same handler")
	}
}

func TestHandler_DelegateToUnderlyingHandler(t *testing.T) {
	logger := audit.New()
	var buf bytes.Buffer

	underlyingHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	handler := NewHandler(logger, HandlerOptions{
		Handler:      underlyingHandler,
		KeyExtractor: AttrExtractor("entity"),
	})

	ctx := context.Background()
	record := slog.Record{
		Message: "Test message",
		Level:   slog.LevelInfo,
	}
	record.AddAttrs(slog.String("entity", "user:123"))

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	// Check that underlying handler received the log
	if buf.Len() == 0 {
		t.Error("Underlying handler should have received the log")
	}

	// Check that audit also received it
	events := logger.Events("user:123")
	if len(events) != 1 {
		t.Error("Audit should also have received the log")
	}
}

func TestDefaultActionExtractor(t *testing.T) {
	tests := []struct {
		name  string
		attrs []slog.Attr
		want  audit.Action
	}{
		{
			name:  "no action attribute",
			attrs: []slog.Attr{},
			want:  audit.ActionCreate,
		},
		{
			name: "create action",
			attrs: []slog.Attr{
				slog.String("action", "create"),
			},
			want: audit.ActionCreate,
		},
		{
			name: "update action",
			attrs: []slog.Attr{
				slog.String("action", "update"),
			},
			want: audit.ActionUpdate,
		},
		{
			name: "delete action",
			attrs: []slog.Attr{
				slog.String("action", "delete"),
			},
			want: audit.ActionDelete,
		},
		{
			name: "unknown action defaults to create",
			attrs: []slog.Attr{
				slog.String("action", "unknown"),
			},
			want: audit.ActionCreate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultActionExtractor(tt.attrs)
			if got != tt.want {
				t.Errorf("DefaultActionExtractor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultAuthorExtractor(t *testing.T) {
	tests := []struct {
		name  string
		attrs []slog.Attr
		want  string
	}{
		{
			name:  "no author attribute",
			attrs: []slog.Attr{},
			want:  "system",
		},
		{
			name: "author attribute",
			attrs: []slog.Attr{
				slog.String("author", "admin"),
			},
			want: "admin",
		},
		{
			name: "user attribute",
			attrs: []slog.Attr{
				slog.String("user", "john"),
			},
			want: "john",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultAuthorExtractor(context.Background(), tt.attrs)
			if got != tt.want {
				t.Errorf("DefaultAuthorExtractor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultPayloadExtractor(t *testing.T) {
	attrs := []slog.Attr{
		slog.String("entity", "user:123"), // should be filtered
		slog.String("action", "create"),   // should be filtered
		slog.String("author", "admin"),    // should be filtered
		slog.String("email", "test@example.com"),
		slog.Int("age", 25),
	}

	payload := DefaultPayloadExtractor(attrs)

	// Should not include reserved keys
	if _, ok := payload["entity"]; ok {
		t.Error("Payload should not include 'entity' key")
	}
	if _, ok := payload["action"]; ok {
		t.Error("Payload should not include 'action' key")
	}
	if _, ok := payload["author"]; ok {
		t.Error("Payload should not include 'author' key")
	}

	// Should include other attributes
	if _, ok := payload["email"]; !ok {
		t.Error("Payload should include 'email' key")
	}
	if _, ok := payload["age"]; !ok {
		t.Error("Payload should include 'age' key")
	}
}

func TestAttrExtractor(t *testing.T) {
	extractor := AttrExtractor("entity")

	attrs := []slog.Attr{
		slog.String("entity", "user:123"),
		slog.String("other", "value"),
	}

	key, found := extractor(attrs)
	if !found {
		t.Error("AttrExtractor should find the attribute")
	}
	if key != "user:123" {
		t.Errorf("expected key 'user:123', got %q", key)
	}

	// Test not found
	attrs2 := []slog.Attr{
		slog.String("other", "value"),
	}
	_, found = extractor(attrs2)
	if found {
		t.Error("AttrExtractor should not find missing attribute")
	}
}

func TestHandler_Enabled(t *testing.T) {
	logger := audit.New()

	// Handler without underlying handler
	handler1 := NewHandler(logger, HandlerOptions{
		KeyExtractor: AttrExtractor("entity"),
	})
	if !handler1.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Handler without underlying handler should always be enabled")
	}

	// Handler with underlying handler
	underlyingHandler := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})
	handler2 := NewHandler(logger, HandlerOptions{
		Handler:      underlyingHandler,
		KeyExtractor: AttrExtractor("entity"),
	})

	if handler2.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Handler should delegate Enabled to underlying handler")
	}
	if !handler2.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("Handler should delegate Enabled to underlying handler")
	}
}
