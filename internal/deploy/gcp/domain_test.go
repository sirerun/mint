package gcp

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type mockDomainMappingAPI struct {
	createFunc func(ctx context.Context, parent, serviceName, domain string) error
}

func (m *mockDomainMappingAPI) CreateDomainMapping(ctx context.Context, parent, serviceName, domain string) error {
	return m.createFunc(ctx, parent, serviceName, domain)
}

func TestMapDomain_Success(t *testing.T) {
	var gotParent, gotService, gotDomain string
	api := &mockDomainMappingAPI{
		createFunc: func(_ context.Context, parent, serviceName, domain string) error {
			gotParent = parent
			gotService = serviceName
			gotDomain = domain
			return nil
		},
	}

	dm := &DomainMapper{
		API:       api,
		ProjectID: "my-project",
		Region:    "us-central1",
	}

	err := dm.MapDomain(context.Background(), "my-service", "api.example.com")
	if err != nil {
		t.Fatalf("MapDomain() unexpected error: %v", err)
	}
	if gotParent != "projects/my-project/locations/us-central1" {
		t.Errorf("parent = %q, want %q", gotParent, "projects/my-project/locations/us-central1")
	}
	if gotService != "my-service" {
		t.Errorf("serviceName = %q, want %q", gotService, "my-service")
	}
	if gotDomain != "api.example.com" {
		t.Errorf("domain = %q, want %q", gotDomain, "api.example.com")
	}
}

func TestMapDomain_InvalidDomain(t *testing.T) {
	api := &mockDomainMappingAPI{
		createFunc: func(_ context.Context, _, _, _ string) error {
			t.Fatal("CreateDomainMapping should not be called with invalid domain")
			return nil
		},
	}

	dm := &DomainMapper{API: api, ProjectID: "p", Region: "r"}

	tests := []struct {
		name    string
		domain  string
		wantErr string
	}{
		{"empty domain", "", "must not be empty"},
		{"ip address", "192.168.1.1", "not an IP address"},
		{"wildcard", "*.example.com", "wildcards"},
		{"single label", "localhost", "at least two labels"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dm.MapDomain(context.Background(), "svc", tt.domain)
			if err == nil {
				t.Fatalf("MapDomain(%q) expected error, got nil", tt.domain)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("MapDomain(%q) error = %q, want substring %q", tt.domain, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestMapDomain_EmptyServiceName(t *testing.T) {
	api := &mockDomainMappingAPI{
		createFunc: func(_ context.Context, _, _, _ string) error {
			t.Fatal("CreateDomainMapping should not be called with empty service name")
			return nil
		},
	}

	dm := &DomainMapper{API: api, ProjectID: "p", Region: "r"}
	err := dm.MapDomain(context.Background(), "", "api.example.com")
	if err == nil {
		t.Fatal("MapDomain() expected error for empty service name, got nil")
	}
	if !strings.Contains(err.Error(), "service name must not be empty") {
		t.Errorf("error = %q, want substring %q", err.Error(), "service name must not be empty")
	}
}

func TestMapDomain_APIError(t *testing.T) {
	api := &mockDomainMappingAPI{
		createFunc: func(_ context.Context, _, _, _ string) error {
			return errors.New("permission denied")
		},
	}

	dm := &DomainMapper{API: api, ProjectID: "p", Region: "r"}
	err := dm.MapDomain(context.Background(), "svc", "api.example.com")
	if err == nil {
		t.Fatal("MapDomain() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error = %q, want substring %q", err.Error(), "permission denied")
	}
}

func TestDNSInstructions(t *testing.T) {
	instructions := DNSInstructions("api.example.com")
	if !strings.Contains(instructions, "api.example.com") {
		t.Error("DNSInstructions() should contain the domain name")
	}
	if !strings.Contains(instructions, "CNAME") {
		t.Error("DNSInstructions() should mention CNAME record type")
	}
	if !strings.Contains(instructions, "ghs.googlehosted.com") {
		t.Error("DNSInstructions() should mention the Google hosted target")
	}
}
