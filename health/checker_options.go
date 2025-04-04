package health

import (
	"context"
)

type CheckerOption func(*checker)

// CheckerWithBaseContext sets the base context for the checker.
func CheckerWithBaseContext(baseCtx context.Context) CheckerOption {
	return func(c *checker) {
		ctx, cancel := context.WithCancel(baseCtx)
		c.baseCtx = ctx
		c.cancel = cancel
	}
}

// CheckerWithCheck sets the checks for the checker.
func CheckerWithCheck(check *Check) CheckerOption {
	return func(c *checker) {
		if check == nil {
			return
		}
		if c.checks == nil {
			c.checks = make([]*Check, 0)
		}
		c.checks = append(c.checks, check)
	}
}

// CheckerWithHTTPStatusCodeUp sets the HTTP status code for the checker when the system is up.
func CheckerWithHTTPStatusCodeUp(code int) CheckerOption {
	return func(c *checker) {
		c.httpStatusCodeUp = code
	}
}

// CheckerWithHTTPStatusCodeDown sets the HTTP status code for the checker when the system is down.
func CheckerWithHTTPStatusCodeDown(code int) CheckerOption {
	return func(c *checker) {
		c.httpStatusCodeDown = code
	}
}
