package aws

import (
	"context"
)

// CodeBuildClient abstracts AWS CodeBuild operations.
type CodeBuildClient interface {
	// CreateProject creates or updates a CodeBuild project.
	CreateProject(ctx context.Context, input *CreateProjectInput) error

	// StartBuild triggers a new build for the given project.
	StartBuild(ctx context.Context, input *StartBuildInput) (*StartBuildOutput, error)

	// BatchGetBuilds returns the status of one or more builds by their IDs.
	BatchGetBuilds(ctx context.Context, buildIDs []string) ([]Build, error)
}

// CreateProjectInput holds parameters for creating a CodeBuild project.
type CreateProjectInput struct {
	ProjectName string
	ServiceRole string
	ImageURI    string
	ComputeType string
}

// StartBuildInput holds parameters for starting a CodeBuild build.
type StartBuildInput struct {
	ProjectName string
	SourceDir   string
	ImageURI    string
}

// StartBuildOutput holds the result of starting a CodeBuild build.
type StartBuildOutput struct {
	BuildID string
}

// Build represents the status of a CodeBuild build.
type Build struct {
	ID       string
	Status   string
	ImageURI string
	LogURL   string
}
