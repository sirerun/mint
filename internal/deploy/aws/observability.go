package aws

import (
	"context"
	"fmt"
)

// ObservabilityConfig holds configuration for AWS ECS observability.
type ObservabilityConfig struct {
	ServiceName    string
	LogGroupPrefix string
	ClusterName    string
	EnableMetrics  bool
}

// ObservabilityConfigurator configures provider-native logging and metrics.
type ObservabilityConfigurator interface {
	ConfigureObservability(ctx context.Context, config ObservabilityConfig) error
}

// observabilityAPI abstracts the AWS SDK methods used by ObservabilityAdapter.
type observabilityAPI interface {
	CreateLogGroup(ctx context.Context, logGroupName string) error
	PutRetentionPolicy(ctx context.Context, logGroupName string, retentionDays int) error
	UpdateClusterSettings(ctx context.Context, clusterName string, containerInsights bool) error
}

// ObservabilityAdapter configures CloudWatch Logs and Container Insights for ECS services.
type ObservabilityAdapter struct {
	client observabilityAPI
}

var _ ObservabilityConfigurator = (*ObservabilityAdapter)(nil)

// NewObservabilityAdapter creates an ObservabilityAdapter.
func NewObservabilityAdapter(client observabilityAPI) *ObservabilityAdapter {
	return &ObservabilityAdapter{client: client}
}

// ConfigureObservability sets up CloudWatch Logs log group with a retention policy
// and optionally enables Container Insights on the ECS cluster.
func (a *ObservabilityAdapter) ConfigureObservability(ctx context.Context, config ObservabilityConfig) error {
	if config.ServiceName == "" {
		return fmt.Errorf("observability: service name is required")
	}
	if config.ClusterName == "" {
		return fmt.Errorf("observability: cluster name is required")
	}

	logGroupName := config.LogGroupPrefix + "/" + config.ServiceName
	if config.LogGroupPrefix == "" {
		logGroupName = "/ecs/" + config.ServiceName
	}

	if err := a.client.CreateLogGroup(ctx, logGroupName); err != nil {
		return fmt.Errorf("observability: create log group: %w", err)
	}

	const defaultRetentionDays = 30
	if err := a.client.PutRetentionPolicy(ctx, logGroupName, defaultRetentionDays); err != nil {
		return fmt.Errorf("observability: set retention policy: %w", err)
	}

	if config.EnableMetrics {
		if err := a.client.UpdateClusterSettings(ctx, config.ClusterName, true); err != nil {
			return fmt.Errorf("observability: enable container insights: %w", err)
		}
	}

	return nil
}
