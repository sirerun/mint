package aws

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	cbtypes "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
)

// CodeBuildAdapter wraps the AWS CodeBuild SDK client.
type CodeBuildAdapter struct {
	client *codebuild.Client
}

var _ CodeBuildClient = (*CodeBuildAdapter)(nil)

// NewCodeBuildAdapter creates a new CodeBuildAdapter from the given AWS config.
func NewCodeBuildAdapter(cfg aws.Config) *CodeBuildAdapter {
	return &CodeBuildAdapter{client: codebuild.NewFromConfig(cfg)}
}

func (a *CodeBuildAdapter) CreateProject(ctx context.Context, input *CreateProjectInput) error {
	_, err := a.client.CreateProject(ctx, &codebuild.CreateProjectInput{
		Name:        aws.String(input.ProjectName),
		ServiceRole: aws.String(input.ServiceRole),
		Source: &cbtypes.ProjectSource{
			Type: cbtypes.SourceTypeNoSource,
			Buildspec: aws.String(`version: 0.2
phases:
  build:
    commands:
      - echo "Building image"
`),
		},
		Artifacts: &cbtypes.ProjectArtifacts{
			Type: cbtypes.ArtifactsTypeNoArtifacts,
		},
		Environment: &cbtypes.ProjectEnvironment{
			ComputeType:          cbtypes.ComputeType(input.ComputeType),
			Image:                aws.String(input.ImageURI),
			Type:                 cbtypes.EnvironmentTypeLinuxContainer,
			PrivilegedMode:       aws.Bool(true),
			ImagePullCredentialsType: cbtypes.ImagePullCredentialsTypeServiceRole,
		},
	})
	return err
}

func (a *CodeBuildAdapter) StartBuild(ctx context.Context, input *StartBuildInput) (*StartBuildOutput, error) {
	out, err := a.client.StartBuild(ctx, &codebuild.StartBuildInput{
		ProjectName:    aws.String(input.ProjectName),
		ImageOverride:  aws.String(input.ImageURI),
		SourceVersion:  aws.String(input.SourceDir),
	})
	if err != nil {
		return nil, err
	}
	return &StartBuildOutput{
		BuildID: aws.ToString(out.Build.Id),
	}, nil
}

func (a *CodeBuildAdapter) BatchGetBuilds(ctx context.Context, buildIDs []string) ([]Build, error) {
	out, err := a.client.BatchGetBuilds(ctx, &codebuild.BatchGetBuildsInput{
		Ids: buildIDs,
	})
	if err != nil {
		return nil, err
	}
	builds := make([]Build, len(out.Builds))
	for i, b := range out.Builds {
		var logURL string
		if b.Logs != nil {
			logURL = aws.ToString(b.Logs.DeepLink)
		}
		builds[i] = Build{
			ID:       aws.ToString(b.Id),
			Status:   string(b.BuildStatus),
			ImageURI: aws.ToString(b.Environment.Image),
			LogURL:   logURL,
		}
	}
	return builds, nil
}

// BuildImage creates (or reuses) a CodeBuild project, starts a build, and polls
// until the build completes. It returns the final Build result.
func BuildImage(ctx context.Context, client CodeBuildClient, projectName, sourceDir, imageURI string) (*Build, error) {
	return buildImageWithInterval(ctx, client, projectName, sourceDir, imageURI, 10*time.Second)
}

func buildImageWithInterval(ctx context.Context, client CodeBuildClient, projectName, sourceDir, imageURI string, pollInterval time.Duration) (*Build, error) {
	// Step 1: Create project (idempotent).
	err := client.CreateProject(ctx, &CreateProjectInput{
		ProjectName: projectName,
		ServiceRole: "", // caller should configure via project defaults
		ImageURI:    imageURI,
		ComputeType: "BUILD_GENERAL1_SMALL",
	})
	if err != nil && !isAlreadyExistsError(err) {
		return nil, fmt.Errorf("codebuild: create project: %w", err)
	}

	// Step 2: Start build.
	startOut, err := client.StartBuild(ctx, &StartBuildInput{
		ProjectName: projectName,
		SourceDir:   sourceDir,
		ImageURI:    imageURI,
	})
	if err != nil {
		return nil, fmt.Errorf("codebuild: start build: %w", err)
	}

	// Step 3: Poll until terminal status.
	fmt.Fprintf(os.Stderr, "CodeBuild: build %s started, polling...\n", startOut.BuildID)
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("codebuild: poll: %w", ctx.Err())
		case <-time.After(pollInterval):
		}

		builds, err := client.BatchGetBuilds(ctx, []string{startOut.BuildID})
		if err != nil {
			return nil, fmt.Errorf("codebuild: poll: %w", err)
		}
		if len(builds) == 0 {
			return nil, fmt.Errorf("codebuild: build %s not found", startOut.BuildID)
		}

		b := builds[0]
		fmt.Fprintf(os.Stderr, "CodeBuild: build %s status: %s\n", b.ID, b.Status)

		switch b.Status {
		case "SUCCEEDED":
			return &b, nil
		case "FAILED", "FAULT", "TIMED_OUT", "STOPPED":
			return nil, fmt.Errorf("codebuild: build %s finished with status %s (logs: %s)", b.ID, b.Status, b.LogURL)
		}
	}
}

// isAlreadyExistsError checks whether an error indicates the resource already exists.
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	// The AWS SDK returns ResourceAlreadyExistsException for duplicate CodeBuild projects.
	return strings.Contains(err.Error(), "ResourceAlreadyExistsException") ||
		strings.Contains(err.Error(), "already exists")
}
