package azure

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers"
)

// environmentAPI abstracts the Azure Managed Environments SDK methods.
type environmentAPI interface {
	Get(ctx context.Context, resourceGroupName string, environmentName string, options *armappcontainers.ManagedEnvironmentsClientGetOptions) (armappcontainers.ManagedEnvironmentsClientGetResponse, error)
	CreateOrUpdate(ctx context.Context, resourceGroupName string, environmentName string, environmentEnvelope armappcontainers.ManagedEnvironment, options *armappcontainers.ManagedEnvironmentsClientBeginCreateOrUpdateOptions) (*armappcontainers.ManagedEnvironmentsClientCreateOrUpdateResponse, error)
}

// environmentSDKAdapter wraps the SDK poller-based CreateOrUpdate.
type environmentSDKAdapter struct {
	client *armappcontainers.ManagedEnvironmentsClient
}

func (a *environmentSDKAdapter) Get(ctx context.Context, resourceGroupName, environmentName string, options *armappcontainers.ManagedEnvironmentsClientGetOptions) (armappcontainers.ManagedEnvironmentsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, environmentName, options)
}

func (a *environmentSDKAdapter) CreateOrUpdate(ctx context.Context, resourceGroupName, environmentName string, envelope armappcontainers.ManagedEnvironment, options *armappcontainers.ManagedEnvironmentsClientBeginCreateOrUpdateOptions) (*armappcontainers.ManagedEnvironmentsClientCreateOrUpdateResponse, error) {
	poller, err := a.client.BeginCreateOrUpdate(ctx, resourceGroupName, environmentName, envelope, options)
	if err != nil {
		return nil, err
	}
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// EnvironmentAdapter implements ManagedEnvironmentClient using the Azure SDK.
type EnvironmentAdapter struct {
	client environmentAPI
}

var _ ManagedEnvironmentClient = (*EnvironmentAdapter)(nil)

// NewEnvironmentAdapter creates a new environment adapter backed by the Azure SDK.
func NewEnvironmentAdapter(subscriptionID string, cred azcore.TokenCredential) (*EnvironmentAdapter, error) {
	client, err := armappcontainers.NewManagedEnvironmentsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create managed environments client: %w", err)
	}
	return &EnvironmentAdapter{client: &environmentSDKAdapter{client: client}}, nil
}

// EnsureEnvironment creates a managed environment if it does not exist and returns its resource ID.
func (a *EnvironmentAdapter) EnsureEnvironment(ctx context.Context, resourceGroup, envName, region string) (string, error) {
	resp, err := a.client.Get(ctx, resourceGroup, envName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == 404 {
			return a.createEnvironment(ctx, resourceGroup, envName, region)
		}
		return "", fmt.Errorf("get environment: %w", err)
	}
	if resp.ID == nil {
		return "", fmt.Errorf("environment %q has no ID", envName)
	}
	return *resp.ID, nil
}

func (a *EnvironmentAdapter) createEnvironment(ctx context.Context, resourceGroup, envName, region string) (string, error) {
	resp, err := a.client.CreateOrUpdate(ctx, resourceGroup, envName, armappcontainers.ManagedEnvironment{
		Location: strPtr(region),
	}, nil)
	if err != nil {
		return "", fmt.Errorf("create environment: %w", err)
	}
	if resp.ID == nil {
		return "", fmt.Errorf("created environment %q has no ID", envName)
	}
	return *resp.ID, nil
}
