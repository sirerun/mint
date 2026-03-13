package aws

import (
	"context"
	"errors"
	"testing"
)

type mockECSClient struct {
	createClusterFn          func(ctx context.Context, clusterName string) (*Cluster, error)
	describeServicesFn       func(ctx context.Context, input *DescribeServicesInput) ([]ECSService, error)
	registerTaskDefinitionFn func(ctx context.Context, input *RegisterTaskDefinitionInput) (*TaskDefinition, error)
	createServiceFn          func(ctx context.Context, input *CreateECSServiceInput) (*ECSService, error)
	updateServiceFn          func(ctx context.Context, input *UpdateECSServiceInput) (*ECSService, error)
	describeTasksFn          func(ctx context.Context, cluster string, taskARNs []string) ([]Task, error)
}

func (m *mockECSClient) CreateCluster(ctx context.Context, clusterName string) (*Cluster, error) {
	return m.createClusterFn(ctx, clusterName)
}

func (m *mockECSClient) DescribeServices(ctx context.Context, input *DescribeServicesInput) ([]ECSService, error) {
	return m.describeServicesFn(ctx, input)
}

func (m *mockECSClient) RegisterTaskDefinition(ctx context.Context, input *RegisterTaskDefinitionInput) (*TaskDefinition, error) {
	return m.registerTaskDefinitionFn(ctx, input)
}

func (m *mockECSClient) CreateService(ctx context.Context, input *CreateECSServiceInput) (*ECSService, error) {
	return m.createServiceFn(ctx, input)
}

func (m *mockECSClient) UpdateService(ctx context.Context, input *UpdateECSServiceInput) (*ECSService, error) {
	return m.updateServiceFn(ctx, input)
}

func (m *mockECSClient) DescribeTasks(ctx context.Context, cluster string, taskARNs []string) ([]Task, error) {
	return m.describeTasksFn(ctx, cluster, taskARNs)
}

func defaultMock() *mockECSClient {
	return &mockECSClient{
		createClusterFn: func(_ context.Context, name string) (*Cluster, error) {
			return &Cluster{ClusterARN: "arn:aws:ecs:us-east-1:123:cluster/" + name, ClusterName: name}, nil
		},
		registerTaskDefinitionFn: func(_ context.Context, input *RegisterTaskDefinitionInput) (*TaskDefinition, error) {
			return &TaskDefinition{
				TaskDefinitionARN: "arn:aws:ecs:us-east-1:123:task-definition/" + input.Family + ":1",
				Family:            input.Family,
				Revision:          1,
			}, nil
		},
		describeServicesFn: func(_ context.Context, _ *DescribeServicesInput) ([]ECSService, error) {
			return nil, ErrServiceNotFound
		},
		createServiceFn: func(_ context.Context, input *CreateECSServiceInput) (*ECSService, error) {
			return &ECSService{
				ServiceARN:        "arn:aws:ecs:us-east-1:123:service/" + input.ServiceName,
				ServiceName:       input.ServiceName,
				Status:            "ACTIVE",
				ClusterARN:        input.Cluster,
				TaskDefinitionARN: input.TaskDefinitionARN,
				DesiredCount:      input.DesiredCount,
			}, nil
		},
		updateServiceFn: func(_ context.Context, input *UpdateECSServiceInput) (*ECSService, error) {
			return &ECSService{
				ServiceARN:        "arn:aws:ecs:us-east-1:123:service/" + input.ServiceName,
				ServiceName:       input.ServiceName,
				Status:            "ACTIVE",
				ClusterARN:        input.Cluster,
				TaskDefinitionARN: input.TaskDefinitionARN,
				DesiredCount:      input.DesiredCount,
			}, nil
		},
		describeTasksFn: func(_ context.Context, _ string, _ []string) ([]Task, error) {
			return nil, nil
		},
	}
}

