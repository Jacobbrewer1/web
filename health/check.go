package health

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jacobbrewer1/web/logging"
)

type CheckFunc = func(ctx context.Context) error

type Check struct {
	// name is the name of the check.
	name string

	// check is the function that performs the check.
	check CheckFunc

	// timeout is the timeout for the check.
	timeout time.Duration

	// maxTimeInError is the maximum time the check can be in an error state.
	maxTimeInError time.Duration

	// maxContiguousFails is the maximum number of contiguous fails.
	maxContiguousFails uint

	// statusListener is the function that will be called when the status changes.
	statusListener func(ctx context.Context, name string, state State)

	// state is the state of the check.
	state *State
}

// NewCheck creates a new Check
func NewCheck(name string, checkerFunc CheckFunc, options ...CheckOption) *Check {
	c := &Check{
		name:    name,
		check:   checkerFunc,
		timeout: 5 * time.Second,
		state:   &State{status: StatusUnknown},
	}

	for _, option := range options {
		option(c)
	}

	return c
}

// NewCheckWithStandardListener creates a new Check with a standard status listener
func NewCheckWithStandardListener(l *slog.Logger, name string, checkerFunc CheckFunc, options ...CheckOption) *Check {
	l = logging.LoggerWithComponent(l, "health-checker")
	c := NewCheck(name, checkerFunc, options...)
	c.statusListener = func(ctx context.Context, name string, state State) {
		l.Info("health check status changed",
			slog.String(logging.KeyName, name),
			slog.String(logging.KeyState, state.Status().String()),
		)
	}
	return c
}

func (c *Check) String() string {
	return c.name
}

func (c *Check) Check(ctx context.Context) error {
	now := Timestamp()
	c.state.lastCheckTime = now

	var (
		checkCtx context.Context
		cancel   context.CancelFunc
	)
	if c.timeout > 0 {
		checkCtx, cancel = context.WithTimeout(ctx, c.timeout)
	} else {
		checkCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	newStatus := StatusUnknown
	defer func() {
		if c.statusListener != nil && c.state.status != newStatus {
			c.state.status = newStatus // Set the new status before calling the listener
			c.statusListener(checkCtx, c.name, *c.state)
		} else {
			c.state.status = newStatus
		}
	}()

	if err := c.check(checkCtx); err != nil {
		newStatus = StatusDown
		c.state.contiguousFails++
		if c.maxContiguousFails > 0 && c.state.contiguousFails < c.maxContiguousFails {
			newStatus = StatusUp
		}

		statusErr := new(StatusError)
		if errors.As(err, &statusErr) {
			newStatus = statusErr.Status
		}

		c.state.checkErr = err
		c.state.lastFail = now

		c.state.checkErr = err
		return err
	} else {
		newStatus = StatusUp
		c.state.lastSuccess = now
		c.state.contiguousFails = 0
		c.state.checkErr = nil
	}

	return nil
}
