package logging

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDedupeHandler(t *testing.T) {
	t.Parallel()

	t.Run("enabled passthrough", func(t *testing.T) {
		t.Parallel()
		buf := bytes.NewBuffer(nil)
		base := slog.NewJSONHandler(buf, nil)
		handler := NewDedupeHandler(base)

		require.Equal(t, base.Enabled(context.Background(), slog.LevelInfo),
			handler.Enabled(context.Background(), slog.LevelInfo))
	})

	t.Run("handle with deduplicated attrs", func(t *testing.T) {
		t.Parallel()
		buf := bytes.NewBuffer(nil)
		base := slog.NewJSONHandler(buf, nil)
		handler := NewDedupeHandler(base)

		// Add some initial attributes
		handler = handler.WithAttrs([]slog.Attr{
			slog.String("key1", "value1"),
			slog.Int("key2", 42),
		})

		record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
		err := handler.Handle(context.Background(), record)
		require.NoError(t, err)

		result := buf.String()
		require.Contains(t, result, `"key1":"value1"`)
		require.Contains(t, result, `"key2":42`)
	})

	t.Run("withAttrs deduplication", func(t *testing.T) {
		t.Parallel()
		buf := bytes.NewBuffer(nil)
		base := slog.NewJSONHandler(buf, nil)
		handler := NewDedupeHandler(base)

		// Add initial attributes
		handler = handler.WithAttrs([]slog.Attr{
			slog.String("key1", "value1"),
			slog.Int("key2", 42),
		})

		// Add overlapping attributes
		handler = handler.WithAttrs([]slog.Attr{
			slog.String("key1", "newvalue"), // Should override
			slog.String("key3", "value3"),   // Should add
		})

		record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
		err := handler.Handle(context.Background(), record)
		require.NoError(t, err)

		result := buf.String()
		require.Contains(t, result, `"key1":"newvalue"`)
		require.Contains(t, result, `"key2":42`)
		require.Contains(t, result, `"key3":"value3"`)
	})

	t.Run("withGroup handling", func(t *testing.T) {
		t.Parallel()
		buf := bytes.NewBuffer(nil)
		base := slog.NewJSONHandler(buf, nil)
		handler := NewDedupeHandler(base)

		// Add attributes and create group
		handler = handler.WithAttrs([]slog.Attr{
			slog.String("key1", "value1"),
		})
		groupHandler := handler.WithGroup("group1")

		// Add more attributes to group
		groupHandler = groupHandler.WithAttrs([]slog.Attr{
			slog.Int("key2", 42),
		})

		record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
		err := groupHandler.Handle(context.Background(), record)
		require.NoError(t, err)

		result := buf.String()
		require.Contains(t, result, `"key1":"value1"`)
		require.Contains(t, result, `"key2":42`)
	})

	t.Run("record immutability", func(t *testing.T) {
		t.Parallel()
		buf := bytes.NewBuffer(nil)
		base := slog.NewJSONHandler(buf, nil)
		handler := NewDedupeHandler(base)

		handler = handler.WithAttrs([]slog.Attr{
			slog.String("key1", "value1"),
		})

		// Create original record
		original := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
		originalAttrs := make([]slog.Attr, 0)
		original.Attrs(func(a slog.Attr) bool {
			originalAttrs = append(originalAttrs, a)
			return true
		})

		// Handle the record
		err := handler.Handle(context.Background(), original)
		require.NoError(t, err)

		// Verify original record wasn't modified
		currentAttrs := make([]slog.Attr, 0)
		original.Attrs(func(a slog.Attr) bool {
			currentAttrs = append(currentAttrs, a)
			return true
		})
		require.Equal(t, originalAttrs, currentAttrs)
	})
}

func BenchmarkDeDupe(b *testing.B) {
	b.Run("WithAttr", func(b *testing.B) {
		buf := bytes.NewBuffer(nil)
		handler := NewLoggerWithWriter(buf, WithAppName("test"), WithComponent("test-component"))

		// Reset the timer after initialization to exclude setup time
		b.ResetTimer()

		// Prefill the attribute to be used in the benchmark
		attr := slog.String("key", "value")

		// Benchmark logging with deduplication and attribute handling
		for range b.N {
			// Use the logger's Info method to log with attributes
			handler.Info("test", attr)
		}
	})

	b.Run("WithAttr Parallel", func(b *testing.B) {
		buf := bytes.NewBuffer(nil)
		handler := NewLoggerWithWriter(buf, WithAppName("test"), WithComponent("test-component"))

		// Pre-fill the attribute to be used in the benchmark
		attr := slog.String("key", "value")

		// Run benchmark with parallelism
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Use the logger's Info method to log with attributes
				handler.Info("test", attr)
			}
		})
	})

	b.Run("Handle", func(b *testing.B) {
		// Set up the buffer and logger
		buf := bytes.NewBuffer(nil)
		handler := NewLoggerWithWriter(buf, WithAppName("test"), WithComponent("test-component"))

		// Prepare the record to log
		record := slog.NewRecord(
			time.Now().UTC(),
			slog.LevelInfo,
			"test",
			0,
		)

		// Reset the timer to exclude setup time from benchmarking
		b.ResetTimer()

		// Benchmark logging with deduplication by calling the handler's Handle method
		for range b.N {
			err := handler.Handler().Handle(context.Background(), record)
			require.NoError(b, err)
		}
	})

	b.Run("Handle Parallel", func(b *testing.B) {
		buf := bytes.NewBuffer(nil)
		handler := NewLoggerWithWriter(buf, WithAppName("test"), WithComponent("test-component"))

		// Pre-fill the record to be logged in the benchmark
		record := slog.NewRecord(time.Now().UTC(), slog.LevelInfo, "test", 0)

		// Run benchmark with parallelism
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := handler.Handler().Handle(context.Background(), record)
				if err != nil {
					b.Errorf("failed to handle log record: %v", err)
				}
			}
		})
	})
}
