package azure

import (
	"context"
	"fmt"
)

// ObservabilityConfig holds configuration for Azure Container Apps observability.
type ObservabilityConfig struct {
	ServiceName   string
	ResourceGroup string
	WorkspaceName string
	EnableMetrics bool
}

// ObservabilityConfigurator configures provider-native logging and metrics.
type ObservabilityConfigurator interface {
	ConfigureObservability(ctx context.Context, config ObservabilityConfig) error
}

// logAnalyticsAPI abstracts the Azure SDK methods used by ObservabilityAdapter.
type logAnalyticsAPI interface {
	EnsureWorkspace(ctx context.Context, resourceGroup, workspaceName string) (string, error)
	LinkEnvironment(ctx context.Context, resourceGroup, workspaceName, environmentName string) error
}

// ObservabilityAdapter configures Log Analytics workspace on Azure Container Apps Environment.
type ObservabilityAdapter struct {
	client logAnalyticsAPI
}

var _ ObservabilityConfigurator = (*ObservabilityAdapter)(nil)

// NewObservabilityAdapter creates an ObservabilityAdapter.
func NewObservabilityAdapter(client logAnalyticsAPI) *ObservabilityAdapter {
	return &ObservabilityAdapter{client: client}
}

// ConfigureObservability creates or ensures a Log Analytics workspace and links it
// to the Container Apps Environment for the given service.
func (a *ObservabilityAdapter) ConfigureObservability(ctx context.Context, config ObservabilityConfig) error {
	if config.ServiceName == "" {
		return fmt.Errorf("observability: service name is required")
	}
	if config.ResourceGroup == "" {
		return fmt.Errorf("observability: resource group is required")
	}

	workspaceName := config.WorkspaceName
	if workspaceName == "" {
		workspaceName = config.ServiceName + "-logs"
	}

	if _, err := a.client.EnsureWorkspace(ctx, config.ResourceGroup, workspaceName); err != nil {
		return fmt.Errorf("observability: ensure workspace: %w", err)
	}

	if err := a.client.LinkEnvironment(ctx, config.ResourceGroup, workspaceName, config.ServiceName); err != nil {
		return fmt.Errorf("observability: link environment: %w", err)
	}

	return nil
}
