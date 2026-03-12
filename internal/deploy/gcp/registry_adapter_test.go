package gcp

import (
	"context"
	"testing"
)

// TestArtifactRegistryAdapterInterface is a compile-time check that
// ArtifactRegistryAdapter satisfies the RegistryClient interface.
func TestArtifactRegistryAdapterInterface(t *testing.T) {
	var _ RegistryClient = (*ArtifactRegistryAdapter)(nil)
}

// TestArtifactRegistryAdapterConstructor verifies that
// NewArtifactRegistryAdapter has the expected signature.
func TestArtifactRegistryAdapterConstructor(t *testing.T) {
	// We only verify the function signature compiles; calling it would
	// require real GCP credentials, so we assign it without invoking.
	var _ func(context.Context) (*ArtifactRegistryAdapter, error) = NewArtifactRegistryAdapter
}
