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
