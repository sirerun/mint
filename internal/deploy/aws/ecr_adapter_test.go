package aws

import (
	"context"
	"errors"
	"testing"
)

type mockECRClient struct {
	describeOut *DescribeRepositoriesOutput
	describeErr error
	createOut   *CreateRepositoryOutput
	createErr   error
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

func TestECRAdapterInterface(t *testing.T) {
	var _ ECRClient = (*ECRAdapter)(nil)
}
