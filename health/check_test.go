package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCheck_Check_Golden(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	statusListenerCalled := false
	c := NewCheck("test", func(ctx context.Context) error {
		return nil
	},
		WithCheckOnStatusChange(func(ctx context.Context, name string, state State) {
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
		contiguousFails: 0,
		checkErr:        nil,
		status:          StatusUp,
	}
	require.Equal(t, expectedState, c.state)
	require.True(t, statusListenerCalled)
}

func TestCheck_Check_Golden_FailCheck(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	statusListenerCalled := false
	c := NewCheck("test", func(ctx context.Context) error {
		return errors.New("test error")
	},
		WithCheckOnStatusChange(func(ctx context.Context, name string, state State) {
			statusListenerCalled = true
			require.Equal(t, "test", name)
			require.Equal(t, StatusDown, state.status)
		}),
	)

	err := c.Check(context.Background())
	require.EqualError(t, err, "test error")

	expectedState := &State{
		lastCheckTime:   now,
		lastSuccess:     time.Time{},
		lastFail:        now,
		contiguousFails: 1,
		checkErr:        errors.New("test error"),
		status:          StatusDown,
	}
	require.Equal(t, expectedState, c.state)
	require.True(t, statusListenerCalled)
}

func TestCheck_Check_Golden_SuccessToFailToSuccess(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	statusListenerCalled := 0
	callNumber := 0
	c := NewCheck("test", func(ctx context.Context) error {
		callNumber++
		if callNumber%2 == 0 {
			return errors.New("test error")
		}
		return nil
	},
		WithCheckOnStatusChange(func(ctx context.Context, name string, state State) {
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
		lastCheckTime:   now,
		lastSuccess:     now,
		lastFail:        time.Time{},
		contiguousFails: 0,
		checkErr:        nil,
		status:          StatusUp,
	}
	require.Equal(t, expectedState, c.state, "First Check() should update the state correctly")

	err = c.Check(context.Background())
	require.EqualError(t, err, "test error", "Second Check() should return the correct error")

	expectedState = &State{
		lastCheckTime:   now,
		lastSuccess:     now,
		lastFail:        now,
		contiguousFails: 1,
		checkErr:        errors.New("test error"),
		status:          StatusDown,
	}
	require.Equal(t, expectedState, c.state, "Second Check() should update the state correctly")

	err = c.Check(context.Background())
	require.NoError(t, err, "Third Check() should not return an error")

	expectedState = &State{
		lastCheckTime:   now,
		lastSuccess:     now,
		lastFail:        now,
		contiguousFails: 0,
		checkErr:        nil,
		status:          StatusUp,
	}
	require.Equal(t, expectedState, c.state)
	require.Equal(t, 3, statusListenerCalled)
}

func TestCheck_Check_Golden_MaxContiguousFails(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	statusListenerCalled := 0
	c := NewCheck("test", func(ctx context.Context) error {
		return errors.New("test error")
	},
		WithCheckMaxFailures(3),
		WithCheckOnStatusChange(func(ctx context.Context, name string, state State) {
			statusListenerCalled++
			require.Equal(t, "test", name)

			if state.contiguousFails < 3 {
				require.Equal(t, StatusUp, state.status)
			} else {
				require.Equal(t, StatusDown, state.status)
			}
		}),
	)

	err := c.Check(context.Background())
	require.EqualError(t, err, "test error", "First Check() should return the correct error")

	expectedState := &State{
		lastCheckTime:   now,
		lastSuccess:     time.Time{},
		lastFail:        now,
		contiguousFails: 1,
		checkErr:        errors.New("test error"),
		status:          StatusUp,
	}
	require.Equal(t, expectedState, c.state, "First Check() should update the state correctly")
	require.Equal(t, 1, statusListenerCalled)

	err = c.Check(context.Background())
	require.EqualError(t, err, "test error", "Second Check() should return the correct error")

	expectedState = &State{
		lastCheckTime:   now,
		lastSuccess:     time.Time{},
		lastFail:        now,
		contiguousFails: 2,
		checkErr:        errors.New("test error"),
		status:          StatusUp,
	}
	require.Equal(t, expectedState, c.state, "Second Check() should update the state correctly")
	require.Equal(t, 1, statusListenerCalled)

	err = c.Check(context.Background())
	require.EqualError(t, err, "test error", "Third Check() should return the correct error")

	expectedState = &State{
		lastCheckTime:   now,
		lastSuccess:     time.Time{},
		lastFail:        now,
		contiguousFails: 3,
		checkErr:        errors.New("test error"),
		status:          StatusDown,
	}
	require.Equal(t, expectedState, c.state, "Third Check() should update the state correctly")
	require.Equal(t, 2, statusListenerCalled)

	err = c.Check(context.Background())
	require.EqualError(t, err, "test error", "Fourth Check() should return the correct error")

	expectedState = &State{
		lastCheckTime:   now,
		lastSuccess:     time.Time{},
		lastFail:        now,
		contiguousFails: 4,
		checkErr:        errors.New("test error"),
		status:          StatusDown,
	}
	require.Equal(t, expectedState, c.state, "Fourth Check() should update the state correctly")
	require.Equal(t, 2, statusListenerCalled)
}

func TestCheck_StatusError(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	c := NewCheck("test", func(ctx context.Context) error {
		return NewStatusError(errors.New("test error"), StatusDegraded)
	})

	err := c.Check(context.Background())
	require.EqualError(t, err, "test error")

	expectedState := &State{
		lastCheckTime:   now,
		lastSuccess:     time.Time{},
		lastFail:        now,
		contiguousFails: 1,
		checkErr:        NewStatusError(errors.New("test error"), StatusDegraded),
		status:          StatusDegraded,
	}
	require.Equal(t, expectedState, c.state)
}

func TestCheck_NoTimeout(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	statusListenerCalled := false
	c := NewCheck("test", func(ctx context.Context) error {
		return nil
	},
		WithCheckOnStatusChange(func(ctx context.Context, name string, state State) {
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
		contiguousFails: 0,
		checkErr:        nil,
		status:          StatusUp,
	}
	require.Equal(t, expectedState, c.state)
	require.True(t, statusListenerCalled)
}

func TestCheck_NoParentContext(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	statusListenerCalled := false
	c := NewCheck("test", func(ctx context.Context) error {
		return nil
	},
		WithCheckOnStatusChange(func(ctx context.Context, name string, state State) {
			statusListenerCalled = true
			require.Equal(t, "test", name)
			require.Equal(t, StatusUp, state.status)
		}),
	)

	err := c.Check(nil) // nolint:staticcheck // This is testing that the function works with a nil context
	require.NoError(t, err)

	expectedState := &State{
		lastCheckTime:   now,
		lastSuccess:     now,
		lastFail:        time.Time{},
		contiguousFails: 0,
		checkErr:        nil,
		status:          StatusUp,
	}
	require.Equal(t, expectedState, c.state)
	require.True(t, statusListenerCalled)
}
