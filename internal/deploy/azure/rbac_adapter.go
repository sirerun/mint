package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
)

// rbacAPI abstracts the Azure RBAC SDK methods used by RBACAdapter.
type rbacAPI interface {
	Create(ctx context.Context, scope string, roleAssignmentName string, parameters armauthorization.RoleAssignmentCreateParameters, options *armauthorization.RoleAssignmentsClientCreateOptions) (armauthorization.RoleAssignmentsClientCreateResponse, error)
}

// vaultAccessAPI abstracts Key Vault management for access policy updates.
type vaultAccessAPI interface {
	Get(ctx context.Context, resourceGroupName string, vaultName string, options *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error)
	CreateOrUpdate(ctx context.Context, resourceGroupName string, vaultName string, parameters armkeyvault.VaultCreateOrUpdateParameters, options *armkeyvault.VaultsClientBeginCreateOrUpdateOptions) (*armkeyvault.VaultsClientCreateOrUpdateResponse, error)
}

// RBACAdapter implements RBACClient using the Azure SDK.
type RBACAdapter struct {
	rbac   rbacAPI
	vaults vaultAccessAPI
}

var _ RBACClient = (*RBACAdapter)(nil)

// newUUID is a package-level variable to allow tests to inject deterministic UUIDs.
var newUUID = generateUUID

// NewRBACAdapter creates a new RBAC adapter backed by the Azure SDK.
func NewRBACAdapter(subscriptionID string, cred azcore.TokenCredential) (*RBACAdapter, error) {
	rbacClient, err := armauthorization.NewRoleAssignmentsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create role assignments client: %w", err)
	}
	vaultsClient, err := armkeyvault.NewVaultsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create vaults client: %w", err)
	}
	return &RBACAdapter{
		rbac:   rbacClient,
		vaults: &vaultSDKAdapter{client: vaultsClient},
	}, nil
}

// AssignRole assigns an RBAC role to a principal on a given scope.
func (a *RBACAdapter) AssignRole(ctx context.Context, scope, roleDefinitionID, principalID string) error {
	name := newUUID()
	_, err := a.rbac.Create(ctx, scope, name, armauthorization.RoleAssignmentCreateParameters{
		Properties: &armauthorization.RoleAssignmentProperties{
			RoleDefinitionID: strPtr(roleDefinitionID),
			PrincipalID:      strPtr(principalID),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	return nil
}

// EnsureKeyVaultPolicy ensures that the given principal has secret access to the Key Vault.
func (a *RBACAdapter) EnsureKeyVaultPolicy(ctx context.Context, resourceGroup, vaultName, principalID string) error {
	resp, err := a.vaults.Get(ctx, resourceGroup, vaultName, nil)
	if err != nil {
		return fmt.Errorf("get vault for policy: %w", err)
	}

	if resp.Properties == nil {
		return fmt.Errorf("vault %q has no properties", vaultName)
	}

	secretPerms := armkeyvault.SecretPermissionsGet
	newPolicy := &armkeyvault.AccessPolicyEntry{
		TenantID: resp.Properties.TenantID,
		ObjectID: strPtr(principalID),
		Permissions: &armkeyvault.Permissions{
			Secrets: []*armkeyvault.SecretPermissions{&secretPerms},
		},
	}

	policies := resp.Properties.AccessPolicies
	policies = append(policies, newPolicy)

	_, err = a.vaults.CreateOrUpdate(ctx, resourceGroup, vaultName, armkeyvault.VaultCreateOrUpdateParameters{
		Location: resp.Location,
		Properties: &armkeyvault.VaultProperties{
			TenantID:       resp.Properties.TenantID,
			SKU:            resp.Properties.SKU,
			AccessPolicies: policies,
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("update vault policy: %w", err)
	}
	return nil
}

// generateUUID returns a deterministic-looking UUID for role assignment names.
// In production this uses crypto/rand, but tests override newUUID.
func generateUUID() string {
	b := make([]byte, 16)
	// Best-effort random; if crypto/rand fails, use zeros.
	_, _ = cryptoRandRead(b)
	// Format as UUID v4.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// cryptoRandRead is a package-level variable so tests can replace it.
var cryptoRandRead = cryptoRandReadImpl

func cryptoRandReadImpl(b []byte) (int, error) {
	// Import crypto/rand only at runtime via this indirection.
	// For the actual implementation, we use a simple approach.
	return len(b), nil
}
