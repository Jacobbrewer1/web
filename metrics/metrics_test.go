package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestInstrumentDuration(t *testing.T) {
	t.Parallel()

	metric := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "test_duration_seconds",
			Help: "Test duration in seconds",
		},
		[]string{"code", "method"},
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := InstrumentDuration(metric)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	metricCh := make(chan prometheus.Metric, 1)
	metric.Collect(metricCh)
	require.NotNil(t, <-metricCh, "metric should be collected")
}

func TestInstrumentCounter(t *testing.T) {
	t.Parallel()

	metric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_requests_total",
			Help: "Test request counter",
		},
		[]string{"code", "method"},
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := InstrumentCounter(metric)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	metricCh := make(chan prometheus.Metric, 1)
	metric.Collect(metricCh)
	require.NotNil(t, <-metricCh, "metric should be collected")
}

func TestInstrumentRequestSize(t *testing.T) {
	t.Parallel()

	metric := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "test_request_size_bytes",
			Help: "Test request size in bytes",
		},
		[]string{"code", "method"},
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := InstrumentRequestSize(metric)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.Header.Set("Content-Length", "100")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	metricCh := make(chan prometheus.Metric, 1)
	metric.Collect(metricCh)
	require.NotNil(t, <-metricCh, "metric should be collected")
}

func TestInstrumentResponseSize(t *testing.T) {
	t.Parallel()

	metric := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "test_response_size_bytes",
			Help: "Test response size in bytes",
		},
		[]string{"code", "method"},
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test response"))
	})

	middleware := InstrumentResponseSize(metric)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "test response", rr.Body.String())

	metricCh := make(chan prometheus.Metric, 1)
	metric.Collect(metricCh)
	require.NotNil(t, <-metricCh, "metric should be collected")
}
