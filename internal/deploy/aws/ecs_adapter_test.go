package aws

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// --- mock for the high-level ECSClient (used by EnsureService tests) ---

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

func TestEnsureService_DescribeServicesUnexpectedError(t *testing.T) {
	t.Parallel()
	mock := defaultMock()
	mock.describeServicesFn = func(_ context.Context, _ *DescribeServicesInput) ([]ECSService, error) {
		return nil, errors.New("network timeout")
	}

	_, err := EnsureService(context.Background(), mock, defaultOpts())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want := "ensure service: describe"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("expected error containing %q, got: %v", want, err)
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

// --- mock for the low-level ecsAPI (used by ECSAdapter method tests) ---

type fakeECSAPI struct {
	createClusterFn          func(ctx context.Context, input *ecs.CreateClusterInput, optFns ...func(*ecs.Options)) (*ecs.CreateClusterOutput, error)
	describeServicesFn       func(ctx context.Context, input *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	registerTaskDefinitionFn func(ctx context.Context, input *ecs.RegisterTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.RegisterTaskDefinitionOutput, error)
	createServiceFn          func(ctx context.Context, input *ecs.CreateServiceInput, optFns ...func(*ecs.Options)) (*ecs.CreateServiceOutput, error)
	updateServiceFn          func(ctx context.Context, input *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error)
	describeTasksFn          func(ctx context.Context, input *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
	listTaskDefinitionsFn    func(ctx context.Context, input *ecs.ListTaskDefinitionsInput, optFns ...func(*ecs.Options)) (*ecs.ListTaskDefinitionsOutput, error)
}

func (f *fakeECSAPI) CreateCluster(ctx context.Context, input *ecs.CreateClusterInput, optFns ...func(*ecs.Options)) (*ecs.CreateClusterOutput, error) {
	return f.createClusterFn(ctx, input, optFns...)
}

func (f *fakeECSAPI) DescribeServices(ctx context.Context, input *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	return f.describeServicesFn(ctx, input, optFns...)
}

func (f *fakeECSAPI) RegisterTaskDefinition(ctx context.Context, input *ecs.RegisterTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.RegisterTaskDefinitionOutput, error) {
	return f.registerTaskDefinitionFn(ctx, input, optFns...)
}

func (f *fakeECSAPI) CreateService(ctx context.Context, input *ecs.CreateServiceInput, optFns ...func(*ecs.Options)) (*ecs.CreateServiceOutput, error) {
	return f.createServiceFn(ctx, input, optFns...)
}

func (f *fakeECSAPI) UpdateService(ctx context.Context, input *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error) {
	return f.updateServiceFn(ctx, input, optFns...)
}

func (f *fakeECSAPI) DescribeTasks(ctx context.Context, input *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
	return f.describeTasksFn(ctx, input, optFns...)
}

func (f *fakeECSAPI) ListTaskDefinitions(ctx context.Context, input *ecs.ListTaskDefinitionsInput, optFns ...func(*ecs.Options)) (*ecs.ListTaskDefinitionsOutput, error) {
	return f.listTaskDefinitionsFn(ctx, input, optFns...)
}

// fakeWaiter implements ecsWaiter for testing.
type fakeWaiter struct {
	err error
}

func (w *fakeWaiter) Wait(_ context.Context, _ *ecs.DescribeServicesInput, _ time.Duration, _ ...func(*ecs.ServicesStableWaiterOptions)) error {
	return w.err
}

func strPtrECS(s string) *string { return &s }

// --- ECSAdapter method tests ---

func TestECSAdapter_CreateCluster(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		clusterName string
		out         *ecs.CreateClusterOutput
		err         error
		wantErr     string
		wantARN     string
		wantName    string
	}{
		{
			name:        "success",
			clusterName: "my-cluster",
			out: &ecs.CreateClusterOutput{
				Cluster: &ecstypes.Cluster{
					ClusterArn:  strPtrECS("arn:aws:ecs:us-east-1:123:cluster/my-cluster"),
					ClusterName: strPtrECS("my-cluster"),
				},
			},
			wantARN:  "arn:aws:ecs:us-east-1:123:cluster/my-cluster",
			wantName: "my-cluster",
		},
		{
			name:        "sdk error",
			clusterName: "bad-cluster",
			err:         errors.New("access denied"),
			wantErr:     "ecs: create cluster",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			api := &fakeECSAPI{
				createClusterFn: func(_ context.Context, input *ecs.CreateClusterInput, _ ...func(*ecs.Options)) (*ecs.CreateClusterOutput, error) {
					if aws.ToString(input.ClusterName) != tt.clusterName {
						t.Errorf("got cluster name %q, want %q", aws.ToString(input.ClusterName), tt.clusterName)
					}
					return tt.out, tt.err
				},
			}
			adapter := &ECSAdapter{client: api}
			got, err := adapter.CreateCluster(context.Background(), tt.clusterName)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ClusterARN != tt.wantARN {
				t.Errorf("ClusterARN = %q, want %q", got.ClusterARN, tt.wantARN)
			}
			if got.ClusterName != tt.wantName {
				t.Errorf("ClusterName = %q, want %q", got.ClusterName, tt.wantName)
			}
		})
	}
}

func TestECSAdapter_DescribeServices(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    *DescribeServicesInput
		out      *ecs.DescribeServicesOutput
		err      error
		wantErr  string
		wantLen  int
		wantName string
	}{
		{
			name:  "success with active service",
			input: &DescribeServicesInput{Cluster: "c1", ServiceName: "s1"},
			out: &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						ServiceArn:     strPtrECS("arn:svc"),
						ServiceName:    strPtrECS("s1"),
						Status:         strPtrECS("ACTIVE"),
						ClusterArn:     strPtrECS("arn:c1"),
						TaskDefinition: strPtrECS("arn:td"),
						DesiredCount:   1,
						RunningCount:   1,
					},
				},
			},
			wantLen:  1,
			wantName: "s1",
		},
		{
			name:  "filters inactive services",
			input: &DescribeServicesInput{Cluster: "c1", ServiceName: "s1"},
			out: &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						ServiceArn:  strPtrECS("arn:svc"),
						ServiceName: strPtrECS("s1"),
						Status:      strPtrECS("INACTIVE"),
					},
				},
			},
			wantErr: "ecs: service not found",
		},
		{
			name:  "no services returns ErrServiceNotFound",
			input: &DescribeServicesInput{Cluster: "c1", ServiceName: "s1"},
			out: &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{},
			},
			wantErr: "ecs: service not found",
		},
		{
			name:    "sdk error",
			input:   &DescribeServicesInput{Cluster: "c1", ServiceName: "s1"},
			err:     errors.New("timeout"),
			wantErr: "ecs: describe services",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			api := &fakeECSAPI{
				describeServicesFn: func(_ context.Context, input *ecs.DescribeServicesInput, _ ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
					if aws.ToString(input.Cluster) != tt.input.Cluster {
						t.Errorf("cluster = %q, want %q", aws.ToString(input.Cluster), tt.input.Cluster)
					}
					if len(input.Services) != 1 || input.Services[0] != tt.input.ServiceName {
						t.Errorf("services = %v, want [%q]", input.Services, tt.input.ServiceName)
					}
					return tt.out, tt.err
				},
			}
			adapter := &ECSAdapter{client: api}
			got, err := adapter.DescribeServices(context.Background(), tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("got %d services, want %d", len(got), tt.wantLen)
			}
			if tt.wantName != "" && got[0].ServiceName != tt.wantName {
				t.Errorf("ServiceName = %q, want %q", got[0].ServiceName, tt.wantName)
			}
		})
	}
}

