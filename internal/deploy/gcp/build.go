package gcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

// BuildClient abstracts Cloud Build operations.
type BuildClient interface {
	CreateBuild(ctx context.Context, projectID string, build *BuildConfig) (*BuildResult, error)
}

// BuildConfig describes a container image build.
type BuildConfig struct {
	SourceDir string // local directory containing Dockerfile and source
	ImageURI  string // full image URI (e.g., us-central1-docker.pkg.dev/proj/repo/name:tag)
	ProjectID string
}

// BuildResult holds the outcome of a build.
type BuildResult struct {
	ImageURI string
	LogURL   string
	Duration time.Duration
	Status   string
}

// BuildImage validates the config, delegates to the client, and returns the result.
// Progress and status messages are printed to stderr.
func BuildImage(ctx context.Context, client BuildClient, config BuildConfig) (*BuildResult, error) {
	if err := validateBuildConfig(config); err != nil {
		return nil, fmt.Errorf("invalid build config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Starting Cloud Build for image %s...\n", config.ImageURI)

	result, err := client.CreateBuild(ctx, config.ProjectID, &config)
	if err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Build completed in %s with status %s\n", result.Duration, result.Status)
	if result.LogURL != "" {
		fmt.Fprintf(os.Stderr, "Logs: %s\n", result.LogURL)
	}

	return result, nil
}

// ImageURI constructs a full Artifact Registry image URI.
func ImageURI(region, projectID, repoName, serviceName, tag string) string {
	return fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s:%s", region, projectID, repoName, serviceName, tag)
}

func validateBuildConfig(c BuildConfig) error {
	if c.SourceDir == "" {
		return errors.New("source directory is required")
	}

	info, err := os.Stat(c.SourceDir)
	if err != nil {
		return fmt.Errorf("source directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source directory %s is not a directory", c.SourceDir)
	}

	if c.ImageURI == "" {
		return errors.New("image URI is required")
	}

	if c.ProjectID == "" {
		return errors.New("project ID is required")
	}

	return nil
}
