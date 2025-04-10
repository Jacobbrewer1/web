package health

type CheckerOption func(*Checker)

// WithCheckerCheck adds a single check to the checker.
func WithCheckerCheck(check *Check) CheckerOption {
	return func(c *Checker) {
		_ = c.AddCheck(check)
	}
}

// WithCheckerChecks adds multiple checks to the checker.
func WithCheckerChecks(checks ...*Check) CheckerOption {
	return func(c *Checker) {
		for _, check := range checks {
			WithCheckerCheck(check)(c)
		}
	}
}

// WithCheckerHTTPCodeUp sets the HTTP status code when the system is up.
func WithCheckerHTTPCodeUp(code int) CheckerOption {
	return func(c *Checker) {
		c.httpStatusCodeUp = code
	}
}

// WithCheckerHTTPCodeDown sets the HTTP status code when the system is down.
func WithCheckerHTTPCodeDown(code int) CheckerOption {
	return func(c *Checker) {
		c.httpStatusCodeDown = code
	}
}
