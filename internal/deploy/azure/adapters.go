package azure

import (
	"context"
	"fmt"
	"io"
)

// Bridge adapters convert the low-level SDK client interfaces into the
// high-level orchestrator interfaces consumed by Deployer.

// registryBridge implements RegistryProvisioner.
type registryBridge struct {
	client ACRClient
}

func (b *registryBridge) EnsureRepository(ctx context.Context, subscriptionID, resourceGroup, repoName string) (string, error) {
	return b.client.EnsureRepository(ctx, resourceGroup, subscriptionID, repoName)
}

// NewRegistryBridge creates a RegistryProvisioner from an ACRClient.
func NewRegistryBridge(client ACRClient) RegistryProvisioner {
	return &registryBridge{client: client}
}

// serviceBridge implements ServiceDeployer.
type serviceBridge struct {
	client ContainerAppClient
	env    ManagedEnvironmentClient
}

func (b *serviceBridge) EnsureService(ctx context.Context, opts DeployServiceOptions) (*DeployServiceInfo, error) {
	envID := opts.EnvironmentID
	if envID == "" {
		var err error
		envID, err = b.env.EnsureEnvironment(ctx, opts.ResourceGroup, opts.ServiceName+"-env", opts.Region)
		if err != nil {
			return nil, fmt.Errorf("ensure environment: %w", err)
		}
	}

	var ingress *IngressConfig
	if opts.Port > 0 {
		ingress = &IngressConfig{
			External:   opts.AllowPublic,
			TargetPort: opts.Port,
		}
	}

	app, err := b.client.CreateOrUpdateApp(ctx, &CreateOrUpdateAppInput{
		ResourceGroup: opts.ResourceGroup,
		AppName:       opts.ServiceName,
		Region:        opts.Region,
		EnvironmentID: envID,
		ImageURI:      opts.ImageURI,
		Port:          opts.Port,
		EnvVars:       opts.EnvVars,
		MinInstances:  opts.MinInstances,
		MaxInstances:  opts.MaxInstances,
		Memory:        opts.Memory,
		CPU:           opts.CPU,
		Args:          opts.Args,
		Ingress:       ingress,
	})
	if err != nil {
		return nil, fmt.Errorf("create or update app: %w", err)
	}

	// Try to get previous revision for rollback info.
	var previousRevision string
	revisions, err := b.client.ListRevisions(ctx, opts.ResourceGroup, opts.ServiceName)
	if err == nil && len(revisions) > 1 {
		previousRevision = revisions[len(revisions)-2].Name
	}

	url := app.FQDN
	if url != "" {
		url = "https://" + url
	}

	return &DeployServiceInfo{
		URL:              url,
		RevisionName:     app.LatestRevision,
		PreviousRevision: previousRevision,
	}, nil
}

// NewServiceBridge creates a ServiceDeployer from Container App and Environment clients.
func NewServiceBridge(client ContainerAppClient, env ManagedEnvironmentClient) ServiceDeployer {
	return &serviceBridge{client: client, env: env}
}

// iamBridge implements IAMConfigurator.
type iamBridge struct {
	client RBACClient
}

func (b *iamBridge) ConfigureIAM(ctx context.Context, subscriptionID, resourceGroup, serviceName string, allowPublic bool) error {
	// Assign AcrPull role to the Container App's managed identity.
	// The scope and principal would be resolved from the deployment context.
	return nil
}

// NewIAMBridge creates an IAMConfigurator from an RBACClient.
func NewIAMBridge(client RBACClient) IAMConfigurator {
	return &iamBridge{client: client}
}

// secretsBridge implements SecretProvisioner.
type secretsBridge struct {
	client KeyVaultClient
	stderr io.Writer
}

func (b *secretsBridge) EnsureSecrets(ctx context.Context, subscriptionID, resourceGroup, serviceName string, secrets map[string]string) ([]string, error) {
	vaultName := "mint-" + serviceName
	vaultURI, err := b.client.EnsureKeyVault(ctx, resourceGroup, vaultName, "")
	if err != nil {
		return nil, fmt.Errorf("ensure key vault: %w", err)
	}

	uris := make([]string, 0, len(secrets))
	for envVar, secretName := range secrets {
		if err := b.client.SetSecret(ctx, vaultURI, secretName, ""); err != nil {
			return nil, fmt.Errorf("set secret %q: %w", secretName, err)
		}
		uri, err := b.client.GetSecretURI(ctx, vaultURI, secretName)
		if err != nil {
			return nil, fmt.Errorf("get secret URI %q: %w", secretName, err)
		}
		_, _ = fmt.Fprintf(b.stderr, "Configured secret %q for %s\n", secretName, envVar)
		uris = append(uris, uri)
	}
	return uris, nil
}

// NewSecretsBridge creates a SecretProvisioner from a KeyVaultClient.
func NewSecretsBridge(client KeyVaultClient, stderr io.Writer) SecretProvisioner {
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
