package aws

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- bridge test mocks (prefixed to avoid collision with other test files) ---

type bridgeMockECRClient struct {
	describeFunc func(ctx context.Context, input *DescribeRepositoriesInput) (*DescribeRepositoriesOutput, error)
	createFunc   func(ctx context.Context, input *CreateRepositoryInput) (*CreateRepositoryOutput, error)
}

func (m *bridgeMockECRClient) DescribeRepositories(ctx context.Context, input *DescribeRepositoriesInput) (*DescribeRepositoriesOutput, error) {
	return m.describeFunc(ctx, input)
}

func (m *bridgeMockECRClient) CreateRepository(ctx context.Context, input *CreateRepositoryInput) (*CreateRepositoryOutput, error) {
	return m.createFunc(ctx, input)
}

func TestRegistryBridge_EnsureRepository(t *testing.T) {
	client := &bridgeMockECRClient{
		describeFunc: func(_ context.Context, input *DescribeRepositoriesInput) (*DescribeRepositoriesOutput, error) {
			if len(input.RepositoryNames) == 1 && input.RepositoryNames[0] == "my-repo" {
				return &DescribeRepositoriesOutput{
					Repositories: []Repository{{RepositoryURI: "123456.dkr.ecr.us-east-1.amazonaws.com/my-repo"}},
				}, nil
			}
			return nil, ErrRepositoryNotFound
		},
	}

	bridge := NewRegistryBridge(client)
	uri, err := bridge.EnsureRepository(context.Background(), "us-east-1", "my-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uri != "123456.dkr.ecr.us-east-1.amazonaws.com/my-repo" {
		t.Fatalf("unexpected URI: %s", uri)
	}
}

func TestRegistryBridge_EnsureRepository_Creates(t *testing.T) {
	client := &bridgeMockECRClient{
		describeFunc: func(_ context.Context, _ *DescribeRepositoriesInput) (*DescribeRepositoriesOutput, error) {
			return nil, ErrRepositoryNotFound
		},
		createFunc: func(_ context.Context, input *CreateRepositoryInput) (*CreateRepositoryOutput, error) {
			return &CreateRepositoryOutput{
				Repository: Repository{RepositoryURI: "123456.dkr.ecr.us-east-1.amazonaws.com/" + input.RepositoryName},
			}, nil
		},
	}

	bridge := NewRegistryBridge(client)
	uri, err := bridge.EnsureRepository(context.Background(), "us-east-1", "new-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uri != "123456.dkr.ecr.us-east-1.amazonaws.com/new-repo" {
		t.Fatalf("unexpected URI: %s", uri)
	}
}

// --- bridge test mock ECSClient ---

type bridgeMockECSClient struct {
	createClusterFunc    func(ctx context.Context, name string) (*Cluster, error)
	describeServicesFunc func(ctx context.Context, input *DescribeServicesInput) ([]ECSService, error)
	registerTaskDefFunc  func(ctx context.Context, input *RegisterTaskDefinitionInput) (*TaskDefinition, error)
	createServiceFunc    func(ctx context.Context, input *CreateECSServiceInput) (*ECSService, error)
	updateServiceFunc    func(ctx context.Context, input *UpdateECSServiceInput) (*ECSService, error)
	describeTasksFunc    func(ctx context.Context, cluster string, taskARNs []string) ([]Task, error)
}

func (m *bridgeMockECSClient) CreateCluster(ctx context.Context, name string) (*Cluster, error) {
	return m.createClusterFunc(ctx, name)
}
func (m *bridgeMockECSClient) DescribeServices(ctx context.Context, input *DescribeServicesInput) ([]ECSService, error) {
	return m.describeServicesFunc(ctx, input)
}
func (m *bridgeMockECSClient) RegisterTaskDefinition(ctx context.Context, input *RegisterTaskDefinitionInput) (*TaskDefinition, error) {
	return m.registerTaskDefFunc(ctx, input)
}
func (m *bridgeMockECSClient) CreateService(ctx context.Context, input *CreateECSServiceInput) (*ECSService, error) {
	return m.createServiceFunc(ctx, input)
}
func (m *bridgeMockECSClient) UpdateService(ctx context.Context, input *UpdateECSServiceInput) (*ECSService, error) {
	return m.updateServiceFunc(ctx, input)
}
func (m *bridgeMockECSClient) DescribeTasks(ctx context.Context, cluster string, taskARNs []string) ([]Task, error) {
	return m.describeTasksFunc(ctx, cluster, taskARNs)
}

