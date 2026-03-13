package azure

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

type stubVaultAPI struct {
	getFunc            func(ctx context.Context, rg, name string, opts *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error)
	createOrUpdateFunc func(ctx context.Context, rg, name string, params armkeyvault.VaultCreateOrUpdateParameters, opts *armkeyvault.VaultsClientBeginCreateOrUpdateOptions) (*armkeyvault.VaultsClientCreateOrUpdateResponse, error)
}

func (s *stubVaultAPI) Get(ctx context.Context, rg, name string, opts *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
	return s.getFunc(ctx, rg, name, opts)
}

func (s *stubVaultAPI) CreateOrUpdate(ctx context.Context, rg, name string, params armkeyvault.VaultCreateOrUpdateParameters, opts *armkeyvault.VaultsClientBeginCreateOrUpdateOptions) (*armkeyvault.VaultsClientCreateOrUpdateResponse, error) {
	return s.createOrUpdateFunc(ctx, rg, name, params, opts)
}

type stubSecretsAPI struct {
	setSecretFunc func(ctx context.Context, name string, params azsecrets.SetSecretParameters, opts *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error)
	getSecretFunc func(ctx context.Context, name, version string, opts *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error)
}

func (s *stubSecretsAPI) SetSecret(ctx context.Context, name string, params azsecrets.SetSecretParameters, opts *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error) {
	return s.setSecretFunc(ctx, name, params, opts)
}

func (s *stubSecretsAPI) GetSecret(ctx context.Context, name, version string, opts *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
	return s.getSecretFunc(ctx, name, version, opts)
}

func TestKeyVaultAdapter_InterfaceCompliance(t *testing.T) {
	var _ KeyVaultClient = (*KeyVaultAdapter)(nil)
}

func TestKeyVaultAdapter_EnsureKeyVault_Exists(t *testing.T) {
	vaultURI := "https://myvault.vault.azure.net/"
	stub := &stubVaultAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
			return armkeyvault.VaultsClientGetResponse{
				Vault: armkeyvault.Vault{
					Properties: &armkeyvault.VaultProperties{
						VaultURI: &vaultURI,
					},
				},
			}, nil
		},
	}
	adapter := &KeyVaultAdapter{vaults: stub, tenantID: "tenant-id"}
	uri, err := adapter.EnsureKeyVault(context.Background(), "rg", "myvault", "eastus")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uri != vaultURI {
		t.Fatalf("got URI %q, want %q", uri, vaultURI)
	}
}

func TestKeyVaultAdapter_EnsureKeyVault_NotFound_Creates(t *testing.T) {
	vaultURI := "https://newvault.vault.azure.net/"
	stub := &stubVaultAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
			return armkeyvault.VaultsClientGetResponse{}, &azcore.ResponseError{StatusCode: 404}
		},
		createOrUpdateFunc: func(_ context.Context, _, _ string, _ armkeyvault.VaultCreateOrUpdateParameters, _ *armkeyvault.VaultsClientBeginCreateOrUpdateOptions) (*armkeyvault.VaultsClientCreateOrUpdateResponse, error) {
			return &armkeyvault.VaultsClientCreateOrUpdateResponse{
				Vault: armkeyvault.Vault{
					Properties: &armkeyvault.VaultProperties{
						VaultURI: &vaultURI,
					},
				},
			}, nil
		},
	}
	adapter := &KeyVaultAdapter{vaults: stub, tenantID: "tenant-id"}
	uri, err := adapter.EnsureKeyVault(context.Background(), "rg", "newvault", "eastus")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uri != vaultURI {
		t.Fatalf("got URI %q, want %q", uri, vaultURI)
	}
}

func TestKeyVaultAdapter_EnsureKeyVault_GetError(t *testing.T) {
	stub := &stubVaultAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
			return armkeyvault.VaultsClientGetResponse{}, errors.New("access denied")
		},
	}
	adapter := &KeyVaultAdapter{vaults: stub, tenantID: "tenant-id"}
	_, err := adapter.EnsureKeyVault(context.Background(), "rg", "myvault", "eastus")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestKeyVaultAdapter_EnsureKeyVault_CreateError(t *testing.T) {
	stub := &stubVaultAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
			return armkeyvault.VaultsClientGetResponse{}, &azcore.ResponseError{StatusCode: 404}
		},
		createOrUpdateFunc: func(_ context.Context, _, _ string, _ armkeyvault.VaultCreateOrUpdateParameters, _ *armkeyvault.VaultsClientBeginCreateOrUpdateOptions) (*armkeyvault.VaultsClientCreateOrUpdateResponse, error) {
			return nil, errors.New("quota exceeded")
		},
	}
	adapter := &KeyVaultAdapter{vaults: stub, tenantID: "tenant-id"}
	_, err := adapter.EnsureKeyVault(context.Background(), "rg", "myvault", "eastus")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestKeyVaultAdapter_EnsureKeyVault_NilURI(t *testing.T) {
	stub := &stubVaultAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
			return armkeyvault.VaultsClientGetResponse{
				Vault: armkeyvault.Vault{
					Properties: &armkeyvault.VaultProperties{VaultURI: nil},
				},
			}, nil
		},
	}
	adapter := &KeyVaultAdapter{vaults: stub, tenantID: "tenant-id"}
	_, err := adapter.EnsureKeyVault(context.Background(), "rg", "myvault", "eastus")
	if err == nil {
		t.Fatal("expected error for nil URI, got nil")
	}
}

