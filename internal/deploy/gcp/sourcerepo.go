// Package gcp provides Google Cloud Platform deployment helpers.
package gcp

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Repo represents a Cloud Source Repository.
type Repo struct {
	// Name is the fully qualified repository name,
	// e.g. "projects/<projectID>/repos/<repoName>".
	Name string
}

// GetRepoRequest is the request for GetRepo.
type GetRepoRequest struct {
	// Name is the fully qualified repository name.
	Name string
}

// CreateRepoRequest is the request for CreateRepo.
type CreateRepoRequest struct {
	// Parent is the project in which to create the repo,
	// e.g. "projects/<projectID>".
	Parent string

	// Repo is the repository to create.
	Repo *Repo
}

// SourceRepoClient abstracts the Cloud Source Repositories API for testability.
type SourceRepoClient interface {
	GetRepo(ctx context.Context, req *GetRepoRequest) (*Repo, error)
	CreateRepo(ctx context.Context, req *CreateRepoRequest) (*Repo, error)
}

// EnsureSourceRepo ensures a Cloud Source Repository exists for the given
// project and repo name. It is idempotent: if the repository already exists,
// it returns the clone URL without modification.
//
// The repository is named "mint-mcp-<repoName>" within the specified project.
// The returned URL has the form:
//
//	https://source.developers.google.com/p/<projectID>/r/mint-mcp-<repoName>
func EnsureSourceRepo(ctx context.Context, client SourceRepoClient, projectID, repoName string) (string, error) {
	qualifiedName := fmt.Sprintf("projects/%s/repos/mint-mcp-%s", projectID, repoName)

	_, err := client.GetRepo(ctx, &GetRepoRequest{Name: qualifiedName})
	if err == nil {
		return cloneURL(projectID, repoName), nil
	}

	if st, ok := status.FromError(err); !ok || st.Code() != codes.NotFound {
		return "", fmt.Errorf("checking repository %s: %w", qualifiedName, err)
	}

	_, err = client.CreateRepo(ctx, &CreateRepoRequest{
		Parent: fmt.Sprintf("projects/%s", projectID),
		Repo:   &Repo{Name: qualifiedName},
	})
	if err != nil {
		return "", fmt.Errorf("creating repository %s: %w", qualifiedName, err)
	}

	return cloneURL(projectID, repoName), nil
}

func cloneURL(projectID, repoName string) string {
	return fmt.Sprintf("https://source.developers.google.com/p/%s/r/mint-mcp-%s", projectID, repoName)
}
