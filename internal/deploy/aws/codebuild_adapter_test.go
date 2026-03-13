package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	cbtypes "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
)

// mockCodeBuildClient implements CodeBuildClient for testing.
type mockCodeBuildClient struct {
	createProjectFn  func(ctx context.Context, input *CreateProjectInput) error
	startBuildFn     func(ctx context.Context, input *StartBuildInput) (*StartBuildOutput, error)
	batchGetBuildsFn func(ctx context.Context, buildIDs []string) ([]Build, error)
}

func (m *mockCodeBuildClient) CreateProject(ctx context.Context, input *CreateProjectInput) error {
	return m.createProjectFn(ctx, input)
}

func (m *mockCodeBuildClient) StartBuild(ctx context.Context, input *StartBuildInput) (*StartBuildOutput, error) {
	return m.startBuildFn(ctx, input)
}

func (m *mockCodeBuildClient) BatchGetBuilds(ctx context.Context, buildIDs []string) ([]Build, error) {
	return m.batchGetBuildsFn(ctx, buildIDs)
}

func TestBuildImage_HappyPath(t *testing.T) {
	pollCount := 0
	mock := &mockCodeBuildClient{
		createProjectFn: func(_ context.Context, _ *CreateProjectInput) error {
			return nil
		},
		startBuildFn: func(_ context.Context, _ *StartBuildInput) (*StartBuildOutput, error) {
			return &StartBuildOutput{BuildID: "build-123"}, nil
		},
		batchGetBuildsFn: func(_ context.Context, ids []string) ([]Build, error) {
			pollCount++
			if pollCount < 2 {
				return []Build{{ID: ids[0], Status: "IN_PROGRESS"}}, nil
			}
			return []Build{{
				ID:       ids[0],
				Status:   "SUCCEEDED",
				ImageURI: "123456.dkr.ecr.us-east-1.amazonaws.com/my-repo:latest",
				LogURL:   "https://console.aws.amazon.com/logs",
			}}, nil
		},
	}

	ctx := context.Background()
	build, err := buildImageWithInterval(ctx, mock, "proj", "/src", "img:latest", 1*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if build.Status != "SUCCEEDED" {
		t.Errorf("expected SUCCEEDED, got %s", build.Status)
	}
	if build.ImageURI != "123456.dkr.ecr.us-east-1.amazonaws.com/my-repo:latest" {
		t.Errorf("unexpected image URI: %s", build.ImageURI)
	}
	if pollCount != 2 {
		t.Errorf("expected 2 polls, got %d", pollCount)
	}
}

func TestBuildImage_BuildFails(t *testing.T) {
	mock := &mockCodeBuildClient{
		createProjectFn: func(_ context.Context, _ *CreateProjectInput) error {
			return nil
		},
		startBuildFn: func(_ context.Context, _ *StartBuildInput) (*StartBuildOutput, error) {
			return &StartBuildOutput{BuildID: "build-fail"}, nil
		},
		batchGetBuildsFn: func(_ context.Context, ids []string) ([]Build, error) {
			return []Build{{
				ID:     ids[0],
				Status: "FAILED",
				LogURL: "https://logs/fail",
			}}, nil
		},
	}

	ctx := context.Background()
	_, err := buildImageWithInterval(ctx, mock, "proj", "/src", "img:latest", 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected error for failed build")
	}
	if want := "finished with status FAILED"; !containsStr(err.Error(), want) {
		t.Errorf("expected error containing %q, got: %v", want, err)
	}
}

func TestBuildImage_CreateProjectAlreadyExists(t *testing.T) {
	mock := &mockCodeBuildClient{
		createProjectFn: func(_ context.Context, _ *CreateProjectInput) error {
			return errors.New("ResourceAlreadyExistsException: project already exists")
		},
		startBuildFn: func(_ context.Context, _ *StartBuildInput) (*StartBuildOutput, error) {
			return &StartBuildOutput{BuildID: "build-456"}, nil
		},
		batchGetBuildsFn: func(_ context.Context, ids []string) ([]Build, error) {
			return []Build{{ID: ids[0], Status: "SUCCEEDED", ImageURI: "img:latest"}}, nil
		},
	}

	ctx := context.Background()
	build, err := buildImageWithInterval(ctx, mock, "proj", "/src", "img:latest", 1*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if build.Status != "SUCCEEDED" {
		t.Errorf("expected SUCCEEDED, got %s", build.Status)
	}
}

func TestBuildImage_ContextCancellation(t *testing.T) {
	mock := &mockCodeBuildClient{
		createProjectFn: func(_ context.Context, _ *CreateProjectInput) error {
			return nil
		},
		startBuildFn: func(_ context.Context, _ *StartBuildInput) (*StartBuildOutput, error) {
			return &StartBuildOutput{BuildID: "build-ctx"}, nil
		},
		batchGetBuildsFn: func(_ context.Context, _ []string) ([]Build, error) {
			return []Build{{ID: "build-ctx", Status: "IN_PROGRESS"}}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately so the poll loop's context check triggers.
	cancel()

	_, err := buildImageWithInterval(ctx, mock, "proj", "/src", "img:latest", 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestBuildImage_StartBuildFails(t *testing.T) {
	mock := &mockCodeBuildClient{
		createProjectFn: func(_ context.Context, _ *CreateProjectInput) error {
			return nil
		},
		startBuildFn: func(_ context.Context, _ *StartBuildInput) (*StartBuildOutput, error) {
			return nil, errors.New("throttling exception")
		},
		batchGetBuildsFn: func(_ context.Context, _ []string) ([]Build, error) {
			t.Fatal("BatchGetBuilds should not be called")
			return nil, nil
		},
	}

	ctx := context.Background()
	_, err := buildImageWithInterval(ctx, mock, "proj", "/src", "img:latest", 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when StartBuild fails")
	}
	if want := "start build"; !containsStr(err.Error(), want) {
		t.Errorf("expected error containing %q, got: %v", want, err)
	}
}

func TestBuildImage_CreateProjectFails(t *testing.T) {
	mock := &mockCodeBuildClient{
		createProjectFn: func(_ context.Context, _ *CreateProjectInput) error {
			return errors.New("access denied")
		},
		startBuildFn: func(_ context.Context, _ *StartBuildInput) (*StartBuildOutput, error) {
			t.Fatal("StartBuild should not be called")
			return nil, nil
		},
		batchGetBuildsFn: func(_ context.Context, _ []string) ([]Build, error) {
			t.Fatal("BatchGetBuilds should not be called")
			return nil, nil
		},
	}

	ctx := context.Background()
	_, err := buildImageWithInterval(ctx, mock, "proj", "/src", "img:latest", 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when CreateProject fails")
	}
	if want := "create project"; !containsStr(err.Error(), want) {
		t.Errorf("expected error containing %q, got: %v", want, err)
	}
}

func TestBuildImage_Wrapper(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}
	mock := &mockCodeBuildClient{
		createProjectFn: func(_ context.Context, _ *CreateProjectInput) error {
			return nil
		},
		startBuildFn: func(_ context.Context, _ *StartBuildInput) (*StartBuildOutput, error) {
			return &StartBuildOutput{BuildID: "build-wrap"}, nil
		},
		batchGetBuildsFn: func(_ context.Context, ids []string) ([]Build, error) {
			return []Build{{ID: ids[0], Status: "SUCCEEDED", ImageURI: "img:latest"}}, nil
		},
	}

	build, err := BuildImage(context.Background(), mock, "proj", "/src", "img:latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if build.Status != "SUCCEEDED" {
		t.Errorf("expected SUCCEEDED, got %s", build.Status)
	}
}

func TestBuildImage_PollReturnsEmptyBuilds(t *testing.T) {
	mock := &mockCodeBuildClient{
		createProjectFn: func(_ context.Context, _ *CreateProjectInput) error {
			return nil
		},
		startBuildFn: func(_ context.Context, _ *StartBuildInput) (*StartBuildOutput, error) {
			return &StartBuildOutput{BuildID: "build-empty"}, nil
		},
		batchGetBuildsFn: func(_ context.Context, _ []string) ([]Build, error) {
			return []Build{}, nil
		},
	}

	_, err := buildImageWithInterval(context.Background(), mock, "proj", "/src", "img:latest", 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when build not found")
	}
	if want := "not found"; !containsStr(err.Error(), want) {
		t.Errorf("expected error containing %q, got: %v", want, err)
	}
}

func TestBuildImage_PollReturnsError(t *testing.T) {
	mock := &mockCodeBuildClient{
		createProjectFn: func(_ context.Context, _ *CreateProjectInput) error {
			return nil
		},
		startBuildFn: func(_ context.Context, _ *StartBuildInput) (*StartBuildOutput, error) {
			return &StartBuildOutput{BuildID: "build-pollerr"}, nil
		},
		batchGetBuildsFn: func(_ context.Context, _ []string) ([]Build, error) {
			return nil, errors.New("service unavailable")
		},
	}

	_, err := buildImageWithInterval(context.Background(), mock, "proj", "/src", "img:latest", 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when BatchGetBuilds fails")
	}
	if want := "codebuild: poll"; !containsStr(err.Error(), want) {
		t.Errorf("expected error containing %q, got: %v", want, err)
	}
}

func TestIsAlreadyExistsError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"already exists", errors.New("project already exists"), true},
		{"ResourceAlreadyExistsException", errors.New("ResourceAlreadyExistsException: dup"), true},
		{"unrelated error", errors.New("access denied"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAlreadyExistsError(tt.err); got != tt.want {
				t.Errorf("isAlreadyExistsError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- SDK-level mock for codebuildAPI ---

type mockCodeBuildAPI struct {
	createProjectFn  func(ctx context.Context, input *codebuild.CreateProjectInput, optFns ...func(*codebuild.Options)) (*codebuild.CreateProjectOutput, error)
	startBuildFn     func(ctx context.Context, input *codebuild.StartBuildInput, optFns ...func(*codebuild.Options)) (*codebuild.StartBuildOutput, error)
	batchGetBuildsFn func(ctx context.Context, input *codebuild.BatchGetBuildsInput, optFns ...func(*codebuild.Options)) (*codebuild.BatchGetBuildsOutput, error)
}

func (m *mockCodeBuildAPI) CreateProject(ctx context.Context, input *codebuild.CreateProjectInput, optFns ...func(*codebuild.Options)) (*codebuild.CreateProjectOutput, error) {
	return m.createProjectFn(ctx, input, optFns...)
}

func (m *mockCodeBuildAPI) StartBuild(ctx context.Context, input *codebuild.StartBuildInput, optFns ...func(*codebuild.Options)) (*codebuild.StartBuildOutput, error) {
	return m.startBuildFn(ctx, input, optFns...)
}

func (m *mockCodeBuildAPI) BatchGetBuilds(ctx context.Context, input *codebuild.BatchGetBuildsInput, optFns ...func(*codebuild.Options)) (*codebuild.BatchGetBuildsOutput, error) {
	return m.batchGetBuildsFn(ctx, input, optFns...)
}

func TestCodeBuildAdapter_CreateProject(t *testing.T) {
	tests := []struct {
		name    string
		sdkErr  error
		wantErr bool
	}{
		{
			name: "success",
		},
		{
			name:    "error",
			sdkErr:  errors.New("access denied"),
			wantErr: true,
		},
		{
			name:    "already exists error passes through",
			sdkErr:  errors.New("ResourceAlreadyExistsException: project already exists"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCodeBuildAPI{
				createProjectFn: func(_ context.Context, input *codebuild.CreateProjectInput, _ ...func(*codebuild.Options)) (*codebuild.CreateProjectOutput, error) {
					if aws.ToString(input.Name) != "my-project" {
						t.Errorf("expected project name %q, got %q", "my-project", aws.ToString(input.Name))
					}
					if tt.sdkErr != nil {
						return nil, tt.sdkErr
					}
					return &codebuild.CreateProjectOutput{}, nil
				},
			}
			adapter := &CodeBuildAdapter{client: mock}
			err := adapter.CreateProject(context.Background(), &CreateProjectInput{
				ProjectName: "my-project",
				ServiceRole: "arn:role",
				ImageURI:    "img:latest",
				ComputeType: "BUILD_GENERAL1_SMALL",
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateProject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCodeBuildAdapter_StartBuild(t *testing.T) {
	tests := []struct {
		name    string
		sdkErr  error
		wantErr bool
		wantID  string
	}{
		{
			name:   "success",
			wantID: "build-123",
		},
		{
			name:    "error",
			sdkErr:  errors.New("throttling"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCodeBuildAPI{
				startBuildFn: func(_ context.Context, input *codebuild.StartBuildInput, _ ...func(*codebuild.Options)) (*codebuild.StartBuildOutput, error) {
					if aws.ToString(input.ProjectName) != "proj" {
						t.Errorf("expected project name %q, got %q", "proj", aws.ToString(input.ProjectName))
					}
					if tt.sdkErr != nil {
						return nil, tt.sdkErr
					}
					return &codebuild.StartBuildOutput{
						Build: &cbtypes.Build{
							Id: aws.String("build-123"),
						},
					}, nil
				},
			}
			adapter := &CodeBuildAdapter{client: mock}
			out, err := adapter.StartBuild(context.Background(), &StartBuildInput{
				ProjectName: "proj",
				SourceDir:   "/src",
				ImageURI:    "img:latest",
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("StartBuild() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && out.BuildID != tt.wantID {
				t.Errorf("StartBuild() BuildID = %q, want %q", out.BuildID, tt.wantID)
			}
		})
	}
}

func TestCodeBuildAdapter_BatchGetBuilds(t *testing.T) {
	tests := []struct {
		name       string
		sdkOut     *codebuild.BatchGetBuildsOutput
		sdkErr     error
		wantErr    bool
		wantBuilds int
		checkLog   string
	}{
		{
			name: "success with logs",
			sdkOut: &codebuild.BatchGetBuildsOutput{
				Builds: []cbtypes.Build{
					{
						Id:          aws.String("build-1"),
						BuildStatus: cbtypes.StatusTypeSucceeded,
						Environment: &cbtypes.ProjectEnvironment{
							Image: aws.String("img:latest"),
						},
						Logs: &cbtypes.LogsLocation{
							DeepLink: aws.String("https://logs/1"),
						},
					},
				},
			},
			wantBuilds: 1,
			checkLog:   "https://logs/1",
		},
		{
			name: "success without logs",
			sdkOut: &codebuild.BatchGetBuildsOutput{
				Builds: []cbtypes.Build{
					{
						Id:          aws.String("build-2"),
						BuildStatus: cbtypes.StatusTypeInProgress,
						Environment: &cbtypes.ProjectEnvironment{
							Image: aws.String("img:latest"),
						},
					},
				},
			},
			wantBuilds: 1,
			checkLog:   "",
		},
		{
			name:    "error",
			sdkErr:  errors.New("service unavailable"),
			wantErr: true,
		},
		{
			name: "empty builds",
			sdkOut: &codebuild.BatchGetBuildsOutput{
				Builds: []cbtypes.Build{},
			},
			wantBuilds: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCodeBuildAPI{
				batchGetBuildsFn: func(_ context.Context, input *codebuild.BatchGetBuildsInput, _ ...func(*codebuild.Options)) (*codebuild.BatchGetBuildsOutput, error) {
					if len(input.Ids) == 0 {
						t.Error("expected non-empty build IDs")
					}
					if tt.sdkErr != nil {
						return nil, tt.sdkErr
					}
					return tt.sdkOut, nil
				},
			}
			adapter := &CodeBuildAdapter{client: mock}
			builds, err := adapter.BatchGetBuilds(context.Background(), []string{"build-1"})
			if (err != nil) != tt.wantErr {
				t.Errorf("BatchGetBuilds() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if len(builds) != tt.wantBuilds {
					t.Errorf("BatchGetBuilds() returned %d builds, want %d", len(builds), tt.wantBuilds)
				}
				if tt.wantBuilds > 0 && builds[0].LogURL != tt.checkLog {
					t.Errorf("expected log URL %q, got %q", tt.checkLog, builds[0].LogURL)
				}
			}
		})
	}
}

func TestNewCodeBuildAdapter(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	adapter := NewCodeBuildAdapter(cfg)
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if adapter.client == nil {
		t.Fatal("expected non-nil client")
	}
}
