package gcp

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Credentials wraps Google Application Default Credentials.
type Credentials struct {
	TokenSource oauth2.TokenSource
	ProjectID   string
}

// Authenticate attempts to find Application Default Credentials.
// Returns a clear error message if credentials are not available.
func Authenticate(ctx context.Context, projectID string) (*Credentials, error) {
	creds, err := google.FindDefaultCredentials(ctx,
		"https://www.googleapis.com/auth/cloud-platform",
	)
	if err != nil {
		return nil, fmt.Errorf("GCP credentials not found. Run 'gcloud auth application-default login' to authenticate: %w", err)
	}

	// Use provided projectID, fall back to creds.ProjectID.
	pid := projectID
	if pid == "" {
		pid = creds.ProjectID
	}
	if pid == "" {
		return nil, fmt.Errorf("GCP project ID is required. Use --project flag or set GOOGLE_CLOUD_PROJECT")
	}

	return &Credentials{
		TokenSource: creds.TokenSource,
		ProjectID:   pid,
	}, nil
}
