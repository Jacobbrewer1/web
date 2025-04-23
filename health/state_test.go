package health

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func compareState(t *testing.T, expected, actual *State) {
	t.Helper()

	require.True(t, expected.lastCheckTime.Equal(actual.lastCheckTime), "lastCheckTime mismatch: expected %v, got %v", expected.lastCheckTime, actual.lastCheckTime)
	require.True(t, expected.lastSuccess.Equal(actual.lastSuccess), "lastSuccess mismatch: expected %v, got %v", expected.lastSuccess, actual.lastSuccess)
	require.True(t, expected.lastFail.Equal(actual.lastFail), "lastFail mismatch: expected %v, got %v", expected.lastFail, actual.lastFail)
	require.Equal(t, expected.contiguousFails.Load(), actual.contiguousFails.Load(), "contiguousFails mismatch")
	require.Equal(t, expected.checkErr, actual.checkErr, "checkErr mismatch")
	require.Equal(t, expected.status, actual.status, "status mismatch")
	require.True(t, expected.firstFailInCycle.Equal(actual.firstFailInCycle), "firstFailInCycle mismatch: expected %v, got %v", expected.firstFailInCycle, actual.firstFailInCycle)
}
