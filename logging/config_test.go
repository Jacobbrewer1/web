package logging

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLoggingConfig(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want *loggingConfig
	}{
		{
			name: "default",
			env:  nil,
			want: &loggingConfig{
				Level: slog.LevelInfo,
			},
		},
		{
			name: "debug",
			env: map[string]string{
				"LOG_LEVEL": "debug",
			},
			want: &loggingConfig{
				Level: slog.LevelDebug,
			},
		},
		{
			name: "info",
			env: map[string]string{
				"LOG_LEVEL": "info",
			},
			want: &loggingConfig{
				Level: slog.LevelInfo,
			},
		},
		{
			name: "warn",
			env: map[string]string{
				"LOG_LEVEL": "warn",
			},
			want: &loggingConfig{
				Level: slog.LevelWarn,
			},
		},
		{
			name: "error",
			env: map[string]string{
				"LOG_LEVEL": "error",
			},
			want: &loggingConfig{
				Level: slog.LevelError,
			},
		},
		{
			name: "invalid",
			env: map[string]string{
				"LOG_LEVEL": "invalid",
			},
			want: &loggingConfig{
				Level: slog.LevelInfo,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != nil {
				for k, v := range tt.env {
					t.Setenv(k, v)
				}
			}

			got := newLoggingConfig()
			require.Equal(t, tt.want, got)
		})
	}
}