func TestECSAdapter_RegisterTaskDefinition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   *RegisterTaskDefinitionInput
		out     *ecs.RegisterTaskDefinitionOutput
		err     error
		wantErr string
		wantARN string
	}{
		{
			name: "success basic",
			input: &RegisterTaskDefinitionInput{
				Family:           "fam",
				ImageURI:         "img:latest",
				ContainerName:    "app",
				Port:             8080,
				CPU:              "256",
				Memory:           "512",
				ExecutionRoleARN: "arn:role",
			},
			out: &ecs.RegisterTaskDefinitionOutput{
				TaskDefinition: &ecstypes.TaskDefinition{
					TaskDefinitionArn: strPtrECS("arn:td:1"),
					Family:            strPtrECS("fam"),
					Revision:          1,
				},
			},
			wantARN: "arn:td:1",
		},
		{
			name: "success with env vars and secrets and task role",
			input: &RegisterTaskDefinitionInput{
				Family:           "fam",
				ImageURI:         "img:latest",
				ContainerName:    "app",
				Port:             8080,
				CPU:              "256",
				Memory:           "512",
				ExecutionRoleARN: "arn:exec-role",
				TaskRoleARN:      "arn:task-role",
				EnvVars:          map[string]string{"KEY": "VAL"},
				SecretARNs:       []string{"arn:secret:1"},
			},
			out: &ecs.RegisterTaskDefinitionOutput{
				TaskDefinition: &ecstypes.TaskDefinition{
					TaskDefinitionArn: strPtrECS("arn:td:2"),
					Family:            strPtrECS("fam"),
					Revision:          2,
				},
			},
			wantARN: "arn:td:2",
		},
		{
			name: "sdk error",
			input: &RegisterTaskDefinitionInput{
				Family:           "fam",
				ImageURI:         "img:latest",
				ContainerName:    "app",
				Port:             8080,
				CPU:              "256",
				Memory:           "512",
				ExecutionRoleARN: "arn:role",
			},
			err:     errors.New("invalid param"),
			wantErr: "ecs: register task definition",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			api := &fakeECSAPI{
				registerTaskDefinitionFn: func(_ context.Context, input *ecs.RegisterTaskDefinitionInput, _ ...func(*ecs.Options)) (*ecs.RegisterTaskDefinitionOutput, error) {
					if aws.ToString(input.Family) != tt.input.Family {
						t.Errorf("Family = %q, want %q", aws.ToString(input.Family), tt.input.Family)
					}
					if tt.input.TaskRoleARN != "" && input.TaskRoleArn == nil {
						t.Error("expected TaskRoleArn to be set")
					}
					if tt.input.TaskRoleARN == "" && input.TaskRoleArn != nil {
						t.Error("expected TaskRoleArn to be nil")
					}
					if len(tt.input.EnvVars) > 0 {
						cd := input.ContainerDefinitions[0]
						if len(cd.Environment) != len(tt.input.EnvVars) {
							t.Errorf("Environment count = %d, want %d", len(cd.Environment), len(tt.input.EnvVars))
						}
					}
					if len(tt.input.SecretARNs) > 0 {
						cd := input.ContainerDefinitions[0]
						if len(cd.Secrets) != len(tt.input.SecretARNs) {
							t.Errorf("Secrets count = %d, want %d", len(cd.Secrets), len(tt.input.SecretARNs))
						}
					}
					return tt.out, tt.err
				},
			}
			adapter := &ECSAdapter{client: api}
			got, err := adapter.RegisterTaskDefinition(context.Background(), tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.TaskDefinitionARN != tt.wantARN {
				t.Errorf("TaskDefinitionARN = %q, want %q", got.TaskDefinitionARN, tt.wantARN)
			}
		})
	}
}

