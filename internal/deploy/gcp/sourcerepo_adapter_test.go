package gcp

import "testing"

// TestSourceRepoAdapterImplementsInterface is a compile-time check that
// SourceRepoAdapter satisfies SourceRepoClient.
func TestSourceRepoAdapterImplementsInterface(t *testing.T) {
	var _ SourceRepoClient = (*SourceRepoAdapter)(nil)
}
