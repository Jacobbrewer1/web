package health

import (
    "context"
    "errors"
    "sync/atomic"
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
}

func TestCheck_Check_Golden_FailCheck(t *testing.T) {
    now := time.Now().UTC()
    timestamp = func() time.Time { return now }

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
    require.Equal(t, expectedState, c.state, "First Check() should update the state correctly")

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
    require.Equal(t, expectedState, c.state, "Second Check() should update the state correctly")

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
}

func TestCheck_Check_Golden_MaxContiguousFails(t *testing.T) {
    now := time.Now().UTC()
    timestamp = func() time.Time { return now }

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
    require.Equal(t, expectedState, c.state, "First Check() should update the state correctly")
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
    require.Equal(t, expectedState, c.state, "Second Check() should update the state correctly")
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
    require.Equal(t, expectedState, c.state, "Third Check() should update the state correctly")
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
}

func TestCheck_NoTimeout(t *testing.T) {
    now := time.Now().UTC()
    timestamp = func() time.Time { return now }

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
}

func TestCheck_NoParentContext(t *testing.T) {
    now := time.Now().UTC()
    timestamp = func() time.Time { return now }

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

    err := c.Check(nil) // nolint:staticcheck // This is testing that the function works with a nil context
    require.NoError(t, err)

    expectedState := &State{
        lastCheckTime:   now,
        lastSuccess:     now,
        lastFail:        time.Time{},
        contiguousFails: atomic.Uint32{},
        checkErr:        nil,
        status:          StatusUp,
    }
    expectedState.contiguousFails.Store(0)
    compareState(t, expectedState, c.state)
    require.True(t, statusListenerCalled)
}

func TestCheck_ErrorGracePeriod(t *testing.T) {
    var currentTime time.Time
    timestamp = func() time.Time { return currentTime }

    statusChanges := make([]Status, 0)
    c := NewCheck("test",
        func(ctx context.Context) error {
            return errors.New("test error")
        },
        WithCheckErrorGracePeriod(5*time.Second),
        WithCheckOnStatusChange(func(ctx context.Context, name string, state *State) {
            statusChanges = append(statusChanges, state.status)
        }),
    )

    // Start at t=0
    currentTime = time.Unix(0, 0)
    err := c.Check(context.Background())
    require.Error(t, err)
    require.Equal(t, StatusUp, c.state.status, "should be UP within grace period")
    require.Equal(t, currentTime, c.state.lastFail)

    // Check at t=3s (within grace period)
    currentTime = time.Unix(3, 0)
    err = c.Check(context.Background())
    require.Error(t, err)
    require.Equal(t, StatusUp, c.state.status, "should still be UP within grace period")
    require.Equal(t, currentTime, c.state.lastFail)

    // Check at t=6s (exceeds grace period)
    currentTime = time.Unix(7, 0)
    err = c.Check(context.Background())
    require.Error(t, err)
    require.Equal(t, StatusDown, c.state.status, "should be DOWN after grace period")
    require.Equal(t, currentTime, c.state.lastFail)

    // Verify status changes
    require.Equal(t, []Status{StatusUp, StatusDown}, statusChanges)

    // Verify recovery
    currentTime = time.Unix(9, 0)
    c.check = func(ctx context.Context) error { return nil }
    err = c.Check(context.Background())
    require.NoError(t, err)
    require.Equal(t, StatusUp, c.state.status, "should recover to UP")
    require.Equal(t, time.Unix(7, 0), c.state.lastFail, "lastFail should remain at last failure time")
    require.Equal(t, []Status{StatusUp, StatusDown, StatusUp}, statusChanges)
}

func TestCheck_ErrorGracePeriod_WithMaxContiguousFails(t *testing.T) {
    var currentTime time.Time
    timestamp = func() time.Time { return currentTime }

    statusChanges := make([]Status, 0)
    c := NewCheck("test",
        func(ctx context.Context) error {
            return errors.New("test error")
        },
        WithCheckErrorGracePeriod(5*time.Second),
        WithCheckMaxFailures(3),
        WithCheckOnStatusChange(func(ctx context.Context, name string, state *State) {
            statusChanges = append(statusChanges, state.status)
        }),
    )

    // Start at t=0
    currentTime = time.Unix(0, 0)
    err := c.Check(context.Background())
    require.Error(t, err)
    require.Equal(t, StatusUp, c.state.status, "should be UP within grace period and max failures")

    // Check at t=3s (failure 2)
    currentTime = time.Unix(3, 0)
    err = c.Check(context.Background())
    require.Error(t, err)
    require.Equal(t, StatusUp, c.state.status, "should be UP within grace period and max failures")

    // Check at t=4s (failure 3)
    currentTime = time.Unix(4, 0)
    err = c.Check(context.Background())
    require.Error(t, err)
    require.Equal(t, StatusUp, c.state.status, "should be UP within grace period despite max failures")

    // Check at t=6s (exceeds grace period)
    currentTime = time.Unix(6, 0)
    err = c.Check(context.Background())
    require.Error(t, err)
    require.Equal(t, StatusDown, c.state.status, "should be DOWN after grace period")

    // Verify status changes
    require.Equal(t, []Status{StatusUp, StatusDown}, statusChanges)
}
