package health

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatus_MarshalJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		status  Status
		want    string
		wantErr error
	}{
		{
			name:    "StatusUp",
			status:  StatusUp,
			want:    `"up"`,
			wantErr: nil,
		},
		{
			name:    "StatusDown",
			status:  StatusDown,
			want:    `"down"`,
			wantErr: nil,
		},
		{
			name:    "StatusDegraded",
			status:  StatusDegraded,
			want:    `"degraded"`,
			wantErr: nil,
		},
		{
			name:    "StatusUnknown",
			status:  StatusUnknown,
			want:    `"unknown"`,
			wantErr: nil,
		},
		{
			name:    "InvalidStatus",
			status:  Status(999),
			want:    ``,
			wantErr: errors.New("invalid is not a valid status"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.status.MarshalJSON()
			require.Equal(t, tt.wantErr, err, "MarshalJSON() should not return an error")

			if tt.wantErr == nil {
				require.JSONEq(t, tt.want, string(got), "MarshalJSON() should return the correct JSON")
			}
		})
	}
}
