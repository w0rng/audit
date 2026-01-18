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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Create("key", "author", "description", payload)
	}
}

func BenchmarkLogger_Update(b *testing.B) {
	logger := audit.New()
	payload := map[string]audit.Value{
		"status": audit.PlainValue("updated"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Update("key", "author", "description", payload)
	}
}

func BenchmarkLogger_Events_NoFilter(b *testing.B) {
	logger := audit.New()
	// Populate with 100 events
	for i := 0; i < 100; i++ {
		logger.Create("key", "author", "desc", map[string]audit.Value{
			"field": audit.PlainValue(i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Events("key")
	}
}

func BenchmarkLogger_Events_WithFilter(b *testing.B) {
	logger := audit.New()
	// Populate with 100 events
	for i := 0; i < 100; i++ {
		logger.Create("key", "author", "desc", map[string]audit.Value{
			"field1": audit.PlainValue(i),
			"field2": audit.PlainValue(i * 2),
			"field3": audit.PlainValue(i * 3),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Events("key", "field1")
	}
}

func BenchmarkLogger_Events_MultipleFilters(b *testing.B) {
	logger := audit.New()
	// Populate with 100 events
	for i := 0; i < 100; i++ {
		logger.Create("key", "author", "desc", map[string]audit.Value{
			"field1": audit.PlainValue(i),
			"field2": audit.PlainValue(i * 2),
			"field3": audit.PlainValue(i * 3),
			"field4": audit.PlainValue(i * 4),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Events("key", "field1", "field3")
	}
}

func BenchmarkLogger_Logs(b *testing.B) {
	logger := audit.New()
	// Populate with 50 events
	for i := 0; i < 50; i++ {
		logger.Create("key", "author", "desc", map[string]audit.Value{
			"field1": audit.PlainValue(i),
			"field2": audit.PlainValue(i * 2),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Logs("key")
	}
}

func BenchmarkLogger_HiddenValue(b *testing.B) {
	logger := audit.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Create("key", "author", "desc", map[string]audit.Value{
			"password": audit.HiddenValue(),
		})
	}
}

func BenchmarkPlainValue(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = audit.PlainValue("test")
	}
}

func BenchmarkHiddenValue(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
	for i := 0; i < 100; i++ {
		storage.Store("key", event)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.Has("key")
	}
}

// Benchmark different payload sizes.
func BenchmarkLogger_Create_SmallPayload(b *testing.B) {
	logger := audit.New()
	payload := map[string]audit.Value{
		"field": audit.PlainValue("value"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Create("key", "author", "desc", payload)
	}
}

func BenchmarkLogger_Create_LargePayload(b *testing.B) {
	logger := audit.New()
	payload := make(map[string]audit.Value)
	for i := 0; i < 50; i++ {
		payload[fmt.Sprintf("field%d", i)] = audit.PlainValue(fmt.Sprintf("value%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Create("key", "author", "desc", payload)
	}
}

// Benchmark with different numbers of keys.
func BenchmarkLogger_Events_10Keys(b *testing.B) {
	logger := audit.New()
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			logger.Create(fmt.Sprintf("key%d", i), "author", "desc", map[string]audit.Value{
				"field": audit.PlainValue(j),
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			_ = logger.Events(fmt.Sprintf("key%d", j))
		}
	}
}

func BenchmarkLogger_Events_100Keys(b *testing.B) {
	logger := audit.New()
	for i := 0; i < 100; i++ {
		for j := 0; j < 10; j++ {
			logger.Create(fmt.Sprintf("key%d", i), "author", "desc", map[string]audit.Value{
				"field": audit.PlainValue(j),
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			_ = logger.Events(fmt.Sprintf("key%d", j))
		}
	}
}