func TestECSAdapter_CreateService(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   *CreateECSServiceInput
		out     *ecs.CreateServiceOutput
		err     error
		wantErr string
		wantSvc string
	}{
		{
			name: "success without LB",
			input: &CreateECSServiceInput{
				Cluster:           "c1",
				ServiceName:       "s1",
				TaskDefinitionARN: "arn:td:1",
				DesiredCount:      1,
				SubnetIDs:         []string{"subnet-1"},
				SecurityGroupIDs:  []string{"sg-1"},
				AssignPublicIP:    false,
			},
			out: &ecs.CreateServiceOutput{
				Service: &ecstypes.Service{
					ServiceArn:     strPtrECS("arn:svc:s1"),
					ServiceName:    strPtrECS("s1"),
					Status:         strPtrECS("ACTIVE"),
					ClusterArn:     strPtrECS("arn:c1"),
					TaskDefinition: strPtrECS("arn:td:1"),
					DesiredCount:   1,
				},
			},
			wantSvc: "s1",
		},
		{
			name: "success with LB and public IP",
			input: &CreateECSServiceInput{
				Cluster:           "c1",
				ServiceName:       "s1",
				TaskDefinitionARN: "arn:td:1",
				DesiredCount:      2,
				SubnetIDs:         []string{"subnet-1"},
				SecurityGroupIDs:  []string{"sg-1"},
				AssignPublicIP:    true,
				TargetGroupARN:    "arn:tg:1",
			},
			out: &ecs.CreateServiceOutput{
				Service: &ecstypes.Service{
					ServiceArn:     strPtrECS("arn:svc:s1"),
					ServiceName:    strPtrECS("s1"),
					Status:         strPtrECS("ACTIVE"),
					ClusterArn:     strPtrECS("arn:c1"),
					TaskDefinition: strPtrECS("arn:td:1"),
					DesiredCount:   2,
				},
			},
			wantSvc: "s1",
		},
		{
			name: "sdk error",
			input: &CreateECSServiceInput{
				Cluster:          "c1",
				ServiceName:      "s1",
				SubnetIDs:        []string{"subnet-1"},
				SecurityGroupIDs: []string{"sg-1"},
			},
			err:     errors.New("quota exceeded"),
			wantErr: "ecs: create service",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			api := &fakeECSAPI{
				createServiceFn: func(_ context.Context, input *ecs.CreateServiceInput, _ ...func(*ecs.Options)) (*ecs.CreateServiceOutput, error) {
					if aws.ToString(input.Cluster) != tt.input.Cluster {
						t.Errorf("Cluster = %q, want %q", aws.ToString(input.Cluster), tt.input.Cluster)
					}
					if tt.input.TargetGroupARN != "" && len(input.LoadBalancers) == 0 {
						t.Error("expected LoadBalancers to be set")
					}
					if tt.input.TargetGroupARN == "" && len(input.LoadBalancers) != 0 {
						t.Error("expected no LoadBalancers")
					}
					if tt.input.AssignPublicIP {
						if input.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIp != ecstypes.AssignPublicIpEnabled {
							t.Error("expected AssignPublicIp = ENABLED")
						}
					} else {
						if input.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIp != ecstypes.AssignPublicIpDisabled {
							t.Error("expected AssignPublicIp = DISABLED")
						}
					}
					return tt.out, tt.err
				},
			}
			adapter := &ECSAdapter{client: api}
			got, err := adapter.CreateService(context.Background(), tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ServiceName != tt.wantSvc {
				t.Errorf("ServiceName = %q, want %q", got.ServiceName, tt.wantSvc)
			}
		})
	}
}

