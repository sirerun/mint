// Package gcp implements Google Cloud Platform deployment orchestration.
package gcp

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/sirerun/mint/internal/deploy"
)

// High-level orchestrator interfaces. These wrap the lower-level SDK client
// interfaces (RegistryClient, BuildClient, etc.) defined in their respective
// files. Production code provides adapters; tests provide mocks.

// RegistryProvisioner provisions Artifact Registry repositories.
type RegistryProvisioner interface {
	EnsureRepository(ctx context.Context, projectID, region, repoName string) (string, error)
}

// ImageBuilder builds container images.
type ImageBuilder interface {
	BuildImage(ctx context.Context, sourceDir, imageURI string) (string, error)
}

// DeployServiceInfo holds information about a deployed Cloud Run service.
type DeployServiceInfo struct {
	URL              string
	RevisionName     string
	PreviousRevision string
}

// DeployServiceOptions holds options for creating or updating a Cloud Run service.
type DeployServiceOptions struct {
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
	Args                 []string
}

// ServiceDeployer deploys Cloud Run services.
type ServiceDeployer interface {
	EnsureService(ctx context.Context, opts DeployServiceOptions) (*DeployServiceInfo, error)
}

// IAMConfigurator configures IAM policies for Cloud Run services.
type IAMConfigurator interface {
	ConfigureIAMPolicy(ctx context.Context, projectID, region, serviceName string, allowUnauthenticated bool) error
}

// SecretProvisioner provisions secrets in Secret Manager.
type SecretProvisioner interface {
	EnsureSecrets(ctx context.Context, projectID, region, serviceName string, secrets map[string]string) error
}

// RepoProvisioner provisions source code repositories.
type RepoProvisioner interface {
	EnsureRepo(ctx context.Context, projectID, repoName string) (string, error)
}

// SourcePusher pushes source code to a remote repository.
type SourcePusher interface {
	Push(ctx context.Context, sourceDir, remoteURL string) error
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

// Deployer orchestrates the full deployment flow.
type Deployer struct {
	Registry   RegistryProvisioner
	Builder    ImageBuilder
	CloudRun   ServiceDeployer
	IAM        IAMConfigurator
	Secrets    SecretProvisioner
	SourceRepo RepoProvisioner
	Git        SourcePusher
	Health     HealthProber
	Stderr     io.Writer // for progress output
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
	ServiceURL   string
	RevisionName string
	ImageURI     string
	RepoURL      string // empty if source repo push was skipped
	Healthy      bool
}

// Deploy executes the full deployment orchestration sequence.
func (d *Deployer) Deploy(ctx context.Context, input DeployInput) (*DeployOutput, error) {
	cfg := input.Config
	out := &DeployOutput{}

	// Step 1: Provision Artifact Registry repository.
	d.log("Provisioning Artifact Registry repository...")
	repoPath, err := d.Registry.EnsureRepository(ctx, cfg.ProjectID, cfg.Region, cfg.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("registry: %w", err)
	}

	// Step 2: Build container image.
	d.log("Building container image...")
	tag := input.CommitSHA
	if tag == "" {
		tag = input.SpecHash
	}
	imageURI := path.Join(repoPath, cfg.ServiceName) + ":" + tag
	builtImage, err := d.Builder.BuildImage(ctx, cfg.SourceDir, imageURI)
	if err != nil {
		return nil, fmt.Errorf("build: %w", err)
	}
	out.ImageURI = builtImage

	// Step 3: Deploy to Cloud Run.
	d.log("Deploying to Cloud Run...")
	svcInfo, err := d.CloudRun.EnsureService(ctx, DeployServiceOptions{
		ProjectID:            cfg.ProjectID,
		Region:               cfg.Region,
		ServiceName:          cfg.ServiceName,
		ImageURI:             builtImage,
		Port:                 cfg.Port,
		EnvVars:              cfg.EnvVars,
		MinInstances:         cfg.MinInstances,
		MaxInstances:         cfg.MaxInstances,
		Memory:               cfg.Memory,
		CPU:                  cfg.CPU,
		AllowUnauthenticated: cfg.Public,
		Args:                 []string{"--transport", "sse"},
	})
	if err != nil {
		return nil, fmt.Errorf("cloud run: %w", err)
	}
	out.ServiceURL = svcInfo.URL
	out.RevisionName = svcInfo.RevisionName

	// Step 4: Configure IAM policy.
	d.log("Configuring IAM policy...")
	if err := d.IAM.ConfigureIAMPolicy(ctx, cfg.ProjectID, cfg.Region, cfg.ServiceName, cfg.Public); err != nil {
		return nil, fmt.Errorf("iam: %w", err)
	}

	// Step 5: Configure secrets if any are specified.
	if len(cfg.Secrets) > 0 {
		d.log("Configuring secrets...")
		secretMap := make(map[string]string, len(cfg.Secrets))
		for _, s := range cfg.Secrets {
			secretMap[s.EnvVar] = s.SecretName
		}
		if err := d.Secrets.EnsureSecrets(ctx, cfg.ProjectID, cfg.Region, cfg.ServiceName, secretMap); err != nil {
			return nil, fmt.Errorf("secrets: %w", err)
		}
	}

	// Step 6: Run health check.
	d.log("Running health check...")
	result, err := d.Health.Check(ctx, svcInfo.URL)
	if err != nil {
		d.log(fmt.Sprintf("Warning: health check error: %v", err))
	} else {
		out.Healthy = result.Healthy
		if !result.Healthy {
			if svcInfo.PreviousRevision != "" {
				d.log(fmt.Sprintf("Warning: service unhealthy (%s), previous revision: %s", result.Message, svcInfo.PreviousRevision))
			} else {
				d.log(fmt.Sprintf("Warning: service unhealthy (%s)", result.Message))
			}
		}
	}

	// Step 7: Push source to Cloud Source Repository if enabled.
	if !cfg.NoSourceRepo {
		d.log("Pushing source to Cloud Source Repository...")
		repoURL, err := d.SourceRepo.EnsureRepo(ctx, cfg.ProjectID, cfg.ServiceName)
		if err != nil {
			return nil, fmt.Errorf("source repo: %w", err)
		}
		if err := d.Git.Push(ctx, cfg.SourceDir, repoURL); err != nil {
			return nil, fmt.Errorf("git push: %w", err)
		}
		out.RepoURL = repoURL
	}

	// Step 8: Print summary.
	status := "healthy"
	if !out.Healthy {
		status = "unhealthy"
	}
	d.log(fmt.Sprintf("Deployment complete: URL=%s revision=%s status=%s", out.ServiceURL, out.RevisionName, status))

	return out, nil
}

func (d *Deployer) log(msg string) {
	if d.Stderr != nil {
		_, _ = fmt.Fprintln(d.Stderr, msg)
	}
}
