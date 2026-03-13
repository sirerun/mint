package aws

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockOIDCClient implements OIDCClient for testing.
type mockOIDCClient struct {
	getProviderFunc    func(ctx context.Context, arn string) error
	createProviderFunc func(ctx context.Context, url string, thumbprints []string) (string, error)
	createRoleFunc     func(ctx context.Context, input *CreateRoleInput) (*Role, error)
	getRoleFunc        func(ctx context.Context, roleName string) (*Role, error)
	attachPolicyFunc   func(ctx context.Context, roleName, policyARN string) error
}

func (m *mockOIDCClient) GetOpenIDConnectProvider(ctx context.Context, arn string) error {
	return m.getProviderFunc(ctx, arn)
}

func (m *mockOIDCClient) CreateOpenIDConnectProvider(ctx context.Context, url string, thumbprints []string) (string, error) {
	return m.createProviderFunc(ctx, url, thumbprints)
}

func (m *mockOIDCClient) CreateRole(ctx context.Context, input *CreateRoleInput) (*Role, error) {
	return m.createRoleFunc(ctx, input)
}

func (m *mockOIDCClient) GetRole(ctx context.Context, roleName string) (*Role, error) {
	return m.getRoleFunc(ctx, roleName)
}

func (m *mockOIDCClient) AttachRolePolicy(ctx context.Context, roleName, policyARN string) error {
	return m.attachPolicyFunc(ctx, roleName, policyARN)
}

func baseOIDCConfig() OIDCConfig {
	return OIDCConfig{
		AccountID: "123456789012",
		Region:    "us-east-1",
		RepoOwner: "sirerun",
		RepoName:  "mint",
	}
}

func defaultMockOIDCClient() *mockOIDCClient {
	return &mockOIDCClient{
		getProviderFunc: func(context.Context, string) error {
			return nil // provider exists
		},
		createProviderFunc: func(context.Context, string, []string) (string, error) {
			return "", nil
		},
		getRoleFunc: func(_ context.Context, roleName string) (*Role, error) {
			return nil, ErrRoleNotFound
		},
		createRoleFunc: func(_ context.Context, input *CreateRoleInput) (*Role, error) {
			return &Role{
				ARN:      "arn:aws:iam::123456789012:role/" + input.RoleName,
				RoleName: input.RoleName,
			}, nil
		},
		attachPolicyFunc: func(context.Context, string, string) error {
			return nil
		},
	}
}

func TestEnsureOIDCProvider_ProviderExists_SkipsCreation(t *testing.T) {
	var createCalled bool
	client := defaultMockOIDCClient()
	client.getProviderFunc = func(context.Context, string) error {
		return nil // provider exists
	}
	client.createProviderFunc = func(context.Context, string, []string) (string, error) {
		createCalled = true
		return "", nil
	}

	var stderr bytes.Buffer
	result, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createCalled {
		t.Error("CreateOpenIDConnectProvider should not be called when provider exists")
	}
	if result.ProviderARN == "" {
		t.Error("expected non-empty provider ARN")
	}
	if !strings.Contains(result.ProviderARN, "123456789012") {
		t.Errorf("provider ARN should contain account ID, got: %s", result.ProviderARN)
	}
}

func TestEnsureOIDCProvider_ProviderNotFound_Creates(t *testing.T) {
	expectedARN := "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com"
	var createCalled bool
	client := defaultMockOIDCClient()
	client.getProviderFunc = func(context.Context, string) error {
		return ErrOIDCProviderNotFound
	}
	client.createProviderFunc = func(_ context.Context, url string, thumbprints []string) (string, error) {
		createCalled = true
		if url != githubOIDCURL {
			t.Errorf("got URL %q, want %q", url, githubOIDCURL)
		}
		if len(thumbprints) != 1 || thumbprints[0] != githubOIDCThumbprint {
			t.Errorf("unexpected thumbprints: %v", thumbprints)
		}
		return expectedARN, nil
	}

	var stderr bytes.Buffer
	result, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !createCalled {
		t.Fatal("expected CreateOpenIDConnectProvider to be called")
	}
	if result.ProviderARN != expectedARN {
		t.Errorf("got provider ARN %q, want %q", result.ProviderARN, expectedARN)
	}
	if !strings.Contains(stderr.String(), "Created OIDC provider") {
		t.Error("expected creation message in stderr")
	}
}

