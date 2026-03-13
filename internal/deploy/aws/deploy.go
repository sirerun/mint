// Package aws implements Amazon Web Services deployment orchestration.
package aws

import (
	"context"
	"fmt"
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

// Deploy executes the full AWS deployment orchestration sequence.
func (d *Deployer) Deploy(ctx context.Context, input DeployInput) (*DeployOutput, error) {
	cfg := input.Config
	out := &DeployOutput{}

	// Step 1: Provision ECR repository.
	d.log("Provisioning ECR repository...")
	repoURI, err := d.Registry.EnsureRepository(ctx, cfg.Region, cfg.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("registry: %w", err)
	}

	// Step 2: Build container image.
	d.log("Building container image...")
	tag := input.CommitSHA
	if tag == "" {
		tag = input.SpecHash
	}
	imageURI := repoURI + ":" + tag
	builtImage, err := d.Builder.BuildImage(ctx, cfg.SourceDir, imageURI)
	if err != nil {
		return nil, fmt.Errorf("build: %w", err)
	}
	out.ImageURI = builtImage

	// Step 3: Configure IAM roles.
	d.log("Configuring IAM roles...")
	if err := d.IAM.ConfigureIAM(ctx, cfg.Region, cfg.ServiceName, cfg.Public); err != nil {
		return nil, fmt.Errorf("iam: %w", err)
	}

	// Step 4: Configure secrets if any are specified.
	if len(cfg.Secrets) > 0 {
		d.log("Configuring secrets...")
		secretMap := make(map[string]string, len(cfg.Secrets))
		for _, s := range cfg.Secrets {
			secretMap[s.EnvVar] = s.SecretName
		}
		if _, err := d.Secrets.EnsureSecrets(ctx, cfg.Region, cfg.ServiceName, secretMap); err != nil {
			return nil, fmt.Errorf("secrets: %w", err)
		}
	}

	// Step 5: Deploy to ECS Fargate.
	d.log("Deploying to ECS Fargate...")
	svcInfo, err := d.ECS.EnsureService(ctx, DeployServiceOptions{
		Region:       cfg.Region,
		ServiceName:  cfg.ServiceName,
		ImageURI:     builtImage,
		Port:         cfg.Port,
		EnvVars:      cfg.EnvVars,
		MinInstances: cfg.MinInstances,
		MaxInstances: cfg.MaxInstances,
		Memory:       cfg.Memory,
		CPU:          cfg.CPU,
		AllowPublic:  cfg.Public,
		Args:         []string{"--transport", "sse"},
	})
	if err != nil {
		return nil, fmt.Errorf("ecs: %w", err)
	}
	out.ServiceURL = svcInfo.URL
	out.TaskARN = svcInfo.TaskARN

	// Step 6: Health check.
	d.log("Running health check...")
	result, err := d.Health.Check(ctx, svcInfo.URL)
	if err != nil {
		d.log(fmt.Sprintf("Warning: health check error: %v", err))
	} else {
		out.Healthy = result.Healthy
		if !result.Healthy {
			if svcInfo.PreviousTaskARN != "" {
				d.log(fmt.Sprintf("Warning: service unhealthy (%s), previous task: %s", result.Message, svcInfo.PreviousTaskARN))
			} else {
				d.log(fmt.Sprintf("Warning: service unhealthy (%s)", result.Message))
			}
		}
	}

	// Step 7: Print summary.
	status := "healthy"
	if !out.Healthy {
		status = "unhealthy"
	}
	d.log(fmt.Sprintf("Deployment complete: URL=%s task=%s status=%s", out.ServiceURL, out.TaskARN, status))

	return out, nil
}

func (d *Deployer) log(msg string) {
	if d.Stderr != nil {
		_, _ = fmt.Fprintln(d.Stderr, msg)
	}
}
