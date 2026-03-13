package gcp

import (
	"context"
	"fmt"
)

// ObservabilityConfig holds configuration for GCP Cloud Run observability.
type ObservabilityConfig struct {
	ServiceName   string
	ProjectID     string
	Region        string
	EnableMetrics bool
}

// ObservabilityConfigurator configures provider-native logging and metrics.
type ObservabilityConfigurator interface {
	ConfigureObservability(ctx context.Context, config ObservabilityConfig) error
}

// loggingAPI abstracts the GCP operations used by ObservabilityAdapter.
type loggingAPI interface {
	UpdateServiceLabels(ctx context.Context, projectID, region, serviceName string, labels map[string]string) error
}

// ObservabilityAdapter configures Cloud Logging and Cloud Monitoring for Cloud Run services.
// Cloud Run has built-in logging, so this primarily adds structured logging labels
// and optionally enables custom metrics.
type ObservabilityAdapter struct {
	client loggingAPI
}

var _ ObservabilityConfigurator = (*ObservabilityAdapter)(nil)

// NewObservabilityAdapter creates an ObservabilityAdapter.
func NewObservabilityAdapter(client loggingAPI) *ObservabilityAdapter {
	return &ObservabilityAdapter{client: client}
}

// ConfigureObservability adds structured logging labels to the Cloud Run service.
// Cloud Run integrates with Cloud Logging by default, so this is a lightweight
// configuration step rather than a full setup.
func (a *ObservabilityAdapter) ConfigureObservability(ctx context.Context, config ObservabilityConfig) error {
	if config.ServiceName == "" {
		return fmt.Errorf("observability: service name is required")
	}
	if config.ProjectID == "" {
		return fmt.Errorf("observability: project ID is required")
	}
	if config.Region == "" {
		return fmt.Errorf("observability: region is required")
	}

	labels := map[string]string{
		"observability": "enabled",
		"service":       SanitizeLabel(config.ServiceName),
	}
	if config.EnableMetrics {
		labels["metrics"] = "enabled"
	}

	if err := a.client.UpdateServiceLabels(ctx, config.ProjectID, config.Region, config.ServiceName, labels); err != nil {
		return fmt.Errorf("observability: update labels: %w", err)
	}

	return nil
}
