package azure

import (
	"context"
	"errors"
)

// ErrRepositoryNotFound indicates that an ACR repository was not found.
var ErrRepositoryNotFound = errors.New("acr: repository not found")

// ACRClient abstracts Azure Container Registry operations.
type ACRClient interface {
	// EnsureRepository creates a repository if it does not already exist
	// and returns the repository URI.
	EnsureRepository(ctx context.Context, resourceGroup, registryName, repoName string) (string, error)

	// GetLoginServer returns the login server URL for the registry.
	GetLoginServer(ctx context.Context, resourceGroup, registryName string) (string, error)
}
