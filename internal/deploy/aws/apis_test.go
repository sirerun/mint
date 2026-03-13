package aws

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestRequiredServicesContainsExpectedServices(t *testing.T) {
	expected := []string{
		"ecs",
		"ecr",
		"elasticloadbalancing",
		"codebuild",
		"secretsmanager",
		"iam",
		"sts",
	}

	if len(requiredServices) != len(expected) {
		t.Fatalf("expected %d required services, got %d", len(expected), len(requiredServices))
	}

	for i, svc := range expected {
		if requiredServices[i] != svc {
			t.Errorf("requiredServices[%d] = %q, want %q", i, requiredServices[i], svc)
		}
	}
}

func TestCheckRequiredServicesSignature(t *testing.T) {
	var fn func(ctx context.Context, checker ServiceChecker, region string) error = CheckRequiredServices
	_ = fn
}

type mockChecker struct {
	failing map[string]bool
}

func (m *mockChecker) CheckService(_ context.Context, service string) error {
	if m.failing[service] {
		return fmt.Errorf("service %s unavailable", service)
	}
	return nil
}

func TestCheckRequiredServicesAllAccessible(t *testing.T) {
	checker := &mockChecker{failing: map[string]bool{}}
	err := CheckRequiredServices(context.Background(), checker, "us-east-1")
	if err != nil {
		t.Fatalf("expected no error when all services accessible, got: %v", err)
	}
}

func TestCheckRequiredServicesSomeInaccessible(t *testing.T) {
	tests := []struct {
		name     string
		failing  []string
		region   string
		wantSvcs []string
	}{
		{
			name:     "single service unavailable",
			failing:  []string{"ecs"},
			region:   "us-west-2",
			wantSvcs: []string{"ecs"},
		},
		{
			name:     "multiple services unavailable",
			failing:  []string{"ecr", "codebuild", "sts"},
			region:   "eu-west-1",
			wantSvcs: []string{"ecr", "codebuild", "sts"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failing := make(map[string]bool)
			for _, s := range tt.failing {
				failing[s] = true
			}
			checker := &mockChecker{failing: failing}

			err := CheckRequiredServices(context.Background(), checker, tt.region)
			if err == nil {
				t.Fatal("expected error when services are inaccessible, got nil")
			}

			msg := err.Error()
			if !strings.Contains(msg, tt.region) {
				t.Errorf("error message should contain region %q, got: %s", tt.region, msg)
			}
			for _, svc := range tt.wantSvcs {
				if !strings.Contains(msg, svc) {
					t.Errorf("error message should list unavailable service %q, got: %s", svc, msg)
				}
			}
			if !strings.Contains(msg, "IAM permissions") {
				t.Errorf("error message should mention IAM permissions, got: %s", msg)
			}
		})
	}
}

func TestCheckRequiredServicesErrorMessageIsHelpful(t *testing.T) {
	failing := map[string]bool{"secretsmanager": true, "iam": true}
	checker := &mockChecker{failing: failing}

	err := CheckRequiredServices(context.Background(), checker, "ap-southeast-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()

	// Must contain region.
	if !strings.Contains(msg, "ap-southeast-1") {
		t.Errorf("error should contain region, got: %s", msg)
	}

	// Must list each unavailable service.
	for _, svc := range []string{"secretsmanager", "iam"} {
		if !strings.Contains(msg, svc) {
			t.Errorf("error should list %q, got: %s", svc, msg)
		}
	}

	// Must not list services that are accessible. Check that each accessible
	// service does not appear as its own line in the formatted list.
	for _, svc := range []string{"ecs", "ecr", "codebuild", "sts", "elasticloadbalancing"} {
		// Each unavailable service is listed on its own line with leading whitespace.
		if strings.Contains(msg, "\n  "+svc+"\n") || strings.HasSuffix(msg, "\n  "+svc) {
			t.Errorf("error should not list accessible service %q, got: %s", svc, msg)
		}
	}

	// Must provide actionable guidance.
	if !strings.Contains(msg, "IAM permissions") || !strings.Contains(msg, "region supports them") {
		t.Errorf("error should provide actionable guidance, got: %s", msg)
	}
}
