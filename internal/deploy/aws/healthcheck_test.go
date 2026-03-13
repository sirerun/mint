package aws

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCheck_HealthyFirstAttempt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	hc := NewHealthChecker(srv.Client())
	result, err := hc.Check(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Healthy {
		t.Fatal("expected healthy")
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}
	if result.Attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", result.Attempts)
	}
	if result.Body != `{"status":"ok"}` {
		t.Fatalf("unexpected body: %s", result.Body)
	}
}

func TestCheck_RetriesThenHealthy(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	hc := NewHealthChecker(srv.Client())
	result, err := hc.Check(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Healthy {
		t.Fatal("expected healthy after retries")
	}
	if result.Attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", result.Attempts)
	}
}

func TestCheck_NeverHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("unavailable"))
	}))
	defer srv.Close()

	hc := NewHealthChecker(srv.Client())
	hc.MaxRetries = 2 // keep test fast

	result, err := hc.Check(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Healthy {
		t.Fatal("expected unhealthy")
	}
	if result.Attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", result.Attempts)
	}
	if result.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", result.StatusCode)
	}
}

func TestCheck_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	hc := NewHealthChecker(srv.Client())
	hc.MaxRetries = 5

	_, err := hc.Check(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

func TestCheck_CustomMaxRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	hc := NewHealthChecker(srv.Client())
	hc.MaxRetries = 1

	result, err := hc.Check(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Healthy {
		t.Fatal("expected unhealthy")
	}
	if result.Attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", result.Attempts)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 1 HTTP call, got %d", got)
	}
}

func TestNewHealthChecker_WithNonNilClient(t *testing.T) {
	custom := &http.Client{Timeout: 30 * time.Second}
	hc := NewHealthChecker(custom)
	if hc.HTTPClient != custom {
		t.Fatal("expected the provided HTTP client to be used")
	}
	if hc.MaxRetries != 5 {
		t.Fatalf("expected MaxRetries 5, got %d", hc.MaxRetries)
	}
}

func TestNewHealthChecker_NilClient(t *testing.T) {
	hc := NewHealthChecker(nil)
	if hc.HTTPClient == nil {
		t.Fatal("expected non-nil HTTP client")
	}
	if hc.HTTPClient.Timeout != 10*time.Second {
		t.Fatalf("expected 10s timeout, got %v", hc.HTTPClient.Timeout)
	}
}

func TestCheck_DefaultMaxRetries(t *testing.T) {
	// MaxRetries <= 0 should default to 5.
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 5 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	hc := &HealthChecker{
		HTTPClient: srv.Client(),
		MaxRetries: 0, // should default to 5
	}
	result, err := hc.Check(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Healthy {
		t.Fatal("expected healthy on 5th attempt")
	}
	if result.Attempts != 5 {
		t.Fatalf("expected 5 attempts, got %d", result.Attempts)
	}
}

func TestCheck_ContextCancelledBeforeLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling Check

	hc := NewHealthChecker(&http.Client{})
	_, err := hc.Check(ctx, "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "health check cancelled") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCheck_InvalidURL(t *testing.T) {
	hc := NewHealthChecker(&http.Client{})
	// Use a URL with an invalid scheme to trigger NewRequestWithContext error.
	_, err := hc.Check(context.Background(), "://invalid")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "invalid health check URL") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCheck_HTTPErrorExhaustsRetries(t *testing.T) {
	// Point to a port that refuses connections; single retry so it exhausts quickly.
	hc := &HealthChecker{
		HTTPClient: &http.Client{Timeout: 100 * time.Millisecond},
		MaxRetries: 1,
	}
	result, err := hc.Check(context.Background(), "http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Healthy {
		t.Fatal("expected unhealthy")
	}
	if result.Attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", result.Attempts)
	}
}

func TestCheck_HTTPErrorRetryThenSucceed(t *testing.T) {
	// First request fails with connection error, second succeeds.
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	// Custom transport that fails the first call then delegates to real transport.
	realTransport := srv.Client().Transport
	srv.Client().Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		n := calls.Add(1)
		if n == 1 {
			return nil, errors.New("connection refused")
		}
		return realTransport.RoundTrip(req)
	})

	hc := &HealthChecker{
		HTTPClient: srv.Client(),
		MaxRetries: 3,
	}
	result, err := hc.Check(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Healthy {
		t.Fatal("expected healthy after retry")
	}
	if result.Attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", result.Attempts)
	}
}

func TestCheck_ContextCancelledDuringDo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Transport that cancels context then returns error.
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			cancel()
			return nil, context.Canceled
		}),
	}

	hc := &HealthChecker{
		HTTPClient: client,
		MaxRetries: 3,
	}
	_, err := hc.Check(ctx, "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error from context cancellation during Do")
	}
	if !strings.Contains(err.Error(), "health check cancelled") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// roundTripFunc is a helper to create http.RoundTripper from a function.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