func TestECSAdapter_UpdateService(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   *UpdateECSServiceInput
		out     *ecs.UpdateServiceOutput
		err     error
		wantErr string
		wantSvc string
	}{
		{
			name: "success",
			input: &UpdateECSServiceInput{
				Cluster:           "c1",
				ServiceName:       "s1",
				TaskDefinitionARN: "arn:td:2",
				DesiredCount:      3,
			},
			out: &ecs.UpdateServiceOutput{
				Service: &ecstypes.Service{
					ServiceArn:     strPtrECS("arn:svc:s1"),
					ServiceName:    strPtrECS("s1"),
					Status:         strPtrECS("ACTIVE"),
					ClusterArn:     strPtrECS("arn:c1"),
					TaskDefinition: strPtrECS("arn:td:2"),
					DesiredCount:   3,
				},
			},
			wantSvc: "s1",
		},
		{
			name: "sdk error",
			input: &UpdateECSServiceInput{
				Cluster:     "c1",
				ServiceName: "s1",
			},
			err:     errors.New("throttled"),
			wantErr: "ecs: update service",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			api := &fakeECSAPI{
				updateServiceFn: func(_ context.Context, input *ecs.UpdateServiceInput, _ ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error) {
					if aws.ToString(input.Cluster) != tt.input.Cluster {
						t.Errorf("Cluster = %q, want %q", aws.ToString(input.Cluster), tt.input.Cluster)
					}
					if aws.ToString(input.Service) != tt.input.ServiceName {
						t.Errorf("Service = %q, want %q", aws.ToString(input.Service), tt.input.ServiceName)
					}
					return tt.out, tt.err
				},
			}
			adapter := &ECSAdapter{client: api}
			got, err := adapter.UpdateService(context.Background(), tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ServiceName != tt.wantSvc {
				t.Errorf("ServiceName = %q, want %q", got.ServiceName, tt.wantSvc)
			}
		})
	}
}

