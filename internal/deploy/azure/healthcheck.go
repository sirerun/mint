package azure

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// HealthChecker performs post-deploy health checks.
type HealthChecker struct {
	HTTPClient *http.Client
	MaxRetries int // default 5
}

// HealthCheckResult holds the outcome of a health check.
type HealthCheckResult struct {
	Healthy    bool
	StatusCode int
	Body       string
	Attempts   int
	Duration   time.Duration
}

// NewHealthChecker creates a HealthChecker with defaults (5 retries).
func NewHealthChecker(httpClient *http.Client) *HealthChecker {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &HealthChecker{
		HTTPClient: httpClient,
		MaxRetries: 5,
	}
}

// Check sends GET requests to <serviceURL>/health with exponential backoff
// retries. It considers status 200 as healthy. It returns error only for
// non-retryable failures such as context cancellation or invalid URL.
func (h *HealthChecker) Check(ctx context.Context, serviceURL string) (*HealthCheckResult, error) {
	start := time.Now()
	url := strings.TrimRight(serviceURL, "/") + "/health"

	maxRetries := h.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 5
	}

	backoff := time.Second
	var lastResult *HealthCheckResult

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Fprintf(os.Stderr, "Health check attempt %d/%d...\n", attempt, maxRetries)

		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("health check cancelled: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("invalid health check URL: %w", err)
		}

		resp, err := h.HTTPClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("health check cancelled: %w", ctx.Err())
			}
			lastResult = &HealthCheckResult{
				Healthy:  false,
				Attempts: attempt,
				Duration: time.Since(start),
			}
			if attempt < maxRetries {
				sleepWithContext(ctx, backoff)
				backoff *= 2
				continue
			}
			return lastResult, nil
		}

		body := readBody(resp)
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return &HealthCheckResult{
				Healthy:    true,
				StatusCode: resp.StatusCode,
				Body:       body,
				Attempts:   attempt,
				Duration:   time.Since(start),
			}, nil
		}

		lastResult = &HealthCheckResult{
			Healthy:    false,
			StatusCode: resp.StatusCode,
			Body:       body,
			Attempts:   attempt,
			Duration:   time.Since(start),
		}
		if attempt < maxRetries {
			sleepWithContext(ctx, backoff)
			backoff *= 2
			continue
		}
	}

	return lastResult, nil
}

// sleepWithContext sleeps for the given duration or until the context is done.
func sleepWithContext(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

// readBody reads and returns the response body as a string, limited to 1KB.
func readBody(resp *http.Response) string {
	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	return string(buf[:n])
}
