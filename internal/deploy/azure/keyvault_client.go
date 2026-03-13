package azure

import (
	"context"
	"errors"
)

// ErrVaultNotFound indicates that a Key Vault was not found.
var ErrVaultNotFound = errors.New("keyvault: vault not found")

// ErrSecretNotFound indicates that a Key Vault secret was not found.
var ErrSecretNotFound = errors.New("keyvault: secret not found")

// KeyVaultClient abstracts Azure Key Vault operations.
type KeyVaultClient interface {
	// EnsureKeyVault creates a Key Vault if it does not already exist
	// and returns its URI.
	EnsureKeyVault(ctx context.Context, resourceGroup, vaultName, region string) (string, error)

	// SetSecret creates or updates a secret in a Key Vault.
	SetSecret(ctx context.Context, vaultURI, secretName, value string) error

	// GetSecretURI returns the full URI to a specific secret version.
	// Returns ErrSecretNotFound if the secret does not exist.
	GetSecretURI(ctx context.Context, vaultURI, secretName string) (string, error)
}
