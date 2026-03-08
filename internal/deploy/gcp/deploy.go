// Package gcp implements Google Cloud Platform deployment orchestration.
package gcp

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/sirerun/mint/internal/deploy"
)

// Deployer orchestrates the full deployment flow.
type Deployer struct {
	Registry    RegistryClient
	Builder     BuildClient
	CloudRun    CloudRunClient
	IAM         IAMPolicyClient
	Secrets     SecretClient
	HealthCheck *HealthChecker
	SourceRepo  SourceRepoClient
	Git         GitClient
	Stderr      io.Writer // for progress output
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
	svcInfo, err := d.CloudRun.EnsureService(ctx, ServiceOptions{
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
		AllowUnauthenticated: cfg.AllowUnauthenticated,
	})
	if err != nil {
		return nil, fmt.Errorf("cloud run: %w", err)
	}
	out.ServiceURL = svcInfo.URL
	out.RevisionName = svcInfo.RevisionName

	// Step 4: Configure IAM policy.
	d.log("Configuring IAM policy...")
	if err := d.IAM.ConfigureIAMPolicy(ctx, cfg.ProjectID, cfg.Region, cfg.ServiceName, cfg.AllowUnauthenticated); err != nil {
		return nil, fmt.Errorf("iam: %w", err)
	}

	// Step 5: Configure secrets if any are specified.
	if len(cfg.Secrets) > 0 {
		d.log("Configuring secrets...")
		if err := d.Secrets.EnsureSecrets(ctx, cfg.ProjectID, cfg.Region, cfg.ServiceName, cfg.Secrets); err != nil {
			return nil, fmt.Errorf("secrets: %w", err)
		}
	}

	// Step 6: Run health check.
	d.log("Running health check...")
	result := d.HealthCheck.Check(ctx, svcInfo.URL)
	out.Healthy = result.Healthy
	if !result.Healthy {
		if svcInfo.PreviousRevision != "" {
			d.log(fmt.Sprintf("Warning: service unhealthy (%s), previous revision: %s", result.Message, svcInfo.PreviousRevision))
		} else {
			d.log(fmt.Sprintf("Warning: service unhealthy (%s)", result.Message))
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
