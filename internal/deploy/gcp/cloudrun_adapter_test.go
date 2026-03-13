package gcp

import (
	"testing"
)

// Compile-time interface checks.
var _ CloudRunClient = (*CloudRunServiceAdapter)(nil)
var _ StatusClient = (*CloudRunStatusAdapter)(nil)
var _ RevisionClient = (*CloudRunRevisionAdapter)(nil)
var _ TrafficClient = (*CloudRunTrafficAdapter)(nil)

func TestCloudRunServiceAdapterInterface(t *testing.T) {
	var _ CloudRunClient = (*CloudRunServiceAdapter)(nil)
}

func TestCloudRunStatusAdapterInterface(t *testing.T) {
	var _ StatusClient = (*CloudRunStatusAdapter)(nil)
}

func TestCloudRunRevisionAdapterInterface(t *testing.T) {
	var _ RevisionClient = (*CloudRunRevisionAdapter)(nil)
}

func TestCloudRunTrafficAdapterInterface(t *testing.T) {
	var _ TrafficClient = (*CloudRunTrafficAdapter)(nil)
}

func TestCloudRunAdapterConstructor(t *testing.T) {
	// Verify the constructor has the expected signature without calling it
	// (calling it requires real GCP credentials).
	var _ = NewCloudRunAdapter
}
