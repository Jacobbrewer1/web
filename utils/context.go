package utils

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jacobbrewer1/web/logging"
)

func GetInterruptedContext(l *slog.Logger) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		got := <-sig
		l.Info("Received signal, shutting down", slog.String(logging.KeySignal, got.String()))
		cancel()
	}()
	return ctx
}
