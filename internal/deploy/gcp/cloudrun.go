package gcp

import (
	"context"
	"errors"
	"fmt"
)

// CloudRunClient abstracts Cloud Run operations.
type CloudRunClient interface {
	GetService(ctx context.Context, name string) (*Service, error)
	CreateService(ctx context.Context, config *ServiceConfig) (*Service, error)
	UpdateService(ctx context.Context, config *ServiceConfig) (*Service, error)
}

// ServiceConfig describes a Cloud Run service to create or update.
type ServiceConfig struct {
	ProjectID    string
	Region       string
	ServiceName  string
	ImageURI     string
	Port         int // default 8080
	Timeout      int // request timeout in seconds
	MaxInstances int
	MinInstances int
	CPUAlways    bool              // allocate CPU even when idle (for SSE)
	EnvVars      map[string]string // environment variables
	Labels       map[string]string // service labels
}

// Service represents a deployed Cloud Run service.
type Service struct {
	Name         string
	URL          string
	RevisionName string
	Status       string
}

// ErrNotFound indicates a Cloud Run service was not found.
var ErrNotFound = errors.New("service not found")

// ServiceFullName constructs the full Cloud Run service resource name.
func ServiceFullName(projectID, region, serviceName string) string {
	return fmt.Sprintf("projects/%s/locations/%s/services/%s", projectID, region, serviceName)
}

// validateConfig checks that required fields are present in the service config.
func validateConfig(config *ServiceConfig) error {
	if config == nil {
		return errors.New("config must not be nil")
	}
	if config.ProjectID == "" {
		return errors.New("config.ProjectID must not be empty")
	}
	if config.Region == "" {
		return errors.New("config.Region must not be empty")
	}
	if config.ServiceName == "" {
		return errors.New("config.ServiceName must not be empty")
	}
	if config.ImageURI == "" {
		return errors.New("config.ImageURI must not be empty")
	}
	return nil
}

// EnsureService creates or updates a Cloud Run service. It tries to get the
// service first; if it does not exist (ErrNotFound), it creates it. If it
// already exists, it updates it with the new configuration.
func EnsureService(ctx context.Context, client CloudRunClient, config *ServiceConfig) (*Service, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid service config: %w", err)
	}

	fullName := ServiceFullName(config.ProjectID, config.Region, config.ServiceName)

	_, err := client.GetService(ctx, fullName)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("get service %s: %w", fullName, err)
		}
		// Service does not exist, create it.
		svc, createErr := client.CreateService(ctx, config)
		if createErr != nil {
			return nil, fmt.Errorf("create service %s: %w", fullName, createErr)
		}
		return svc, nil
	}

	// Service exists, update it.
	svc, err := client.UpdateService(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("update service %s: %w", fullName, err)
	}
	return svc, nil
}
