package azure

import (
	"context"
)

// ManagedEnvironmentClient abstracts Azure Container Apps Managed Environment operations.
type ManagedEnvironmentClient interface {
	// EnsureEnvironment creates a managed environment if it does not already exist
	// and returns its resource ID.
	EnsureEnvironment(ctx context.Context, resourceGroup, envName, region string) (string, error)
}