func defaultOpts() *EnsureServiceOptions {
	return &EnsureServiceOptions{
		ClusterName: "test-cluster",
		ServiceName: "test-service",
		TaskDefinitionInput: &RegisterTaskDefinitionInput{
			Family:        "test-family",
			ImageURI:      "123.dkr.ecr.us-east-1.amazonaws.com/app:latest",
			ContainerName: "app",
			Port:          8080,
			CPU:           "256",
			Memory:        "512",
		},
		DesiredCount:     1,
		SubnetIDs:        []string{"subnet-1"},
		SecurityGroupIDs: []string{"sg-1"},
		AssignPublicIP:   true,
	}
}

func TestEnsureService_CreatesNewService(t *testing.T) {
	t.Parallel()
	mock := defaultMock()
	ctx := context.Background()

	svc, err := EnsureService(ctx, mock, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.ServiceName != "test-service" {
		t.Errorf("got service name %q, want %q", svc.ServiceName, "test-service")
	}
	if svc.Status != "ACTIVE" {
		t.Errorf("got status %q, want %q", svc.Status, "ACTIVE")
	}
}

func TestEnsureService_UpdatesExistingService(t *testing.T) {
	t.Parallel()
	mock := defaultMock()
	mock.describeServicesFn = func(_ context.Context, _ *DescribeServicesInput) ([]ECSService, error) {
		return []ECSService{{
			ServiceARN:        "arn:aws:ecs:us-east-1:123:service/test-service",
			ServiceName:       "test-service",
			Status:            "ACTIVE",
			TaskDefinitionARN: "arn:aws:ecs:us-east-1:123:task-definition/test-family:0",
			DesiredCount:      1,
		}}, nil
	}

	var updateCalled bool
	mock.updateServiceFn = func(_ context.Context, input *UpdateECSServiceInput) (*ECSService, error) {
		updateCalled = true
		return &ECSService{
			ServiceARN:        "arn:aws:ecs:us-east-1:123:service/" + input.ServiceName,
			ServiceName:       input.ServiceName,
			Status:            "ACTIVE",
			TaskDefinitionARN: input.TaskDefinitionARN,
			DesiredCount:      input.DesiredCount,
		}, nil
	}

	ctx := context.Background()
	svc, err := EnsureService(ctx, mock, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Fatal("expected UpdateService to be called")
	}
	if svc.TaskDefinitionARN != "arn:aws:ecs:us-east-1:123:task-definition/test-family:1" {
		t.Errorf("got task def %q, want updated ARN", svc.TaskDefinitionARN)
	}
}

func TestEnsureService_ClusterCreationFails(t *testing.T) {
	t.Parallel()
	mock := defaultMock()
	mock.createClusterFn = func(_ context.Context, _ string) (*Cluster, error) {
		return nil, errors.New("access denied")
	}

	_, err := EnsureService(context.Background(), mock, defaultOpts())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errors.Unwrap(err)) && err.Error() == "" {
		t.Fatal("expected wrapped error")
	}
}

func TestEnsureService_TaskDefinitionRegistrationFails(t *testing.T) {
	t.Parallel()
	mock := defaultMock()
	mock.registerTaskDefinitionFn = func(_ context.Context, _ *RegisterTaskDefinitionInput) (*TaskDefinition, error) {
		return nil, errors.New("invalid CPU")
	}

	_, err := EnsureService(context.Background(), mock, defaultOpts())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnsureService_ServiceCreationFails(t *testing.T) {
	t.Parallel()
	mock := defaultMock()
	mock.createServiceFn = func(_ context.Context, _ *CreateECSServiceInput) (*ECSService, error) {
		return nil, errors.New("subnet not found")
	}

	_, err := EnsureService(context.Background(), mock, defaultOpts())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnsureService_ServiceUpdateFails(t *testing.T) {
	t.Parallel()
	mock := defaultMock()
	mock.describeServicesFn = func(_ context.Context, _ *DescribeServicesInput) ([]ECSService, error) {
		return []ECSService{{ServiceName: "test-service", Status: "ACTIVE"}}, nil
	}
	mock.updateServiceFn = func(_ context.Context, _ *UpdateECSServiceInput) (*ECSService, error) {
		return nil, errors.New("throttled")
	}

	_, err := EnsureService(context.Background(), mock, defaultOpts())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
