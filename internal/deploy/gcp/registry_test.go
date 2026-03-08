package gcp

import (
	"context"
	"errors"
	"testing"

	artifactregistrypb "cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockRegistryClient struct {
	getRepo   *artifactregistrypb.Repository
	getErr    error
	createErr error
	createReq *artifactregistrypb.CreateRepositoryRequest
}

func (m *mockRegistryClient) GetRepository(_ context.Context, _ string) (*artifactregistrypb.Repository, error) {
	return m.getRepo, m.getErr
}

func (m *mockRegistryClient) CreateRepository(_ context.Context, req *artifactregistrypb.CreateRepositoryRequest) (*artifactregistrypb.Repository, error) {
	m.createReq = req
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &artifactregistrypb.Repository{Name: req.GetParent() + "/repositories/" + req.GetRepositoryId()}, nil
}

func TestEnsureRepository(t *testing.T) {
	ctx := context.Background()
	const (
		project = "my-project"
		region  = "us-central1"
		repo    = "mint-mcp-servers"
	)
	wantURI := "us-central1-docker.pkg.dev/my-project/mint-mcp-servers"

	t.Run("exists", func(t *testing.T) {
		client := &mockRegistryClient{
			getRepo: &artifactregistrypb.Repository{
				Name: "projects/my-project/locations/us-central1/repositories/mint-mcp-servers",
			},
		}
		uri, err := EnsureRepository(ctx, client, project, region, repo)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uri != wantURI {
			t.Fatalf("got URI %q, want %q", uri, wantURI)
		}
		if client.createReq != nil {
			t.Fatal("CreateRepository should not have been called")
		}
	})

	t.Run("not_found_creates", func(t *testing.T) {
		client := &mockRegistryClient{
			getErr: status.Error(codes.NotFound, "not found"),
		}
		uri, err := EnsureRepository(ctx, client, project, region, repo)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uri != wantURI {
			t.Fatalf("got URI %q, want %q", uri, wantURI)
		}
		if client.createReq == nil {
			t.Fatal("CreateRepository should have been called")
		}
		if client.createReq.GetRepositoryId() != repo {
			t.Fatalf("got repo ID %q, want %q", client.createReq.GetRepositoryId(), repo)
		}
		if client.createReq.GetRepository().GetFormat() != artifactregistrypb.Repository_DOCKER {
			t.Fatalf("got format %v, want DOCKER", client.createReq.GetRepository().GetFormat())
		}
	})

	t.Run("get_unexpected_error", func(t *testing.T) {
		client := &mockRegistryClient{
			getErr: status.Error(codes.PermissionDenied, "permission denied"),
		}
		_, err := EnsureRepository(ctx, client, project, region, repo)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("create_fails", func(t *testing.T) {
		client := &mockRegistryClient{
			getErr:    status.Error(codes.NotFound, "not found"),
			createErr: errors.New("quota exceeded"),
		}
		_, err := EnsureRepository(ctx, client, project, region, repo)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("default_repo_name", func(t *testing.T) {
		client := &mockRegistryClient{
			getRepo: &artifactregistrypb.Repository{
				Name: "projects/my-project/locations/us-central1/repositories/mint-mcp-servers",
			},
		}
		uri, err := EnsureRepository(ctx, client, project, region, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uri != wantURI {
			t.Fatalf("got URI %q, want %q", uri, wantURI)
		}
	})
}
