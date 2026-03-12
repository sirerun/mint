package gcp

import (
	"testing"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"cloud.google.com/go/storage"
)

// Compile-time interface check.
var _ BuildClient = (*CloudBuildAdapter)(nil)

func TestCloudBuildAdapterConstructor(t *testing.T) {
	// Verify the constructor accepts the expected client types and returns
	// the correct adapter type. We pass nil clients since we are only
	// checking the signature, not making real API calls.
	adapter := NewCloudBuildAdapter((*cloudbuild.Client)(nil), (*storage.Client)(nil))
	if adapter == nil {
		t.Fatal("NewCloudBuildAdapter returned nil")
	}
}
