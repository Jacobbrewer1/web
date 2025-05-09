package health

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewCheck_Check(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		statusListenerCalled := false
		c := NewCheck("test", func(ctx context.Context) error {
			return nil
		},
			WithCheckOnStatusChange(func(ctx context.Context, name string, state *State) {
				statusListenerCalled = true
				require.Equal(t, "test", name)
				require.Equal(t, StatusUp, state.status)
			}),
		)

		err := c.Check(context.Background())
		require.NoError(t, err)

		expectedState := &State{
			lastCheckTime:   now,
			lastSuccess:     now,
			lastFail:        time.Time{},
			contiguousFails: atomic.Uint32{},
			checkErr:        nil,
			status:          StatusUp,
		}
		compareState(t, expectedState, c.state)
		require.True(t, statusListenerCalled)
	})

	t.Run("fail check", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		statusListenerCalled := false
		c := NewCheck("test", func(ctx context.Context) error {
			return errors.New("test error")
		},
			WithCheckOnStatusChange(func(ctx context.Context, name string, state *State) {
				statusListenerCalled = true
				require.Equal(t, "test", name)
				require.Equal(t, StatusDown, state.status)
			}),
		)

		err := c.Check(context.Background())
		require.EqualError(t, err, "test error")

		expectedState := &State{
			lastCheckTime:    now,
			lastSuccess:      time.Time{},
			lastFail:         now,
			contiguousFails:  atomic.Uint32{},
			checkErr:         errors.New("test error"),
			status:           StatusDown,
			firstFailInCycle: now,
		}
		expectedState.contiguousFails.Store(1)
		compareState(t, expectedState, c.state)
		require.True(t, statusListenerCalled)
	})

	t.Run("success to fail to success", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		statusListenerCalled := 0
		callNumber := 0
		c := NewCheck("test", func(ctx context.Context) error {
			callNumber++
			if callNumber%2 == 0 {
				return errors.New("test error")
			}
			return nil
		},
			WithCheckOnStatusChange(func(ctx context.Context, name string, state *State) {
				statusListenerCalled++
				if statusListenerCalled%2 == 0 {
					require.Equal(t, "test", name)
					require.Equal(t, StatusDown, state.status)
				} else {
					require.Equal(t, "test", name)
					require.Equal(t, StatusUp, state.status)
				}
			}),
		)

		err := c.Check(context.Background())
		require.NoError(t, err, "First Check() should not return an error")

		expectedState := &State{
			lastCheckTime:    now,
			lastSuccess:      now,
			lastFail:         time.Time{},
			contiguousFails:  atomic.Uint32{},
			checkErr:         nil,
			status:           StatusUp,
			firstFailInCycle: time.Time{},
		}
		expectedState.contiguousFails.Store(0)
		compareState(t, expectedState, c.state)

		err = c.Check(context.Background())
		require.EqualError(t, err, "test error", "Second Check() should return the correct error")

		expectedState = &State{
			lastCheckTime:    now,
			lastSuccess:      now,
			lastFail:         now,
			contiguousFails:  atomic.Uint32{},
			checkErr:         errors.New("test error"),
			status:           StatusDown,
			firstFailInCycle: now,
		}
		expectedState.contiguousFails.Store(1)
		compareState(t, expectedState, c.state)

		err = c.Check(context.Background())
		require.NoError(t, err, "Third Check() should not return an error")

		expectedState = &State{
			lastCheckTime:    now,
			lastSuccess:      now,
			lastFail:         now,
			contiguousFails:  atomic.Uint32{},
			checkErr:         nil,
			status:           StatusUp,
			firstFailInCycle: time.Time{},
		}
		expectedState.contiguousFails.Store(0)
		compareState(t, expectedState, c.state)
		require.Equal(t, 3, statusListenerCalled)
	})

	t.Run("max contiguous fails", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		statusListenerCalled := 0
		c := NewCheck("test", func(ctx context.Context) error {
			return errors.New("test error")
		},
			WithCheckMaxFailures(3),
			WithCheckOnStatusChange(func(ctx context.Context, name string, state *State) {
				statusListenerCalled++
				require.Equal(t, "test", name)

				if state.contiguousFails.Load() < 3 {
					require.Equal(t, StatusUp, state.status)
				} else {
					require.Equal(t, StatusDown, state.status)
				}
			}),
		)

		err := c.Check(context.Background())
		require.EqualError(t, err, "test error", "First Check() should return the correct error")

		expectedState := &State{
			lastCheckTime:    now,
			lastSuccess:      time.Time{},
			lastFail:         now,
			contiguousFails:  atomic.Uint32{},
			checkErr:         errors.New("test error"),
			status:           StatusUp,
			firstFailInCycle: now,
		}
		expectedState.contiguousFails.Store(1)
		compareState(t, expectedState, c.state)
		require.Equal(t, 1, statusListenerCalled)

		err = c.Check(context.Background())
		require.EqualError(t, err, "test error", "Second Check() should return the correct error")

		expectedState = &State{
			lastCheckTime:    now,
			lastSuccess:      time.Time{},
			lastFail:         now,
			contiguousFails:  atomic.Uint32{},
			checkErr:         errors.New("test error"),
			status:           StatusUp,
			firstFailInCycle: now,
		}
		expectedState.contiguousFails.Store(2)
		compareState(t, expectedState, c.state)
		require.Equal(t, 1, statusListenerCalled)

		err = c.Check(context.Background())
		require.EqualError(t, err, "test error", "Third Check() should return the correct error")

		expectedState = &State{
			lastCheckTime:    now,
			lastSuccess:      time.Time{},
			lastFail:         now,
			contiguousFails:  atomic.Uint32{},
			checkErr:         errors.New("test error"),
			status:           StatusDown,
			firstFailInCycle: now,
		}
		expectedState.contiguousFails.Store(3)
		compareState(t, expectedState, c.state)
		require.Equal(t, 2, statusListenerCalled)

		err = c.Check(context.Background())
		require.EqualError(t, err, "test error", "Fourth Check() should return the correct error")

		expectedState = &State{
			lastCheckTime:    now,
			lastSuccess:      time.Time{},
			lastFail:         now,
			contiguousFails:  atomic.Uint32{},
			checkErr:         errors.New("test error"),
			status:           StatusDown,
			firstFailInCycle: now,
		}
		expectedState.contiguousFails.Store(4)
		compareState(t, expectedState, c.state)
		require.Equal(t, 2, statusListenerCalled)
	})

	t.Run("status error", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		c := NewCheck("test", func(ctx context.Context) error {
			return NewStatusError(errors.New("test error"), StatusDegraded)
		})

		err := c.Check(context.Background())
		require.EqualError(t, err, "test error")

		expectedState := &State{
			lastCheckTime:    now,
			lastSuccess:      time.Time{},
			lastFail:         now,
			contiguousFails:  atomic.Uint32{},
			checkErr:         NewStatusError(errors.New("test error"), StatusDegraded),
			status:           StatusDegraded,
			firstFailInCycle: now,
		}
		expectedState.contiguousFails.Store(1)
		compareState(t, expectedState, c.state)
	})

	t.Run("no timeout", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		statusListenerCalled := false
		c := NewCheck("test", func(ctx context.Context) error {
			return nil
		},
			WithCheckOnStatusChange(func(ctx context.Context, name string, state *State) {
				statusListenerCalled = true
				require.Equal(t, "test", name)
				require.Equal(t, StatusUp, state.status)
			}),
			WithNoCheckTimeout(),
		)

		err := c.Check(context.Background())
		require.NoError(t, err)

		expectedState := &State{
			lastCheckTime:   now,
			lastSuccess:     now,
			lastFail:        time.Time{},
			contiguousFails: atomic.Uint32{},
			checkErr:        nil,
			status:          StatusUp,
		}
		compareState(t, expectedState, c.state)
		require.True(t, statusListenerCalled)
	})

	t.Run("nil context", func(t *testing.T) {
		t.Parallel()

		c := NewCheck("test", func(ctx context.Context) error {
			return nil
		})

		err := c.Check(nil) // nolint:staticcheck // This is testing that the function works with a nil context
		require.EqualError(t, err, "context cannot be nil")
	})

	t.Run("error grace period", func(t *testing.T) {
		t.Parallel()

		statusChanges := make([]Status, 0)
		c := NewCheck("test",
			func(ctx context.Context) error {
				return errors.New("test error")
			},
			WithCheckErrorGracePeriod(2*time.Second),
			WithCheckOnStatusChange(func(ctx context.Context, name string, state *State) {
				statusChanges = append(statusChanges, state.status)
			}),
		)

		err := c.Check(context.Background())
		require.Error(t, err)
		require.Equal(t, StatusUp, c.state.status, "should be UP within grace period")
		require.WithinDuration(t, time.Now(), c.state.lastFail, 1*time.Second, "lastFail should be set to current time")

		err = c.Check(context.Background())
		require.Error(t, err)
		require.Equal(t, StatusUp, c.state.status, "should still be UP within grace period")
		require.WithinDuration(t, time.Now(), c.state.lastFail, 1*time.Second, "lastFail should be set to current time")

		time.Sleep(3 * time.Second) // Sleep for 3 seconds to exceed the grace period

		err = c.Check(context.Background())
		require.Error(t, err)
		require.Equal(t, StatusDown, c.state.status, "should be DOWN after grace period")
		require.WithinDuration(t, time.Now(), c.state.lastFail, 1*time.Second, "lastFail should be set to current time")

		// Verify status changes
		require.Equal(t, []Status{StatusUp, StatusDown}, statusChanges)

		// Verify recovery
		c.check = func(ctx context.Context) error { return nil }
		err = c.Check(context.Background())
		require.NoError(t, err)
		require.Equal(t, StatusUp, c.state.status, "should recover to UP")
		require.WithinDuration(t, time.Now(), c.state.lastFail, 1*time.Second, "lastFail should remain at last failure time")
		require.Equal(t, []Status{StatusUp, StatusDown, StatusUp}, statusChanges)
	})

	t.Run("error grace period with max contiguous fails", func(t *testing.T) {
		t.Parallel()

		statusChanges := make([]Status, 0)
		c := NewCheck("test",
			func(ctx context.Context) error {
				return errors.New("test error")
			},
			WithCheckErrorGracePeriod(2*time.Second),
			WithCheckMaxFailures(3),
			WithCheckOnStatusChange(func(ctx context.Context, name string, state *State) {
				statusChanges = append(statusChanges, state.status)
			}),
		)

		err := c.Check(context.Background())
		require.Error(t, err)
		require.Equal(t, StatusUp, c.state.status, "should be UP within grace period and max failures")

		err = c.Check(context.Background())
		require.Error(t, err)
		require.Equal(t, StatusUp, c.state.status, "should be UP within grace period and max failures")

		err = c.Check(context.Background())
		require.Error(t, err)
		require.Equal(t, StatusUp, c.state.status, "should be UP within grace period despite max failures")

		time.Sleep(3 * time.Second) // Sleep for 3 seconds to exceed the grace period

		err = c.Check(context.Background())
		require.Error(t, err)
		require.Equal(t, StatusDown, c.state.status, "should be DOWN after grace period")

		// Verify status changes
		require.Equal(t, []Status{StatusUp, StatusDown}, statusChanges)
	})
}
