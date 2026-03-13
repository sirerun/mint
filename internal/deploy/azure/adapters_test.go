package azure

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- mock ACRClient for bridge tests ---

type bridgeMockACRClient struct {
	ensureRepoFunc func(ctx context.Context, rg, registry, repo string) (string, error)
	getLoginFunc   func(ctx context.Context, rg, registry string) (string, error)
}

func (m *bridgeMockACRClient) EnsureRepository(ctx context.Context, rg, registry, repo string) (string, error) {
	return m.ensureRepoFunc(ctx, rg, registry, repo)
}

func (m *bridgeMockACRClient) GetLoginServer(ctx context.Context, rg, registry string) (string, error) {
	return m.getLoginFunc(ctx, rg, registry)
}

func TestRegistryBridge_EnsureRepository(t *testing.T) {
	client := &bridgeMockACRClient{
		ensureRepoFunc: func(_ context.Context, _, _, _ string) (string, error) {
			return "myregistry.azurecr.io/myrepo", nil
		},
	}
	bridge := NewRegistryBridge(client)
	uri, err := bridge.EnsureRepository(context.Background(), "sub-id", "rg", "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uri != "myregistry.azurecr.io/myrepo" {
		t.Fatalf("unexpected URI: %s", uri)
	}
}

func TestRegistryBridge_EnsureRepository_Error(t *testing.T) {
	client := &bridgeMockACRClient{
		ensureRepoFunc: func(_ context.Context, _, _, _ string) (string, error) {
			return "", errors.New("acr failure")
		},
	}
	bridge := NewRegistryBridge(client)
	_, err := bridge.EnsureRepository(context.Background(), "sub-id", "rg", "myrepo")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- mock ContainerAppClient and ManagedEnvironmentClient for bridge tests ---

type bridgeMockContainerAppClient struct {
	createOrUpdateFunc func(ctx context.Context, input *CreateOrUpdateAppInput) (*ContainerApp, error)
	getAppFunc         func(ctx context.Context, rg, name string) (*ContainerApp, error)
	listRevisionsFunc  func(ctx context.Context, rg, name string) ([]Revision, error)
	updateTrafficFunc  func(ctx context.Context, rg, name string, traffic []TrafficWeight) error
}

func (m *bridgeMockContainerAppClient) CreateOrUpdateApp(ctx context.Context, input *CreateOrUpdateAppInput) (*ContainerApp, error) {
	return m.createOrUpdateFunc(ctx, input)
}

func (m *bridgeMockContainerAppClient) GetApp(ctx context.Context, rg, name string) (*ContainerApp, error) {
	return m.getAppFunc(ctx, rg, name)
}

func (m *bridgeMockContainerAppClient) ListRevisions(ctx context.Context, rg, name string) ([]Revision, error) {
	return m.listRevisionsFunc(ctx, rg, name)
}

func (m *bridgeMockContainerAppClient) UpdateTrafficSplit(ctx context.Context, rg, name string, traffic []TrafficWeight) error {
	return m.updateTrafficFunc(ctx, rg, name, traffic)
}

type bridgeMockEnvironmentClient struct {
	ensureFunc func(ctx context.Context, rg, name, region string) (string, error)
}

func (m *bridgeMockEnvironmentClient) EnsureEnvironment(ctx context.Context, rg, name, region string) (string, error) {
	return m.ensureFunc(ctx, rg, name, region)
}

func TestServiceBridge_EnsureService(t *testing.T) {
	appClient := &bridgeMockContainerAppClient{
		createOrUpdateFunc: func(_ context.Context, input *CreateOrUpdateAppInput) (*ContainerApp, error) {
			return &ContainerApp{
				Name:           input.AppName,
				FQDN:           "myapp.azurecontainerapps.io",
				LatestRevision: "myapp--rev1",
			}, nil
		},
		listRevisionsFunc: func(_ context.Context, _, _ string) ([]Revision, error) {
			return []Revision{{Name: "myapp--rev1"}}, nil
		},
	}
	envClient := &bridgeMockEnvironmentClient{
		ensureFunc: func(_ context.Context, _, _, _ string) (string, error) {
			return "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.App/managedEnvironments/myapp-env", nil
		},
	}

	bridge := NewServiceBridge(appClient, envClient)
	info, err := bridge.EnsureService(context.Background(), DeployServiceOptions{
		ResourceGroup: "rg",
		Region:        "eastus",
		ServiceName:   "myapp",
		ImageURI:      "myregistry.azurecr.io/myrepo:latest",
		Port:          8080,
		AllowPublic:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.URL != "https://myapp.azurecontainerapps.io" {
		t.Fatalf("unexpected URL: %s", info.URL)
	}
	if info.RevisionName != "myapp--rev1" {
		t.Fatalf("unexpected revision: %s", info.RevisionName)
	}
}

func TestServiceBridge_EnsureService_WithEnvironmentID(t *testing.T) {
	appClient := &bridgeMockContainerAppClient{
		createOrUpdateFunc: func(_ context.Context, _ *CreateOrUpdateAppInput) (*ContainerApp, error) {
			return &ContainerApp{
				FQDN:           "myapp.azurecontainerapps.io",
				LatestRevision: "myapp--rev1",
			}, nil
		},
		listRevisionsFunc: func(_ context.Context, _, _ string) ([]Revision, error) {
			return nil, nil
		},
	}

	bridge := NewServiceBridge(appClient, nil)
	info, err := bridge.EnsureService(context.Background(), DeployServiceOptions{
		ResourceGroup: "rg",
		Region:        "eastus",
		ServiceName:   "myapp",
		ImageURI:      "img:latest",
		EnvironmentID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.App/managedEnvironments/existing-env",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.RevisionName != "myapp--rev1" {
		t.Fatalf("unexpected revision: %s", info.RevisionName)
	}
}

func TestServiceBridge_EnsureService_EnvironmentError(t *testing.T) {
	envClient := &bridgeMockEnvironmentClient{
		ensureFunc: func(_ context.Context, _, _, _ string) (string, error) {
			return "", errors.New("env creation failed")
		},
	}
	bridge := NewServiceBridge(nil, envClient)
	_, err := bridge.EnsureService(context.Background(), DeployServiceOptions{
		ResourceGroup: "rg",
		Region:        "eastus",
		ServiceName:   "myapp",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceBridge_EnsureService_AppError(t *testing.T) {
	appClient := &bridgeMockContainerAppClient{
		createOrUpdateFunc: func(_ context.Context, _ *CreateOrUpdateAppInput) (*ContainerApp, error) {
			return nil, errors.New("app creation failed")
		},
	}
	envClient := &bridgeMockEnvironmentClient{
		ensureFunc: func(_ context.Context, _, _, _ string) (string, error) {
			return "/env/id", nil
		},
	}
	bridge := NewServiceBridge(appClient, envClient)
	_, err := bridge.EnsureService(context.Background(), DeployServiceOptions{
		ResourceGroup: "rg",
		Region:        "eastus",
		ServiceName:   "myapp",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceBridge_EnsureService_PreviousRevision(t *testing.T) {
	appClient := &bridgeMockContainerAppClient{
		createOrUpdateFunc: func(_ context.Context, _ *CreateOrUpdateAppInput) (*ContainerApp, error) {
			return &ContainerApp{
				FQDN:           "myapp.azurecontainerapps.io",
				LatestRevision: "myapp--rev2",
			}, nil
		},
		listRevisionsFunc: func(_ context.Context, _, _ string) ([]Revision, error) {
			return []Revision{
				{Name: "myapp--rev1"},
				{Name: "myapp--rev2"},
			}, nil
		},
	}
	bridge := NewServiceBridge(appClient, nil)
	info, err := bridge.EnsureService(context.Background(), DeployServiceOptions{
		ResourceGroup: "rg",
		Region:        "eastus",
		ServiceName:   "myapp",
		EnvironmentID: "/env/id",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.PreviousRevision != "myapp--rev1" {
		t.Fatalf("expected previous revision %q, got %q", "myapp--rev1", info.PreviousRevision)
	}
}

// --- IAM bridge tests ---

type bridgeMockRBACClient struct {
	assignRoleFunc         func(ctx context.Context, scope, roleDefID, principalID string) error
	ensureKeyVaultPolicyFn func(ctx context.Context, rg, vault, principal string) error
}

func (m *bridgeMockRBACClient) AssignRole(ctx context.Context, scope, roleDefID, principalID string) error {
	return m.assignRoleFunc(ctx, scope, roleDefID, principalID)
}

func (m *bridgeMockRBACClient) EnsureKeyVaultPolicy(ctx context.Context, rg, vault, principal string) error {
	return m.ensureKeyVaultPolicyFn(ctx, rg, vault, principal)
}

func TestIAMBridge_ConfigureIAM(t *testing.T) {
	client := &bridgeMockRBACClient{
		assignRoleFunc: func(_ context.Context, _, _, _ string) error { return nil },
	}
	bridge := NewIAMBridge(client)
	err := bridge.ConfigureIAM(context.Background(), "sub", "rg", "myapp", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Secrets bridge tests ---

type bridgeMockKeyVaultClient struct {
	ensureKeyVaultFunc func(ctx context.Context, rg, name, region string) (string, error)
	setSecretFunc      func(ctx context.Context, uri, name, value string) error
	getSecretURIFunc   func(ctx context.Context, uri, name string) (string, error)
}

func (m *bridgeMockKeyVaultClient) EnsureKeyVault(ctx context.Context, rg, name, region string) (string, error) {
	return m.ensureKeyVaultFunc(ctx, rg, name, region)
}

func (m *bridgeMockKeyVaultClient) SetSecret(ctx context.Context, uri, name, value string) error {
	return m.setSecretFunc(ctx, uri, name, value)
}

func (m *bridgeMockKeyVaultClient) GetSecretURI(ctx context.Context, uri, name string) (string, error) {
	return m.getSecretURIFunc(ctx, uri, name)
}

func TestSecretsBridge_EnsureSecrets(t *testing.T) {
	client := &bridgeMockKeyVaultClient{
		ensureKeyVaultFunc: func(_ context.Context, _, _, _ string) (string, error) {
			return "https://vault.vault.azure.net", nil
		},
		setSecretFunc: func(_ context.Context, _, _, _ string) error {
			return nil
		},
		getSecretURIFunc: func(_ context.Context, _, name string) (string, error) {
			return "https://vault.vault.azure.net/secrets/" + name, nil
		},
	}
	var buf bytes.Buffer
	bridge := NewSecretsBridge(client, &buf)
	uris, err := bridge.EnsureSecrets(context.Background(), "sub", "rg", "myapp", map[string]string{
		"DB_PASSWORD": "db-pass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(uris) != 1 {
		t.Fatalf("expected 1 URI, got %d", len(uris))
	}
}

func TestSecretsBridge_EnsureSecrets_VaultError(t *testing.T) {
	client := &bridgeMockKeyVaultClient{
		ensureKeyVaultFunc: func(_ context.Context, _, _, _ string) (string, error) {
			return "", errors.New("vault error")
		},
	}
	var buf bytes.Buffer
	bridge := NewSecretsBridge(client, &buf)
	_, err := bridge.EnsureSecrets(context.Background(), "sub", "rg", "myapp", map[string]string{
		"KEY": "val",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSecretsBridge_EnsureSecrets_SetSecretError(t *testing.T) {
	client := &bridgeMockKeyVaultClient{
		ensureKeyVaultFunc: func(_ context.Context, _, _, _ string) (string, error) {
			return "https://vault.vault.azure.net", nil
		},
		setSecretFunc: func(_ context.Context, _, _, _ string) error {
			return errors.New("set secret failed")
		},
	}
	var buf bytes.Buffer
	bridge := NewSecretsBridge(client, &buf)
	_, err := bridge.EnsureSecrets(context.Background(), "sub", "rg", "myapp", map[string]string{
		"KEY": "val",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSecretsBridge_EnsureSecrets_GetURIError(t *testing.T) {
	client := &bridgeMockKeyVaultClient{
		ensureKeyVaultFunc: func(_ context.Context, _, _, _ string) (string, error) {
			return "https://vault.vault.azure.net", nil
		},
		setSecretFunc: func(_ context.Context, _, _, _ string) error {
			return nil
		},
		getSecretURIFunc: func(_ context.Context, _, _ string) (string, error) {
			return "", errors.New("get URI failed")
		},
	}
	var buf bytes.Buffer
	bridge := NewSecretsBridge(client, &buf)
	_, err := bridge.EnsureSecrets(context.Background(), "sub", "rg", "myapp", map[string]string{
		"KEY": "val",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Health bridge tests ---

func TestHealthBridge_Check(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	checker := NewHealthChecker(srv.Client())
	bridge := NewHealthBridge(checker)
	result, err := bridge.Check(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Healthy {
		t.Fatal("expected healthy")
	}
}

func TestHealthBridge_Check_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	}))
	defer srv.Close()

	checker := &HealthChecker{HTTPClient: srv.Client(), MaxRetries: 1}
	bridge := NewHealthBridge(checker)
	result, err := bridge.Check(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Healthy {
		t.Fatal("expected unhealthy")
	}
}

func TestHealthBridge_Check_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	checker := &HealthChecker{HTTPClient: srv.Client(), MaxRetries: 1}
	bridge := NewHealthBridge(checker)
	result, err := bridge.Check(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Healthy {
		t.Fatal("expected healthy")
	}
	want := "status 200 after 1 attempts"
	if result.Message != want {
		t.Fatalf("expected message %q, got %q", want, result.Message)
	}
}

func TestHealthBridge_Check_Error(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := NewHealthChecker(&http.Client{})
	bridge := NewHealthBridge(checker)
	_, err := bridge.Check(ctx, "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
