package gcp

import "context"

// SecretClient manages secrets in Secret Manager.
type SecretClient interface {
	// EnsureSecrets creates or updates the specified secrets and grants
	// the Cloud Run service account access to them.
	EnsureSecrets(ctx context.Context, projectID, region, serviceName string, secrets map[string]string) error
}
