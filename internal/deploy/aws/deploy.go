// Package aws implements Amazon Web Services deployment orchestration.
package aws

import (
	"context"
	"io"

	"github.com/sirerun/mint/internal/deploy"
)

// RegistryProvisioner provisions ECR repositories.
type RegistryProvisioner interface {
	EnsureRepository(ctx context.Context, region, repoName string) (string, error)
}

// ImageBuilder builds container images via CodeBuild.
type ImageBuilder interface {
	BuildImage(ctx context.Context, sourceDir, imageURI string) (string, error)
}

// DeployServiceInfo holds information about a deployed ECS Fargate service.
type DeployServiceInfo struct {
	URL             string
	TaskARN         string
	PreviousTaskARN string
}

// DeployServiceOptions holds options for creating or updating an ECS Fargate service.
type DeployServiceOptions struct {
	Region       string
	ServiceName  string
	ImageURI     string
	Port         int
	EnvVars      map[string]string
	MinInstances int
	MaxInstances int
	Memory       string
	CPU          string
	AllowPublic  bool
	Args         []string
	VPCID        string
	ClusterARN   string
}

// ServiceDeployer deploys ECS Fargate services.
type ServiceDeployer interface {
	EnsureService(ctx context.Context, opts DeployServiceOptions) (*DeployServiceInfo, error)
}

// IAMConfigurator configures IAM roles and security groups for ECS tasks.
type IAMConfigurator interface {
	ConfigureIAM(ctx context.Context, region, serviceName string, allowPublic bool) error
}

// SecretProvisioner provisions secrets in AWS Secrets Manager.
type SecretProvisioner interface {
	EnsureSecrets(ctx context.Context, region, serviceName string, secrets map[string]string) ([]string, error)
}

// HealthProber checks service health.
type HealthProber interface {
	Check(ctx context.Context, url string) (*HealthProbeResult, error)
}

// HealthProbeResult holds the outcome of a health probe.
type HealthProbeResult struct {
	Healthy    bool
	StatusCode int
	Message    string
}

// Deployer orchestrates the full AWS deployment flow.
type Deployer struct {
	Registry RegistryProvisioner
	Builder  ImageBuilder
	ECS      ServiceDeployer
	IAM      IAMConfigurator
	Secrets  SecretProvisioner
	Health   HealthProber
	Stderr   io.Writer
}

// DeployInput holds all inputs for a deployment.
type DeployInput struct {
	Config      *deploy.DeployConfig
	MintVersion string
	SpecHash    string
	CommitSHA   string
}

// DeployOutput holds the result of a deployment.
type DeployOutput struct {
	ServiceURL string
	TaskARN    string
	ImageURI   string
	Healthy    bool
}
