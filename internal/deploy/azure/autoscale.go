package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers"
)

// DefaultConcurrentRequests is the default HTTP concurrent requests threshold
// for KEDA auto-scaling rules.
const DefaultConcurrentRequests = 100

// AutoScaleConfig holds configuration for Container Apps auto-scaling.
type AutoScaleConfig struct {
	AppName            string
	ResourceGroup      string
	MinReplicas        int
	MaxReplicas        int
	ConcurrentRequests int
}

// AutoScaler configures auto-scaling policies for Azure Container Apps.
type AutoScaler interface {
	ConfigureAutoScaling(ctx context.Context, config AutoScaleConfig) error
}

// autoscaleAPI abstracts the Azure Container Apps SDK methods used by
// the auto-scaling implementation.
type autoscaleAPI interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, containerAppName string, containerAppEnvelope armappcontainers.ContainerApp, options *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error)
}

// KEDAAutoScaler configures KEDA-based auto-scaling on Azure Container Apps
// using HTTP concurrent requests as the scaling rule.
type KEDAAutoScaler struct {
	API autoscaleAPI
}

var _ AutoScaler = (*KEDAAutoScaler)(nil)

// ConfigureAutoScaling applies KEDA HTTP concurrent requests scaling rules
// to the specified Container App.
func (s *KEDAAutoScaler) ConfigureAutoScaling(ctx context.Context, config AutoScaleConfig) error {
	if err := validateAutoScaleConfig(config); err != nil {
		return err
	}

	concurrent := config.ConcurrentRequests
	if concurrent == 0 {
		concurrent = DefaultConcurrentRequests
	}

	concurrentStr := fmt.Sprintf("%d", concurrent)

	envelope := armappcontainers.ContainerApp{
		Properties: &armappcontainers.ContainerAppProperties{
			Template: &armappcontainers.Template{
				Scale: &armappcontainers.Scale{
					MinReplicas: int32Ptr(int32(config.MinReplicas)),
					MaxReplicas: int32Ptr(int32(config.MaxReplicas)),
					Rules: []*armappcontainers.ScaleRule{
						{
							Name: strPtr("http-concurrency"),
							HTTP: &armappcontainers.HTTPScaleRule{
								Metadata: map[string]*string{
									"concurrentRequests": &concurrentStr,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := s.API.CreateOrUpdate(ctx, config.ResourceGroup, config.AppName, envelope, nil)
	if err != nil {
		return fmt.Errorf("configure auto-scaling for %s: %w", config.AppName, err)
	}

	return nil
}

func validateAutoScaleConfig(config AutoScaleConfig) error {
	if config.AppName == "" {
		return fmt.Errorf("app name must not be empty")
	}
	if config.ResourceGroup == "" {
		return fmt.Errorf("resource group must not be empty")
	}
	if config.MinReplicas < 0 {
		return fmt.Errorf("min replicas must be non-negative, got %d", config.MinReplicas)
	}
	if config.MaxReplicas < 1 {
		return fmt.Errorf("max replicas must be at least 1, got %d", config.MaxReplicas)
	}
	if config.MinReplicas > config.MaxReplicas {
		return fmt.Errorf("min replicas (%d) must not exceed max replicas (%d)", config.MinReplicas, config.MaxReplicas)
	}
	if config.ConcurrentRequests < 0 {
		return fmt.Errorf("concurrent requests must be non-negative, got %d", config.ConcurrentRequests)
	}
	return nil
}
