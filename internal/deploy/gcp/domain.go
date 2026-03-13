package gcp

import (
	"context"
	"fmt"

	"github.com/sirerun/mint/internal/deploy"
)

// DomainMappingAPI abstracts the Cloud Run domain mapping SDK calls.
type DomainMappingAPI interface {
	// CreateDomainMapping creates a domain mapping for a Cloud Run service.
	CreateDomainMapping(ctx context.Context, parent, serviceName, domain string) error
}

// DomainMapper maps custom domains to Cloud Run services with managed TLS.
type DomainMapper struct {
	API       DomainMappingAPI
	ProjectID string
	Region    string
}

// MapDomain creates a Cloud Run domain mapping with managed TLS for the given
// service. It validates the domain, calls the SDK, and prints DNS instructions.
func (m *DomainMapper) MapDomain(ctx context.Context, serviceName, domain string) error {
	if err := deploy.ValidateDomain(domain); err != nil {
		return fmt.Errorf("invalid domain: %w", err)
	}
	if serviceName == "" {
		return fmt.Errorf("service name must not be empty")
	}

	parent := fmt.Sprintf("projects/%s/locations/%s", m.ProjectID, m.Region)
	if err := m.API.CreateDomainMapping(ctx, parent, serviceName, domain); err != nil {
		return fmt.Errorf("create domain mapping for %s: %w", domain, err)
	}

	return nil
}

// DNSInstructions returns human-readable DNS setup instructions for a
// Cloud Run custom domain.
func DNSInstructions(domain string) string {
	return fmt.Sprintf(`Custom domain %q mapped successfully.

To complete setup, add the following DNS record:

  Type:  CNAME
  Name:  %s
  Value: ghs.googlehosted.com.

TLS certificates are provisioned automatically by Google-managed TLS.
It may take up to 24 hours for the certificate to be issued.`, domain, domain)
}
