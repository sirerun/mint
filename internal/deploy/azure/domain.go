package azure

import (
	"context"
	"fmt"

	"github.com/sirerun/mint/internal/deploy"
)

// CustomDomainAPI abstracts Azure Container Apps custom domain SDK calls.
type CustomDomainAPI interface {
	// AddCustomDomain adds a custom domain with a managed certificate to a Container App.
	AddCustomDomain(ctx context.Context, resourceGroup, appName, domain string) error
}

// DomainMapper maps custom domains to Azure Container Apps with managed certificates.
type DomainMapper struct {
	API CustomDomainAPI
}

// MapDomain adds a custom domain with a managed TLS certificate to the
// specified Container App. It validates the domain, calls the SDK, and
// returns nil on success.
func (m *DomainMapper) MapDomain(ctx context.Context, appName, resourceGroup, domain string) error {
	if err := deploy.ValidateDomain(domain); err != nil {
		return fmt.Errorf("invalid domain: %w", err)
	}
	if appName == "" {
		return fmt.Errorf("app name must not be empty")
	}
	if resourceGroup == "" {
		return fmt.Errorf("resource group must not be empty")
	}

	if err := m.API.AddCustomDomain(ctx, resourceGroup, appName, domain); err != nil {
		return fmt.Errorf("add custom domain %s to %s: %w", domain, appName, err)
	}

	return nil
}

// DNSInstructions returns human-readable DNS setup instructions for an
// Azure Container Apps custom domain.
func DNSInstructions(domain, appFQDN string) string {
	return fmt.Sprintf(`Custom domain %q configured successfully.

To complete setup, add the following DNS records:

1. Verify domain ownership:

   Type:  TXT
   Name:  asuid.%s
   Value: (copy from Azure Portal or CLI)

2. Route traffic to the Container App:

   Type:  CNAME
   Name:  %s
   Value: %s

Managed TLS certificates are provisioned automatically by Azure.
It may take up to 24 hours for the certificate to be issued.`, domain, domain, domain, appFQDN)
}
