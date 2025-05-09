package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func ptr[T any](t *testing.T, v T) *T {
	t.Helper()
	return &v
}

func TestNewChecker(t *testing.T) {
	t.Parallel()

	gotCheck := NewCheck("test", func(_ context.Context) error {
		return nil
	})

	c, err := NewChecker(WithCheckerCheck(gotCheck))
	require.NoError(t, err)

	require.NotNil(t, c)
	require.Equal(t, http.StatusOK, c.httpStatusCodeUp)
	require.Equal(t, http.StatusServiceUnavailable, c.httpStatusCodeDown)
}

func TestNewChecker_Handler(t *testing.T) {
	t.Run("single success", func(t *testing.T) {
		t.Parallel()

		gotCheck := NewCheck("test_check", func(_ context.Context) error {
			return nil
		})

		got, err := NewChecker(WithCheckerCheck(gotCheck))
		require.NoError(t, err)

		handler := got.Handler()
		require.NotNil(t, handler)

		// Call the handler and check the response
		req := httptest.NewRequest("GET", "/", http.NoBody)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))

		result := new(Result)
		err = json.NewDecoder(rec.Body).Decode(result)
		require.NoError(t, err)

		expectedResult := &Result{
			Status: StatusUp,
			Details: map[string]*Result{
				"test_check": {
					Status:    StatusUp,
					Timestamp: ptr(t, time.Now().UTC()),
					Details:   nil,
					Error:     "",
				},
			},
		}
		compareResult(t, expectedResult, result)
	})

	t.Run("single error", func(t *testing.T) {
		t.Parallel()

		gotCheck := NewCheck("test_check", func(_ context.Context) error {
			return errors.New("test")
		})

		got, err := NewChecker(WithCheckerCheck(gotCheck))
		require.NoError(t, err)

		handler := got.Handler()
		require.NotNil(t, handler)

		// Call the handler and check the response
		req := httptest.NewRequest("GET", "/", http.NoBody)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusServiceUnavailable, rec.Code)
		require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))

		result := new(Result)
		err = json.NewDecoder(rec.Body).Decode(result)
		require.NoError(t, err)

		expectedResult := &Result{
			Status: StatusDown,
			Details: map[string]*Result{
				"test_check": {
					Status:    StatusDown,
					Timestamp: ptr(t, time.Now().UTC()),
					Details:   nil,
					Error:     "test",
				},
			},
		}
		compareResult(t, expectedResult, result)
	})

	t.Run("single status error", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		gotCheck := NewCheck("test_check", func(_ context.Context) error {
			return NewStatusError(errors.New("test error"), StatusDegraded)
		})

		got, err := NewChecker(WithCheckerCheck(gotCheck))
		require.NoError(t, err)

		handler := got.Handler()
		require.NotNil(t, handler)

		// Call the handler and check the response
		req := httptest.NewRequest("GET", "/", http.NoBody)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusServiceUnavailable, rec.Code)
		require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))

		result := new(Result)
		err = json.NewDecoder(rec.Body).Decode(result)
		require.NoError(t, err)

		expectedResult := &Result{
			Status: StatusDegraded,
			Details: map[string]*Result{
				"test_check": {
					Status:    StatusDegraded,
					Timestamp: ptr(t, now),
					Details:   nil,
					Error:     "test error",
				},
			},
		}
		compareResult(t, expectedResult, result)
	})

	t.Run("single status error invalid status", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		gotCheck := NewCheck("test_check", func(_ context.Context) error {
			return NewStatusError(errors.New("test error"), 123)
		})

		got, err := NewChecker(WithCheckerCheck(gotCheck))
		require.NoError(t, err)

		handler := got.Handler()
		require.NotNil(t, handler)

		// Call the handler and check the response
		req := httptest.NewRequest("GET", "/", http.NoBody)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusServiceUnavailable, rec.Code)
		require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))

		result := new(Result)
		err = json.NewDecoder(rec.Body).Decode(result)
		require.NoError(t, err)

		expectedResult := &Result{
			Status: StatusUnknown,
			Details: map[string]*Result{
				"test_check": {
					Status:    StatusUnknown,
					Timestamp: ptr(t, now),
					Details:   nil,
					Error:     "test error",
				},
			},
		}
		compareResult(t, expectedResult, result)
	})

	t.Run("multiple checks success", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		gotCheck := NewCheck("test_check", func(_ context.Context) error {
			return nil
		})

		secondCheck := NewCheck("second_check", func(_ context.Context) error {
			return nil
		})

		got, err := NewChecker(WithCheckerChecks([]*Check{gotCheck, secondCheck}...))
		require.NoError(t, err)

		handler := got.Handler()
		require.NotNil(t, handler)

		// Call the handler and check the response
		req := httptest.NewRequest("GET", "/", http.NoBody)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))

		result := new(Result)
		err = json.NewDecoder(rec.Body).Decode(result)
		require.NoError(t, err)

		expectedResult := &Result{
			Status: StatusUp,
			Details: map[string]*Result{
				"test_check": {
					Status:    StatusUp,
					Timestamp: ptr(t, now),
					Details:   nil,
					Error:     "",
				},
				"second_check": {
					Status:    StatusUp,
					Timestamp: ptr(t, now),
					Details:   nil,
					Error:     "",
				},
			},
		}
		compareResult(t, expectedResult, result)
	})

	t.Run("single error", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		gotCheck := NewCheck("test_check", func(_ context.Context) error {
			return errors.New("test error")
		})

		got, err := NewChecker(WithCheckerCheck(gotCheck))
		require.NoError(t, err)

		handler := got.Handler()
		require.NotNil(t, handler)

		// Call the handler and check the response
		req := httptest.NewRequest("GET", "/", http.NoBody)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusServiceUnavailable, rec.Code)
		require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))

		result := new(Result)
		err = json.NewDecoder(rec.Body).Decode(result)
		require.NoError(t, err)

		expectedResult := &Result{
			Status: StatusDown,
			Details: map[string]*Result{
				"test_check": {
					Status:    StatusDown,
					Timestamp: ptr(t, now),
					Details:   nil,
					Error:     "test error",
				},
			},
		}
		compareResult(t, expectedResult, result)
	})

	t.Run("no parent context", func(t *testing.T) {
		t.Parallel()

		gotCheck := NewCheck("test_check", func(_ context.Context) error {
			return nil
		})

		got, err := NewChecker(WithCheckerCheck(gotCheck))
		require.NoError(t, err)

		handler := got.Handler()
		require.NotNil(t, handler)

		res := got.Check(nil) // nolint:staticcheck // This is testing that the function works with a nil context
		require.Equal(t, StatusUp, res.Status)

		require.NotNil(t, res)

		require.NotNil(t, res.Details)
		require.Len(t, res.Details, 1)
		require.Equal(t, StatusUp, res.Details["test_check"].Status)
	})
}

