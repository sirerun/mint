package gcp

import (
	"context"
	"fmt"
	"os"
	"time"
)

// GitClient abstracts git operations for testability.
type GitClient interface {
	Init(dir string) error
	AddAll(dir string) error
	Commit(dir string, message string) error
	AddRemote(dir string, name, url string) error
	Push(dir string, remote, branch string) error
	HasRemote(dir string, name string) (bool, error)
}

// SourcePushConfig holds configuration for pushing source code.
type SourcePushConfig struct {
	SourceDir string
	ProjectID string
	RepoName  string // e.g., "mint-mcp-<serviceName>"
	SpecHash  string // for commit message
}

// SourcePushResult holds the outcome.
type SourcePushResult struct {
	RepoURL    string
	CommitHash string
}

// RepoURL constructs a Cloud Source Repositories URL from a project ID and repo name.
func RepoURL(projectID, repoName string) string {
	return fmt.Sprintf("https://source.developers.google.com/p/%s/r/%s", projectID, repoName)
}

// PushSource pushes generated source code to a Google Cloud Source Repository.
// It initializes a git repo in the source directory, adds a remote, commits all
// files, and pushes to the "main" branch.
func PushSource(_ context.Context, client GitClient, config SourcePushConfig) (*SourcePushResult, error) {
	repoURL := RepoURL(config.ProjectID, config.RepoName)

	fmt.Fprintf(os.Stderr, "Initializing git repository in %s\n", config.SourceDir)
	if err := client.Init(config.SourceDir); err != nil {
		return nil, fmt.Errorf("git init: %w", err)
	}

	hasRemote, err := client.HasRemote(config.SourceDir, "google")
	if err != nil {
		return nil, fmt.Errorf("check remote: %w", err)
	}

	if !hasRemote {
		fmt.Fprintf(os.Stderr, "Adding remote \"google\" -> %s\n", repoURL)
		if err := client.AddRemote(config.SourceDir, "google", repoURL); err != nil {
			return nil, fmt.Errorf("add remote: %w", err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Remote \"google\" already exists, skipping")
	}

	fmt.Fprintln(os.Stderr, "Staging all files")
	if err := client.AddAll(config.SourceDir); err != nil {
		return nil, fmt.Errorf("git add: %w", err)
	}

	commitMsg := fmt.Sprintf("deploy: spec-hash=%s at %s", config.SpecHash, time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(os.Stderr, "Committing: %s\n", commitMsg)
	if err := client.Commit(config.SourceDir, commitMsg); err != nil {
		return nil, fmt.Errorf("git commit: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Pushing to google/main")
	if err := client.Push(config.SourceDir, "google", "main"); err != nil {
		return nil, fmt.Errorf("git push: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Push complete")
	return &SourcePushResult{
		RepoURL: repoURL,
	}, nil
}
