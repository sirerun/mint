package azure

import (
	"context"
	"errors"
)

// ErrAppNotFound indicates that a Container App was not found.
var ErrAppNotFound = errors.New("containerapp: app not found")

// ContainerAppClient abstracts Azure Container Apps operations.
type ContainerAppClient interface {
	// CreateOrUpdateApp creates or updates a Container App.
	CreateOrUpdateApp(ctx context.Context, input *CreateOrUpdateAppInput) (*ContainerApp, error)

	// GetApp returns metadata for a Container App.
	// Returns ErrAppNotFound if the app does not exist.
	GetApp(ctx context.Context, resourceGroup, appName string) (*ContainerApp, error)

	// ListRevisions returns all revisions for a Container App.
	ListRevisions(ctx context.Context, resourceGroup, appName string) ([]Revision, error)

	// UpdateTrafficSplit updates the traffic distribution across revisions.
	UpdateTrafficSplit(ctx context.Context, resourceGroup, appName string, traffic []TrafficWeight) error
}

// CreateOrUpdateAppInput holds parameters for creating or updating a Container App.
type CreateOrUpdateAppInput struct {
	ResourceGroup string
	AppName       string
	Region        string
	EnvironmentID string
	ImageURI      string
	Port          int
	EnvVars       map[string]string
	SecretRefs    []SecretRef
	MinInstances  int
	MaxInstances  int
	Memory        string
	CPU           string
	Args          []string
	Ingress       *IngressConfig
}

// IngressConfig holds ingress configuration for a Container App.
type IngressConfig struct {
	External   bool
	TargetPort int
}

// ContainerApp represents an Azure Container App.
type ContainerApp struct {
	ID                string
	Name              string
	FQDN              string
	LatestRevision    string
	ProvisioningState string
}

// Revision represents a Container App revision.
type Revision struct {
	Name          string
	Active        bool
	TrafficWeight int
	CreatedTime   string
}

// TrafficWeight assigns a traffic percentage to a revision.
type TrafficWeight struct {
	RevisionName string
	Weight       int
	Latest       bool
}

// SecretRef references a secret for use in environment variables.
type SecretRef struct {
	Name     string
	KeyVault string
	Identity string
}
