package health

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func compareResult(t *testing.T, expected, actual *Result) {
	t.Helper()

	if expected.Timestamp != nil && actual.Timestamp != nil {
		require.WithinDuration(t, *expected.Timestamp, *actual.Timestamp, 1*time.Second, "Timestamp mismatch: expected %v, got %v", expected.Timestamp, actual.Timestamp)
	} else {
		require.Equal(t, expected.Timestamp, actual.Timestamp, "Timestamp mismatch: expected %v, got %v", expected.Timestamp, actual.Timestamp)
	}

	require.Equal(t, expected.Status, actual.Status, "Status mismatch: expected %v, got %v", expected.Status, actual.Status)
	require.Equal(t, expected.Error, actual.Error, "Error mismatch: expected %v, got %v", expected.Error, actual.Error)

	require.Len(t, expected.Details, len(actual.Details), "Details length mismatch: expected %d, got %d", len(expected.Details), len(actual.Details))

	for k := range expected.Details {
		compareResult(t, expected.Details[k], actual.Details[k])
	}
}

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

			r := NewResult()
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

			r := NewResult()
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
