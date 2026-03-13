package azure

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockOIDCClient implements OIDCClient for testing.
type mockOIDCClient struct {
	getAppFunc     func(ctx context.Context, displayName string) (*AppRegistration, error)
	createAppFunc  func(ctx context.Context, displayName string) (*AppRegistration, error)
	createCredFunc func(ctx context.Context, appID string, input *FederatedCredentialInput) error
	assignRoleFunc func(ctx context.Context, input *RoleAssignmentInput) error
}

func (m *mockOIDCClient) GetAppRegistration(ctx context.Context, displayName string) (*AppRegistration, error) {
	return m.getAppFunc(ctx, displayName)
}

func (m *mockOIDCClient) CreateAppRegistration(ctx context.Context, displayName string) (*AppRegistration, error) {
	return m.createAppFunc(ctx, displayName)
}

func (m *mockOIDCClient) CreateFederatedCredential(ctx context.Context, appID string, input *FederatedCredentialInput) error {
	return m.createCredFunc(ctx, appID, input)
}

func (m *mockOIDCClient) AssignRole(ctx context.Context, input *RoleAssignmentInput) error {
	return m.assignRoleFunc(ctx, input)
}

func baseOIDCConfig() OIDCConfig {
	return OIDCConfig{
		SubscriptionID: "00000000-0000-0000-0000-000000000000",
		TenantID:       "11111111-1111-1111-1111-111111111111",
		RepoOwner:      "sirerun",
		RepoName:       "mint",
	}
}

func defaultMockOIDCClient() *mockOIDCClient {
	return &mockOIDCClient{
		getAppFunc: func(_ context.Context, _ string) (*AppRegistration, error) {
			return nil, ErrADAppNotFound
		},
		createAppFunc: func(_ context.Context, displayName string) (*AppRegistration, error) {
			return &AppRegistration{
				AppID:    "app-id-12345",
				ObjectID: "obj-id-12345",
			}, nil
		},
		createCredFunc: func(_ context.Context, _ string, _ *FederatedCredentialInput) error {
			return nil
		},
		assignRoleFunc: func(_ context.Context, _ *RoleAssignmentInput) error {
			return nil
		},
	}
}

func TestEnsureOIDCProvider_AppExists_SkipsCreation(t *testing.T) {
	existingApp := &AppRegistration{
		AppID:    "existing-app-id",
		ObjectID: "existing-obj-id",
	}
	var createCalled bool
	client := defaultMockOIDCClient()
	client.getAppFunc = func(_ context.Context, _ string) (*AppRegistration, error) {
		return existingApp, nil
	}
	client.createAppFunc = func(_ context.Context, _ string) (*AppRegistration, error) {
		createCalled = true
		return nil, nil
	}

	var stderr bytes.Buffer
	result, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createCalled {
		t.Error("CreateAppRegistration should not be called when app exists")
	}
	if result.ClientID != existingApp.AppID {
		t.Errorf("got client ID %q, want %q", result.ClientID, existingApp.AppID)
	}
}

func TestEnsureOIDCProvider_AppNotFound_Creates(t *testing.T) {
	var createCalled bool
	client := defaultMockOIDCClient()
	client.getAppFunc = func(_ context.Context, _ string) (*AppRegistration, error) {
		return nil, ErrADAppNotFound
	}
	client.createAppFunc = func(_ context.Context, displayName string) (*AppRegistration, error) {
		createCalled = true
		if !strings.Contains(displayName, "mint") {
			t.Errorf("display name should contain repo name, got %q", displayName)
		}
		return &AppRegistration{
			AppID:    "new-app-id",
			ObjectID: "new-obj-id",
		}, nil
	}

	var stderr bytes.Buffer
	result, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !createCalled {
		t.Fatal("expected CreateAppRegistration to be called")
	}
	if result.ClientID != "new-app-id" {
		t.Errorf("got client ID %q, want %q", result.ClientID, "new-app-id")
	}
	if !strings.Contains(stderr.String(), "Created Azure AD app registration") {
		t.Error("expected creation message in stderr")
	}
}

func TestEnsureOIDCProvider_FederatedCredentialParams(t *testing.T) {
	var capturedInput *FederatedCredentialInput
	var capturedAppID string
	client := defaultMockOIDCClient()
	client.createCredFunc = func(_ context.Context, appID string, input *FederatedCredentialInput) error {
		capturedAppID = appID
		capturedInput = input
		return nil
	}

	var stderr bytes.Buffer
	_, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedAppID != "app-id-12345" {
		t.Errorf("got app ID %q, want %q", capturedAppID, "app-id-12345")
	}
	if capturedInput == nil {
		t.Fatal("expected federated credential input to be captured")
	}
	if capturedInput.Issuer != githubOIDCIssuer {
		t.Errorf("got issuer %q, want %q", capturedInput.Issuer, githubOIDCIssuer)
	}
	if !strings.Contains(capturedInput.Subject, "repo:sirerun/mint:") {
		t.Errorf("subject should contain repo reference, got %q", capturedInput.Subject)
	}
	if len(capturedInput.Audiences) != 1 || capturedInput.Audiences[0] != "api://AzureADTokenExchange" {
		t.Errorf("unexpected audiences: %v", capturedInput.Audiences)
	}
}

