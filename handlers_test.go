package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapHandler(t *testing.T) {
	t.Parallel()

	t.Run("no middleware", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := func(w http.ResponseWriter, r *http.Request) {
			called = true
		}

		wrapped := WrapHandler(handler)
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)
		require.True(t, called, "handler should be called")
	})

	t.Run("single middleware", func(t *testing.T) {
		t.Parallel()

		handlerCalled := false
		middlewareCalled := false

		handler := func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		}

		middleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				middlewareCalled = true
				next.ServeHTTP(w, r)
			})
		}

		wrapped := WrapHandler(handler, middleware)
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)
		require.True(t, handlerCalled, "handler should be called")
		require.True(t, middlewareCalled, "middleware should be called")
	})

	t.Run("multiple middlewares execution order", func(t *testing.T) {
		t.Parallel()

		order := make([]int, 0, 3)
		handler := func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 0)
		}

		middleware1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, 1)
				next.ServeHTTP(w, r)
			})
		}

		middleware2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, 2)
				next.ServeHTTP(w, r)
			})
		}

		wrapped := WrapHandler(handler, middleware1, middleware2)
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)
		require.Equal(t, []int{2, 1, 0}, order, "middleware execution order should be reversed")
	})

	t.Run("nil middleware", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := func(w http.ResponseWriter, r *http.Request) {
			called = true
		}

		wrapped := WrapHandler(handler, nil)
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)
		require.True(t, called, "handler should be called even with nil middleware")
	})
}
