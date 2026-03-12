package gcp

import (
	"context"
	"fmt"
	"io"
)

// Bridge adapters convert the low-level SDK client interfaces into the
// high-level orchestrator interfaces consumed by Deployer.

// registryBridge implements RegistryProvisioner.
type registryBridge struct{ client RegistryClient }

func (b *registryBridge) EnsureRepository(ctx context.Context, projectID, region, repoName string) (string, error) {
	return EnsureRepository(ctx, b.client, projectID, region, repoName)
}

// NewRegistryBridge creates a RegistryProvisioner from a RegistryClient.
func NewRegistryBridge(client RegistryClient) RegistryProvisioner {
	return &registryBridge{client: client}
}

// buildBridge implements ImageBuilder.
type buildBridge struct {
	client    BuildClient
	projectID string
}

func (b *buildBridge) BuildImage(ctx context.Context, sourceDir, imageURI string) (string, error) {
	result, err := BuildImage(ctx, b.client, BuildConfig{
		SourceDir: sourceDir,
		ImageURI:  imageURI,
		ProjectID: b.projectID,
	})
	if err != nil {
		return "", err
	}
	return result.ImageURI, nil
}

// NewBuildBridge creates an ImageBuilder from a BuildClient.
func NewBuildBridge(client BuildClient, projectID string) ImageBuilder {
	return &buildBridge{client: client, projectID: projectID}
}

// cloudRunBridge implements ServiceDeployer.
type cloudRunBridge struct{ client CloudRunClient }

func (b *cloudRunBridge) EnsureService(ctx context.Context, opts DeployServiceOptions) (*DeployServiceInfo, error) {
	svc, err := EnsureService(ctx, b.client, &ServiceConfig{
		ProjectID:    opts.ProjectID,
		Region:       opts.Region,
		ServiceName:  opts.ServiceName,
		ImageURI:     opts.ImageURI,
		Port:         opts.Port,
		MaxInstances: opts.MaxInstances,
		MinInstances: opts.MinInstances,
		EnvVars:      opts.EnvVars,
	})
	if err != nil {
		return nil, err
	}
	return &DeployServiceInfo{
		URL:          svc.URL,
		RevisionName: svc.RevisionName,
	}, nil
}

// NewCloudRunBridge creates a ServiceDeployer from a CloudRunClient.
func NewCloudRunBridge(client CloudRunClient) ServiceDeployer {
	return &cloudRunBridge{client: client}
}

// iamBridge implements IAMConfigurator.
type iamBridge struct{ client IAMPolicyClient }

func (b *iamBridge) ConfigureIAMPolicy(ctx context.Context, projectID, region, serviceName string, allowUnauthenticated bool) error {
	fullName := ServiceFullName(projectID, region, serviceName)
	return ConfigureIAMPolicy(ctx, b.client, ServiceAccountConfig{
		ProjectID:   projectID,
		ServiceName: serviceName,
		Public:      allowUnauthenticated,
	}, fullName)
}

// NewIAMBridge creates an IAMConfigurator from an IAMPolicyClient.
func NewIAMBridge(client IAMPolicyClient) IAMConfigurator {
	return &iamBridge{client: client}
}

// secretsBridge implements SecretProvisioner.
type secretsBridge struct {
	client SecretClient
	stderr io.Writer
}

func (b *secretsBridge) EnsureSecrets(ctx context.Context, projectID, _, _ string, secrets map[string]string) error {
	mounts := make([]SecretMount, 0, len(secrets))
	for envVar, secretName := range secrets {
		mounts = append(mounts, SecretMount{EnvVar: envVar, SecretName: secretName})
	}
	_, err := EnsureSecrets(ctx, b.client, SecretConfig{
		ProjectID: projectID,
		Secrets:   mounts,
	}, b.stderr)
	return err
}

// NewSecretsBridge creates a SecretProvisioner from a SecretClient.
func NewSecretsBridge(client SecretClient, stderr io.Writer) SecretProvisioner {
	return &secretsBridge{client: client, stderr: stderr}
}

// sourceRepoBridge implements RepoProvisioner.
type sourceRepoBridge struct{ client SourceRepoClient }

func (b *sourceRepoBridge) EnsureRepo(ctx context.Context, projectID, repoName string) (string, error) {
	return EnsureSourceRepo(ctx, b.client, projectID, repoName)
}

// NewSourceRepoBridge creates a RepoProvisioner from a SourceRepoClient.
func NewSourceRepoBridge(client SourceRepoClient) RepoProvisioner {
	return &sourceRepoBridge{client: client}
}

// gitBridge implements SourcePusher.
type gitBridge struct{ client GitClient }

func (b *gitBridge) Push(ctx context.Context, sourceDir, remoteURL string) error {
	_, err := PushSource(ctx, b.client, SourcePushConfig{
		SourceDir: sourceDir,
	})
	return err
}

// NewGitBridge creates a SourcePusher from a GitClient.
func NewGitBridge(client GitClient) SourcePusher {
	return &gitBridge{client: client}
}

// healthBridge implements HealthProber.
type healthBridge struct{ checker *HealthChecker }

func (b *healthBridge) Check(ctx context.Context, url string) (*HealthProbeResult, error) {
	result, err := b.checker.Check(ctx, url)
	if err != nil {
		return nil, err
	}
	msg := result.Body
	if msg == "" {
		msg = fmt.Sprintf("status %d after %d attempts", result.StatusCode, result.Attempts)
	}
	return &HealthProbeResult{
		Healthy:    result.Healthy,
		StatusCode: result.StatusCode,
		Message:    msg,
	}, nil
}

// NewHealthBridge creates a HealthProber from a HealthChecker.
func NewHealthBridge(checker *HealthChecker) HealthProber {
	return &healthBridge{checker: checker}
}