func TestChecker_HttpCodeFromStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		status         Status
		expectedStatus int
	}{
		{
			name:           "StatusUp",
			status:         StatusUp,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "StatusDown",
			status:         StatusDown,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "StatusDegraded",
			status:         StatusDegraded,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "StatusUnknown",
			status:         StatusUnknown,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "InvalidStatus",
			status:         123,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := NewChecker()
			require.NoError(t, err)
			require.NotNil(t, c)
			require.Equal(t, tt.expectedStatus, c.httpCodeFromStatus(tt.status))
		})
	}
}

func TestChecker_AddCheck(t *testing.T) {
	t.Parallel()

	t.Run("valid check", func(t *testing.T) {
		t.Parallel()

		c, err := NewChecker()
		require.NoError(t, err)
		require.NotNil(t, c)

		check := NewCheck("test_check", func(_ context.Context) error {
			return nil
		})

		err = c.AddCheck(check)
		require.NoError(t, err)

		// Check if the check was added
		c.checks.Range(func(key, value any) bool {
			require.Equal(t, "test_check", key)
			return false
		})
	})
}

func TestChecker_AddTest(t *testing.T) {
	t.Parallel()

	t.Run("nil check", func(t *testing.T) {
		t.Parallel()

		c, err := NewChecker()
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.AddCheck(nil)
		require.Error(t, err)
		require.Equal(t, "check is nil", err.Error())
	})

	t.Run("empty name", func(t *testing.T) {
		t.Parallel()

		c, err := NewChecker()
		require.NoError(t, err)
		require.NotNil(t, c)

		check := NewCheck("", func(_ context.Context) error {
			return nil
		})

		err = c.AddCheck(check)
		require.Error(t, err)
		require.Equal(t, "check name is empty", err.Error())
	})

	t.Run("test name already exists", func(t *testing.T) {
		t.Parallel()

		c, err := NewChecker()
		require.NoError(t, err)
		require.NotNil(t, c)

		check := NewCheck("test_check", func(_ context.Context) error {
			return nil
		})

		err = c.AddCheck(check)
		require.NoError(t, err)

		err = c.AddCheck(check)
		require.Error(t, err)
		require.Equal(t, "check already exists: test_check", err.Error())
	})
}

func TestChecker_ErrorGracePeriod(t *testing.T) {
	t.Parallel()

	t.Run("within grace period", func(t *testing.T) {
		t.Parallel()

		c, err := NewChecker(WithCheckerErrorGracePeriod(5 * time.Second))
		require.NoError(t, err)
		require.NotNil(t, c)

		check := NewCheck("test_check", func(_ context.Context) error {
			return errors.New("test error")
		})

		err = c.AddCheck(check)
		require.NoError(t, err)

		// Simulate a failure
		c.firstFailInCycle.Store(time.Now().UTC().Add(-2 * time.Second))

		res := c.Check(context.Background())
		require.Equal(t, StatusUp, res.Status)
	})

	t.Run("outside grace period", func(t *testing.T) {
		t.Parallel()

		c, err := NewChecker(WithCheckerErrorGracePeriod(5 * time.Second))
		require.NoError(t, err)
		require.NotNil(t, c)

		check := NewCheck("test_check", func(_ context.Context) error {
			return errors.New("test error")
		})

		err = c.AddCheck(check)
		require.NoError(t, err)

		// Simulate a failure
		c.firstFailInCycle.Store(time.Now().UTC().Add(-10 * time.Second))

		res := c.Check(context.Background())
		require.Equal(t, StatusDown, res.Status)
	})

	t.Run("multiple async checks", func(t *testing.T) {
		t.Parallel()

		c, err := NewChecker(WithCheckerErrorGracePeriod(5 * time.Second))
		require.NoError(t, err)
		require.NotNil(t, c)

		check1 := NewCheck("test_check_1", func(_ context.Context) error {
			return errors.New("test error")
		})

		check2 := NewCheck("test_check_2", func(_ context.Context) error {
			return errors.New("test error 2")
		})

		check3 := NewCheck("test_check_3", func(_ context.Context) error {
			return errors.New("test error 3")
		})

		err = c.AddCheck(check1)
		require.NoError(t, err)

		err = c.AddCheck(check2)
		require.NoError(t, err)

		err = c.AddCheck(check3)
		require.NoError(t, err)

		res := c.Check(context.Background())
		require.Equal(t, StatusUp, res.Status)
	})
}
