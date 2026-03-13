package managed

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"
)

// DeployFromSource creates a tarball of sourceDir, uploads it, deploys via the API,
// and polls until the server reaches a terminal state. Progress is written to stderr.
func DeployFromSource(ctx context.Context, client HostingClient, sourceDir, serviceName string, public bool, stderr io.Writer) (*DeployOutput, error) {
	// Step 1: Create tarball.
	_, _ = fmt.Fprintln(stderr, "creating source tarball...")
	var tarBuf bytes.Buffer
	if err := CreateSourceTarball(sourceDir, &tarBuf); err != nil {
		return nil, fmt.Errorf("creating tarball: %w", err)
	}

	// Step 2: Upload tarball.
	hc, ok := client.(*httpClient)
	if !ok {
		return nil, fmt.Errorf("upload requires an HTTP-based client")
	}

	_, _ = fmt.Fprintln(stderr, "uploading source...")
	sourceID, err := uploadSource(ctx, hc, &tarBuf, int64(tarBuf.Len()), stderr)
	if err != nil {
		return nil, fmt.Errorf("uploading source: %w", err)
	}

	// Step 3: Deploy.
	_, _ = fmt.Fprintln(stderr, "deploying...")
	out, err := client.Deploy(ctx, &DeployInput{
		Source:      sourceID,
		ServiceName: serviceName,
		Public:      public,
	})
	if err != nil {
		return nil, fmt.Errorf("deploying: %w", err)
	}

	// Step 4: Poll for status.
	_, _ = fmt.Fprintln(stderr, "waiting for deployment...")
	if err := pollDeployment(ctx, client, out.ServiceID, stderr); err != nil {
		return nil, err
	}

	return out, nil
}

// pollDeployment polls the server status with exponential backoff until a terminal state is reached.
func pollDeployment(ctx context.Context, client HostingClient, serviceID string, stderr io.Writer) error {
	const (
		initialInterval = 2 * time.Second
		maxInterval     = 10 * time.Second
		multiplier      = 2
	)

	interval := initialInterval
	lastState := ""

	for {
		status, err := client.Status(ctx, serviceID)
		if err != nil {
			return fmt.Errorf("polling status: %w", err)
		}

		if status.State != lastState {
			_, _ = fmt.Fprintf(stderr, "status: %s\n", status.State)
			lastState = status.State
		}

		switch status.State {
		case "ready", "running":
			return nil
		case "failed", "error":
			return fmt.Errorf("deployment failed: server state is %q", status.State)
		}

		// Wait with exponential backoff.
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("deployment timed out: %w", ctx.Err())
		case <-timer.C:
		}

		interval *= multiplier
		if interval > maxInterval {
			interval = maxInterval
		}
	}
}