func TestEnsureOIDCProvider_RoleExists_SkipsCreation(t *testing.T) {
	existingRole := &Role{
		ARN:      "arn:aws:iam::123456789012:role/mint-github-deploy-mint",
		RoleName: "mint-github-deploy-mint",
	}
	var createRoleCalled bool
	client := defaultMockOIDCClient()
	client.getRoleFunc = func(_ context.Context, roleName string) (*Role, error) {
		return existingRole, nil
	}
	client.createRoleFunc = func(context.Context, *CreateRoleInput) (*Role, error) {
		createRoleCalled = true
		return nil, nil
	}

	var stderr bytes.Buffer
	result, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createRoleCalled {
		t.Error("CreateRole should not be called when role exists")
	}
	if result.RoleARN != existingRole.ARN {
		t.Errorf("got role ARN %q, want %q", result.RoleARN, existingRole.ARN)
	}
}

func TestEnsureOIDCProvider_RoleNotFound_CreatesWithTrustPolicy(t *testing.T) {
	var capturedInput *CreateRoleInput
	client := defaultMockOIDCClient()
	client.getRoleFunc = func(context.Context, string) (*Role, error) {
		return nil, ErrRoleNotFound
	}
	client.createRoleFunc = func(_ context.Context, input *CreateRoleInput) (*Role, error) {
		capturedInput = input
		return &Role{
			ARN:      "arn:aws:iam::123456789012:role/" + input.RoleName,
			RoleName: input.RoleName,
		}, nil
	}

	var stderr bytes.Buffer
	result, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedInput == nil {
		t.Fatal("expected CreateRole to be called")
	}
	if capturedInput.RoleName != "mint-github-deploy-mint" {
		t.Errorf("got role name %q, want %q", capturedInput.RoleName, "mint-github-deploy-mint")
	}

	// Verify trust policy contains required fields.
	policy := capturedInput.AssumeRolePolicyDocument
	checks := []string{
		"sts:AssumeRoleWithWebIdentity",
		"token.actions.githubusercontent.com:aud",
		"sts.amazonaws.com",
		"token.actions.githubusercontent.com:sub",
		"repo:sirerun/mint:*",
	}
	for _, want := range checks {
		if !strings.Contains(policy, want) {
			t.Errorf("trust policy missing %q", want)
		}
	}

	if result.RoleName != "mint-github-deploy-mint" {
		t.Errorf("got role name %q, want %q", result.RoleName, "mint-github-deploy-mint")
	}
	if !strings.Contains(stderr.String(), "Created IAM role") {
		t.Error("expected role creation message in stderr")
	}
}

func TestEnsureOIDCProvider_GetProviderError(t *testing.T) {
	client := defaultMockOIDCClient()
	client.getProviderFunc = func(context.Context, string) error {
		return fmt.Errorf("access denied")
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

func TestEnsureOIDCProvider_CreateProviderError(t *testing.T) {
	client := defaultMockOIDCClient()
	client.getProviderFunc = func(context.Context, string) error {
		return ErrOIDCProviderNotFound
	}
	client.createProviderFunc = func(context.Context, string, []string) (string, error) {
		return "", fmt.Errorf("quota exceeded")
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

func TestEnsureOIDCProvider_CreateRoleError(t *testing.T) {
	client := defaultMockOIDCClient()
	client.getRoleFunc = func(context.Context, string) (*Role, error) {
		return nil, ErrRoleNotFound
	}
	client.createRoleFunc = func(context.Context, *CreateRoleInput) (*Role, error) {
		return nil, fmt.Errorf("limit exceeded")
	}

	var stderr bytes.Buffer
	_, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "limit exceeded") {
		t.Errorf("error should contain 'limit exceeded', got: %v", err)
	}
}

func TestEnsureOIDCProvider_AttachPolicyError(t *testing.T) {
	client := defaultMockOIDCClient()
	client.attachPolicyFunc = func(_ context.Context, _, policyARN string) error {
		return fmt.Errorf("policy not found: %s", policyARN)
	}

	var stderr bytes.Buffer
	_, err := EnsureOIDCProvider(context.Background(), client, baseOIDCConfig(), &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "policy not found") {
		t.Errorf("error should contain 'policy not found', got: %v", err)
	}
}

func TestEnsureOIDCProvider_MissingConfig(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*OIDCConfig)
		want   string
	}{
		{
			name:   "missing AccountID",
			modify: func(c *OIDCConfig) { c.AccountID = "" },
			want:   "accountID",
		},
		{
			name:   "missing Region",
			modify: func(c *OIDCConfig) { c.Region = "" },
			want:   "region",
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
