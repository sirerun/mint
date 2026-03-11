package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !rl.Allow("key1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	if rl.Allow("key1") {
		t.Error("4th request should be denied")
	}

	// Different key should be allowed.
	if !rl.Allow("key2") {
		t.Error("different key should be allowed")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	rl := NewRateLimiter(1, 10*time.Millisecond)

	if !rl.Allow("key1") {
		t.Error("first request should be allowed")
	}
	if rl.Allow("key1") {
		t.Error("second request should be denied")
	}

	time.Sleep(15 * time.Millisecond)

	if !rl.Allow("key1") {
		t.Error("request after window reset should be allowed")
	}
}

func TestRateLimiter_Remaining(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)

	if r := rl.Remaining("key1"); r != 5 {
		t.Errorf("remaining = %d, want 5", r)
	}

	rl.Allow("key1")
	rl.Allow("key1")

	if r := rl.Remaining("key1"); r != 3 {
		t.Errorf("remaining = %d, want 3", r)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RateLimit(rl, func(r *http.Request) string {
		return "test-key"
	})(handler)

	// First two requests should pass.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i+1, w.Code, http.StatusOK)
		}
	}

	// Third should be rate limited.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestPublisherKeyFunc(t *testing.T) {
	// Without publisher context.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	key := PublisherKeyFunc(req)
	if key != "ip:1.2.3.4:5678" {
		t.Errorf("key = %q, want %q", key, "ip:1.2.3.4:5678")
	}
}

func TestIPKeyFunc(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	key := IPKeyFunc(req)
	if key != "ip:10.0.0.1:1234" {
		t.Errorf("key = %q, want %q", key, "ip:10.0.0.1:1234")
	}
}
