package azure

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type mockCustomDomainAPI struct {
	addFunc func(ctx context.Context, resourceGroup, appName, domain string) error
}

func (m *mockCustomDomainAPI) AddCustomDomain(ctx context.Context, resourceGroup, appName, domain string) error {
	return m.addFunc(ctx, resourceGroup, appName, domain)
}

func TestMapDomain_Success(t *testing.T) {
	var gotRG, gotApp, gotDomain string
	api := &mockCustomDomainAPI{
		addFunc: func(_ context.Context, rg, app, domain string) error {
			gotRG = rg
			gotApp = app
			gotDomain = domain
			return nil
		},
	}

	dm := &DomainMapper{API: api}
	err := dm.MapDomain(context.Background(), "my-app", "my-rg", "api.example.com")
	if err != nil {
		t.Fatalf("MapDomain() unexpected error: %v", err)
	}
	if gotRG != "my-rg" {
		t.Errorf("resourceGroup = %q, want %q", gotRG, "my-rg")
	}
	if gotApp != "my-app" {
		t.Errorf("appName = %q, want %q", gotApp, "my-app")
	}
	if gotDomain != "api.example.com" {
		t.Errorf("domain = %q, want %q", gotDomain, "api.example.com")
	}
}

func TestMapDomain_InvalidDomain(t *testing.T) {
	api := &mockCustomDomainAPI{
		addFunc: func(_ context.Context, _, _, _ string) error {
			t.Fatal("AddCustomDomain should not be called with invalid domain")
			return nil
		},
	}

	dm := &DomainMapper{API: api}

	tests := []struct {
		name    string
		domain  string
		wantErr string
	}{
		{"empty domain", "", "must not be empty"},
		{"ip address", "172.16.0.1", "not an IP address"},
		{"wildcard", "*.example.com", "wildcards"},
		{"single label", "localhost", "at least two labels"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dm.MapDomain(context.Background(), "app", "rg", tt.domain)
			if err == nil {
				t.Fatalf("MapDomain(%q) expected error, got nil", tt.domain)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("MapDomain(%q) error = %q, want substring %q", tt.domain, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestMapDomain_EmptyAppName(t *testing.T) {
	api := &mockCustomDomainAPI{
		addFunc: func(_ context.Context, _, _, _ string) error {
			t.Fatal("AddCustomDomain should not be called with empty app name")
			return nil
		},
	}

	dm := &DomainMapper{API: api}
	err := dm.MapDomain(context.Background(), "", "rg", "api.example.com")
	if err == nil {
		t.Fatal("expected error for empty app name")
	}
	if !strings.Contains(err.Error(), "app name must not be empty") {
		t.Errorf("error = %q, want substring about empty app name", err.Error())
	}
}

func TestMapDomain_EmptyResourceGroup(t *testing.T) {
	api := &mockCustomDomainAPI{
		addFunc: func(_ context.Context, _, _, _ string) error {
			t.Fatal("AddCustomDomain should not be called with empty resource group")
			return nil
		},
	}

	dm := &DomainMapper{API: api}
	err := dm.MapDomain(context.Background(), "app", "", "api.example.com")
	if err == nil {
		t.Fatal("expected error for empty resource group")
	}
	if !strings.Contains(err.Error(), "resource group must not be empty") {
		t.Errorf("error = %q, want substring about empty resource group", err.Error())
	}
}

func TestMapDomain_APIError(t *testing.T) {
	api := &mockCustomDomainAPI{
		addFunc: func(_ context.Context, _, _, _ string) error {
			return errors.New("conflict: domain already bound")
		},
	}

	dm := &DomainMapper{API: api}
	err := dm.MapDomain(context.Background(), "app", "rg", "api.example.com")
	if err == nil {
		t.Fatal("expected error when API fails")
	}
	if !strings.Contains(err.Error(), "conflict: domain already bound") {
		t.Errorf("error = %q, want substring %q", err.Error(), "conflict: domain already bound")
	}
}

func TestDNSInstructions(t *testing.T) {
	instructions := DNSInstructions("api.example.com", "my-app.azurecontainerapps.io")
	if !strings.Contains(instructions, "api.example.com") {
		t.Error("DNSInstructions() should contain the domain name")
	}
	if !strings.Contains(instructions, "CNAME") {
		t.Error("DNSInstructions() should mention CNAME record type")
	}
	if !strings.Contains(instructions, "my-app.azurecontainerapps.io") {
		t.Error("DNSInstructions() should contain the app FQDN")
	}
	if !strings.Contains(instructions, "TXT") {
		t.Error("DNSInstructions() should mention TXT record for verification")
	}
	if !strings.Contains(instructions, "asuid.api.example.com") {
		t.Error("DNSInstructions() should contain the asuid TXT record name")
	}
}
