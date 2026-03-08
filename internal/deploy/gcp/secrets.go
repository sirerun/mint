package gcp

import (
	"context"
	"fmt"
	"io"
)

// SecretClient abstracts Secret Manager operations.
type SecretClient interface {
	// GetSecret checks if a secret exists. Returns nil if found,
	// a NotFoundErr if not found, or another error on failure.
	GetSecret(ctx context.Context, name string) error

	// CreateSecret creates a new secret in the given project.
	CreateSecret(ctx context.Context, projectID, secretID string) error
}

// NotFoundErr indicates a secret was not found.
type NotFoundErr struct {
	Name string
}

func (e *NotFoundErr) Error() string {
	return fmt.Sprintf("secret not found: %s", e.Name)
}

// IsNotFound reports whether err is a NotFoundErr.
func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundErr)
	return ok
}

// SecretConfig holds configuration for secret mounting.
type SecretConfig struct {
	ProjectID string
	Secrets   []SecretMount
}

// SecretMount maps an environment variable to a Secret Manager secret.
type SecretMount struct {
	EnvVar     string // environment variable name
	SecretName string // Secret Manager secret ID
}

// SecretResourceName returns the fully qualified Secret Manager resource name.
func SecretResourceName(projectID, secretID string) string {
	return fmt.Sprintf("projects/%s/secrets/%s", projectID, secretID)
}

// EnsureSecrets verifies that all configured secrets exist, creating any that
// are missing. Created secrets are empty; the user must set their values
// separately. Messages about newly created secrets are written to stderr.
func EnsureSecrets(ctx context.Context, client SecretClient, config SecretConfig, stderr io.Writer) ([]SecretMount, error) {
	for _, s := range config.Secrets {
		name := SecretResourceName(config.ProjectID, s.SecretName)
		err := client.GetSecret(ctx, name)
		if err == nil {
			continue
		}
		if !IsNotFound(err) {
			return nil, fmt.Errorf("checking secret %s: %w", s.SecretName, err)
		}
		// Secret does not exist; create it.
		if createErr := client.CreateSecret(ctx, config.ProjectID, s.SecretName); createErr != nil {
			return nil, fmt.Errorf("creating secret %s: %w", s.SecretName, createErr)
		}
		_, _ = fmt.Fprintf(stderr, "Secret %s created. Set its value with: gcloud secrets versions add %s --data-file=- --project=%s\n",
			s.SecretName, s.SecretName, config.ProjectID)
	}
	return config.Secrets, nil
}
