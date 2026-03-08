package gcp

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HealthResult holds the result of a health check.
type HealthResult struct {
	Healthy    bool
	StatusCode int
	Message    string
}

// HealthChecker performs HTTP health checks against a deployed service.
type HealthChecker struct {
	// Client is the HTTP client to use for health checks.
	Client *http.Client

	// Timeout is the maximum time to wait for the service to become healthy.
	Timeout time.Duration

	// Interval is the time between health check attempts.
	Interval time.Duration
}

// Check performs a health check against the given URL.
// It retries until the service responds with a 2xx status or the timeout expires.
func (h *HealthChecker) Check(ctx context.Context, url string) *HealthResult {
	if h.Timeout == 0 {
		h.Timeout = 30 * time.Second
	}
	if h.Interval == 0 {
		h.Interval = 2 * time.Second
	}

	deadline := time.Now().Add(h.Timeout)
	var lastErr error
	var lastStatus int

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return &HealthResult{
				Healthy: false,
				Message: fmt.Sprintf("context cancelled: %v", ctx.Err()),
			}
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return &HealthResult{
				Healthy: false,
				Message: fmt.Sprintf("failed to create request: %v", err),
			}
		}

		resp, err := h.Client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(h.Interval)
			continue
		}
		_ = resp.Body.Close()
		lastStatus = resp.StatusCode

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return &HealthResult{
				Healthy:    true,
				StatusCode: resp.StatusCode,
				Message:    "service is healthy",
			}
		}

		time.Sleep(h.Interval)
	}

	msg := fmt.Sprintf("health check timed out after %s", h.Timeout)
	if lastErr != nil {
		msg = fmt.Sprintf("%s: last error: %v", msg, lastErr)
	} else if lastStatus != 0 {
		msg = fmt.Sprintf("%s: last status code: %d", msg, lastStatus)
	}

	return &HealthResult{
		Healthy:    false,
		StatusCode: lastStatus,
		Message:    msg,
	}
}
