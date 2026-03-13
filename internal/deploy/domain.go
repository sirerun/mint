package deploy

import (
	"fmt"
	"net"
	"strings"
)

// ValidateDomain checks that a domain name is syntactically valid.
// It rejects empty strings, IP addresses, wildcard domains, and names
// that do not contain at least one dot separating two labels.
func ValidateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain must not be empty")
	}
	if net.ParseIP(domain) != nil {
		return fmt.Errorf("domain %q must be a hostname, not an IP address", domain)
	}
	if strings.Contains(domain, "*") {
		return fmt.Errorf("domain %q must not contain wildcards", domain)
	}
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return fmt.Errorf("domain %q must contain at least two labels (e.g., api.example.com)", domain)
	}
	for _, label := range labels {
		if label == "" {
			return fmt.Errorf("domain %q contains an empty label", domain)
		}
		if len(label) > 63 {
			return fmt.Errorf("domain %q has a label longer than 63 characters", domain)
		}
		for i, c := range label {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
				return fmt.Errorf("domain %q contains invalid character %q in label %q", domain, string(c), label)
			}
			if c == '-' && (i == 0 || i == len(label)-1) {
				return fmt.Errorf("domain %q has a label %q that starts or ends with a hyphen", domain, label)
			}
		}
	}
	return nil
}
