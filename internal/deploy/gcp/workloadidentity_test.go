package gcp

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockIAMClient implements IAMClient for testing.
type mockIAMClient struct {
	getFunc    func(ctx context.Context, name string) (string, error)
	createFunc func(ctx context.Context, projectID, accountID, displayName string) (string, error)
}

func (m *mockIAMClient) GetServiceAccount(ctx context.Context, name string) (string, error) {
	return m.getFunc(ctx, name)
}

func (m *mockIAMClient) CreateServiceAccount(ctx context.Context, projectID, accountID, displayName string) (string, error) {
	return m.createFunc(ctx, projectID, accountID, displayName)
}

func baseConfig() WorkloadIdentityConfig {
	return WorkloadIdentityConfig{
		ProjectID:     "my-project",
		ProjectNumber: "123456789",
		GitHubOrg:     "sirerun",
		GitHubRepo:    "mint",
	}
}

func TestEnsureWorkloadIdentity_SAAlreadyExists(t *testing.T) {
	existingEmail := "mint-deploy@my-project.iam.gserviceaccount.com"
	client := &mockIAMClient{
		getFunc: func(_ context.Context, name string) (string, error) {
			if !strings.Contains(name, existingEmail) {
				t.Fatalf("unexpected resource name: %s", name)
			}
			return existingEmail, nil
		},
		createFunc: func(context.Context, string, string, string) (string, error) {
			t.Fatal("CreateServiceAccount should not be called when SA exists")
			return "", nil
		},
	}

	var stderr bytes.Buffer
	result, err := EnsureWorkloadIdentity(context.Background(), client, baseConfig(), &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ServiceAccount != existingEmail {
		t.Errorf("got SA %q, want %q", result.ServiceAccount, existingEmail)
	}
	if !strings.Contains(result.ProviderName, "mint-github-pool") {
		t.Errorf("provider name missing pool ID: %s", result.ProviderName)
	}
	if !strings.Contains(result.ProviderName, "mint-github-provider") {
		t.Errorf("provider name missing provider ID: %s", result.ProviderName)
	}
	if !strings.Contains(stderr.String(), "gcloud iam workload-identity-pools create") {
		t.Error("expected gcloud instructions in stderr output")
	}
}

func TestEnsureWorkloadIdentity_SACreated(t *testing.T) {
	createdEmail := "mint-deploy@my-project.iam.gserviceaccount.com"
	var createCalled bool
	client := &mockIAMClient{
		getFunc: func(context.Context, string) (string, error) {
			return "", nil // SA does not exist
		},
		createFunc: func(_ context.Context, projectID, accountID, displayName string) (string, error) {
			createCalled = true
			if projectID != "my-project" {
				t.Errorf("got projectID %q, want %q", projectID, "my-project")
			}
			if accountID != "mint-deploy" {
				t.Errorf("got accountID %q, want %q", accountID, "mint-deploy")
			}
			return createdEmail, nil
		},
	}

	var stderr bytes.Buffer
	result, err := EnsureWorkloadIdentity(context.Background(), client, baseConfig(), &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !createCalled {
		t.Fatal("expected CreateServiceAccount to be called")
	}
	if result.ServiceAccount != createdEmail {
		t.Errorf("got SA %q, want %q", result.ServiceAccount, createdEmail)
	}
}

func TestEnsureWorkloadIdentity_CreateFails(t *testing.T) {
	client := &mockIAMClient{
		getFunc: func(context.Context, string) (string, error) {
			return "", nil // SA does not exist
		},
		createFunc: func(context.Context, string, string, string) (string, error) {
			return "", fmt.Errorf("permission denied")
		},
	}

	var stderr bytes.Buffer
	_, err := EnsureWorkloadIdentity(context.Background(), client, baseConfig(), &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error should contain 'permission denied', got: %v", err)
	}
}

func TestEnsureWorkloadIdentity_MissingProjectID(t *testing.T) {
	config := baseConfig()
	config.ProjectID = ""
	client := &mockIAMClient{
		getFunc:    func(context.Context, string) (string, error) { return "", nil },
		createFunc: func(context.Context, string, string, string) (string, error) { return "", nil },
	}

	var stderr bytes.Buffer
	_, err := EnsureWorkloadIdentity(context.Background(), client, config, &stderr)
	if err == nil {
		t.Fatal("expected error for missing project ID")
	}
}

func TestEnsureWorkloadIdentity_MissingProjectNumber(t *testing.T) {
	config := baseConfig()
	config.ProjectNumber = ""
	client := &mockIAMClient{
		getFunc:    func(context.Context, string) (string, error) { return "", nil },
		createFunc: func(context.Context, string, string, string) (string, error) { return "", nil },
	}

	var stderr bytes.Buffer
	_, err := EnsureWorkloadIdentity(context.Background(), client, config, &stderr)
	if err == nil {
		t.Fatal("expected error for missing project number")
	}
}

func TestEnsureWorkloadIdentity_CustomPoolAndProvider(t *testing.T) {
	config := baseConfig()
	config.PoolID = "custom-pool"
	config.ProviderID = "custom-provider"

	client := &mockIAMClient{
		getFunc: func(context.Context, string) (string, error) {
			return "mint-deploy@my-project.iam.gserviceaccount.com", nil
		},
		createFunc: func(context.Context, string, string, string) (string, error) {
			return "", nil
		},
	}

	var stderr bytes.Buffer
	result, err := EnsureWorkloadIdentity(context.Background(), client, config, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.ProviderName, "custom-pool") {
		t.Errorf("provider name should contain custom-pool: %s", result.ProviderName)
	}
	if !strings.Contains(result.ProviderName, "custom-provider") {
		t.Errorf("provider name should contain custom-provider: %s", result.ProviderName)
	}
	if !strings.Contains(stderr.String(), "custom-pool") {
		t.Error("stderr instructions should reference custom pool")
	}
}

func TestEnsureWorkloadIdentity_GetFails(t *testing.T) {
	client := &mockIAMClient{
		getFunc: func(context.Context, string) (string, error) {
			return "", fmt.Errorf("network error")
		},
		createFunc: func(context.Context, string, string, string) (string, error) {
			t.Fatal("CreateServiceAccount should not be called when Get fails")
			return "", nil
		},
	}

	var stderr bytes.Buffer
	_, err := EnsureWorkloadIdentity(context.Background(), client, baseConfig(), &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "network error") {
		t.Errorf("error should contain 'network error', got: %v", err)
	}
}
