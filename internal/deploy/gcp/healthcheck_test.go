package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	// Use short backoff by setting MaxRetries and using a custom approach.
	// We can't easily shorten backoff, but we can set MaxRetries high enough.
	// The test will take ~1+2 = 3 seconds for 2 retries before the 3rd succeeds.
	// To keep tests fast, we'll use a wrapper that overrides the sleep.
	// Instead, let's just accept the test takes a few seconds or use a simpler approach.

	// Actually, the backoff starts at 1s. With 2 failures the wait is 1s + 2s = 3s.
	// Let's keep it simple and accept the duration for correctness.
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
	// Cancel immediately so the second iteration detects it.
	// The first request will succeed (503), then before the second request
	// the context check catches it.
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
