package health

import (
	"context"
	"time"
)

type CheckOption = func(*Check)

// CheckWithTimeout sets the timeout for the check.
func CheckWithTimeout(timeout time.Duration) CheckOption {
	return func(c *Check) {
		c.timeout = timeout
	}
}

// CheckWithMaxTimeInError sets the maximum time the check can be in an error state before it is marked as down.
// The check will be marked as down if the time in error exceeds this value.
// This is useful for checks that may take a long time to recover from an error.
// For example, if a check takes 5 minutes to recover from an error, you can set this value to 10 minutes.
// This will allow the check to be in an error state for up to 10 minutes before it is marked as down.
func CheckWithMaxTimeInError(maxTimeInError time.Duration) CheckOption {
	return func(c *Check) {
		c.maxTimeInError = maxTimeInError
	}
}

// CheckWithMaxContiguousFails sets the maximum number of contiguous fails.
//
// The check will be marked as up until the number of contiguous fails equals or exceeds this value.
func CheckWithMaxContiguousFails(maxContiguousFails uint) CheckOption {
	return func(c *Check) {
		c.maxContiguousFails = maxContiguousFails
	}
}

// CheckWithStatusListener sets the status listener for the check.
func CheckWithStatusListener(statusListener func(ctx context.Context, name string, state State)) CheckOption {
	return func(c *Check) {
		c.statusListener = statusListener
	}
}
