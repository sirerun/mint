package gcp

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockSourceRepoClient implements SourceRepoClient for testing.
type mockSourceRepoClient struct {
	getRepo    func(ctx context.Context, req *GetRepoRequest) (*Repo, error)
	createRepo func(ctx context.Context, req *CreateRepoRequest) (*Repo, error)
}

func (m *mockSourceRepoClient) GetRepo(ctx context.Context, req *GetRepoRequest) (*Repo, error) {
	return m.getRepo(ctx, req)
}

func (m *mockSourceRepoClient) CreateRepo(ctx context.Context, req *CreateRepoRequest) (*Repo, error) {
	return m.createRepo(ctx, req)
}

func TestEnsureSourceRepo_AlreadyExists(t *testing.T) {
	createCalled := false
	client := &mockSourceRepoClient{
		getRepo: func(_ context.Context, req *GetRepoRequest) (*Repo, error) {
			want := "projects/my-project/repos/mint-mcp-petstore"
			if req.Name != want {
				t.Errorf("GetRepo name = %q, want %q", req.Name, want)
			}
			return &Repo{Name: req.Name}, nil
		},
		createRepo: func(_ context.Context, _ *CreateRepoRequest) (*Repo, error) {
			createCalled = true
			return nil, errors.New("should not be called")
		},
	}

	url, err := EnsureSourceRepo(context.Background(), client, "my-project", "petstore")
	if err != nil {
		t.Fatalf("EnsureSourceRepo() error = %v", err)
	}

	want := "https://source.developers.google.com/p/my-project/r/mint-mcp-petstore"
	if url != want {
		t.Errorf("URL = %q, want %q", url, want)
	}

	if createCalled {
		t.Error("CreateRepo should not have been called when repo exists")
	}
}

func TestEnsureSourceRepo_NotFound_Creates(t *testing.T) {
	client := &mockSourceRepoClient{
		getRepo: func(_ context.Context, _ *GetRepoRequest) (*Repo, error) {
			return nil, status.Error(codes.NotFound, "repository not found")
		},
		createRepo: func(_ context.Context, req *CreateRepoRequest) (*Repo, error) {
			wantParent := "projects/my-project"
			if req.Parent != wantParent {
				t.Errorf("CreateRepo parent = %q, want %q", req.Parent, wantParent)
			}
			wantName := "projects/my-project/repos/mint-mcp-petstore"
			if req.Repo.Name != wantName {
				t.Errorf("CreateRepo repo name = %q, want %q", req.Repo.Name, wantName)
			}
			return &Repo{Name: req.Repo.Name}, nil
		},
	}

	url, err := EnsureSourceRepo(context.Background(), client, "my-project", "petstore")
	if err != nil {
		t.Fatalf("EnsureSourceRepo() error = %v", err)
	}

	want := "https://source.developers.google.com/p/my-project/r/mint-mcp-petstore"
	if url != want {
		t.Errorf("URL = %q, want %q", url, want)
	}
}

func TestEnsureSourceRepo_GetRepoUnexpectedError(t *testing.T) {
	client := &mockSourceRepoClient{
		getRepo: func(_ context.Context, _ *GetRepoRequest) (*Repo, error) {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		},
		createRepo: func(_ context.Context, _ *CreateRepoRequest) (*Repo, error) {
			t.Fatal("CreateRepo should not be called on unexpected error")
			return nil, nil
		},
	}

	_, err := EnsureSourceRepo(context.Background(), client, "my-project", "petstore")
	if err == nil {
		t.Fatal("EnsureSourceRepo() expected error, got nil")
	}
}

func TestEnsureSourceRepo_CreateRepoFails(t *testing.T) {
	client := &mockSourceRepoClient{
		getRepo: func(_ context.Context, _ *GetRepoRequest) (*Repo, error) {
			return nil, status.Error(codes.NotFound, "repository not found")
		},
		createRepo: func(_ context.Context, _ *CreateRepoRequest) (*Repo, error) {
			return nil, status.Error(codes.Internal, "internal server error")
		},
	}

	_, err := EnsureSourceRepo(context.Background(), client, "my-project", "petstore")
	if err == nil {
		t.Fatal("EnsureSourceRepo() expected error, got nil")
	}
}
