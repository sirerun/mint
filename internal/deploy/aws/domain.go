package aws

import (
	"context"
	"fmt"

	"github.com/sirerun/mint/internal/deploy"
)

// ACMAPI abstracts AWS Certificate Manager operations needed for custom domains.
type ACMAPI interface {
	// RequestCertificate requests a public TLS certificate for the domain.
	RequestCertificate(ctx context.Context, domain string) (certificateARN string, err error)
}

// ELBAPI abstracts the ELBv2 operations needed for adding an HTTPS listener.
type ELBAPI interface {
	// CreateHTTPSListener adds an HTTPS listener to an ALB using the given certificate.
	CreateHTTPSListener(ctx context.Context, loadBalancerARN, targetGroupARN, certificateARN string) error
}

// DomainMapper maps custom domains to AWS ALB services using ACM certificates.
type DomainMapper struct {
	ACM ACMAPI
	ELB ELBAPI
}

// MapDomain requests an ACM certificate for the domain and creates an HTTPS
// listener on the ALB. The caller must configure DNS (CNAME) to point the
// domain to the ALB's DNS name and validate the certificate via DNS.
func (m *DomainMapper) MapDomain(ctx context.Context, albARN, targetGroupARN, domain string) (string, error) {
	if err := deploy.ValidateDomain(domain); err != nil {
		return "", fmt.Errorf("invalid domain: %w", err)
	}
	if albARN == "" {
		return "", fmt.Errorf("ALB ARN must not be empty")
	}
	if targetGroupARN == "" {
		return "", fmt.Errorf("target group ARN must not be empty")
	}

	certARN, err := m.ACM.RequestCertificate(ctx, domain)
	if err != nil {
		return "", fmt.Errorf("request ACM certificate for %s: %w", domain, err)
	}

	if err := m.ELB.CreateHTTPSListener(ctx, albARN, targetGroupARN, certARN); err != nil {
		return "", fmt.Errorf("create HTTPS listener on %s: %w", albARN, err)
	}

	return certARN, nil
}

// DNSInstructions returns human-readable DNS setup instructions for an
// AWS ALB custom domain.
func DNSInstructions(domain, albDNSName string) string {
	return fmt.Sprintf(`Custom domain %q certificate requested.

To complete setup:

1. Add a DNS CNAME record to validate the ACM certificate:
   Check the ACM console for the validation CNAME record.

2. Add a DNS record to route traffic to the ALB:

   Type:  CNAME
   Name:  %s
   Value: %s

TLS certificates are provisioned automatically by ACM after DNS validation.
It may take up to 30 minutes for the certificate to be issued.`, domain, domain, albDNSName)
}
