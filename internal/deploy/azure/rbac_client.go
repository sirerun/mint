package azure

import (
	"context"
)

// RBACClient abstracts Azure Role-Based Access Control operations.
type RBACClient interface {
	// AssignRole assigns an RBAC role to a principal on a given scope.
	AssignRole(ctx context.Context, scope, roleDefinitionID, principalID string) error

	// EnsureKeyVaultPolicy ensures that the given principal has access to
	// the specified Key Vault for secret operations.
	EnsureKeyVaultPolicy(ctx context.Context, resourceGroup, vaultName, principalID string) error
}
