package aws

import (
	"context"
	"errors"
	"testing"
	"time"
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
