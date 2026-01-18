// Package slog provides integration between Go's structured logging (slog) and audit logging.
// It allows audit events to be automatically created from slog log entries based on
// configurable extraction and filtering rules.
package slog

import (
	"context"
	"log/slog"

	"github.com/w0rng/audit"
)

// Attribute keys used for audit logging.
// Use these constants when logging to ensure correct extraction.
const (
	// AttrEntity is the key for the entity identifier (required for audit).
	// Example: slog.Info("...", slog.AttrEntity, "user:123")
	AttrEntity = "entity"

	// AttrAction is the key for the action type (create, update, delete).
	// Example: slog.Info("...", slog.AttrAction, "update")
	AttrAction = "action"

	// AttrAuthor is the key for the author/user who performed the action.
	// Example: slog.Info("...", slog.AttrAuthor, "admin")
	AttrAuthor = "author"

	// AttrUser is an alternative key for the author (use either AttrAuthor or AttrUser).
	// Example: slog.Info("...", slog.AttrUser, "john.doe")
	AttrUser = "user"
)

// Handler is a slog.Handler that writes audit logs based on slog records.
// It delegates to another handler for normal logging while optionally
// sending matching records to an audit logger.
type Handler struct {
	logger  *audit.Logger
	handler slog.Handler
	opts    HandlerOptions
	attrs   []slog.Attr
	groups  []string
}

// HandlerOptions configures how slog records are converted to audit logs.
type HandlerOptions struct {
	// Handler is the underlying slog.Handler to delegate to for normal logging.
	// If nil, logs will only go to audit (no regular logging).
	Handler slog.Handler

	// ShouldAudit determines whether a log record should be sent to audit.
	// If nil, all records are audited.
	ShouldAudit func(record slog.Record) bool

	// KeyExtractor extracts the entity key from log attributes.
	// Required. Must return (key, true) if found, ("", false) otherwise.
	KeyExtractor func(attrs []slog.Attr) (string, bool)

	// ActionExtractor extracts the action from log attributes.
	// If nil, uses ActionCreate by default.
	ActionExtractor func(attrs []slog.Attr) audit.Action

	// AuthorExtractor extracts the author from log attributes or context.
	// If nil, uses "system" as default.
	AuthorExtractor func(ctx context.Context, attrs []slog.Attr) string

	// PayloadExtractor extracts the payload from log attributes.
	// If nil, includes all attributes except those used for key/action/author.
	PayloadExtractor func(attrs []slog.Attr) map[string]audit.Value
}

// NewHandler creates a new slog.Handler that sends matching records to audit.
//
// Example:
//
//	handler := slog.NewHandler(auditLogger, slog.HandlerOptions{
//	    Handler: slog.NewJSONHandler(os.Stdout, nil),
//	    KeyExtractor: func(attrs []slog.Attr) (string, bool) {
//	        for _, attr := range attrs {
//	            if attr.Key == "entity" {
//	                return attr.Value.String(), true
//	            }
//	        }
//	        return "", false
//	    },
//	    ShouldAudit: func(record slog.Record) bool {
//	        return record.Level >= slog.LevelInfo
//	    },
//	})
func NewHandler(logger *audit.Logger, opts HandlerOptions) *Handler {
	if opts.KeyExtractor == nil {
		panic("slog: KeyExtractor is required")
	}

	// Set defaults
	if opts.ActionExtractor == nil {
		opts.ActionExtractor = DefaultActionExtractor
	}
	if opts.AuthorExtractor == nil {
		opts.AuthorExtractor = DefaultAuthorExtractor
	}
	if opts.PayloadExtractor == nil {
		opts.PayloadExtractor = DefaultPayloadExtractor
	}

	return &Handler{
		logger:  logger,
		opts:    opts,
		handler: opts.Handler,
		attrs:   []slog.Attr{},
		groups:  []string{},
	}
}

// Enabled reports whether the handler handles records at the given level.
// It delegates to the underlying handler if present.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.handler != nil {
		return h.handler.Enabled(ctx, level)
	}
	return true
}

// Handle processes a slog.Record, optionally sending it to audit.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	// Delegate to underlying handler first
	if h.handler != nil {
		if err := h.handler.Handle(ctx, record); err != nil {
			return err
		}
	}

	// Check if this record should be audited
	if h.opts.ShouldAudit != nil && !h.opts.ShouldAudit(record) {
		return nil
	}

	// Collect all attributes (handler-level + record-level)
	allAttrs := make([]slog.Attr, 0, len(h.attrs)+record.NumAttrs())
	allAttrs = append(allAttrs, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		allAttrs = append(allAttrs, attr)
		return true
	})

	// Extract entity key
	key, ok := h.opts.KeyExtractor(allAttrs)
	if !ok {
		// No entity key found, skip audit logging
		return nil
	}

	// Extract other fields
	action := h.opts.ActionExtractor(allAttrs)
	author := h.opts.AuthorExtractor(ctx, allAttrs)
	payload := h.opts.PayloadExtractor(allAttrs)

	// Log to audit
	h.logger.LogChange(key, action, author, record.Message, payload)

	return nil
}

// WithAttrs returns a new Handler with additional attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := *h
	newHandler.attrs = make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newHandler.attrs, h.attrs)
	copy(newHandler.attrs[len(h.attrs):], attrs)

	if h.handler != nil {
		newHandler.handler = h.handler.WithAttrs(attrs)
	}

	return &newHandler
}

// WithGroup returns a new Handler with a group name.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newHandler := *h
	newHandler.groups = make([]string, len(h.groups)+1)
	copy(newHandler.groups, h.groups)
	newHandler.groups[len(h.groups)] = name

	if h.handler != nil {
		newHandler.handler = h.handler.WithGroup(name)
	}

	return &newHandler
}

// DefaultActionExtractor extracts action from AttrAction attribute.
// Defaults to ActionCreate if not found.
func DefaultActionExtractor(attrs []slog.Attr) audit.Action {
	for _, attr := range attrs {
		if attr.Key == AttrAction {
			switch attr.Value.String() {
			case "create":
				return audit.ActionCreate
			case "update":
				return audit.ActionUpdate
			case "delete":
				return audit.ActionDelete
			default:
				return audit.ActionCreate
			}
		}
	}
	return audit.ActionCreate
}

// DefaultAuthorExtractor extracts author from AttrAuthor or AttrUser attribute.
// Defaults to "system" if not found.
func DefaultAuthorExtractor(ctx context.Context, attrs []slog.Attr) string {
	for _, attr := range attrs {
		if attr.Key == AttrAuthor || attr.Key == AttrUser {
			return attr.Value.String()
		}
	}
	return "system"
}

// DefaultPayloadExtractor includes all attributes except reserved keys.
// Reserved keys: AttrEntity, AttrAction, AttrAuthor, AttrUser
func DefaultPayloadExtractor(attrs []slog.Attr) map[string]audit.Value {
	payload := make(map[string]audit.Value)
	reservedKeys := map[string]bool{
		AttrEntity: true,
		AttrAction: true,
		AttrAuthor: true,
		AttrUser:   true,
	}

	for _, attr := range attrs {
		if reservedKeys[attr.Key] {
			continue
		}

		// Convert slog.Value to audit.Value
		payload[attr.Key] = audit.PlainValue(attr.Value.Any())
	}

	return payload
}

// AttrExtractor is a helper to extract a specific attribute by key.
func AttrExtractor(key string) func(attrs []slog.Attr) (string, bool) {
	return func(attrs []slog.Attr) (string, bool) {
		for _, attr := range attrs {
			if attr.Key == key {
				return attr.Value.String(), true
			}
		}
		return "", false
	}
}
