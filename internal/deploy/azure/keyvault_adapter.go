package azure

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

// vaultAPI abstracts the Azure Key Vault management SDK methods.
type vaultAPI interface {
	Get(ctx context.Context, resourceGroupName string, vaultName string, options *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error)
	CreateOrUpdate(ctx context.Context, resourceGroupName string, vaultName string, parameters armkeyvault.VaultCreateOrUpdateParameters, options *armkeyvault.VaultsClientBeginCreateOrUpdateOptions) (*armkeyvault.VaultsClientCreateOrUpdateResponse, error)
}

// secretsAPI abstracts the Azure Key Vault secrets data-plane SDK methods.
type secretsAPI interface {
	SetSecret(ctx context.Context, secretName string, parameters azsecrets.SetSecretParameters, options *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error)
	GetSecret(ctx context.Context, secretName string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error)
}

// vaultSDKAdapter wraps the SDK poller-based CreateOrUpdate.
type vaultSDKAdapter struct {
	client *armkeyvault.VaultsClient
}

func (a *vaultSDKAdapter) Get(ctx context.Context, resourceGroupName, vaultName string, options *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, vaultName, options)
}

func (a *vaultSDKAdapter) CreateOrUpdate(ctx context.Context, resourceGroupName, vaultName string, parameters armkeyvault.VaultCreateOrUpdateParameters, options *armkeyvault.VaultsClientBeginCreateOrUpdateOptions) (*armkeyvault.VaultsClientCreateOrUpdateResponse, error) {
	poller, err := a.client.BeginCreateOrUpdate(ctx, resourceGroupName, vaultName, parameters, options)
	if err != nil {
		return nil, err
	}
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// KeyVaultAdapter implements KeyVaultClient using the Azure SDK.
type KeyVaultAdapter struct {
	vaults         vaultAPI
	tenantID       string
	newSecretsFunc func(vaultURI string) (secretsAPI, error)
}

var _ KeyVaultClient = (*KeyVaultAdapter)(nil)

// NewKeyVaultAdapter creates a new Key Vault adapter backed by the Azure SDK.
func NewKeyVaultAdapter(subscriptionID, tenantID string, cred azcore.TokenCredential) (*KeyVaultAdapter, error) {
	vaultsClient, err := armkeyvault.NewVaultsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create vaults client: %w", err)
	}
	return &KeyVaultAdapter{
		vaults:   &vaultSDKAdapter{client: vaultsClient},
		tenantID: tenantID,
		newSecretsFunc: func(vaultURI string) (secretsAPI, error) {
			return azsecrets.NewClient(vaultURI, cred, nil)
		},
	}, nil
}

// EnsureKeyVault creates a Key Vault if it does not exist and returns its URI.
func (a *KeyVaultAdapter) EnsureKeyVault(ctx context.Context, resourceGroup, vaultName, region string) (string, error) {
	resp, err := a.vaults.Get(ctx, resourceGroup, vaultName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == 404 {
			return a.createVault(ctx, resourceGroup, vaultName, region)
		}
		return "", fmt.Errorf("get vault: %w", err)
	}
	if resp.Properties == nil || resp.Properties.VaultURI == nil {
		return "", fmt.Errorf("vault %q has no URI", vaultName)
	}
	return *resp.Properties.VaultURI, nil
}

// SetSecret creates or updates a secret in a Key Vault.
func (a *KeyVaultAdapter) SetSecret(ctx context.Context, vaultURI, secretName, value string) error {
	client, err := a.newSecretsFunc(vaultURI)
	if err != nil {
		return fmt.Errorf("create secrets client: %w", err)
	}
	_, err = client.SetSecret(ctx, secretName, azsecrets.SetSecretParameters{
		Value: strPtr(value),
	}, nil)
	if err != nil {
		return fmt.Errorf("set secret: %w", err)
	}
	return nil
}

// GetSecretURI returns the full URI to a specific secret.
func (a *KeyVaultAdapter) GetSecretURI(ctx context.Context, vaultURI, secretName string) (string, error) {
	client, err := a.newSecretsFunc(vaultURI)
	if err != nil {
		return "", fmt.Errorf("create secrets client: %w", err)
	}
	resp, err := client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == 404 {
			return "", ErrSecretNotFound
		}
		return "", fmt.Errorf("get secret: %w", err)
	}
	if resp.ID == nil {
		return "", fmt.Errorf("secret %q has no ID", secretName)
	}
	return string(*resp.ID), nil
}

func (a *KeyVaultAdapter) createVault(ctx context.Context, resourceGroup, vaultName, region string) (string, error) {
	resp, err := a.vaults.CreateOrUpdate(ctx, resourceGroup, vaultName, armkeyvault.VaultCreateOrUpdateParameters{
		Location: strPtr(region),
		Properties: &armkeyvault.VaultProperties{
			TenantID: strPtr(a.tenantID),
			SKU: &armkeyvault.SKU{
				Family: skuFamilyPtr(armkeyvault.SKUFamilyA),
				Name:   skuNamePtr(armkeyvault.SKUNameStandard),
			},
		},
	}, nil)
	if err != nil {
		return "", fmt.Errorf("create vault: %w", err)
	}
	if resp.Properties == nil || resp.Properties.VaultURI == nil {
		return "", fmt.Errorf("created vault %q has no URI", vaultName)
	}
	return *resp.Properties.VaultURI, nil
}

func skuFamilyPtr(f armkeyvault.SKUFamily) *armkeyvault.SKUFamily { return &f }
func skuNamePtr(n armkeyvault.SKUName) *armkeyvault.SKUName       { return &n }