func TestECSAdapter_DescribeTasks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cluster  string
		taskARNs []string
		out      *ecs.DescribeTasksOutput
		err      error
		wantErr  string
		wantLen  int
	}{
		{
			name:     "success",
			cluster:  "c1",
			taskARNs: []string{"arn:task:1", "arn:task:2"},
			out: &ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{
					{TaskArn: strPtrECS("arn:task:1"), LastStatus: strPtrECS("RUNNING")},
					{TaskArn: strPtrECS("arn:task:2"), LastStatus: strPtrECS("STOPPED")},
				},
			},
			wantLen: 2,
		},
		{
			name:     "empty tasks",
			cluster:  "c1",
			taskARNs: []string{},
			out: &ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{},
			},
			wantLen: 0,
		},
		{
			name:     "sdk error",
			cluster:  "c1",
			taskARNs: []string{"arn:task:1"},
			err:      errors.New("not found"),
			wantErr:  "ecs: describe tasks",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			api := &fakeECSAPI{
				describeTasksFn: func(_ context.Context, input *ecs.DescribeTasksInput, _ ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
					if aws.ToString(input.Cluster) != tt.cluster {
						t.Errorf("Cluster = %q, want %q", aws.ToString(input.Cluster), tt.cluster)
					}
					return tt.out, tt.err
				},
			}
			adapter := &ECSAdapter{client: api}
			got, err := adapter.DescribeTasks(context.Background(), tt.cluster, tt.taskARNs)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("got %d tasks, want %d", len(got), tt.wantLen)
			}
			if tt.wantLen > 0 {
				if got[0].TaskARN != "arn:task:1" {
					t.Errorf("TaskARN = %q, want %q", got[0].TaskARN, "arn:task:1")
				}
				if got[0].LastStatus != "RUNNING" {
					t.Errorf("LastStatus = %q, want %q", got[0].LastStatus, "RUNNING")
				}
			}
		})
	}
}