func TestECSBridge_EnsureService(t *testing.T) {
	client := &bridgeMockECSClient{
		createClusterFunc: func(_ context.Context, _ string) (*Cluster, error) {
			return &Cluster{ClusterARN: "arn:aws:ecs:us-east-1:123456:cluster/test"}, nil
		},
		registerTaskDefFunc: func(_ context.Context, input *RegisterTaskDefinitionInput) (*TaskDefinition, error) {
			return &TaskDefinition{TaskDefinitionARN: "arn:aws:ecs:us-east-1:123456:task-def/" + input.Family + ":1"}, nil
		},
		describeServicesFunc: func(_ context.Context, _ *DescribeServicesInput) ([]ECSService, error) {
			return nil, ErrServiceNotFound
		},
		createServiceFunc: func(_ context.Context, input *CreateECSServiceInput) (*ECSService, error) {
			return &ECSService{
				ServiceARN:        "arn:aws:ecs:us-east-1:123456:service/my-svc",
				ServiceName:       input.ServiceName,
				TaskDefinitionARN: input.TaskDefinitionARN,
			}, nil
		},
	}

	bridge := NewECSBridge(client)
	info, err := bridge.EnsureService(context.Background(), DeployServiceOptions{
		Region:       "us-east-1",
		ServiceName:  "my-svc",
		ImageURI:     "123456.dkr.ecr.us-east-1.amazonaws.com/my-repo:latest",
		Port:         8080,
		MinInstances: 1,
		CPU:          "256",
		Memory:       "512",
		ClusterARN:   "test-cluster",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.TaskARN == "" {
		t.Fatal("expected non-empty TaskARN")
	}
}

// --- bridge test mock IAMClient ---

type bridgeMockIAMClient struct {
	getRoleFunc          func(ctx context.Context, name string) (*Role, error)
	createRoleFunc       func(ctx context.Context, input *CreateRoleInput) (*Role, error)
	attachRolePolicyFunc func(ctx context.Context, roleName, policyARN string) error
}

func (m *bridgeMockIAMClient) GetRole(ctx context.Context, name string) (*Role, error) {
	return m.getRoleFunc(ctx, name)
}
func (m *bridgeMockIAMClient) CreateRole(ctx context.Context, input *CreateRoleInput) (*Role, error) {
	return m.createRoleFunc(ctx, input)
}
func (m *bridgeMockIAMClient) AttachRolePolicy(ctx context.Context, roleName, policyARN string) error {
	return m.attachRolePolicyFunc(ctx, roleName, policyARN)
}

func TestIAMBridge_ConfigureIAM(t *testing.T) {
	client := &bridgeMockIAMClient{
		getRoleFunc: func(_ context.Context, name string) (*Role, error) {
			return &Role{ARN: "arn:aws:iam::123456:role/" + name, RoleName: name}, nil
		},
		attachRolePolicyFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
	}

	bridge := NewIAMBridge(client)
	err := bridge.ConfigureIAM(context.Background(), "us-east-1", "my-svc", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIAMBridge_ConfigureIAM_Error(t *testing.T) {
	client := &bridgeMockIAMClient{
		getRoleFunc: func(_ context.Context, _ string) (*Role, error) {
			return nil, errors.New("iam failure")
		},
	}

	bridge := NewIAMBridge(client)
	err := bridge.ConfigureIAM(context.Background(), "us-east-1", "my-svc", false)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- bridge test mock SecretsClient ---

type bridgeMockSecretsClient struct {
	describeSecretFunc func(ctx context.Context, id string) (*SecretInfo, error)
	createSecretFunc   func(ctx context.Context, input *CreateSecretInput) (*SecretInfo, error)
	getSecretValueFunc func(ctx context.Context, id string) (string, error)
}

func (m *bridgeMockSecretsClient) DescribeSecret(ctx context.Context, id string) (*SecretInfo, error) {
	return m.describeSecretFunc(ctx, id)
}
func (m *bridgeMockSecretsClient) CreateSecret(ctx context.Context, input *CreateSecretInput) (*SecretInfo, error) {
	return m.createSecretFunc(ctx, input)
}
func (m *bridgeMockSecretsClient) GetSecretValue(ctx context.Context, id string) (string, error) {
	return m.getSecretValueFunc(ctx, id)
}

func TestSecretsBridge_EnsureSecrets(t *testing.T) {
	client := &bridgeMockSecretsClient{
		describeSecretFunc: func(_ context.Context, id string) (*SecretInfo, error) {
			return &SecretInfo{ARN: "arn:aws:secretsmanager:us-east-1:123456:secret:" + id, Name: id}, nil
		},
	}

	var buf bytes.Buffer
	bridge := NewSecretsBridge(client, &buf)
	arns, err := bridge.EnsureSecrets(context.Background(), "us-east-1", "my-svc", map[string]string{
		"DB_PASSWORD": "prod/db-password",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(arns) != 1 {
		t.Fatalf("expected 1 ARN, got %d", len(arns))
	}
}

func TestSecretsBridge_EnsureSecrets_Creates(t *testing.T) {
	client := &bridgeMockSecretsClient{
		describeSecretFunc: func(_ context.Context, _ string) (*SecretInfo, error) {
			return nil, ErrSecretNotFound
		},
		createSecretFunc: func(_ context.Context, input *CreateSecretInput) (*SecretInfo, error) {
			return &SecretInfo{ARN: "arn:aws:secretsmanager:us-east-1:123456:secret:" + input.Name, Name: input.Name}, nil
		},
	}

	var buf bytes.Buffer
	bridge := NewSecretsBridge(client, &buf)
	arns, err := bridge.EnsureSecrets(context.Background(), "us-east-1", "my-svc", map[string]string{
		"API_KEY": "prod/api-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(arns) != 1 {
		t.Fatalf("expected 1 ARN, got %d", len(arns))
	}
}

func TestHealthBridge_Check(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}
}

func TestHealthBridge_Check_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	}))
	defer srv.Close()

	checker := &HealthChecker{
		HTTPClient: srv.Client(),
		MaxRetries: 1,
	}
	bridge := NewHealthBridge(checker)
	result, err := bridge.Check(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Healthy {
		t.Fatal("expected unhealthy")
	}
}
