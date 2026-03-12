package gcp

import (
	"context"
	"fmt"

	artifactregistry "cloud.google.com/go/artifactregistry/apiv1"
	artifactregistrypb "cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
)

// ArtifactRegistryAdapter implements RegistryClient using the real GCP SDK.
type ArtifactRegistryAdapter struct {
	client *artifactregistry.Client
}

var _ RegistryClient = (*ArtifactRegistryAdapter)(nil)

// NewArtifactRegistryAdapter creates a new adapter backed by the Artifact
// Registry gRPC client. The caller should call Close when done.
func NewArtifactRegistryAdapter(ctx context.Context) (*ArtifactRegistryAdapter, error) {
	client, err := artifactregistry.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating artifact registry client: %w", err)
	}
	return &ArtifactRegistryAdapter{client: client}, nil
}

// Close releases the underlying gRPC connection.
func (a *ArtifactRegistryAdapter) Close() error {
	return a.client.Close()
}

// GetRepository returns the named Artifact Registry repository.
func (a *ArtifactRegistryAdapter) GetRepository(ctx context.Context, name string) (*artifactregistrypb.Repository, error) {
	return a.client.GetRepository(ctx, &artifactregistrypb.GetRepositoryRequest{Name: name})
}

// CreateRepository creates a new Artifact Registry repository, waiting for
// the long-running operation to complete before returning.
func (a *ArtifactRegistryAdapter) CreateRepository(ctx context.Context, req *artifactregistrypb.CreateRepositoryRequest) (*artifactregistrypb.Repository, error) {
	op, err := a.client.CreateRepository(ctx, req)
	if err != nil {
		return nil, err
	}
	return op.Wait(ctx)
}
