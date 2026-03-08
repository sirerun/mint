package gcp

import "context"

// RegistryClient manages Artifact Registry repositories.
type RegistryClient interface {
	// EnsureRepository creates the Artifact Registry repository if it does not exist.
	// It returns the repository path (e.g., "us-central1-docker.pkg.dev/project/repo").
	EnsureRepository(ctx context.Context, projectID, region, repoName string) (string, error)
}