func TestKeyVaultAdapter_EnsureKeyVault_CreateNilURI(t *testing.T) {
	stub := &stubVaultAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
			return armkeyvault.VaultsClientGetResponse{}, &azcore.ResponseError{StatusCode: 404}
		},
		createOrUpdateFunc: func(_ context.Context, _, _ string, _ armkeyvault.VaultCreateOrUpdateParameters, _ *armkeyvault.VaultsClientBeginCreateOrUpdateOptions) (*armkeyvault.VaultsClientCreateOrUpdateResponse, error) {
			return &armkeyvault.VaultsClientCreateOrUpdateResponse{
				Vault: armkeyvault.Vault{
					Properties: &armkeyvault.VaultProperties{VaultURI: nil},
				},
			}, nil
		},
	}
	adapter := &KeyVaultAdapter{vaults: stub, tenantID: "tenant-id"}
	_, err := adapter.EnsureKeyVault(context.Background(), "rg", "myvault", "eastus")
	if err == nil {
		t.Fatal("expected error for nil URI on created vault, got nil")
	}
}

func TestKeyVaultAdapter_SetSecret(t *testing.T) {
	secrets := &stubSecretsAPI{
		setSecretFunc: func(_ context.Context, _ string, _ azsecrets.SetSecretParameters, _ *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error) {
			return azsecrets.SetSecretResponse{}, nil
		},
	}
	adapter := &KeyVaultAdapter{
		newSecretsFunc: func(_ string) (secretsAPI, error) { return secrets, nil },
	}
	err := adapter.SetSecret(context.Background(), "https://vault.vault.azure.net", "mysecret", "myvalue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKeyVaultAdapter_SetSecret_Error(t *testing.T) {
	secrets := &stubSecretsAPI{
		setSecretFunc: func(_ context.Context, _ string, _ azsecrets.SetSecretParameters, _ *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error) {
			return azsecrets.SetSecretResponse{}, errors.New("forbidden")
		},
	}
	adapter := &KeyVaultAdapter{
		newSecretsFunc: func(_ string) (secretsAPI, error) { return secrets, nil },
	}
	err := adapter.SetSecret(context.Background(), "https://vault.vault.azure.net", "mysecret", "myvalue")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestKeyVaultAdapter_SetSecret_ClientError(t *testing.T) {
	adapter := &KeyVaultAdapter{
		newSecretsFunc: func(_ string) (secretsAPI, error) { return nil, errors.New("client init failed") },
	}
	err := adapter.SetSecret(context.Background(), "https://vault.vault.azure.net", "mysecret", "myvalue")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestKeyVaultAdapter_GetSecretURI(t *testing.T) {
	secretID := azsecrets.ID("https://vault.vault.azure.net/secrets/mysecret/abc123")
	secrets := &stubSecretsAPI{
		getSecretFunc: func(_ context.Context, _, _ string, _ *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
			return azsecrets.GetSecretResponse{
				Secret: azsecrets.Secret{
					ID: &secretID,
				},
			}, nil
		},
	}
	adapter := &KeyVaultAdapter{
		newSecretsFunc: func(_ string) (secretsAPI, error) { return secrets, nil },
	}
	uri, err := adapter.GetSecretURI(context.Background(), "https://vault.vault.azure.net", "mysecret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://vault.vault.azure.net/secrets/mysecret/abc123"
	if uri != want {
		t.Fatalf("got URI %q, want %q", uri, want)
	}
}

func TestKeyVaultAdapter_GetSecretURI_NotFound(t *testing.T) {
	secrets := &stubSecretsAPI{
		getSecretFunc: func(_ context.Context, _, _ string, _ *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
			return azsecrets.GetSecretResponse{}, &azcore.ResponseError{StatusCode: 404}
		},
	}
	adapter := &KeyVaultAdapter{
		newSecretsFunc: func(_ string) (secretsAPI, error) { return secrets, nil },
	}
	_, err := adapter.GetSecretURI(context.Background(), "https://vault.vault.azure.net", "mysecret")
	if !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("expected ErrSecretNotFound, got %v", err)
	}
}

func TestKeyVaultAdapter_GetSecretURI_Error(t *testing.T) {
	secrets := &stubSecretsAPI{
		getSecretFunc: func(_ context.Context, _, _ string, _ *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
			return azsecrets.GetSecretResponse{}, errors.New("network error")
		},
	}
	adapter := &KeyVaultAdapter{
		newSecretsFunc: func(_ string) (secretsAPI, error) { return secrets, nil },
	}
	_, err := adapter.GetSecretURI(context.Background(), "https://vault.vault.azure.net", "mysecret")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestKeyVaultAdapter_GetSecretURI_ClientError(t *testing.T) {
	adapter := &KeyVaultAdapter{
		newSecretsFunc: func(_ string) (secretsAPI, error) { return nil, errors.New("client init failed") },
	}
	_, err := adapter.GetSecretURI(context.Background(), "https://vault.vault.azure.net", "mysecret")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestKeyVaultAdapter_GetSecretURI_NilID(t *testing.T) {
	secrets := &stubSecretsAPI{
		getSecretFunc: func(_ context.Context, _, _ string, _ *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
			return azsecrets.GetSecretResponse{
				Secret: azsecrets.Secret{ID: nil},
			}, nil
		},
	}
	adapter := &KeyVaultAdapter{
		newSecretsFunc: func(_ string) (secretsAPI, error) { return secrets, nil },
	}
	_, err := adapter.GetSecretURI(context.Background(), "https://vault.vault.azure.net", "mysecret")
	if err == nil {
		t.Fatal("expected error for nil ID, got nil")
	}
}
