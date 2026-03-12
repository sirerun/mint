package gcp

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/serviceusage/v1"
)

var requiredAPIs = []string{
	"run.googleapis.com",
	"cloudbuild.googleapis.com",
	"artifactregistry.googleapis.com",
	"secretmanager.googleapis.com",
	"iam.googleapis.com",
}

// CheckAPIsEnabled verifies that all required GCP APIs are enabled for the
// given project. If any are disabled, it returns an error listing them along
// with the gcloud command to enable them.
func CheckAPIsEnabled(ctx context.Context, projectID string) error {
	svc, err := serviceusage.NewService(ctx)
	if err != nil {
		return fmt.Errorf("creating service usage client: %w", err)
	}

	var disabled []string
	for _, api := range requiredAPIs {
		name := fmt.Sprintf("projects/%s/services/%s", projectID, api)
		resp, err := svc.Services.Get(name).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("checking API %s: %w", api, err)
		}
		if resp.State != "ENABLED" {
			disabled = append(disabled, api)
		}
	}

	if len(disabled) > 0 {
		return fmt.Errorf("required APIs not enabled:\n  %s\n\nEnable them with:\n  gcloud services enable %s --project=%s",
			strings.Join(disabled, "\n  "),
			strings.Join(disabled, " "),
			projectID)
	}
	return nil
}
