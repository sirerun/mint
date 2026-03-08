// Package deploy defines shared configuration types for deployment.
package deploy

// DeployConfig holds the configuration for a deployment.
type DeployConfig struct {
	// ProjectID is the GCP project ID.
	ProjectID string

	// Region is the GCP region to deploy to.
	Region string

	// ServiceName is the Cloud Run service name.
	ServiceName string

	// Port is the container port to expose.
	Port int

	// EnvVars are environment variables to set on the Cloud Run service.
	EnvVars map[string]string

	// Secrets maps secret names to their mount paths or env var names.
	Secrets map[string]string

	// NoSourceRepo disables pushing source to Cloud Source Repositories.
	NoSourceRepo bool

	// AllowUnauthenticated allows unauthenticated access to the Cloud Run service.
	AllowUnauthenticated bool

	// MinInstances is the minimum number of instances to keep warm.
	MinInstances int

	// MaxInstances is the maximum number of instances to scale to.
	MaxInstances int

	// Memory is the memory limit for each instance (e.g., "512Mi").
	Memory string

	// CPU is the CPU limit for each instance (e.g., "1").
	CPU string

	// SourceDir is the local directory containing the source code to deploy.
	SourceDir string
}
