package aws

import (
	"context"
	"errors"
)

// ErrRepositoryNotFound indicates that an ECR repository was not found.
var ErrRepositoryNotFound = errors.New("ecr: repository not found")

// ECRClient abstracts Amazon Elastic Container Registry operations.
type ECRClient interface {
	// DescribeRepositories returns metadata for the named repositories.
	// Returns ErrRepositoryNotFound if none of the requested repositories exist.
	DescribeRepositories(ctx context.Context, input *DescribeRepositoriesInput) (*DescribeRepositoriesOutput, error)

	// CreateRepository creates a new ECR repository.
	CreateRepository(ctx context.Context, input *CreateRepositoryInput) (*CreateRepositoryOutput, error)
}

// DescribeRepositoriesInput holds parameters for describing ECR repositories.
type DescribeRepositoriesInput struct {
	RepositoryNames []string
}

// DescribeRepositoriesOutput holds the result of describing ECR repositories.
type DescribeRepositoriesOutput struct {
	Repositories []Repository
}

// Repository represents an ECR repository.
type Repository struct {
	RepositoryURI string
	RepositoryARN string
}

// CreateRepositoryInput holds parameters for creating an ECR repository.
type CreateRepositoryInput struct {
	RepositoryName string
}

// CreateRepositoryOutput holds the result of creating an ECR repository.
type CreateRepositoryOutput struct {
	Repository Repository
}
