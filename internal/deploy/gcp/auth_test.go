package gcp

import (
	"context"
	"strings"
	"testing"
)

func TestAuthenticateNoCredentials(t *testing.T) {
	// Ensure no credentials are available by clearing relevant env vars.
	for _, env := range []string{
		"GOOGLE_APPLICATION_CREDENTIALS",
		"GOOGLE_CLOUD_PROJECT",
		"GCLOUD_PROJECT",
		"CLOUDSDK_CORE_PROJECT",
	} {
		t.Setenv(env, "")
	}
	// Point to a nonexistent credentials file to force failure.
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/nonexistent/credentials.json")

	_, err := Authenticate(context.Background(), "test-project")
	if err == nil {
		t.Fatal("Authenticate() expected error when no credentials available, got nil")
	}

	errMsg := err.Error()

	if !strings.Contains(errMsg, "gcloud auth application-default login") {
		t.Errorf("error message should mention 'gcloud auth application-default login', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "GCP credentials not found") {
		t.Errorf("error message should mention 'GCP credentials not found', got: %s", errMsg)
	}
}

func TestAuthenticateErrorIsHelpful(t *testing.T) {
	tests := []struct {
		name         string
		projectID    string
		wantContains []string
	}{
		{
			name:      "error mentions gcloud command",
			projectID: "my-project",
			wantContains: []string{
				"gcloud auth application-default login",
				"GCP credentials not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/nonexistent/credentials.json")

			_, err := Authenticate(context.Background(), tt.projectID)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q should contain %q", err.Error(), want)
				}
			}
		})
	}
}
