package aws

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type mockACMAPI struct {
	requestFunc func(ctx context.Context, domain string) (string, error)
}

func (m *mockACMAPI) RequestCertificate(ctx context.Context, domain string) (string, error) {
	return m.requestFunc(ctx, domain)
}

type mockELBAPI struct {
	createFunc func(ctx context.Context, albARN, tgARN, certARN string) error
}

func (m *mockELBAPI) CreateHTTPSListener(ctx context.Context, albARN, tgARN, certARN string) error {
	return m.createFunc(ctx, albARN, tgARN, certARN)
}

func TestMapDomain_Success(t *testing.T) {
	var gotDomain, gotALB, gotTG, gotCert string

	acm := &mockACMAPI{
		requestFunc: func(_ context.Context, domain string) (string, error) {
			gotDomain = domain
			return "arn:aws:acm:us-east-1:123:certificate/abc", nil
		},
	}
	elb := &mockELBAPI{
		createFunc: func(_ context.Context, albARN, tgARN, certARN string) error {
			gotALB = albARN
			gotTG = tgARN
			gotCert = certARN
			return nil
		},
	}

	dm := &DomainMapper{ACM: acm, ELB: elb}
	certARN, err := dm.MapDomain(context.Background(), "arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/my-alb/abc", "arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/my-tg/def", "api.example.com")
	if err != nil {
		t.Fatalf("MapDomain() unexpected error: %v", err)
	}

	if certARN != "arn:aws:acm:us-east-1:123:certificate/abc" {
		t.Errorf("certARN = %q, want %q", certARN, "arn:aws:acm:us-east-1:123:certificate/abc")
	}
	if gotDomain != "api.example.com" {
		t.Errorf("domain = %q, want %q", gotDomain, "api.example.com")
	}
	if gotALB != "arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/my-alb/abc" {
		t.Errorf("albARN passed to ELB was incorrect")
	}
	if gotTG != "arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/my-tg/def" {
		t.Errorf("targetGroupARN passed to ELB was incorrect")
	}
	if gotCert != "arn:aws:acm:us-east-1:123:certificate/abc" {
		t.Errorf("certARN passed to ELB = %q, want the ACM cert ARN", gotCert)
	}
}

func TestMapDomain_InvalidDomain(t *testing.T) {
	acm := &mockACMAPI{
		requestFunc: func(_ context.Context, _ string) (string, error) {
			t.Fatal("RequestCertificate should not be called with invalid domain")
			return "", nil
		},
	}
	elb := &mockELBAPI{
		createFunc: func(_ context.Context, _, _, _ string) error {
			t.Fatal("CreateHTTPSListener should not be called with invalid domain")
			return nil
		},
	}

	dm := &DomainMapper{ACM: acm, ELB: elb}

	tests := []struct {
		name    string
		domain  string
		wantErr string
	}{
		{"empty domain", "", "must not be empty"},
		{"ip address", "10.0.0.1", "not an IP address"},
		{"wildcard", "*.example.com", "wildcards"},
		{"single label", "localhost", "at least two labels"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dm.MapDomain(context.Background(), "arn:alb", "arn:tg", tt.domain)
			if err == nil {
				t.Fatalf("MapDomain(%q) expected error, got nil", tt.domain)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("MapDomain(%q) error = %q, want substring %q", tt.domain, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestMapDomain_EmptyALBARN(t *testing.T) {
	dm := &DomainMapper{
		ACM: &mockACMAPI{requestFunc: func(_ context.Context, _ string) (string, error) { return "", nil }},
		ELB: &mockELBAPI{createFunc: func(_ context.Context, _, _, _ string) error { return nil }},
	}
	_, err := dm.MapDomain(context.Background(), "", "arn:tg", "api.example.com")
	if err == nil {
		t.Fatal("expected error for empty ALB ARN")
	}
	if !strings.Contains(err.Error(), "ALB ARN must not be empty") {
		t.Errorf("error = %q, want substring about empty ALB ARN", err.Error())
	}
}

func TestMapDomain_EmptyTargetGroupARN(t *testing.T) {
	dm := &DomainMapper{
		ACM: &mockACMAPI{requestFunc: func(_ context.Context, _ string) (string, error) { return "", nil }},
		ELB: &mockELBAPI{createFunc: func(_ context.Context, _, _, _ string) error { return nil }},
	}
	_, err := dm.MapDomain(context.Background(), "arn:alb", "", "api.example.com")
	if err == nil {
		t.Fatal("expected error for empty target group ARN")
	}
	if !strings.Contains(err.Error(), "target group ARN must not be empty") {
		t.Errorf("error = %q, want substring about empty target group ARN", err.Error())
	}
}

func TestMapDomain_ACMError(t *testing.T) {
	acm := &mockACMAPI{
		requestFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("throttled")
		},
	}
	elb := &mockELBAPI{
		createFunc: func(_ context.Context, _, _, _ string) error {
			t.Fatal("CreateHTTPSListener should not be called when ACM fails")
			return nil
		},
	}

	dm := &DomainMapper{ACM: acm, ELB: elb}
	_, err := dm.MapDomain(context.Background(), "arn:alb", "arn:tg", "api.example.com")
	if err == nil {
		t.Fatal("expected error when ACM fails")
	}
	if !strings.Contains(err.Error(), "throttled") {
		t.Errorf("error = %q, want substring %q", err.Error(), "throttled")
	}
}

func TestMapDomain_ELBError(t *testing.T) {
	acm := &mockACMAPI{
		requestFunc: func(_ context.Context, _ string) (string, error) {
			return "arn:cert", nil
		},
	}
	elb := &mockELBAPI{
		createFunc: func(_ context.Context, _, _, _ string) error {
			return errors.New("listener limit reached")
		},
	}

	dm := &DomainMapper{ACM: acm, ELB: elb}
	_, err := dm.MapDomain(context.Background(), "arn:alb", "arn:tg", "api.example.com")
	if err == nil {
		t.Fatal("expected error when ELB fails")
	}
	if !strings.Contains(err.Error(), "listener limit reached") {
		t.Errorf("error = %q, want substring %q", err.Error(), "listener limit reached")
	}
}

func TestDNSInstructions(t *testing.T) {
	instructions := DNSInstructions("api.example.com", "my-alb-1234.us-east-1.elb.amazonaws.com")
	if !strings.Contains(instructions, "api.example.com") {
		t.Error("DNSInstructions() should contain the domain name")
	}
	if !strings.Contains(instructions, "CNAME") {
		t.Error("DNSInstructions() should mention CNAME record type")
	}
	if !strings.Contains(instructions, "my-alb-1234.us-east-1.elb.amazonaws.com") {
		t.Error("DNSInstructions() should contain the ALB DNS name")
	}
}
