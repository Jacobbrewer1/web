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
			name:     "full path with line number",
			attr:     slog.String(slog.SourceKey, "/path/to/project/internal/pkg/file.go 111"),
			expected: "pkg/file.go 111",
		},
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
		{
			name:     "bazel path",
			attr:     slog.String(slog.SourceKey, "gazelle~~go_deps~com_github_jacobbrewer1_web/app.go"),
			expected: "web/app.go",
		},
		{
			name:     "bazel path with line number",
			attr:     slog.String(slog.SourceKey, "gazelle~~go_deps~com_github_jacobbrewer1_web/app.go 271"),
			expected: "web/app.go 271",
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
			name: "full path with line number",
			attr: slog.String(slog.SourceKey, "/path/to/project/internal/pkg/file.go 111"),
		},
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
		{
			name: "bazel path",
			attr: slog.String(slog.SourceKey, "gazelle~~go_deps~com_github_jacobbrewer1_web/app.go"),
		},
		{
			name: "bazel path with line number",
			attr: slog.String(slog.SourceKey, "gazelle~~go_deps~com_github_jacobbrewer1_web/app.go 271"),
		},
	}

	b.ResetTimer()
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				replaceAttrs(nil, bm.attr)
			}
		})
	}
}
