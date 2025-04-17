package utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// CoreContext returns a context that is cancelled when an interrupt signal is received.
var CoreContext = func() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