func TestEnsureOIDCProvider_RoleAssignment(t *testing.T) {
	var capturedRole *RoleAssignmentInput
	client := defaultMockOIDCClient()
	client.assignRoleFunc = func(_ context.Context, input *RoleAssignmentInput) error {
		capturedRole = input
		return nil
	}

	config := baseOIDCConfig()
	var stderr bytes.Buffer
	_, err := EnsureOIDCProvider(context.Background(), client, config, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedRole == nil {
		t.Fatal("expected role assignment to be captured")
	}
	if capturedRole.RoleName != "Contributor" {
		t.Errorf("got role %q, want %q", capturedRole.RoleName, "Contributor")
	}
	if capturedRole.PrincipalID != "obj-id-12345" {
		t.Errorf("got principal ID %q, want %q", capturedRole.PrincipalID, "obj-id-12345")
	}
	if !strings.Contains(capturedRole.Scope, config.SubscriptionID) {
		t.Errorf("scope should contain subscription ID, got %q", capturedRole.Scope)
	}
}

func TestEnsureOIDCProvider_GetAppError(t *testing.T) {
	client := defaultMockOIDCClient()
	client.getAppFunc = func(_ context.Context, _ string) (*AppRegistration, error) {
		return nil, fmt.Errorf("access denied")
	}

	var stderr bytes.Buffer
	_, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Errorf("error should contain 'access denied', got: %v", err)
	}
}

func TestEnsureOIDCProvider_CreateAppError(t *testing.T) {
	client := defaultMockOIDCClient()
	client.getAppFunc = func(_ context.Context, _ string) (*AppRegistration, error) {
		return nil, ErrADAppNotFound
	}
	client.createAppFunc = func(_ context.Context, _ string) (*AppRegistration, error) {
		return nil, fmt.Errorf("quota exceeded")
	}

	var stderr bytes.Buffer
	_, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "quota exceeded") {
		t.Errorf("error should contain 'quota exceeded', got: %v", err)
	}
}

func TestEnsureOIDCProvider_CreateFederatedCredentialError(t *testing.T) {
	client := defaultMockOIDCClient()
	client.createCredFunc = func(_ context.Context, _ string, _ *FederatedCredentialInput) error {
		return fmt.Errorf("credential limit reached")
	}

	var stderr bytes.Buffer
	_, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "credential limit reached") {
		t.Errorf("error should contain 'credential limit reached', got: %v", err)
	}
}

func TestEnsureOIDCProvider_AssignRoleError(t *testing.T) {
	client := defaultMockOIDCClient()
	client.assignRoleFunc = func(_ context.Context, _ *RoleAssignmentInput) error {
		return fmt.Errorf("authorization failed")
	}

	var stderr bytes.Buffer
	_, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "authorization failed") {
		t.Errorf("error should contain 'authorization failed', got: %v", err)
	}
}

func TestEnsureOIDCProvider_ResultFields(t *testing.T) {
	client := defaultMockOIDCClient()
	config := baseOIDCConfig()

	var stderr bytes.Buffer
	result, err := EnsureOIDCProvider(context.Background(), client, config, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ClientID != "app-id-12345" {
		t.Errorf("got client ID %q, want %q", result.ClientID, "app-id-12345")
	}
	if result.TenantID != config.TenantID {
		t.Errorf("got tenant ID %q, want %q", result.TenantID, config.TenantID)
	}
	if result.SubscriptionID != config.SubscriptionID {
		t.Errorf("got subscription ID %q, want %q", result.SubscriptionID, config.SubscriptionID)
	}
}

func TestPrintOIDCInstructions(t *testing.T) {
	result := &OIDCResult{
		ClientID:       "test-client-id",
		TenantID:       "test-tenant-id",
		SubscriptionID: "test-subscription-id",
	}

	var buf bytes.Buffer
	PrintOIDCInstructions(&buf, result)
	output := buf.String()

	checks := []struct {
		label string
		want  string
	}{
		{"client ID", result.ClientID},
		{"tenant ID", result.TenantID},
		{"subscription ID", result.SubscriptionID},
		{"AZURE_CLIENT_ID", "AZURE_CLIENT_ID"},
		{"AZURE_TENANT_ID", "AZURE_TENANT_ID"},
		{"AZURE_SUBSCRIPTION_ID", "AZURE_SUBSCRIPTION_ID"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.want) {
			t.Errorf("output missing %s (%q)", c.label, c.want)
		}
	}
}

func TestEnsureOIDCProvider_MissingConfig(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*OIDCConfig)
		want   string
	}{
		{
			name:   "missing SubscriptionID",
			modify: func(c *OIDCConfig) { c.SubscriptionID = "" },
			want:   "subscriptionID",
		},
		{
			name:   "missing TenantID",
			modify: func(c *OIDCConfig) { c.TenantID = "" },
			want:   "tenantID",
		},
		{
			name:   "missing RepoOwner",
			modify: func(c *OIDCConfig) { c.RepoOwner = "" },
			want:   "repoOwner",
		},
		{
			name:   "missing RepoName",
			modify: func(c *OIDCConfig) { c.RepoName = "" },
			want:   "repoName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := baseOIDCConfig()
			tt.modify(&config)

			client := defaultMockOIDCClient()
			var stderr bytes.Buffer
			_, err := EnsureOIDCProvider(context.Background(), client, config, &stderr)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q should mention %q", err, tt.want)
			}
		})
	}
}
