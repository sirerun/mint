package aws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"

	sdkaws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// SecretsManagerAdapter implements SecretsClient using the AWS SDK v2.
type SecretsManagerAdapter struct {
	client *secretsmanager.Client
}

var _ SecretsClient = (*SecretsManagerAdapter)(nil)

// NewSecretsManagerAdapter creates a new adapter backed by the AWS Secrets Manager SDK client.
func NewSecretsManagerAdapter(cfg sdkaws.Config) *SecretsManagerAdapter {
	return &SecretsManagerAdapter{client: secretsmanager.NewFromConfig(cfg)}
}

// DescribeSecret returns metadata about a secret.
func (a *SecretsManagerAdapter) DescribeSecret(ctx context.Context, secretID string) (*SecretInfo, error) {
	out, err := a.client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: &secretID,
	})
	if err != nil {
		var notFound *types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return nil, ErrSecretNotFound
		}
		return nil, err
	}
	return &SecretInfo{
		ARN:  derefStr(out.ARN),
		Name: derefStr(out.Name),
	}, nil
}

// CreateSecret creates a new secret.
func (a *SecretsManagerAdapter) CreateSecret(ctx context.Context, input *CreateSecretInput) (*SecretInfo, error) {
	out, err := a.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:        &input.Name,
		Description: &input.Description,
	})
	if err != nil {
		return nil, err
	}
	return &SecretInfo{
		ARN:  derefStr(out.ARN),
		Name: derefStr(out.Name),
	}, nil
}

// GetSecretValue retrieves the current value of a secret.
func (a *SecretsManagerAdapter) GetSecretValue(ctx context.Context, secretID string) (string, error) {
	out, err := a.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretID,
	})
	if err != nil {
		var notFound *types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return "", ErrSecretNotFound
		}
		return "", err
	}
	return derefStr(out.SecretString), nil
}

// EnsureSecrets verifies that each secret in the map exists, creating any that
// are missing. The map keys are environment variable names and values are secret
// names in Secrets Manager. It returns the list of secret ARNs.
func EnsureSecrets(ctx context.Context, client SecretsClient, secrets map[string]string, stderr io.Writer) ([]string, error) {
	// Sort keys for deterministic output.
	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	arns := make([]string, 0, len(secrets))
	for _, envVar := range keys {
		secretName := secrets[envVar]
		info, err := client.DescribeSecret(ctx, secretName)
		if err == nil {
			arns = append(arns, info.ARN)
			continue
		}
		if !errors.Is(err, ErrSecretNotFound) {
			return nil, fmt.Errorf("describe secret %q: %w", secretName, err)
		}

		info, err = client.CreateSecret(ctx, &CreateSecretInput{
			Name:        secretName,
			Description: fmt.Sprintf("Mint managed secret for %s", envVar),
		})
		if err != nil {
			return nil, fmt.Errorf("create secret %q: %w", secretName, err)
		}
		fmt.Fprintf(stderr, "Created secret %q (%s)\n", secretName, info.ARN)
		arns = append(arns, info.ARN)
	}
	return arns, nil
}
