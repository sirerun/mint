package azure

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
)

type stubRBACAPI struct {
	createFunc func(ctx context.Context, scope, name string, params armauthorization.RoleAssignmentCreateParameters, opts *armauthorization.RoleAssignmentsClientCreateOptions) (armauthorization.RoleAssignmentsClientCreateResponse, error)
}

func (s *stubRBACAPI) Create(ctx context.Context, scope, name string, params armauthorization.RoleAssignmentCreateParameters, opts *armauthorization.RoleAssignmentsClientCreateOptions) (armauthorization.RoleAssignmentsClientCreateResponse, error) {
	return s.createFunc(ctx, scope, name, params, opts)
}

type stubVaultAccessAPI struct {
	getFunc            func(ctx context.Context, rg, name string, opts *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error)
	createOrUpdateFunc func(ctx context.Context, rg, name string, params armkeyvault.VaultCreateOrUpdateParameters, opts *armkeyvault.VaultsClientBeginCreateOrUpdateOptions) (*armkeyvault.VaultsClientCreateOrUpdateResponse, error)
}

func (s *stubVaultAccessAPI) Get(ctx context.Context, rg, name string, opts *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
	return s.getFunc(ctx, rg, name, opts)
}

func (s *stubVaultAccessAPI) CreateOrUpdate(ctx context.Context, rg, name string, params armkeyvault.VaultCreateOrUpdateParameters, opts *armkeyvault.VaultsClientBeginCreateOrUpdateOptions) (*armkeyvault.VaultsClientCreateOrUpdateResponse, error) {
	return s.createOrUpdateFunc(ctx, rg, name, params, opts)
}

func TestRBACAdapter_InterfaceCompliance(t *testing.T) {
	var _ RBACClient = (*RBACAdapter)(nil)
}

