package gcp

import "context"

// SourceRepoClient manages Cloud Source Repositories.
type SourceRepoClient interface {
	// EnsureRepo creates the Cloud Source Repository if it does not exist
	// and returns the repository URL.
	EnsureRepo(ctx context.Context, projectID, repoName string) (string, error)
}

// GitClient pushes source code to a Git remote.
type GitClient interface {
	// Push pushes the source directory to the given remote URL.
	Push(ctx context.Context, sourceDir, remoteURL string) error
}
