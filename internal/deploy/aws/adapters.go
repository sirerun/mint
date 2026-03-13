package aws

import (
	"context"
	"fmt"
	"io"
)

// Bridge adapters convert the low-level SDK client interfaces into the
// high-level orchestrator interfaces consumed by Deployer.

// registryBridge implements RegistryProvisioner.
type registryBridge struct{ client ECRClient }

func (b *registryBridge) EnsureRepository(ctx context.Context, region, repoName string) (string, error) {
	return EnsureRepository(ctx, b.client, region, repoName)
}

// NewRegistryBridge creates a RegistryProvisioner from an ECRClient.
func NewRegistryBridge(client ECRClient) RegistryProvisioner {
	return &registryBridge{client: client}
}

// buildBridge implements ImageBuilder.
type buildBridge struct {
	client      CodeBuildClient
	projectName string
}

func (b *buildBridge) BuildImage(ctx context.Context, sourceDir, imageURI string) (string, error) {
	result, err := BuildImage(ctx, b.client, b.projectName, sourceDir, imageURI)
	if err != nil {
		return "", err
	}
	return result.ImageURI, nil
}

// NewBuildBridge creates an ImageBuilder from a CodeBuildClient.
func NewBuildBridge(client CodeBuildClient, projectName string) ImageBuilder {
	return &buildBridge{client: client, projectName: projectName}
}

// ecsBridge implements ServiceDeployer.
type ecsBridge struct{ client ECSClient }

func (b *ecsBridge) EnsureService(ctx context.Context, opts DeployServiceOptions) (*DeployServiceInfo, error) {
	svc, err := EnsureService(ctx, b.client, &EnsureServiceOptions{
		ClusterName: opts.ClusterARN,
		ServiceName: opts.ServiceName,
		TaskDefinitionInput: &RegisterTaskDefinitionInput{
			Family:        opts.ServiceName,
			ImageURI:      opts.ImageURI,
			ContainerName: opts.ServiceName,
			Port:          opts.Port,
			CPU:           opts.CPU,
			Memory:        opts.Memory,
			EnvVars:       opts.EnvVars,
			Args:          opts.Args,
		},
		DesiredCount:   opts.MinInstances,
		AssignPublicIP: opts.AllowPublic,
	})
	if err != nil {
		return nil, err
	}
	return &DeployServiceInfo{
		URL:     svc.ServiceARN,
		TaskARN: svc.TaskDefinitionARN,
	}, nil
}

// NewECSBridge creates a ServiceDeployer from an ECSClient.
func NewECSBridge(client ECSClient) ServiceDeployer {
	return &ecsBridge{client: client}
}

// iamBridge implements IAMConfigurator.
type iamBridge struct{ client IAMClient }

func (b *iamBridge) ConfigureIAM(ctx context.Context, region, serviceName string, allowPublic bool) error {
	_, err := EnsureTaskRoles(ctx, b.client, serviceName)
	return err
}

// NewIAMBridge creates an IAMConfigurator from an IAMClient.
func NewIAMBridge(client IAMClient) IAMConfigurator {
	return &iamBridge{client: client}
}

// secretsBridge implements SecretProvisioner.
type secretsBridge struct {
	client SecretsClient
	stderr io.Writer
}

func (b *secretsBridge) EnsureSecrets(ctx context.Context, region, serviceName string, secrets map[string]string) ([]string, error) {
	return EnsureSecrets(ctx, b.client, secrets, b.stderr)
}

// NewSecretsBridge creates a SecretProvisioner from a SecretsClient.
func NewSecretsBridge(client SecretsClient, stderr io.Writer) SecretProvisioner {
	return &secretsBridge{client: client, stderr: stderr}
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
