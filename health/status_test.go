package health

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatus_MarshalJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status Status
		want   string
	}{
		{
			name:   "StatusUp",
			status: StatusUp,
			want:   `"up"`,
		},
		{
			name:   "StatusDown",
			status: StatusDown,
			want:   `"down"`,
		},
		{
			name:   "StatusDegraded",
			status: StatusDegraded,
			want:   `"degraded"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.status.MarshalJSON()
			require.NoError(t, err, "MarshalJSON() should not return an error")
			require.JSONEq(t, tt.want, string(got), "MarshalJSON() should return the correct JSON")
		})
	}
}
