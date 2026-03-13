package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

// DefaultRepoName is the default ECR repository name.
const DefaultRepoName = "mint-mcp-servers"

// ecrAPI abstracts the AWS ECR SDK methods used by ECRAdapter.
type ecrAPI interface {
	DescribeRepositories(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error)
	CreateRepository(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error)
}

// ECRAdapter implements ECRClient using the AWS SDK v2.
type ECRAdapter struct {
	client ecrAPI
}

var _ ECRClient = (*ECRAdapter)(nil)

// NewECRAdapter creates a new adapter backed by the AWS ECR SDK client.
func NewECRAdapter(cfg aws.Config) *ECRAdapter {
	return &ECRAdapter{client: ecr.NewFromConfig(cfg)}
}

// DescribeRepositories returns metadata for the named repositories.
func (a *ECRAdapter) DescribeRepositories(ctx context.Context, input *DescribeRepositoriesInput) (*DescribeRepositoriesOutput, error) {
	out, err := a.client.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{
		RepositoryNames: input.RepositoryNames,
	})
	if err != nil {
		var notFound *types.RepositoryNotFoundException
		if errors.As(err, &notFound) {
			return nil, ErrRepositoryNotFound
		}
		return nil, err
	}
	repos := make([]Repository, len(out.Repositories))
	for i, r := range out.Repositories {
		repos[i] = Repository{
			RepositoryURI: derefStr(r.RepositoryUri),
			RepositoryARN: derefStr(r.RepositoryArn),
		}
	}
	return &DescribeRepositoriesOutput{Repositories: repos}, nil
}

// CreateRepository creates a new ECR repository.
func (a *ECRAdapter) CreateRepository(ctx context.Context, input *CreateRepositoryInput) (*CreateRepositoryOutput, error) {
	out, err := a.client.CreateRepository(ctx, &ecr.CreateRepositoryInput{
		RepositoryName: &input.RepositoryName,
	})
	if err != nil {
		return nil, err
	}
	return &CreateRepositoryOutput{
		Repository: Repository{
			RepositoryURI: derefStr(out.Repository.RepositoryUri),
			RepositoryARN: derefStr(out.Repository.RepositoryArn),
		},
	}, nil
}

// EnsureRepository checks whether an ECR repository exists and creates it if
// it does not. It returns the repository URI.
func EnsureRepository(ctx context.Context, client ECRClient, region, repoName string) (string, error) {
	if repoName == "" {
		repoName = DefaultRepoName
	}

	out, err := client.DescribeRepositories(ctx, &DescribeRepositoriesInput{
		RepositoryNames: []string{repoName},
	})
	if err == nil && len(out.Repositories) > 0 {
		return out.Repositories[0].RepositoryURI, nil
	}
	if err != nil && !errors.Is(err, ErrRepositoryNotFound) {
		return "", fmt.Errorf("describe repositories: %w", err)
	}

	createOut, err := client.CreateRepository(ctx, &CreateRepositoryInput{
		RepositoryName: repoName,
	})
	if err != nil {
		return "", fmt.Errorf("create repository: %w", err)
	}
	return createOut.Repository.RepositoryURI, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
