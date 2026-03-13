package azure

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
)

// acrAPI abstracts the Azure Container Registry SDK methods used by ACRAdapter.
type acrAPI interface {
	Get(ctx context.Context, resourceGroupName string, registryName string, options *armcontainerregistry.RegistriesClientGetOptions) (armcontainerregistry.RegistriesClientGetResponse, error)
	Create(ctx context.Context, resourceGroupName string, registryName string, registry armcontainerregistry.Registry, options *armcontainerregistry.RegistriesClientBeginCreateOptions) (*armcontainerregistry.RegistriesClientCreateResponse, error)
}

// acrSDKAdapter wraps the SDK poller-based Create to match our synchronous acrAPI.
type acrSDKAdapter struct {
	client *armcontainerregistry.RegistriesClient
}

func (a *acrSDKAdapter) Get(ctx context.Context, resourceGroupName, registryName string, options *armcontainerregistry.RegistriesClientGetOptions) (armcontainerregistry.RegistriesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, registryName, options)
}

func (a *acrSDKAdapter) Create(ctx context.Context, resourceGroupName, registryName string, registry armcontainerregistry.Registry, options *armcontainerregistry.RegistriesClientBeginCreateOptions) (*armcontainerregistry.RegistriesClientCreateResponse, error) {
	poller, err := a.client.BeginCreate(ctx, resourceGroupName, registryName, registry, options)
	if err != nil {
		return nil, err
	}
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ACRAdapter implements ACRClient using the Azure SDK.
type ACRAdapter struct {
	client acrAPI
}

var _ ACRClient = (*ACRAdapter)(nil)

// NewACRAdapter creates a new ACR adapter backed by the Azure Container Registry SDK.
func NewACRAdapter(subscriptionID string, cred azcore.TokenCredential) (*ACRAdapter, error) {
	client, err := armcontainerregistry.NewRegistriesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create ACR client: %w", err)
	}
	return &ACRAdapter{client: &acrSDKAdapter{client: client}}, nil
}

// EnsureRepository ensures an ACR registry exists and returns the repository URI.
// Azure ACR does not require explicit repository creation; pushing an image creates
// the repository automatically. This method ensures the registry itself exists.
func (a *ACRAdapter) EnsureRepository(ctx context.Context, resourceGroup, registryName, repoName string) (string, error) {
	resp, err := a.client.Get(ctx, resourceGroup, registryName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == 404 {
			return a.createRegistryAndReturnURI(ctx, resourceGroup, registryName, repoName)
		}
		return "", fmt.Errorf("get registry: %w", err)
	}
	loginServer := ""
	if resp.Properties != nil && resp.Properties.LoginServer != nil {
		loginServer = *resp.Properties.LoginServer
	}
	return loginServer + "/" + repoName, nil
}

// GetLoginServer returns the login server URL for the given ACR registry.
func (a *ACRAdapter) GetLoginServer(ctx context.Context, resourceGroup, registryName string) (string, error) {
	resp, err := a.client.Get(ctx, resourceGroup, registryName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == 404 {
			return "", ErrRepositoryNotFound
		}
		return "", fmt.Errorf("get registry: %w", err)
	}
	if resp.Properties == nil || resp.Properties.LoginServer == nil {
		return "", fmt.Errorf("registry %q has no login server", registryName)
	}
	return *resp.Properties.LoginServer, nil
}

func (a *ACRAdapter) createRegistryAndReturnURI(ctx context.Context, resourceGroup, registryName, repoName string) (string, error) {
	sku := armcontainerregistry.SKUNameBasic
	resp, err := a.client.Create(ctx, resourceGroup, registryName, armcontainerregistry.Registry{
		Location: strPtr(""), // caller must set region via resource group default
		SKU:      &armcontainerregistry.SKU{Name: &sku},
	}, nil)
	if err != nil {
		return "", fmt.Errorf("create registry: %w", err)
	}
	loginServer := ""
	if resp.Properties != nil && resp.Properties.LoginServer != nil {
		loginServer = *resp.Properties.LoginServer
	}
	return loginServer + "/" + repoName, nil
}

func strPtr(s string) *string { return &s }
