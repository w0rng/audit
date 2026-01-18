package audit_test

import (
	"fmt"
	"testing"

	"github.com/w0rng/audit"
)

func BenchmarkLogger_Create(b *testing.B) {
	logger := audit.New()
	payload := map[string]audit.Value{
		"field1": audit.PlainValue("value1"),
		"field2": audit.PlainValue("value2"),
		"field3": audit.PlainValue(123),
	}

	for b.Loop() {
		logger.Create("key", "author", "description", payload)
	}
}

func BenchmarkLogger_Update(b *testing.B) {
	logger := audit.New()
	payload := map[string]audit.Value{
		"status": audit.PlainValue("updated"),
	}

	for b.Loop() {
		logger.Update("key", "author", "description", payload)
	}
}

func BenchmarkLogger_Events_NoFilter(b *testing.B) {
	logger := audit.New()
	// Populate with 100 events
	for i := range 100 {
		logger.Create("key", "author", "desc", map[string]audit.Value{
			"field": audit.PlainValue(i),
		})
	}

	for b.Loop() {
		_ = logger.Events("key")
	}
}

func BenchmarkLogger_Events_WithFilter(b *testing.B) {
	logger := audit.New()
	// Populate with 100 events
	for i := range 100 {
		logger.Create("key", "author", "desc", map[string]audit.Value{
			"field1": audit.PlainValue(i),
			"field2": audit.PlainValue(i * 2),
			"field3": audit.PlainValue(i * 3),
		})
	}

	for b.Loop() {
		_ = logger.Events("key", "field1")
	}
}

func BenchmarkLogger_Events_MultipleFilters(b *testing.B) {
	logger := audit.New()
	// Populate with 100 events
	for i := range 100 {
		logger.Create("key", "author", "desc", map[string]audit.Value{
			"field1": audit.PlainValue(i),
			"field2": audit.PlainValue(i * 2),
			"field3": audit.PlainValue(i * 3),
			"field4": audit.PlainValue(i * 4),
		})
	}

	for b.Loop() {
		_ = logger.Events("key", "field1", "field3")
	}
}

func BenchmarkLogger_Logs(b *testing.B) {
	logger := audit.New()
	// Populate with 50 events
	for i := range 50 {
		logger.Create("key", "author", "desc", map[string]audit.Value{
			"field1": audit.PlainValue(i),
			"field2": audit.PlainValue(i * 2),
		})
	}

	for b.Loop() {
		_ = logger.Logs("key")
	}
}

func BenchmarkLogger_HiddenValue(b *testing.B) {
	logger := audit.New()

	for b.Loop() {
		logger.Create("key", "author", "desc", map[string]audit.Value{
			"password": audit.HiddenValue(),
		})
	}
}

func BenchmarkPlainValue(b *testing.B) {
	for b.Loop() {
		_ = audit.PlainValue("test")
	}
}

func BenchmarkHiddenValue(b *testing.B) {
	for b.Loop() {
		_ = audit.HiddenValue()
	}
}

func BenchmarkInMemoryStorage_Store(b *testing.B) {
	storage := audit.NewInMemoryStorage()
	event := audit.Event{
		Action:      audit.ActionCreate,
		Author:      "test",
		Description: "Test",
		Payload: map[string]audit.Value{
			"field": audit.PlainValue("value"),
		},
	}

	for b.Loop() {
		storage.Store("key", event)
	}
}

func BenchmarkInMemoryStorage_Get(b *testing.B) {
	storage := audit.NewInMemoryStorage()
	event := audit.Event{
		Action:      audit.ActionCreate,
		Author:      "test",
		Description: "Test",
		Payload: map[string]audit.Value{
			"field": audit.PlainValue("value"),
		},
	}

	// Populate with 100 events
	for range 100 {
		storage.Store("key", event)
	}

	for b.Loop() {
		_ = storage.Get("key")
	}
}

func BenchmarkInMemoryStorage_Has(b *testing.B) {
	storage := audit.NewInMemoryStorage()
	event := audit.Event{
		Action:      audit.ActionCreate,
		Author:      "test",
		Description: "Test",
		Payload:     map[string]audit.Value{},
	}
	storage.Store("key", event)

	for b.Loop() {
		_ = storage.Has("key")
	}
}

// Benchmark different payload sizes.
func BenchmarkLogger_Create_SmallPayload(b *testing.B) {
	logger := audit.New()
	payload := map[string]audit.Value{
		"field": audit.PlainValue("value"),
	}

	for b.Loop() {
		logger.Create("key", "author", "desc", payload)
	}
}

func BenchmarkLogger_Create_MediumPayload(b *testing.B) {
	logger := audit.New()
	payload := map[string]audit.Value{
		"field1":  audit.PlainValue("value1"),
		"field2":  audit.PlainValue("value2"),
		"field3":  audit.PlainValue("value3"),
		"field4":  audit.PlainValue("value4"),
		"field5":  audit.PlainValue("value5"),
		"field6":  audit.PlainValue("value6"),
		"field7":  audit.PlainValue("value7"),
		"field8":  audit.PlainValue("value8"),
		"field9":  audit.PlainValue("value9"),
		"field10": audit.PlainValue("value10"),
	}

	for b.Loop() {
		logger.Create("key", "author", "desc", payload)
	}
}

func BenchmarkLogger_Create_LargePayload(b *testing.B) {
	logger := audit.New()
	payload := make(map[string]audit.Value)
	for i := range 50 {
		payload[fmt.Sprintf("field%d", i)] = audit.PlainValue(fmt.Sprintf("value%d", i))
	}

	for b.Loop() {
		logger.Create("key", "author", "desc", payload)
	}
}

// Benchmark with different numbers of keys.
func BenchmarkLogger_Events_10Keys(b *testing.B) {
	logger := audit.New()
	for i := range 10 {
		for j := range 10 {
			logger.Create(fmt.Sprintf("key%d", i), "author", "desc", map[string]audit.Value{
				"field": audit.PlainValue(j),
			})
		}
	}

	for b.Loop() {
		for j := range 10 {
			_ = logger.Events(fmt.Sprintf("key%d", j))
		}
	}
}

func BenchmarkLogger_Events_100Keys(b *testing.B) {
	logger := audit.New()
	for i := range 100 {
		for j := range 10 {
			logger.Create(fmt.Sprintf("key%d", i), "author", "desc", map[string]audit.Value{
				"field": audit.PlainValue(j),
			})
		}
	}

	for b.Loop() {
		for j := range 100 {
			_ = logger.Events(fmt.Sprintf("key%d", j))
		}
	}
}
