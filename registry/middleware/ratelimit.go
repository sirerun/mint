package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter tracks request counts per key within a time window.
type RateLimiter struct {
	mu     sync.Mutex
	counts map[string]*bucket
	limit  int
	window time.Duration
}

type bucket struct {
	count   int
	resetAt time.Time
}

// NewRateLimiter creates a rate limiter that allows limit requests per window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		counts: make(map[string]*bucket),
		limit:  limit,
		window: window,
	}
}

// Allow checks if a request from the given key is allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.counts[key]
	if !ok || now.After(b.resetAt) {
		rl.counts[key] = &bucket{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	if b.count >= rl.limit {
		return false
	}
	b.count++
	return true
}

// Remaining returns how many requests remain for the given key.
func (rl *RateLimiter) Remaining(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.counts[key]
	if !ok || now.After(b.resetAt) {
		return rl.limit
	}
	rem := rl.limit - b.count
	if rem < 0 {
		return 0
	}
	return rem
}

// RateLimit returns middleware that rate-limits based on a key extractor.
func RateLimit(rl *RateLimiter, keyFunc func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if !rl.Allow(key) {
				remaining := rl.Remaining(key)
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// PublisherKeyFunc extracts the publisher ID from context for rate limiting.
func PublisherKeyFunc(r *http.Request) string {
	p := PublisherFromContext(r.Context())
	if p != nil {
		return "publisher:" + p.ID
	}
	return "ip:" + r.RemoteAddr
}

// IPKeyFunc extracts the client IP for rate limiting.
func IPKeyFunc(r *http.Request) string {
	return "ip:" + r.RemoteAddr
}
