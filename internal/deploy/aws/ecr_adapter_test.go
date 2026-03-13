package aws

import (
	"context"
	"errors"
	"testing"

	sdkaws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

type mockECRClient struct {
	describeOut  *DescribeRepositoriesOutput
	describeErr  error
	createOut    *CreateRepositoryOutput
	createErr    error
	createCalled bool
}

func (m *mockECRClient) DescribeRepositories(_ context.Context, _ *DescribeRepositoriesInput) (*DescribeRepositoriesOutput, error) {
	return m.describeOut, m.describeErr
}

func (m *mockECRClient) CreateRepository(_ context.Context, input *CreateRepositoryInput) (*CreateRepositoryOutput, error) {
	m.createCalled = true
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createOut, nil
}

func TestEnsureRepository_Exists(t *testing.T) {
	client := &mockECRClient{
		describeOut: &DescribeRepositoriesOutput{
			Repositories: []Repository{
				{RepositoryURI: "123456789012.dkr.ecr.us-east-1.amazonaws.com/mint-mcp-servers", RepositoryARN: "arn:aws:ecr:us-east-1:123456789012:repository/mint-mcp-servers"},
			},
		},
	}
	uri, err := EnsureRepository(context.Background(), client, "us-east-1", "mint-mcp-servers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "123456789012.dkr.ecr.us-east-1.amazonaws.com/mint-mcp-servers"
	if uri != want {
		t.Fatalf("got URI %q, want %q", uri, want)
	}
	if client.createCalled {
		t.Fatal("CreateRepository should not have been called")
	}
}

func TestEnsureRepository_NotFound_Creates(t *testing.T) {
	wantURI := "123456789012.dkr.ecr.us-east-1.amazonaws.com/mint-mcp-servers"
	client := &mockECRClient{
		describeErr: ErrRepositoryNotFound,
		createOut: &CreateRepositoryOutput{
			Repository: Repository{RepositoryURI: wantURI, RepositoryARN: "arn:aws:ecr:us-east-1:123456789012:repository/mint-mcp-servers"},
		},
	}
	uri, err := EnsureRepository(context.Background(), client, "us-east-1", "mint-mcp-servers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uri != wantURI {
		t.Fatalf("got URI %q, want %q", uri, wantURI)
	}
	if !client.createCalled {
		t.Fatal("CreateRepository should have been called")
	}
}

func TestEnsureRepository_GetUnexpectedError(t *testing.T) {
	client := &mockECRClient{
		describeErr: errors.New("access denied"),
	}
	_, err := EnsureRepository(context.Background(), client, "us-east-1", "mint-mcp-servers")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnsureRepository_CreateFails(t *testing.T) {
	client := &mockECRClient{
		describeErr: ErrRepositoryNotFound,
		createErr:   errors.New("quota exceeded"),
	}
	_, err := EnsureRepository(context.Background(), client, "us-east-1", "mint-mcp-servers")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnsureRepository_DefaultRepoName(t *testing.T) {
	wantURI := "123456789012.dkr.ecr.us-east-1.amazonaws.com/mint-mcp-servers"
	client := &mockECRClient{
		describeOut: &DescribeRepositoriesOutput{
			Repositories: []Repository{
				{RepositoryURI: wantURI},
			},
		},
	}
	uri, err := EnsureRepository(context.Background(), client, "us-east-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uri != wantURI {
		t.Fatalf("got URI %q, want %q", uri, wantURI)
	}
}

func TestDerefStr(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"nil pointer", nil, ""},
		{"non-nil pointer", strPtr("hello"), "hello"},
		{"empty string pointer", strPtr(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := derefStr(tt.in); got != tt.want {
				t.Errorf("derefStr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func TestECRAdapterInterface(t *testing.T) {
	var _ ECRClient = (*ECRAdapter)(nil)
}

// --- ecrAPI mock for adapter-level tests ---

type stubECRAPI struct {
	describeFunc func(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error)
	createFunc   func(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error)
}

func (s *stubECRAPI) DescribeRepositories(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
	return s.describeFunc(ctx, params, optFns...)
}

func (s *stubECRAPI) CreateRepository(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error) {
	return s.createFunc(ctx, params, optFns...)
}

func TestNewECRAdapter(t *testing.T) {
	adapter := NewECRAdapter(sdkaws.Config{Region: "us-east-1"})
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if adapter.client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestECRAdapter_DescribeRepositories(t *testing.T) {
	uri := "123456.dkr.ecr.us-east-1.amazonaws.com/my-repo"
	arn := "arn:aws:ecr:us-east-1:123456:repository/my-repo"

	tests := []struct {
		name      string
		stubFn    func(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error)
		wantErr   error
		wantRepos int
		wantURI   string
	}{
		{
			name: "success",
			stubFn: func(_ context.Context, _ *ecr.DescribeRepositoriesInput, _ ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
				return &ecr.DescribeRepositoriesOutput{
					Repositories: []ecrtypes.Repository{
						{RepositoryUri: &uri, RepositoryArn: &arn},
					},
				}, nil
			},
			wantRepos: 1,
			wantURI:   uri,
		},
		{
			name: "not found mapped to sentinel",
			stubFn: func(_ context.Context, _ *ecr.DescribeRepositoriesInput, _ ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
				return nil, &ecrtypes.RepositoryNotFoundException{Message: strPtr("not found")}
			},
			wantErr: ErrRepositoryNotFound,
		},
		{
			name: "other error passed through",
			stubFn: func(_ context.Context, _ *ecr.DescribeRepositoriesInput, _ ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
				return nil, errors.New("access denied")
			},
			wantErr: nil, // non-nil error but not the sentinel
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &ECRAdapter{client: &stubECRAPI{describeFunc: tt.stubFn}}
			out, err := adapter.DescribeRepositories(context.Background(), &DescribeRepositoriesInput{
				RepositoryNames: []string{"my-repo"},
			})

			if tt.name == "other error passed through" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if errors.Is(err, ErrRepositoryNotFound) {
					t.Fatal("expected non-sentinel error")
				}
				return
			}

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(out.Repositories) != tt.wantRepos {
				t.Fatalf("expected %d repos, got %d", tt.wantRepos, len(out.Repositories))
			}
			if out.Repositories[0].RepositoryURI != tt.wantURI {
				t.Fatalf("expected URI %q, got %q", tt.wantURI, out.Repositories[0].RepositoryURI)
			}
		})
	}
}

func TestECRAdapter_CreateRepository(t *testing.T) {
	uri := "123456.dkr.ecr.us-east-1.amazonaws.com/new-repo"
	arn := "arn:aws:ecr:us-east-1:123456:repository/new-repo"

	tests := []struct {
		name    string
		stubFn  func(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error)
		wantErr bool
		wantURI string
	}{
		{
			name: "success",
			stubFn: func(_ context.Context, _ *ecr.CreateRepositoryInput, _ ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error) {
				return &ecr.CreateRepositoryOutput{
					Repository: &ecrtypes.Repository{
						RepositoryUri: &uri,
						RepositoryArn: &arn,
					},
				}, nil
			},
			wantURI: uri,
		},
		{
			name: "error",
			stubFn: func(_ context.Context, _ *ecr.CreateRepositoryInput, _ ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error) {
				return nil, errors.New("quota exceeded")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &ECRAdapter{client: &stubECRAPI{createFunc: tt.stubFn}}
			out, err := adapter.CreateRepository(context.Background(), &CreateRepositoryInput{
				RepositoryName: "new-repo",
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.Repository.RepositoryURI != tt.wantURI {
				t.Fatalf("expected URI %q, got %q", tt.wantURI, out.Repository.RepositoryURI)
			}
		})
	}
}
