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

	got := NewChecker(CheckerWithCheck(gotCheck))

	c, ok := got.(*checker)
	require.True(t, ok, "NewChecker() should return a *checker")
	require.NotNil(t, c, "NewChecker() should return a non-nil Checker")
	require.Equal(t, http.StatusOK, c.httpStatusCodeUp, "NewChecker() should set the default HTTP status code up")
	require.Equal(t, http.StatusServiceUnavailable, c.httpStatusCodeDown, "NewChecker() should set the default HTTP status code down")
	require.NotNil(t, c.baseCtx, "NewChecker() should set the base context")
}

func TestNewCheckerHandler_Single(t *testing.T) {
	now := time.Now().UTC()
	Timestamp = func() time.Time { return now }

	gotCheck := NewCheck("test_check", func(_ context.Context) error {
		return nil
	})

	got := NewChecker(CheckerWithCheck(gotCheck))

	handler := got.Handler()
	require.NotNil(t, handler, "Handler() should return a non-nil http.HandlerFunc")

	// Call the handler and check the response
	req := httptest.NewRequest("GET", "/", http.NoBody)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "Handler() should return a 200 OK status code")
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"), "Handler() should return a JSON content type")

	expectedJSON := `{"status":"up","details":{"test_check":{"status":"up","timestamp":"` + now.Format(time.RFC3339Nano) + `"}}}`
	require.JSONEq(t, expectedJSON, rec.Body.String(), "Handler() should return the expected JSON response")
}

func TestNewCheckerHandler_Multiple(t *testing.T) {
	now := time.Now().UTC()
	Timestamp = func() time.Time { return now }

	gotCheck := NewCheck("test_check", func(_ context.Context) error {
		return nil
	})

	secondCheck := NewCheck("second_check", func(_ context.Context) error {
		return nil
	})

	got := NewChecker(CheckerWithCheck(gotCheck), CheckerWithCheck(secondCheck))

	handler := got.Handler()
	require.NotNil(t, handler, "Handler() should return a non-nil http.HandlerFunc")

	// Call the handler and check the response
	req := httptest.NewRequest("GET", "/", http.NoBody)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "Handler() should return a 200 OK status code")
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"), "Handler() should return a JSON content type")

	expectedJSON := `{"status":"up","details":{"test_check":{"status":"up","timestamp":"` + now.Format(time.RFC3339Nano) + `"},"second_check":{"status":"up","timestamp":"` + now.Format(time.RFC3339Nano) + `"}}}`
	require.JSONEq(t, expectedJSON, rec.Body.String(), "Handler() should return the expected JSON response")
}

func TestNewCheckerHandler_Single_Error(t *testing.T) {
	now := time.Now().UTC()
	Timestamp = func() time.Time { return now }

	gotCheck := NewCheck("test_check", func(_ context.Context) error {
		return errors.New("test error")
	})

	got := NewChecker(CheckerWithCheck(gotCheck))

	handler := got.Handler()
	require.NotNil(t, handler, "Handler() should return a non-nil http.HandlerFunc")

	// Call the handler and check the response
	req := httptest.NewRequest("GET", "/", http.NoBody)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code, "Handler() should return a 200 OK status code")
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"), "Handler() should return a JSON content type")

	expectedJSON := `{"status":"down","details":{"test_check":{"status":"down","error":"test error","timestamp":"` + now.Format(time.RFC3339Nano) + `"}}}`
	require.JSONEq(t, expectedJSON, rec.Body.String(), "Handler() should return the expected JSON response")
}
