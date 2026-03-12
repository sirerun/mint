package gcp

import (
	"context"
	"testing"
)

func TestRequiredAPIsContainsExpectedAPIs(t *testing.T) {
	expected := []string{
		"run.googleapis.com",
		"cloudbuild.googleapis.com",
		"artifactregistry.googleapis.com",
		"secretmanager.googleapis.com",
		"iam.googleapis.com",
	}

	if len(requiredAPIs) != len(expected) {
		t.Fatalf("expected %d required APIs, got %d", len(expected), len(requiredAPIs))
	}

	for i, api := range expected {
		if requiredAPIs[i] != api {
			t.Errorf("requiredAPIs[%d] = %q, want %q", i, requiredAPIs[i], api)
		}
	}
}

func TestCheckAPIsEnabledSignature(t *testing.T) {
	// Verify the function signature matches the expected contract.
	var fn func(ctx context.Context, projectID string) error = CheckAPIsEnabled
	_ = fn
}
