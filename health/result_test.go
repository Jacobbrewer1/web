package health

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResult_SetStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status Status
		expect Status
	}{
		{
			name:   "StatusUp",
			status: StatusUp,
			expect: StatusUp,
		},
		{
			name:   "StatusDown",
			status: StatusDown,
			expect: StatusDown,
		},
		{
			name:   "StatusDegraded",
			status: StatusDegraded,
			expect: StatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newResult()
			r.SetStatus(tt.status)
			require.Equal(t, tt.expect, r.Status)
		})
	}
}

func TestResult_SetStatusOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		statusOne Status
		statusTwo Status
		expect    Status
	}{
		{
			name:      "StatusUp to StatusDown",
			statusOne: StatusUp,
			statusTwo: StatusDown,
			expect:    StatusDown,
		},
		{
			name:      "StatusDown to StatusUp",
			statusOne: StatusDown,
			statusTwo: StatusUp,
			expect:    StatusDown,
		},
		{
			name:      "StatusDegraded to StatusUp",
			statusOne: StatusDegraded,
			statusTwo: StatusUp,
			expect:    StatusDegraded,
		},
		{
			name:      "StatusUp to StatusDegraded",
			statusOne: StatusUp,
			statusTwo: StatusDegraded,
			expect:    StatusDegraded,
		},
		{
			name:      "StatusDegraded to StatusDown",
			statusOne: StatusDegraded,
			statusTwo: StatusDown,
			expect:    StatusDown,
		},
		{
			name:      "StatusDown to StatusDegraded",
			statusOne: StatusDown,
			statusTwo: StatusDegraded,
			expect:    StatusDown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newResult()
			r.SetStatus(tt.statusOne)
			r.SetStatus(tt.statusTwo)
			require.Equal(t, tt.expect, r.Status)
		})
	}
}

func TestResult_AddDetail_NilMap(t *testing.T) {
	t.Parallel()

	res := new(Result)
	res.mtx = new(sync.RWMutex)

	require.NotPanics(t, func() {
		res.addDetail("test", &Result{Status: StatusUp})
	})

	require.NotNil(t, res.Details, "Details map should be initialized")
	require.Len(t, res.Details, 1)
}
