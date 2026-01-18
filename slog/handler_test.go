package slog_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/w0rng/audit"
	"github.com/w0rng/audit/internal/be"
	auditslog "github.com/w0rng/audit/slog"
)

func TestNewHandler_PanicsWithoutKeyExtractor(t *testing.T) {
	t.Parallel()
	defer func() {
		be.True(t, recover() != nil)
	}()

	logger := audit.New()
	auditslog.NewHandler(logger, auditslog.HandlerOptions{})
}

func TestHandler_Handle_Basic(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	handler := auditslog.NewHandler(logger, auditslog.HandlerOptions{
		KeyExtractor: auditslog.AttrExtractor("entity"),
	})

	record := slog.Record{
		Message: "User created",
	}
	record.AddAttrs(
		slog.String("entity", "user:123"),
		slog.String("email", "test@example.com"),
	)

	be.Err(t, handler.Handle(t.Context(), record), nil)

	// Check that event was logged
	events := logger.Events("user:123")
	be.Equal(t, len(events), 1)

	event := events[0]
	be.Equal(t, event.Description, "User created")
	be.Equal(t, event.Author, "system")
	be.Equal(t, event.Action, audit.ActionCreate)
}

func TestHandler_Handle_WithAction(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	handler := auditslog.NewHandler(logger, auditslog.HandlerOptions{
		KeyExtractor: auditslog.AttrExtractor("entity"),
	})

	record := slog.Record{
		Message: "User updated",
	}
	record.AddAttrs(
		slog.String("entity", "user:123"),
		slog.String("action", "update"),
		slog.String("status", "active"),
	)

	be.Err(t, handler.Handle(t.Context(), record), nil)

	events := logger.Events("user:123")
	be.Equal(t, len(events), 1)
	be.Equal(t, events[0].Action, audit.ActionUpdate)
}

func TestHandler_Handle_WithAuthor(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	handler := auditslog.NewHandler(logger, auditslog.HandlerOptions{
		KeyExtractor: auditslog.AttrExtractor("entity"),
	})
	const author = "admin"

	record := slog.Record{
		Message: "User created",
	}
	record.AddAttrs(
		slog.String("entity", "user:123"),
		slog.String("author", author),
	)

	be.Err(t, handler.Handle(t.Context(), record), nil)

	events := logger.Events("user:123")
	be.Equal(t, len(events), 1)
	be.Equal(t, events[0].Author, author)
}

func TestHandler_Handle_ShouldAudit(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	handler := auditslog.NewHandler(logger, auditslog.HandlerOptions{
		KeyExtractor: auditslog.AttrExtractor("entity"),
		ShouldAudit: func(record slog.Record) bool {
			// Only audit Info level and above
			return record.Level >= slog.LevelInfo
		},
	})

	// Debug level - should not be audited
	debugRecord := slog.Record{
		Message: "Debug message",
		Level:   slog.LevelDebug,
	}
	debugRecord.AddAttrs(slog.String("entity", "user:123"))
	be.Err(t, handler.Handle(t.Context(), debugRecord), nil)

	// Info level - should be audited
	infoRecord := slog.Record{
		Message: "Info message",
		Level:   slog.LevelInfo,
	}
	infoRecord.AddAttrs(slog.String("entity", "user:456"))
	be.Err(t, handler.Handle(t.Context(), infoRecord), nil)

	// Check results
	debugEvents := logger.Events("user:123")
	be.Equal(t, len(debugEvents), 0)

	infoEvents := logger.Events("user:456")
	be.Equal(t, len(infoEvents), 1)
}

func TestHandler_Handle_NoEntityKey(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	handler := auditslog.NewHandler(logger, auditslog.HandlerOptions{
		KeyExtractor: auditslog.AttrExtractor("entity"),
	})

	record := slog.Record{
		Message: "Regular log without entity",
	}
	record.AddAttrs(slog.String("other", "value"))

	err := handler.Handle(t.Context(), record)
	be.Err(t, err, nil)

	// Should not create any audit events
	events := logger.Events("nonexistent")
	be.Equal(t, len(events), 0)
}

func TestHandler_WithAttrs(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	baseHandler := auditslog.NewHandler(logger, auditslog.HandlerOptions{
		KeyExtractor: auditslog.AttrExtractor("entity"),
	})

	// Add attributes
	handlerWithAttrs := baseHandler.WithAttrs([]slog.Attr{
		slog.String("entity", "user:123"),
	})

	record := slog.Record{
		Message: "User updated",
	}
	record.AddAttrs(slog.String("status", "active"))

	handlerWithAttrs.Handle(t.Context(), record)

	// Check that entity from handler attrs was used
	events := logger.Events("user:123")
	be.Equal(t, len(events), 1)
}

