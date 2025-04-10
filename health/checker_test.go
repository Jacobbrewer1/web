package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewChecker(t *testing.T) {
	t.Parallel()

	gotCheck := NewCheck("test", func(_ context.Context) error {
		return nil
	})

	got := NewChecker(WithCheckerCheck(gotCheck))

	c, ok := got.(*checker)
	require.True(t, ok, "NewChecker() should return a *checker")
	require.NotNil(t, c, "NewChecker() should return a non-nil Checker")
	require.Equal(t, http.StatusOK, c.httpStatusCodeUp, "NewChecker() should set the default HTTP status code up")
	require.Equal(t, http.StatusServiceUnavailable, c.httpStatusCodeDown, "NewChecker() should set the default HTTP status code down")
	require.NotNil(t, c.baseCtx, "NewChecker() should set the base context")
}

func TestNewCheckerHandler_Single(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	gotCheck := NewCheck("test_check", func(_ context.Context) error {
		return nil
	})

	got := NewChecker(WithCheckerCheck(gotCheck))

	handler := got.Handler()
	require.NotNil(t, handler, "Handler() should return a non-nil http.HandlerFunc")

	// Call the handler and check the response
	req := httptest.NewRequest("GET", "/", http.NoBody)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "Handler() should return a 200 OK status code")
	require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"), "Handler() should return a JSON content type")

	expectedJSON := `{"status":"up","details":{"test_check":{"status":"up","timestamp":"` + now.Format(time.RFC3339Nano) + `"}}}`
	require.JSONEq(t, expectedJSON, rec.Body.String(), "Handler() should return the expected JSON response")
}

func TestNewCheckerHandler_Single_StatusError(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	gotCheck := NewCheck("test_check", func(_ context.Context) error {
		return NewStatusError(errors.New("test error"), StatusDegraded)
	})

	got := NewChecker(WithCheckerCheck(gotCheck))

	handler := got.Handler()
	require.NotNil(t, handler, "Handler() should return a non-nil http.HandlerFunc")

	// Call the handler and check the response
	req := httptest.NewRequest("GET", "/", http.NoBody)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"), "Handler() should return a JSON content type")

	expectedJSON := `{"status":"degraded","details":{"test_check":{"status":"degraded","error":"test error","timestamp":"` + now.Format(time.RFC3339Nano) + `"}}}`
	require.JSONEq(t, expectedJSON, rec.Body.String(), "Handler() should return the expected JSON response")
}

func TestNewCheckerHandler_Single_StatusError_InvalidStatus(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	gotCheck := NewCheck("test_check", func(_ context.Context) error {
		return NewStatusError(errors.New("test error"), 123)
	})

	got := NewChecker(WithCheckerCheck(gotCheck))

	handler := got.Handler()
	require.NotNil(t, handler, "Handler() should return a non-nil http.HandlerFunc")

	// Call the handler and check the response
	req := httptest.NewRequest("GET", "/", http.NoBody)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code, "Handler() should return a 200 OK status code")
	require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"), "Handler() should return a JSON content type")

	expectedJSON := `{"status":"unknown","details":{"test_check":{"status":"unknown","error":"test error","timestamp":"` + now.Format(time.RFC3339Nano) + `"}}}`
	require.JSONEq(t, expectedJSON, rec.Body.String(), "Handler() should return the expected JSON response")
}

func TestNewCheckerHandler_Multiple(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	gotCheck := NewCheck("test_check", func(_ context.Context) error {
		return nil
	})

	secondCheck := NewCheck("second_check", func(_ context.Context) error {
		return nil
	})

	got := NewChecker(WithCheckerChecks([]*Check{gotCheck, secondCheck}...))

	handler := got.Handler()
	require.NotNil(t, handler, "Handler() should return a non-nil http.HandlerFunc")

	// Call the handler and check the response
	req := httptest.NewRequest("GET", "/", http.NoBody)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "Handler() should return a 200 OK status code")
	require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"), "Handler() should return a JSON content type")

	expectedJSON := `{"status":"up","details":{"test_check":{"status":"up","timestamp":"` + now.Format(time.RFC3339Nano) + `"},"second_check":{"status":"up","timestamp":"` + now.Format(time.RFC3339Nano) + `"}}}`
	require.JSONEq(t, expectedJSON, rec.Body.String(), "Handler() should return the expected JSON response")
}

func TestNewCheckerHandler_Single_Error(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	gotCheck := NewCheck("test_check", func(_ context.Context) error {
		return errors.New("test error")
	})

	got := NewChecker(WithCheckerCheck(gotCheck))

	handler := got.Handler()
	require.NotNil(t, handler, "Handler() should return a non-nil http.HandlerFunc")

	// Call the handler and check the response
	req := httptest.NewRequest("GET", "/", http.NoBody)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code, "Handler() should return a 200 OK status code")
	require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"), "Handler() should return a JSON content type")

	expectedJSON := `{"status":"down","details":{"test_check":{"status":"down","error":"test error","timestamp":"` + now.Format(time.RFC3339Nano) + `"}}}`
	require.JSONEq(t, expectedJSON, rec.Body.String(), "Handler() should return the expected JSON response")
}

func TestNewCheckerHandler_NoParentContext(t *testing.T) {
	now := time.Now().UTC()
	timestamp = func() time.Time { return now }

	gotCheck := NewCheck("test_check", func(_ context.Context) error {
		return nil
	})

	got := NewChecker(WithCheckerCheck(gotCheck))

	handler := got.Handler()
	require.NotNil(t, handler, "Handler() should return a non-nil http.HandlerFunc")

	res := got.Check(nil) // nolint:staticcheck // This is testing that the function works with a nil context
	require.Equal(t, StatusUp, res.Status, "Check() should return a status of up when no parent context is provided")

	require.NotNil(t, res)

	require.NotNil(t, res.Details)
	require.Len(t, res.Details, 1, "Check() should return one detail")
	require.Equal(t, StatusUp, res.Details["test_check"].Status, "Check() should return a status of up for the test_check detail")
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

			got := NewChecker()
			c, ok := got.(*checker)
			require.True(t, ok, "NewChecker() should return a *checker")
			require.NotNil(t, c, "NewChecker() should return a non-nil Checker")

			require.Equal(t, tt.expectedStatus, c.httpCodeFromStatus(tt.status), "httpCodeFromStatus() should return the expected status code")
		})
	}
}