func TestECSAdapter_ListTaskDefinitions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		family  string
		out     *ecs.ListTaskDefinitionsOutput
		err     error
		wantErr string
		wantLen int
	}{
		{
			name:   "success",
			family: "my-fam",
			out: &ecs.ListTaskDefinitionsOutput{
				TaskDefinitionArns: []string{"arn:td:2", "arn:td:1"},
			},
			wantLen: 2,
		},
		{
			name:   "empty",
			family: "my-fam",
			out: &ecs.ListTaskDefinitionsOutput{
				TaskDefinitionArns: nil,
			},
			wantLen: 0,
		},
		{
			name:    "sdk error",
			family:  "my-fam",
			err:     errors.New("access denied"),
			wantErr: "ecs: list task definitions",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			api := &fakeECSAPI{
				listTaskDefinitionsFn: func(_ context.Context, input *ecs.ListTaskDefinitionsInput, _ ...func(*ecs.Options)) (*ecs.ListTaskDefinitionsOutput, error) {
					if aws.ToString(input.FamilyPrefix) != tt.family {
						t.Errorf("FamilyPrefix = %q, want %q", aws.ToString(input.FamilyPrefix), tt.family)
					}
					if input.Sort != ecstypes.SortOrderDesc {
						t.Errorf("Sort = %v, want DESC", input.Sort)
					}
					return tt.out, tt.err
				},
			}
			adapter := &ECSAdapter{client: api}
			got, err := adapter.ListTaskDefinitions(context.Background(), tt.family)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("got %d arns, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestECSAdapter_WaitForStableService(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cluster     string
		serviceName string
		waiterErr   error
		wantErr     string
	}{
		{
			name:        "success",
			cluster:     "c1",
			serviceName: "s1",
		},
		{
			name:        "waiter error",
			cluster:     "c1",
			serviceName: "s1",
			waiterErr:   errors.New("exceeded max wait"),
			wantErr:     "ecs: wait for stable service",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			adapter := &ECSAdapter{
				waiter: &fakeWaiter{err: tt.waiterErr},
			}
			err := adapter.WaitForStableService(context.Background(), tt.cluster, tt.serviceName)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEcsServiceFromSDK(t *testing.T) {
	t.Parallel()
	svc := ecsServiceFromSDK(ecstypes.Service{
		ServiceArn:     strPtrECS("arn:aws:ecs:us-east-1:123:service/my-svc"),
		ServiceName:    strPtrECS("my-svc"),
		Status:         strPtrECS("ACTIVE"),
		ClusterArn:     strPtrECS("arn:aws:ecs:us-east-1:123:cluster/my-cluster"),
		TaskDefinition: strPtrECS("arn:aws:ecs:us-east-1:123:task-definition/my-family:1"),
		DesiredCount:   2,
		RunningCount:   1,
	})
	if svc.ServiceARN != "arn:aws:ecs:us-east-1:123:service/my-svc" {
		t.Errorf("unexpected ServiceARN: %s", svc.ServiceARN)
	}
	if svc.ServiceName != "my-svc" {
		t.Errorf("unexpected ServiceName: %s", svc.ServiceName)
	}
	if svc.Status != "ACTIVE" {
		t.Errorf("unexpected Status: %s", svc.Status)
	}
	if svc.ClusterARN != "arn:aws:ecs:us-east-1:123:cluster/my-cluster" {
		t.Errorf("unexpected ClusterARN: %s", svc.ClusterARN)
	}
	if svc.TaskDefinitionARN != "arn:aws:ecs:us-east-1:123:task-definition/my-family:1" {
		t.Errorf("unexpected TaskDefinitionARN: %s", svc.TaskDefinitionARN)
	}
	if svc.DesiredCount != 2 {
		t.Errorf("unexpected DesiredCount: %d", svc.DesiredCount)
	}
	if svc.RunningCount != 1 {
		t.Errorf("unexpected RunningCount: %d", svc.RunningCount)
	}
}

func TestEcsServiceFromSDK_NilFields(t *testing.T) {
	t.Parallel()
	svc := ecsServiceFromSDK(ecstypes.Service{})
	if svc.ServiceARN != "" {
		t.Errorf("expected empty ServiceARN, got %q", svc.ServiceARN)
	}
	if svc.ServiceName != "" {
		t.Errorf("expected empty ServiceName, got %q", svc.ServiceName)
	}
	if svc.DesiredCount != 0 {
		t.Errorf("expected 0 DesiredCount, got %d", svc.DesiredCount)
	}
}

func TestNewECSAdapter(t *testing.T) {
	adapter := NewECSAdapter(aws.Config{})
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if adapter.client == nil {
		t.Fatal("expected non-nil client")
	}
	if adapter.waiter == nil {
		t.Fatal("expected non-nil waiter")
	}
}
