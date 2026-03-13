package deploy

import (
	"strings"
	"testing"
)

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr string
	}{
		{"valid domain", "api.example.com", ""},
		{"valid two labels", "example.com", ""},
		{"valid subdomain", "a.b.c.example.com", ""},
		{"empty", "", "must not be empty"},
		{"ipv4", "192.168.1.1", "not an IP address"},
		{"ipv6", "::1", "not an IP address"},
		{"wildcard prefix", "*.example.com", "wildcards"},
		{"wildcard middle", "api.*.com", "wildcards"},
		{"single label", "localhost", "at least two labels"},
		{"trailing dot empty label", "example.com.", "empty label"},
		{"leading dot empty label", ".example.com", "empty label"},
		{"double dot", "example..com", "empty label"},
		{"label too long", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmn.com", "longer than 63"},
		{"hyphen start", "-example.com", "starts or ends with a hyphen"},
		{"hyphen end", "example-.com", "starts or ends with a hyphen"},
		{"underscore", "ex_ample.com", "invalid character"},
		{"space", "ex ample.com", "invalid character"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDomain(tt.domain)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateDomain(%q) unexpected error: %v", tt.domain, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateDomain(%q) expected error containing %q, got nil", tt.domain, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ValidateDomain(%q) error = %q, want substring %q", tt.domain, err.Error(), tt.wantErr)
			}
		})
	}
}
