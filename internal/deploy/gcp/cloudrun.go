package gcp

import "context"

// ServiceInfo holds information about a deployed Cloud Run service.
type ServiceInfo struct {
	// URL is the service URL.
	URL string

	// RevisionName is the name of the latest revision.
	RevisionName string

	// PreviousRevision is the name of the previous revision, if any.
	PreviousRevision string
}

// CloudRunClient manages Cloud Run services.
type CloudRunClient interface {
	// EnsureService creates or updates a Cloud Run service with the given
	// configuration. It returns information about the deployed service.
	EnsureService(ctx context.Context, opts ServiceOptions) (*ServiceInfo, error)
}

// ServiceOptions holds options for creating or updating a Cloud Run service.
type ServiceOptions struct {
	ProjectID            string
	Region               string
	ServiceName          string
	ImageURI             string
	Port                 int
	EnvVars              map[string]string
	MinInstances         int
	MaxInstances         int
	Memory               string
	CPU                  string
	AllowUnauthenticated bool
}
