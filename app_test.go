package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/jacobbrewer1/web/logging"
	"github.com/stretchr/testify/require"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	app, err := NewApp(logging.NewLoggerWithWriter(io.Discard))
	require.NoError(t, err)
	return app
}

func TestNewApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		logger  *slog.Logger
		wantErr bool
	}{
		{
			name:    "nil logger",
			logger:  nil,
			wantErr: true,
		},
		{
			name:    "valid logger",
			logger:  logging.NewLoggerWithWriter(io.Discard),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app, err := NewApp(tt.logger)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, app)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, app)
			require.NotNil(t, app.baseCtx)
			require.NotNil(t, app.baseCtxCancel)
			require.NotNil(t, app.shutdownWg)
			require.True(t, app.metricsEnabled)
		})
	}
}

func TestApp_Shutdown(t *testing.T) {
	t.Run("single shutdown", func(t *testing.T) {
		t.Parallel()

		app := newTestApp(t)

		server := &http.Server{
			Addr: ":0",
		}
		err := app.StartServer("test", server)
		require.NoError(t, err)

		// Give the server time to start
		time.Sleep(100 * time.Millisecond)

		// Shutdown should complete without error
		app.Shutdown()
	})

	t.Run("multiple shutdowns", func(t *testing.T) {
		t.Parallel()

		app := newTestApp(t)

		// Call shutdown multiple times
		done := make(chan struct{})
		go func() {
			app.Shutdown()
			app.Shutdown()
			app.Shutdown()
			close(done)
		}()

		select {
		case <-done:
			// Success - multiple shutdowns completed
		case <-time.After(2 * time.Second):
			t.Fatal("multiple shutdowns timed out")
		}
	})
}

func TestApp_StartServer(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	svr1 := &http.Server{
		Addr: ":8080",
	}

	svr2 := &http.Server{
		Addr: ":8081",
	}

	err := app.StartServer("test1", svr1)
	require.NoError(t, err)

	err = app.StartServer("test2", svr2)
	require.NoError(t, err)

	// Try and start the same server again
	err = app.StartServer("test1", svr1)
	require.Error(t, err)
}

func TestApp_ChildContext(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	ctx, cancel := app.ChildContext()
	defer cancel()

	require.NotNil(t, ctx)
	require.NotNil(t, cancel)

	// Verify context inheritance
	app.baseCtxCancel()
	select {
	case <-ctx.Done():
		// Success - child context was cancelled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("child context not cancelled when parent cancelled")
	}
}

func TestApp_WaitForEnd(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	callbackCalled := false
	callback := func() {
		callbackCalled = true
	}

	done := make(chan struct{})
	go func() {
		app.WaitForEnd(callback)
		close(done)
	}()

	// Cancel the context after a short delay
	time.Sleep(100 * time.Millisecond)
	app.baseCtxCancel()

	select {
	case <-done:
		require.True(t, callbackCalled, "callback should have been called")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("WaitForEnd did not complete")
	}
}

func TestApp_IsLeader(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	// When no leader election is configured
	require.True(t, app.IsLeader(), "should be leader when no election configured")
}

func TestApp_Panics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(*App)
		panicMsg string
	}{
		{
			name: "Logger",
			testFunc: func(a *App) {
				a.l = nil
				a.Logger()
			},
			panicMsg: "logger has not been registered",
		},
		{
			name: "VaultClient",
			testFunc: func(a *App) {
				a.VaultClient()
			},
			panicMsg: "vault client has not been registered",
		},
		{
			name: "Viper",
			testFunc: func(a *App) {
				a.Viper()
			},
			panicMsg: "viper instance has not been registered",
		},
		{
			name: "DBConn",
			testFunc: func(a *App) {
				a.DBConn()
			},
			panicMsg: "database connection has not been registered",
		},
		{
			name: "KubeClient",
			testFunc: func(a *App) {
				a.KubeClient()
			},
			panicMsg: "kubernetes client has not been registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := newTestApp(t)
			require.PanicsWithValue(t, tt.panicMsg, func() {
				tt.testFunc(app)
			})
		})
	}
}

func TestApp_Start(t *testing.T) {
	t.Parallel()

	t.Run("successful start", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)
		err := app.Start()
		require.NoError(t, err)
	})

	t.Run("multiple starts - options not executed", func(t *testing.T) {
		t.Parallel()

		app := newTestApp(t)

		// First start
		err := app.Start()
		require.NoError(t, err)

		// Second start with option
		optionCalled := false
		err = app.Start(func(a *App) error {
			optionCalled = true
			return nil
		})
		require.NoError(t, err)
		require.False(t, optionCalled, "option should not be called on second start")
	})

	t.Run("failing option aborts startup", func(t *testing.T) {
		t.Parallel()

		app := newTestApp(t)

		called := false
		err := app.Start(
			func(a *App) error {
				return errors.New("first error")
			},
			func(a *App) error {
				called = true
				return nil
			},
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "first error")
		require.False(t, called, "subsequent options should not be called")
	})

	t.Run("with async tasks", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)

		taskCalled := false
		app.indefiniteAsyncTasks.Store("test", AsyncTaskFunc(func(ctx context.Context) {
			taskCalled = true
			<-ctx.Done()
		}))

		err := app.Start()
		require.NoError(t, err)
		time.Sleep(50 * time.Millisecond)
		require.True(t, taskCalled, "async task should be called")
	})

	t.Run("async shutdown on error", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t)

		app.servers.Store("test", &http.Server{
			Addr: ":0",
		})

		err := app.Start(
			func(a *App) error {
				return errors.New("error during startup")
			},
		)
		require.EqualError(t, err, "failed to apply option: error during startup")

		app.WaitForEnd(app.Shutdown)
	})
}
