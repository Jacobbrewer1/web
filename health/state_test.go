package health

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func compareState(t *testing.T, expected, actual *State) {
	t.Helper()

	require.WithinDuration(t, expected.lastCheckTime, actual.lastCheckTime, 1*time.Second, "lastCheckTime mismatch: expected %v, got %v", expected.lastCheckTime, actual.lastCheckTime)
	require.WithinDuration(t, expected.lastSuccess, actual.lastSuccess, 1*time.Second, "lastSuccess mismatch: expected %v, got %v", expected.lastSuccess, actual.lastSuccess)
	require.WithinDuration(t, expected.lastFail, actual.lastFail, 1*time.Second, "lastFail mismatch: expected %v, got %v", expected.lastFail, actual.lastFail)
	require.WithinDuration(t, expected.firstFailInCycle, actual.firstFailInCycle, 1*time.Second, "firstFailInCycle mismatch: expected %v, got %v", expected.firstFailInCycle, actual.firstFailInCycle)

	require.Equal(t, expected.contiguousFails.Load(), actual.contiguousFails.Load(), "contiguousFails mismatch")
	require.Equal(t, expected.checkErr, actual.checkErr, "checkErr mismatch")
	require.Equal(t, expected.status, actual.status, "status mismatch")
}
