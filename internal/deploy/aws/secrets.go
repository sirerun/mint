package aws

import (
	"context"
	"errors"
)

// ErrSecretNotFound indicates that a secret was not found in Secrets Manager.
var ErrSecretNotFound = errors.New("secrets: secret not found")

// SecretsClient abstracts AWS Secrets Manager operations.
type SecretsClient interface {
	// DescribeSecret returns metadata about a secret.
	// Returns ErrSecretNotFound if the secret does not exist.
	DescribeSecret(ctx context.Context, secretID string) (*SecretInfo, error)

	// CreateSecret creates a new secret.
	CreateSecret(ctx context.Context, input *CreateSecretInput) (*SecretInfo, error)

	// GetSecretValue retrieves the current value of a secret.
	GetSecretValue(ctx context.Context, secretID string) (string, error)
}

// SecretInfo represents metadata about a Secrets Manager secret.
type SecretInfo struct {
	ARN  string
	Name string
}

// CreateSecretInput holds parameters for creating a secret.
type CreateSecretInput struct {
	Name        string
	Description string
}
