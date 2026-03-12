package gcp

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SecretManagerAdapter implements SecretClient using the GCP Secret Manager API.
type SecretManagerAdapter struct {
	client *secretmanager.Client
}

var _ SecretClient = (*SecretManagerAdapter)(nil)

// NewSecretManagerAdapter creates a new adapter backed by the GCP Secret Manager service.
func NewSecretManagerAdapter(ctx context.Context) (*SecretManagerAdapter, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating secret manager client: %w", err)
	}
	return &SecretManagerAdapter{client: client}, nil
}

// Close releases the underlying gRPC connection.
func (a *SecretManagerAdapter) Close() error {
	return a.client.Close()
}

// GetSecret checks whether a secret exists by name. Returns nil if the secret
// exists, a *NotFoundErr if it does not, or another error on failure.
func (a *SecretManagerAdapter) GetSecret(ctx context.Context, name string) error {
	_, err := a.client.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{Name: name})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return &NotFoundErr{Name: name}
		}
		return err
	}
	return nil
}

// CreateSecret creates a new secret with automatic replication in the given project.
func (a *SecretManagerAdapter) CreateSecret(ctx context.Context, projectID, secretID string) error {
	_, err := a.client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", projectID),
		SecretId: secretID,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	})
	return err
}