func TestHandler_WithGroup(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	handler := auditslog.NewHandler(logger, auditslog.HandlerOptions{
		KeyExtractor: auditslog.AttrExtractor("entity"),
	})

	groupHandler := handler.WithGroup("mygroup")
	be.True(t, groupHandler != nil)

	// Empty group should return same handler
	sameHandler := handler.WithGroup("")
	if sameHandler != handler {
		t.Error("WithGroup with empty name should return same handler")
	}
}

func TestHandler_DelegateToUnderlyingHandler(t *testing.T) {
	t.Parallel()
	logger := audit.New()
	var buf bytes.Buffer

	underlyingHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	handler := auditslog.NewHandler(logger, auditslog.HandlerOptions{
		Handler:      underlyingHandler,
		KeyExtractor: auditslog.AttrExtractor("entity"),
	})

	record := slog.Record{
		Message: "Test message",
		Level:   slog.LevelInfo,
	}
	record.AddAttrs(slog.String("entity", "user:123"))

	err := handler.Handle(t.Context(), record)
	be.Err(t, err, nil)

	// Check that underlying handler received the log
	be.True(t, buf.Len() > 0)

	// Check that audit also received it
	events := logger.Events("user:123")
	be.Equal(t, len(events), 1)
}

func TestDefaultActionExtractor(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := auditslog.DefaultActionExtractor(tt.attrs)
			be.Equal(t, got, tt.want)
		})
	}
}

func TestDefaultAuthorExtractor(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
			got := auditslog.DefaultAuthorExtractor(t.Context(), tt.attrs)
			be.Equal(t, got, tt.want)
		})
	}
}

func TestDefaultPayloadExtractor(t *testing.T) {
	t.Parallel()

	attrs := []slog.Attr{
		slog.String("entity", "user:123"), // should be filtered
		slog.String("action", "create"),   // should be filtered
		slog.String("author", "admin"),    // should be filtered
		slog.String("email", "test@example.com"),
		slog.Int("age", 25),
	}

	payload := auditslog.DefaultPayloadExtractor(attrs)

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
	t.Parallel()

	extractor := auditslog.AttrExtractor("entity")

	attrs := []slog.Attr{
		slog.String("entity", "user:123"),
		slog.String("other", "value"),
	}

	key, found := extractor(attrs)
	be.True(t, found)
	be.Equal(t, key, "user:123")

	// Test not found
	attrs2 := []slog.Attr{
		slog.String("other", "value"),
	}
	_, found = extractor(attrs2)
	be.True(t, !found)
}

func TestHandler_Enabled(t *testing.T) {
	t.Parallel()

	logger := audit.New()

	// Handler without underlying handler
	handler1 := auditslog.NewHandler(logger, auditslog.HandlerOptions{
		KeyExtractor: auditslog.AttrExtractor("entity"),
	})
	be.True(t, handler1.Enabled(t.Context(), slog.LevelInfo))

	// Handler with underlying handler
	underlyingHandler := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})
	handler2 := auditslog.NewHandler(logger, auditslog.HandlerOptions{
		Handler:      underlyingHandler,
		KeyExtractor: auditslog.AttrExtractor("entity"),
	})

	be.True(t, !handler2.Enabled(t.Context(), slog.LevelInfo))
	be.True(t, handler2.Enabled(t.Context(), slog.LevelWarn))
}

func TestHandler_WithConstants(t *testing.T) {
	t.Parallel()

	logger := audit.New()
	handler := auditslog.NewHandler(logger, auditslog.HandlerOptions{
		KeyExtractor: auditslog.AttrExtractor(auditslog.AttrEntity),
	})

	record := slog.Record{
		Message: "Test with constants",
	}
	record.AddAttrs(
		slog.String(auditslog.AttrEntity, "user:123"),
		slog.String(auditslog.AttrAction, "create"),
		slog.String(auditslog.AttrAuthor, "admin"),
	)

	err := handler.Handle(t.Context(), record)
	be.Err(t, err, nil)

	events := logger.Events("user:123")
	be.Equal(t, len(events), 1)
	be.Equal(t, events[0].Action, audit.ActionCreate)
	be.Equal(t, events[0].Author, "admin")
}
