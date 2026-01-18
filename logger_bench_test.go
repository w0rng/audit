package audit

import (
	"fmt"
	"testing"
)

func BenchmarkLogger_Create(b *testing.B) {
	logger := New()
	payload := map[string]Value{
		"field1": PlainValue("value1"),
		"field2": PlainValue("value2"),
		"field3": PlainValue(123),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Create("key", "author", "description", payload)
	}
}

func BenchmarkLogger_Update(b *testing.B) {
	logger := New()
	payload := map[string]Value{
		"status": PlainValue("updated"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Update("key", "author", "description", payload)
	}
}

func BenchmarkLogger_Events_NoFilter(b *testing.B) {
	logger := New()
	// Populate with 100 events
	for i := 0; i < 100; i++ {
		logger.Create("key", "author", "desc", map[string]Value{
			"field": PlainValue(i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Events("key")
	}
}

func BenchmarkLogger_Events_WithFilter(b *testing.B) {
	logger := New()
	// Populate with 100 events
	for i := 0; i < 100; i++ {
		logger.Create("key", "author", "desc", map[string]Value{
			"field1": PlainValue(i),
			"field2": PlainValue(i * 2),
			"field3": PlainValue(i * 3),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Events("key", "field1")
	}
}

func BenchmarkLogger_Events_MultipleFilters(b *testing.B) {
	logger := New()
	// Populate with 100 events
	for i := 0; i < 100; i++ {
		logger.Create("key", "author", "desc", map[string]Value{
			"field1": PlainValue(i),
			"field2": PlainValue(i * 2),
			"field3": PlainValue(i * 3),
			"field4": PlainValue(i * 4),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Events("key", "field1", "field3")
	}
}

func BenchmarkLogger_Logs(b *testing.B) {
	logger := New()
	// Populate with 50 events
	for i := 0; i < 50; i++ {
		logger.Create("key", "author", "desc", map[string]Value{
			"field1": PlainValue(i),
			"field2": PlainValue(i * 2),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Logs("key")
	}
}

func BenchmarkLogger_HiddenValue(b *testing.B) {
	logger := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Create("key", "author", "desc", map[string]Value{
			"password": HiddenValue(),
		})
	}
}

func BenchmarkPlainValue(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PlainValue("test")
	}
}

func BenchmarkHiddenValue(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HiddenValue()
	}
}

func BenchmarkInMemoryStorage_Store(b *testing.B) {
	storage := NewInMemoryStorage()
	event := Event{
		Action:      ActionCreate,
		Author:      "test",
		Description: "Test",
		Payload: map[string]Value{
			"field": PlainValue("value"),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.Store("key", event)
	}
}

func BenchmarkInMemoryStorage_Get(b *testing.B) {
	storage := NewInMemoryStorage()
	event := Event{
		Action:      ActionCreate,
		Author:      "test",
		Description: "Test",
		Payload: map[string]Value{
			"field": PlainValue("value"),
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
	storage := NewInMemoryStorage()
	event := Event{
		Action:      ActionCreate,
		Author:      "test",
		Description: "Test",
		Payload:     map[string]Value{},
	}
	storage.Store("key", event)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.Has("key")
	}
}

// Benchmark different payload sizes
func BenchmarkLogger_Create_SmallPayload(b *testing.B) {
	logger := New()
	payload := map[string]Value{
		"field": PlainValue("value"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Create("key", "author", "desc", payload)
	}
}

func BenchmarkLogger_Create_MediumPayload(b *testing.B) {
	logger := New()
	payload := map[string]Value{
		"field1":  PlainValue("value1"),
		"field2":  PlainValue("value2"),
		"field3":  PlainValue("value3"),
		"field4":  PlainValue("value4"),
		"field5":  PlainValue("value5"),
		"field6":  PlainValue("value6"),
		"field7":  PlainValue("value7"),
		"field8":  PlainValue("value8"),
		"field9":  PlainValue("value9"),
		"field10": PlainValue("value10"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Create("key", "author", "desc", payload)
	}
}

func BenchmarkLogger_Create_LargePayload(b *testing.B) {
	logger := New()
	payload := make(map[string]Value)
	for i := 0; i < 50; i++ {
		payload[fmt.Sprintf("field%d", i)] = PlainValue(fmt.Sprintf("value%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Create("key", "author", "desc", payload)
	}
}

// Benchmark with different numbers of keys
func BenchmarkLogger_Events_10Keys(b *testing.B) {
	logger := New()
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			logger.Create(fmt.Sprintf("key%d", i), "author", "desc", map[string]Value{
				"field": PlainValue(j),
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
	logger := New()
	for i := 0; i < 100; i++ {
		for j := 0; j < 10; j++ {
			logger.Create(fmt.Sprintf("key%d", i), "author", "desc", map[string]Value{
				"field": PlainValue(j),
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