func TestRBACAdapter_AssignRole(t *testing.T) {
	orig := newUUID
	newUUID = func() string { return "test-uuid-1234" }
	defer func() { newUUID = orig }()

	stub := &stubRBACAPI{
		createFunc: func(_ context.Context, scope, name string, params armauthorization.RoleAssignmentCreateParameters, _ *armauthorization.RoleAssignmentsClientCreateOptions) (armauthorization.RoleAssignmentsClientCreateResponse, error) {
			if scope != "/subscriptions/sub/resourceGroups/rg" {
				t.Fatalf("unexpected scope: %s", scope)
			}
			if name != "test-uuid-1234" {
				t.Fatalf("unexpected name: %s", name)
			}
			if *params.Properties.RoleDefinitionID != "role-def-id" {
				t.Fatalf("unexpected role definition ID: %s", *params.Properties.RoleDefinitionID)
			}
			if *params.Properties.PrincipalID != "principal-id" {
				t.Fatalf("unexpected principal ID: %s", *params.Properties.PrincipalID)
			}
			return armauthorization.RoleAssignmentsClientCreateResponse{}, nil
		},
	}
	adapter := &RBACAdapter{rbac: stub}
	err := adapter.AssignRole(context.Background(), "/subscriptions/sub/resourceGroups/rg", "role-def-id", "principal-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRBACAdapter_AssignRole_Error(t *testing.T) {
	orig := newUUID
	newUUID = func() string { return "test-uuid" }
	defer func() { newUUID = orig }()

	stub := &stubRBACAPI{
		createFunc: func(_ context.Context, _, _ string, _ armauthorization.RoleAssignmentCreateParameters, _ *armauthorization.RoleAssignmentsClientCreateOptions) (armauthorization.RoleAssignmentsClientCreateResponse, error) {
			return armauthorization.RoleAssignmentsClientCreateResponse{}, errors.New("forbidden")
		},
	}
	adapter := &RBACAdapter{rbac: stub}
	err := adapter.AssignRole(context.Background(), "/scope", "role", "principal")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRBACAdapter_EnsureKeyVaultPolicy(t *testing.T) {
	tenantID := "tenant-123"
	location := "eastus"
	family := armkeyvault.SKUFamilyA
	skuName := armkeyvault.SKUNameStandard

	vaultStub := &stubVaultAccessAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
			return armkeyvault.VaultsClientGetResponse{
				Vault: armkeyvault.Vault{
					Location: &location,
					Properties: &armkeyvault.VaultProperties{
						TenantID:       &tenantID,
						SKU:            &armkeyvault.SKU{Family: &family, Name: &skuName},
						AccessPolicies: []*armkeyvault.AccessPolicyEntry{},
					},
				},
			}, nil
		},
		createOrUpdateFunc: func(_ context.Context, _, _ string, params armkeyvault.VaultCreateOrUpdateParameters, _ *armkeyvault.VaultsClientBeginCreateOrUpdateOptions) (*armkeyvault.VaultsClientCreateOrUpdateResponse, error) {
			if len(params.Properties.AccessPolicies) != 1 {
				t.Fatalf("expected 1 access policy, got %d", len(params.Properties.AccessPolicies))
			}
			if *params.Properties.AccessPolicies[0].ObjectID != "principal-123" {
				t.Fatalf("unexpected object ID: %s", *params.Properties.AccessPolicies[0].ObjectID)
			}
			return &armkeyvault.VaultsClientCreateOrUpdateResponse{}, nil
		},
	}
	adapter := &RBACAdapter{vaults: vaultStub}
	err := adapter.EnsureKeyVaultPolicy(context.Background(), "rg", "myvault", "principal-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRBACAdapter_EnsureKeyVaultPolicy_GetError(t *testing.T) {
	vaultStub := &stubVaultAccessAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
			return armkeyvault.VaultsClientGetResponse{}, errors.New("not found")
		},
	}
	adapter := &RBACAdapter{vaults: vaultStub}
	err := adapter.EnsureKeyVaultPolicy(context.Background(), "rg", "myvault", "principal-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRBACAdapter_EnsureKeyVaultPolicy_NilProperties(t *testing.T) {
	vaultStub := &stubVaultAccessAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
			return armkeyvault.VaultsClientGetResponse{
				Vault: armkeyvault.Vault{Properties: nil},
			}, nil
		},
	}
	adapter := &RBACAdapter{vaults: vaultStub}
	err := adapter.EnsureKeyVaultPolicy(context.Background(), "rg", "myvault", "principal-123")
	if err == nil {
		t.Fatal("expected error for nil properties, got nil")
	}
}

func TestRBACAdapter_EnsureKeyVaultPolicy_UpdateError(t *testing.T) {
	tenantID := "tenant-123"
	family := armkeyvault.SKUFamilyA
	skuName := armkeyvault.SKUNameStandard

	vaultStub := &stubVaultAccessAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
			return armkeyvault.VaultsClientGetResponse{
				Vault: armkeyvault.Vault{
					Properties: &armkeyvault.VaultProperties{
						TenantID:       &tenantID,
						SKU:            &armkeyvault.SKU{Family: &family, Name: &skuName},
						AccessPolicies: []*armkeyvault.AccessPolicyEntry{},
					},
				},
			}, nil
		},
		createOrUpdateFunc: func(_ context.Context, _, _ string, _ armkeyvault.VaultCreateOrUpdateParameters, _ *armkeyvault.VaultsClientBeginCreateOrUpdateOptions) (*armkeyvault.VaultsClientCreateOrUpdateResponse, error) {
			return nil, errors.New("update failed")
		},
	}
	adapter := &RBACAdapter{vaults: vaultStub}
	err := adapter.EnsureKeyVaultPolicy(context.Background(), "rg", "myvault", "principal-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGenerateUUID(t *testing.T) {
	uuid := generateUUID()
	if len(uuid) != 36 {
		t.Fatalf("expected UUID length 36, got %d: %s", len(uuid), uuid)
	}
	// Check dashes at correct positions.
	if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		t.Fatalf("UUID format incorrect: %s", uuid)
	}
}
