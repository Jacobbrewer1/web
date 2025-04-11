package logging

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func BenchmarkDeDupe_WithAttrs(b *testing.B) {
	buf := bytes.NewBuffer(nil)
	handler := NewLoggerWithWriter(buf, WithAppName("test"), WithComponent("test-component"))

	// Reset the timer after initialization to exclude setup time
	b.ResetTimer()

	// Prefill the attribute to be used in the benchmark
	attr := slog.String("key", "value")

	// Benchmark logging with deduplication and attribute handling
	for i := 0; i < b.N; i++ {
		// Use the logger's Info method to log with attributes
		handler.Info("test", attr)
	}
}

func BenchmarkDeDupe_Handle_Parallel(b *testing.B) {
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
}

func BenchmarkDeDupe_Handle(b *testing.B) {
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
	for i := 0; i < b.N; i++ {
		err := handler.Handler().Handle(context.Background(), record)
		require.NoError(b, err)
	}
}

func BenchmarkDeDupe_WithAttrs_Parallel(b *testing.B) {
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
}
