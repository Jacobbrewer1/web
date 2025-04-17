package logging

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewLoggerWithWriter(buf, WithAppName("test"), WithComponent("test-component"))
	l.Info("test")

	got := buf.String()

	require.Contains(t, got, `"app":"test"`)
	require.Contains(t, got, `"component":"test-component"`)
	require.Contains(t, got, `"level":"INFO"`)
}

func TestNewLogger_With(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewLoggerWithWriter(buf, WithAppName("test"), WithComponent("test-component"))
	l = l.With(
		slog.String(KeyHandler, "test-handler"),
	)
	l.Info("test")

	got := buf.String()

	require.Contains(t, got, `"app":"test"`)
	require.Contains(t, got, `"component":"test-component"`)
	require.Contains(t, got, `"level":"INFO"`)
	require.Contains(t, got, `"handler":"test-handler"`)
}

func TestDuplicateKey(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewLoggerWithWriter(buf, WithAppName("test"), WithComponent("test-component"))
	l = LoggerWithComponent(l, "test-component-2") // This will duplicate the component key
	l.Info("test")

	got := buf.String()

	require.Contains(t, got, `"app":"test"`)
	require.Contains(t, got, `"component":"test-component-2"`)
	require.NotContains(t, got, `"component":"test-component"`)
	require.Contains(t, got, `"level":"INFO"`)
}

func TestReplaceAttrs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		attr     slog.Attr
		expected string
	}{
		{
			name:     "full path",
			attr:     slog.String(slog.SourceKey, "/path/to/project/internal/pkg/file.go"),
			expected: "pkg/file.go",
		},
		{
			name:     "with braces",
			attr:     slog.String(slog.SourceKey, "pkg/{test}/file{1}.go"),
			expected: "test/file1.go",
		},
		{
			name:     "single component",
			attr:     slog.String(slog.SourceKey, "file.go"),
			expected: "file.go",
		},
		{
			name:     "two components",
			attr:     slog.String(slog.SourceKey, "pkg/file.go"),
			expected: "pkg/file.go",
		},
		{
			name:     "non-source key",
			attr:     slog.String("other", "/path/to/file.go"),
			expected: "/path/to/file.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := replaceAttrs(nil, tt.attr)
			require.Equal(t, tt.expected, result.Value.String())
		})
	}
}

func BenchmarkReplaceAttrs(b *testing.B) {
	benchmarks := []struct {
		name string
		attr slog.Attr
	}{
		{
			name: "full path",
			attr: slog.String(slog.SourceKey, "/path/to/project/internal/pkg/file.go"),
		},
		{
			name: "with braces",
			attr: slog.String(slog.SourceKey, "pkg/{test}/file{1}.go"),
		},
		{
			name: "single component",
			attr: slog.String(slog.SourceKey, "file.go"),
		},
		{
			name: "two components",
			attr: slog.String(slog.SourceKey, "pkg/file.go"),
		},
		{
			name: "non-source key",
			attr: slog.String("other", "/path/to/file.go"),
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				replaceAttrs(nil, bm.attr)
			}
		})
	}
}
